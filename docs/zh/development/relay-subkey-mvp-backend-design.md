# 自托管 Relay + Sub-Key 共享容量 MVP：后端与数据设计

## 背景

AxonHub 当前已经具备自托管 Relay MVP 所需的基础骨架：

- `channels` 已经承载上游 provider 凭证、模型能力、路由设置与健康状态。
- `api_keys` 已经承载下游 API 鉴权，并天然带有 `project_id` 隔离能力。
- `requests`、`request_executions`、`usage_logs` 已经覆盖请求主记录、上游执行记录、token 与成本记录。
- `provider_quota_status` 已经为 Claude Code / Codex 这类 provider 提供额度状态探测入口。

本 MVP 不做“上游 Key 直接转售”，而是让运营方部署一套 AxonHub Relay，对外发放自己的 Sub-Key。多个 Sub-Key 共享一组上游渠道容量池，但额度、余额、停用状态、风控限制仍按 Sub-Key 独立管理。

## MVP 范围

1. 仅支持自托管 Relay 场景，不引入去中心化撮合或多卖家市场。
2. 运营侧手动配置产品、上游渠道、Sub-Key、充值与冻结操作。
3. 下游继续走 AxonHub 现有 OpenAI / Anthropic / Codex 兼容入口，不新增协议层分支。
4. 计费先支持“预付余额 + 日级硬配额 + 月度成本 preflight guard + settlement hard cap”模型；支付网关、发票、代理分润留到后续阶段。
5. 共享容量通过“一个产品绑定多个上游渠道”实现，而不是把某个上游 API Key 直接暴露给终端用户。

## 设计原则

- **请求链路复用优先**：`requests` / `request_executions` / `usage_logs` 继续作为真实请求事实来源，不重复造主流水。
- **认证与业务解耦**：对外鉴权继续使用 `api_keys`，Relay 业务能力放到独立表，避免污染现有通用 Key 语义。
- **账务不可变**：余额变更全部落到不可变流水表，禁止直接覆盖消费结果。
- **路由基于池而不是单 Key**：Sub-Key 绑定产品，产品绑定共享上游渠道池，路由时再做健康度与额度过滤。
- **同步链路分层限额**：请求入口只查状态、余额、日级硬限额与月度 preflight guard；月度成本 hard cap 在 settlement 事务内通过同 Key 串行化/原子检查兜底，复杂统计和报表走异步汇总表。

## 复用现有表

| 表                                                      | 在 MVP 中的角色                           | 处理方式                                                |
| ------------------------------------------------------- | ----------------------------------------- | ------------------------------------------------------- |
| `projects`                                              | 买方租户隔离、管理台作用域                | 保持不变，继续作为项目级隔离主键                        |
| `users`                                                 | 管理员 / 运营人员账号                     | 保持不变                                                |
| `api_keys`                                              | 对外发放给终端用户的鉴权凭证              | 保持不变，通过新表 `relay_keys.api_key_id` 绑定业务属性 |
| `channels`                                              | 上游 provider 账号 / sub-key / OAuth 渠道 | 保持不变，继续承载真实上游凭证                          |
| `provider_quota_status`                                 | 上游额度轮询状态                          | 保持不变，作为路由过滤条件之一                          |
| `requests`                                              | 下游请求主记录                            | 保持不变，作为消费与审计锚点                            |
| `request_executions`                                    | 实际命中的上游渠道与执行结果              | 保持不变，作为路由结果事实表                            |
| `usage_logs`                                            | 实际 token / upstream cost 事实表         | 保持不变，继续记录上游成本                              |
| `channel_model_prices` / `channel_model_price_versions` | 上游成本基准                              | 保持不变，用于计算 upstream cost                        |
| `system`                                                | Relay 开关、默认策略                      | 可新增系统配置项，但不新增独立系统表                    |

### 为什么不上来就新增“上游账号表”

当前 `channels` 已经承担了“一个上游账号 / 一个上游 sub-key / 一个 provider 凭证”的职责，并且已经与请求执行、额度状态、模型价格、自动禁用等机制集成。MVP 阶段继续复用 `channels`，可以避免：

- 再维护一套与 `channels` 平行的凭证生命周期；
- 在请求链路里引入双重路由模型；
- 为 quota、health、fallback 逻辑做两遍适配。

## 新增表清单

MVP 建议新增 6 张表，全部放在 `internal/ent/schema/` 下，由 Ent 管理迁移。

### 1. `relay_products`

建议 schema 文件：`internal/ent/schema/relay_product.go`

用途：定义对外销售的“共享容量产品”，一个产品对应一类 provider 能力与计费策略。

核心字段：

- `id`
- `code`：产品编码，唯一索引，用于后台引用与外部展示
- `name`
- `provider_type`：如 `claudecode`、`codex`、`openai_compatible`
- `access_mode`：固定为 `shared_capacity`
- `billing_mode`：`prepaid` / `quota_only`
- `status`：`draft` / `active` / `archived`
- `currency`
- `list_price_config`：JSON，描述面向终端的计费规则
- `allowed_models`：JSON 数组，产品层允许的模型列表
- `request_timeout_seconds`
- `created_at` / `updated_at` / `deleted_at`

索引建议：

- `code` 唯一索引
- `status`
- `provider_type`

说明：产品是“售卖面”的聚合概念，不直接保存上游凭证。

### 2. `relay_product_channels`

建议 schema 文件：`internal/ent/schema/relay_product_channel.go`

用途：把一个产品绑定到多个上游 `channels`，形成共享容量池。

核心字段：

- `id`
- `product_id`
- `channel_id`
- `priority`：优先级，数值越小越优先
- `weight`：同优先级下的权重
- `status`：`active` / `paused`
- `allow_fallback`
- `model_filter`：JSON 或字符串规则，用于限制该渠道可承接的模型
- `max_inflight`
- `created_at` / `updated_at`

索引建议：

- `(product_id, channel_id)` 唯一索引
- `(product_id, status, priority)` 复合索引

说明：共享容量的本质就是“产品 -> 渠道池”映射，这张表是路由核心。

### 3. `relay_keys`

建议 schema 文件：`internal/ent/schema/relay_key.go`

用途：给现有 `api_keys` 增加 Relay 业务语义，代表“售出的 Sub-Key”。

核心字段：

- `id`
- `api_key_id`：唯一，绑定现有 `api_keys`
- `project_id`
- `product_id`
- `owner_user_id`：可空，支持运营代建或项目机器人账号
- `display_name`
- `status`：`active` / `suspended` / `exhausted` / `archived`
- `balance_mode`：`prepaid` / `quota_only`
- `daily_request_limit`
- `daily_token_limit`
- `monthly_cost_limit`
- `concurrency_limit`
- `expires_at`
- `last_used_at`
- `created_at` / `updated_at` / `deleted_at`

索引建议：

- `api_key_id` 唯一索引
- `(project_id, status)`
- `(product_id, status)`
- `expires_at`

说明：

- `api_keys` 继续负责协议层鉴权；
- `relay_keys` 负责余额、限额、产品绑定、风控状态；
- `monthly_cost_limit` 在 P1 选项 1 中同时承担请求前 preflight guard 与 settlement hard cap：入口基于 `relay_daily_usage_summaries` 月度聚合快速拒绝，结算事务内以锁后的月度聚合 + 本次 charge 原子判断为准；
- 正向 `concurrency_limit` 在 MVP 中只作为配置、展示和后续 tracker 预留；`<= 0` 可表达拒绝/停用语义，真正超额 in-flight 阻塞留给 Post-MVP Begin/Release tracker；
- 这样可以避免让 `api_keys.profiles` 同时承担鉴权、模型映射、商业计费三套职责。

### 4. `relay_wallets`

建议 schema 文件：`internal/ent/schema/relay_wallet.go`

用途：保存每个 Relay Key 的余额快照，用于快速同步校验。

核心字段：

- `id`
- `relay_key_id`：唯一
- `currency`
- `available_amount`
- `frozen_amount`
- `overdraft_limit`
- `version`：乐观锁版本号
- `updated_at`

索引建议：

- `relay_key_id` 唯一索引

说明：

- 钱包表只保存“当前快照”；
- 真正的账务来源仍是流水表；
- 扣费时通过 `version` 做 CAS/乐观锁，避免并发请求透支。

### 5. `relay_wallet_ledger_entries`

建议 schema 文件：`internal/ent/schema/relay_wallet_ledger_entry.go`

用途：记录充值、消费、退款、冻结、人工调整等不可变账务流水。

核心字段：

- `id`
- `relay_key_id`
- `request_id`：可空，请求消费时写入
- `usage_log_id`：可空，和 usage 结算绑定
- `direction`：`credit` / `debit`
- `scene`：`recharge` / `consume` / `refund` / `freeze` / `unfreeze` / `manual_adjust`
- `amount`
- `balance_before`
- `balance_after`
- `upstream_cost`：可空，保留真实上游成本
- `price_snapshot`：JSON，记录当时的销售计价规则
- `idempotency_key`：唯一，用于防重复扣费
- `operator_user_id`：人工调整时写入
- `remark`
- `created_at`

索引建议：

- `idempotency_key` 唯一索引
- `(relay_key_id, created_at)`
- `request_id`
- `usage_log_id`

说明：推荐用 `usage-log:{id}:consume` 作为扣费幂等键，确保重试时不重复记账。

### 6. `relay_daily_usage_summaries`

建议 schema 文件：`internal/ent/schema/relay_daily_usage_summary.go`

用途：为入口限额检查和运营看板提供轻量聚合，不必每次扫 `usage_logs`。

核心字段：

- `id`
- `relay_key_id`
- `stat_date`
- `request_count`
- `total_tokens`
- `total_charge`
- `total_upstream_cost`
- `last_request_id`
- `updated_at`

索引建议：

- `(relay_key_id, stat_date)` 唯一索引
- `(stat_date, total_charge)`

说明：

- 日级聚合足够覆盖 MVP 的日限额与基础报表；
- 小时级或模型级聚合可以后续再拆。

## 请求与结算主链路

### 1. 认证阶段

1. 现有中间件继续通过 `api_keys` 完成 `X-API-Key` / Bearer Key 鉴权。
2. 在鉴权成功后追加加载 `relay_keys` 与 `relay_wallets`。
3. 如果 `relay_keys.status != active`、已过期、余额不足、日级硬配额已满，或月度成本 preflight guard 已判定超限，则直接拒绝请求；未被 preflight 拦截的请求仍需在结算阶段执行 monthly hard cap。

### 2. 路由阶段

1. 根据 `relay_keys.product_id` 找到 `relay_products`。
2. 从 `relay_product_channels` 取出可用渠道池。
3. 用以下条件做过滤：
   - `channels.status = enabled`
   - `relay_product_channels.status = active`
   - 模型命中 `model_filter`
   - `provider_quota_status.ready = true`（如果该 provider 支持额度探测）
   - 渠道未进入自动禁用窗口
4. 把过滤后的候选集合交给现有 `ChannelService` 健康度与优先级逻辑继续选择。

### 3. 请求落库阶段

1. `RequestService.CreateRequest` 继续写 `requests`。
2. `RequestService.CreateRequestExecution` 继续写 `request_executions`。
3. 不新增平行的 Relay 请求主表，避免双写主流水。

### 4. 结算阶段

1. `UsageLogService.CreateUsageLogFromRequest` 写入 `usage_logs`，得到真实 token 与 upstream cost。
2. `RelaySettlementService` 先用 `usage_log_id` 做幂等判断；已结算的同一 `usage_log_id` 直接成功 no-op。
3. 设置 `monthly_cost_limit` 时，结算事务内先按 `relay_key_id` 加锁并读取当月 `relay_daily_usage_summaries` 聚合：`current_monthly + charge == limit` 允许，`>` 拒绝且不得写 ledger、daily summary、wallet debit 或 `last_used_at`。
4. hard cap 通过后，`prepaid` 更新 `relay_wallets` 快照，`quota_only` 跳过钱包扣减；随后写扣费流水并 upsert `relay_daily_usage_summaries`。
5. 如果请求失败且没有 `usage_logs`，默认不扣费；后续再补“异常执行人工对账”流程。

### 5. MVP 限额语义

- `monthly_cost_limit` 在请求入口仍是基于 `relay_daily_usage_summaries` 月度聚合的 preflight guard，用于快速拦截已经明显超限的请求。
- `monthly_cost_limit` 的最终 hard cap 在 `RelaySettlementService.SettleUsage` 事务内完成：Postgres/MySQL 对 `relay_keys` 执行 `SELECT ... FOR UPDATE`，SQLite 用事务内 no-op `UPDATE relay_keys SET updated_at = updated_at WHERE id = ?` 获得同 Key 串行化效果，再读取当月聚合并判断 `current_monthly + charge`。
- hard cap 边界：正向 charge 下 `current_monthly + charge == limit` 允许；`>` 拒绝。已成功结算的同一 `usage_log_id` 因幂等检查先于 cap 检查而继续 success/no-op；首次被 cap 拒绝的结算不写幂等标记，数据不变时重试仍拒绝。
- 如果上游调用已经完成，settlement 再拒绝会引入“用户已得到响应但无法扣费”的体验和对账风险；页面和运维告警必须把请求成功与结算成功拆开展示。
- 正向 `concurrency_limit` 在 MVP 中暂不强制，只用于配置、展示和未来 tracker 预留；`<= 0` 可以表达拒绝/停用语义。真正的超额 in-flight 阻塞需要 Post-MVP 的 Begin/Release tracker 覆盖完整请求生命周期。

### 6. 为什么不直接在 `usage_logs` 里加“销售金额”字段

MVP 推荐把“上游成本”和“下游销售金额”分开放：

- `usage_logs.total_cost` 继续表示上游成本事实；
- `relay_wallet_ledger_entries.amount` 表示面向用户的实际扣费；
- 这样以后支持促销价、包月、免费额度、人工退款时，不会污染基础 usage 事实表。

## 后端模块拆分

### 复用模块

| 模块                   | 继续承担的职责                                         |
| ---------------------- | ------------------------------------------------------ |
| `APIKeyService`        | 生成 / 查询 / 缓存对外鉴权 Key                         |
| `ChannelService`       | 渠道缓存、健康度、fallback、模型能力判断               |
| `RequestService`       | 创建 `requests` / `request_executions`                 |
| `UsageLogService`      | 记录真实 token 与 upstream cost                        |
| `QuotaService`         | 保留现有 API Key quota 能力，可作为 Relay 限额实现参考 |
| `ProviderQuotaService` | 继续轮询上游 provider 配额状态                         |
| `ProjectService`       | 买方项目管理与隔离                                     |

### 新增业务模块

| 模块           | 建议文件                                     | 职责                                                                   | 关键依赖                                                          |
| -------------- | -------------------------------------------- | ---------------------------------------------------------------------- | ----------------------------------------------------------------- |
| 产品目录服务   | `internal/server/biz/relay_catalog.go`       | 产品 CRUD、产品与渠道池绑定                                            | `relay_products`、`relay_product_channels`、`ChannelService`      |
| Relay Key 服务 | `internal/server/biz/relay_key.go`           | Sub-Key 开通、停用、过期、绑定现有 `api_keys`                          | `relay_keys`、`APIKeyService`                                     |
| 钱包服务       | `internal/server/biz/relay_wallet.go`        | 充值、冻结、余额校验、乐观锁更新                                       | `relay_wallets`、`relay_wallet_ledger_entries`                    |
| 访问守卫       | `internal/server/biz/relay_access.go`        | 请求入口校验 Sub-Key 状态、余额、日限额与月度 preflight guard          | `relay_keys`、`relay_wallets`、`relay_daily_usage_summaries`      |
| 路由服务       | `internal/server/biz/relay_router.go`        | 根据产品池筛选候选渠道并委托 `ChannelService` 选择                     | `relay_product_channels`、`channels`、`provider_quota_status`     |
| 结算服务       | `internal/server/biz/relay_settlement.go`    | 根据 `usage_logs` 执行月度 hard cap、扣费、退款、写账本                | `UsageLogService`、`relay_wallets`、`relay_wallet_ledger_entries` |
| 汇总服务       | `internal/server/biz/relay_usage_summary.go` | 增量 upsert 日聚合，支撑 dashboard、日限额、月度 preflight 与 hard cap | `relay_daily_usage_summaries`、`usage_logs`                       |
| 管理编排服务   | `internal/server/biz/relay_admin.go`         | 把产品、Key、钱包、汇总组合成后台视图                                  | 上述全部 Relay 模块                                               |

### GraphQL / API 分层建议

1. 协议接入层（`internal/server/api/openai.go`、`anthropic.go`、`codex.go` 等）保持轻量，不直接写余额或产品逻辑。
2. Relay 管理台优先走 GraphQL：
   - `internal/server/gql/relay.graphql`
   - `internal/server/gql/relay.resolvers.go`
3. 典型 GraphQL 能力：
   - 产品列表 / 创建 / 上下架
   - 产品绑定渠道池
   - Relay Key 开通 / 停用 / 续期
   - 钱包充值 / 冻结 / 手工调账
   - 账本、日汇总、渠道池使用情况查询

### 对现有请求链路的最小侵入点

推荐只改 3 个关键钩子：

1. **鉴权后钩子**：在 API Key 鉴权成功后补充加载 `relay_keys`。
2. **路由前钩子**：在真正选择 channel 前按产品池过滤候选渠道。
3. **usage 落库后钩子**：在 `UsageLogService.CreateUsageLog` 成功后触发结算。

这样可以避免把 Relay 业务逻辑散落到所有协议 handler 中。

## 典型查询与索引关注点

### 同步链路必须命中的查询

1. `api_keys -> relay_keys`：通过 `api_key_id` 唯一索引。
2. `relay_keys -> relay_wallets`：通过 `relay_key_id` 唯一索引。
3. `relay_keys -> relay_daily_usage_summaries`：按 `(relay_key_id, stat_date)` 取当天聚合。
4. `relay_product_channels -> channels`：按 `(product_id, status, priority)` 取候选池。
5. `provider_quota_status`：按 `channel_id` 唯一索引取额度状态。

### 异步链路重点查询

1. `usage_logs` 按 `request_id` / `api_key_id` / `created_at` 聚合。
2. `relay_wallet_ledger_entries` 按 `relay_key_id + created_at` 查账单。
3. `relay_daily_usage_summaries` 按日期区间做报表。

## MVP 落地顺序

### 阶段 1：数据结构

1. 新增 6 个 Ent schema。
2. 跑 `make generate` 生成模型与迁移。
3. 后台只先做最小 CRUD 与列表查询。

### 阶段 2：入口校验与路由

1. 在现有 API Key 鉴权后加载 `relay_keys`。
2. 新增 `RelayAccessService` 做状态、过期、余额、日级硬限额和月度成本 preflight guard 检查。
3. 新增 `RelayRouterService`，在现有 `ChannelService` 之前按产品池过滤渠道。

### 阶段 3：结算闭环

1. `usage_logs` 成功写入后触发账本扣费。
2. 用 `idempotency_key` 保证重试不重复扣费，已结算重试必须早于月度 hard cap 拒绝判定返回 no-op。
3. 设置 `monthly_cost_limit` 时在 settlement 事务内按 Relay Key 加锁，原子判断月度聚合 + 本次 charge 不超过上限。
4. 通过 cap 后同步更新钱包快照并 upsert 日汇总；`quota_only` 跳过钱包扣减但仍执行 cap。

### 阶段 4：运营台能力

1. 产品管理
2. 渠道池绑定
3. Sub-Key 管理
4. 钱包充值 / 冻结 / 对账视图

## 明确延期的能力

以下能力建议明确不进 MVP：

- 第三方支付回调表与支付网关对接
- 分销商 / 代理商层级与分润结算
- 用户自助注册、优惠券、套餐版本管理
- 多区域容量池与跨机房调度
- 面向终端的精细化账单导出与税务字段
- 请求级预授权 / 额度保留 / 部分退款复杂策略
- 正向并发限额所需的 inflight Begin/Release tracker
- 用户贡献 API / 多供给方共享容量市场
- 模型真实性验证、随机抽检、质量评分与积分奖惩

## 后续实验：用户贡献 API 与质量治理

如果要把 AxonHub 扩展为“用户共享自己的 API 换取平台积分，再用积分兑换或抵扣 token 使用”的共享容量平台，需要在本 MVP 之外补充贡献者账户、贡献渠道、模型探针、随机抽检、质量事件和用户积分账本。

该方向的正式设计与小范围试点验收清单见：

- [用户贡献 API：模型真实性验证、随机抽检与积分奖惩设计](./user-contributed-api-quality-plan.md)
- [用户贡献 API 质量治理：QA 与小范围试点验收清单](./user-contributed-api-quality-qa-checklist.md)

## 结论

这个 MVP 的核心不是重写 AxonHub 请求链路，而是在现有 `api_keys + channels + requests + usage_logs` 之上，补出一层“产品、Sub-Key、钱包、账本、日汇总”模型。

如果按这个拆法落地：

- 同步请求链路只新增少量索引查询；
- 上游路由仍复用已有 `ChannelService` 与 `ProviderQuotaService`；
- 账务与销售策略被限制在 Relay 领域模块内，不会把通用请求模型做脏；
- 后续从 MVP 扩展到包月、支付回调、代理体系时，也不需要推翻主数据结构。
