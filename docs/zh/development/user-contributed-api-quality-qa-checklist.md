# 用户贡献 API 质量治理：QA 与小范围试点验收清单

本文配套 `user-contributed-api-quality-plan.md`，用于验证“用户共享自己的 API 换取平台积分，再用积分兑换或抵扣 token 使用”方向中的模型真实性验证、随机抽检、质量评分、积分奖惩和申诉流程。

## 验收原则

- 小范围试点优先验证风控闭环，不追求公开市场规模。
- 所有积分流水必须幂等、不可变、可追溯。
- 模型真实性只声明“通过平台抽检标准”，不声明绝对真实。
- 严重处罚前必须保留人工复核或申诉通道。
- 探针题库、标准答案和反作弊细节不得暴露给贡献者。
- 劣质 API 必须能快速降权或下线，避免影响消费方体验。

## 试点前置条件

- [ ] 已创建 `user-contributed-api-quality-plan.md` 并冻结试点边界。
- [ ] 已确定种子用户范围，建议 3-10 人。
- [ ] 已限制每个用户最多贡献 1-3 个 API Key。
- [ ] 已限制每个贡献 API 的每日 token 和请求上限。
- [ ] 已确定试点 provider / model 范围，建议先只选 1-2 类。
- [ ] 已准备平台可信基线渠道，用于对照模型比较和 LLM-as-judge。
- [ ] 已准备正常 API、低质 API、伪造 API、间歇故障 API 四类测试样本。
- [ ] 已设定每日抽检预算，避免 probe 成本失控。

## 数据模型验收

### `contributed_channels`

- [ ] 每个贡献渠道能关联到现有 `channels.id`。
- [ ] 每个贡献渠道能追溯 `contributor_user_id`。
- [ ] `declared_provider` 与 `declared_models` 可记录用户声明。
- [ ] `share_status` 覆盖 `pending_verification`、`verification_timeout`、`active`、`watch`、`suspended`、`blocked`、`withdrawn`。
- [ ] `quality_score` 范围限制为 0-100。
- [ ] `quality_status` 覆盖 `unverified`、`verified`、`degraded`、`blocked`。
- [ ] 支持 `daily_token_limit`、`daily_request_limit` 和 `max_inflight`。
- [ ] 支持 `last_verified_at`、`last_probe_at` 和 `penalty_active_until`。
- [ ] 贡献者不能读取其他贡献者的贡献渠道。

### `model_authenticity_probe_cases`

- [ ] 探针题能按 provider、模型族、类别和难度筛选。
- [ ] 支持 `exact`、`regex`、`json_schema`、`judge_model`、`embedding_similarity`、`human_review` 等校验类型。
- [ ] 支持题目版本和启停。
- [ ] 支持 nonce / 随机变量注入。
- [ ] 支持成本权重，用于抽检预算控制。
- [ ] 普通贡献者无权读取完整 prompt、标准答案和 private rubric。

### `model_authenticity_probe_runs`

- [ ] 每次模型真实性抽检都有唯一 `authenticity_probe_run_id` 或等价任务 ID。
- [ ] 每次抽检都有 `trigger_type`。
- [ ] 抽检请求和响应只存 hash 或脱敏摘要，不默认保存敏感明文。
- [ ] 记录 `status`、`score`、`latency_ms`、`token_usage` 和 `failure_reason`。
- [ ] 重复执行同一模型真实性抽检任务不会重复处罚。
- [ ] `inconclusive` 不应直接触发严重处罚。

### `channel_quality_events`

- [ ] 质量分变动、降权、冻结、扣罚、恢复、申诉都落事件。
- [ ] 每个事件可关联模型真实性抽检任务、request 或积分流水。
- [ ] 贡献者可看摘要原因，不可看私密探针内容。
- [ ] 人工操作记录 `operator_user_id`。
- [ ] 事件不可硬删除，只能追加纠正事件。

### `user_point_accounts` / `user_point_ledger_entries`

- [ ] 用户积分账户区分 `available_points`、`pending_points`、`frozen_points`。
- [ ] 所有积分变动都有不可变流水。
- [ ] 每条流水有 `idempotency_key`。
- [ ] 并发释放、冻结、扣罚不会导致负数余额。
- [ ] 同一 request / usage / 模型真实性抽检任务不能重复生成积分。
- [ ] 人工调整必须记录操作人和备注。

## 入驻验证验收

- [ ] 提交后立即创建 `pending_verification` 状态的贡献渠道，不同步等待完整验证完成。
- [ ] 入驻验证由后台任务异步执行，任务状态、开始时间、结束时间和失败原因可追踪。
- [ ] 前端能轮询、刷新或订阅验证状态，不依赖提交接口长时间阻塞。
- [ ] 无效 API Key 不能通过入驻验证。
- [ ] provider / base URL 不可达时给出可解释错误。
- [ ] 声明模型不存在时给出可解释错误。
- [ ] 至少执行 3-5 个轻量探针，其中包含基础可用性探针和 1 个反缓存或 nonce 探针。
- [ ] 超时后，例如 5 分钟未完成，转为可解释失败或 `verification_timeout`。
- [ ] 重复提交或重试同一验证任务具备幂等键，不会重复创建渠道、扣费或处罚。
- [ ] 入驻验证成功后 `share_status` 从 `pending_verification` 进入 `active` 或 `watch`。
- [ ] 入驻验证失败或超时不会进入正常路由候选池。
- [ ] 入驻验证产生的 probe 成本不扣消费方积分。
- [ ] 入驻验证结果能在运营侧追溯。

## 模型真实性与探针验收

### 程序化校验

- [ ] JSON schema 校验失败能标记 probe failed。
- [ ] regex / exact 校验失败能给出 reason code。
- [ ] 数值答案支持容差。
- [ ] 未引用 nonce 的响应能被识别。
- [ ] 空响应、过短响应、超长响应能被识别。

### 对照模型比较

- [ ] 系统可用可信渠道生成基线答案。
- [ ] 被测 API 与基线答案差异过大时进入 `inconclusive` 或 `failed`。
- [ ] 对照模型失败时不直接处罚被测渠道。
- [ ] 对照结果记录 judge metadata。

### LLM-as-judge

- [ ] 裁判模型必须来自可信渠道。
- [ ] judge 失败时 probe 不应自动视为贡献渠道失败。
- [ ] judge 分数可影响 `probe_score`。
- [ ] 单次 judge 低分不能触发 P3/P4 处罚。

### 人工复核

- [ ] P3/P4 处罚前可进入人工复核。
- [ ] 人工复核能查看完整证据链。
- [ ] 人工复核结论能写入 `channel_quality_events`。
- [ ] 人工复核通过后可恢复质量分和冻结积分。

## 随机抽检验收

- [ ] 入驻验证 100% 触发。
- [ ] 活跃渠道每日可触发 1-3 次轻量抽检。
- [ ] 按流量抽检可按真实成功请求量比例触发。
- [ ] 风险渠道抽检频率可自动提高。
- [ ] 大额积分释放前可触发 `pre_reward_release` 抽检。
- [ ] 抽检不会展示给普通消费方。
- [ ] 抽检不会扣消费方余额或积分。
- [ ] 抽检成本受每日预算约束。
- [ ] 抽检题目不会在贡献者页面泄漏。
- [ ] 优质渠道抽检频率可降低但不能为 0。

## 质量分验收

- [ ] `availability_score` 能根据成功率、错误率、超时率计算。
- [ ] `probe_score` 能根据抽检通过率和严重失败次数计算。
- [ ] `latency_score` 能根据 P50 / P95 / P99 延迟计算。
- [ ] `settlement_integrity_score` 能根据 usage / token / 请求事实完整性计算。
- [ ] `complaint_score` 能根据投诉和人工反馈计算。
- [ ] 综合 `quality_score` 限制在 0-100。
- [ ] `excellent`、`good`、`watch`、`degraded`、`blocked` 状态转换符合阈值。
- [ ] 质量分变更写入 `channel_quality_events`。
- [ ] 质量分不会因单次轻微失败大幅波动。

## 路由降权验收

- [ ] `blocked` 渠道不进入正常候选池。
- [ ] `degraded` 渠道默认不承接新流量，除非运营明确允许低优先级测试流量。
- [ ] `watch` 渠道权重降低，例如乘以 0.3。
- [ ] `good` 渠道按正常权重进入候选池。
- [ ] `excellent` 渠道可小幅加权，但仍受额度、公平性和成本限制。
- [ ] `penalty_active_until` 未过期时不能恢复正常权重。
- [ ] 质量状态变化后路由缓存能及时失效。
- [ ] 消费侧请求失败时能区分上游池不可用、贡献渠道异常、余额不足和模型不可用。

## 积分奖励验收

- [ ] 成功命中贡献渠道后生成 `pending_points`，不直接进入 `available_points`。
- [ ] 积分计算包含模型倍率、质量倍率、可用性倍率和平台调整系数。
- [ ] 释放窗口结束后 `pending_points` 可转入 `available_points`。
- [ ] 大额释放前触发抽检。
- [ ] 抽检失败时可冻结相关 `pending_points`。
- [ ] 真实请求失败时不应给贡献者正常奖励。
- [ ] 由平台或消费方原因导致失败时，不应错误处罚贡献者。
- [ ] 积分流水可通过 request / usage / channel 追溯。
- [ ] 同一 request 不会重复奖励。
- [ ] 贡献者 user/project 与消费方 user/project 判定为同主体时，不发放或降低贡献奖励，且原因可追溯。

## 积分与 Relay 钱包兑换验收

- [ ] 积分抵扣 token 前先写入 `user_point_ledger_entries` 扣减流水，`scene` 为 `consume` 或 `point_redeem`，并带 `idempotency_key`。
- [ ] 积分扣减与 Relay 钱包侧充值/抵扣流水能通过 `related_wallet_ledger_entry_id` 或等价关联记录互相追溯。
- [ ] 保存 `conversion_rate_snapshot`、`related_relay_key_id`、`related_project_id` 等兑换快照，便于审计和回滚。
- [ ] 同一兑换请求重复执行时不会重复扣积分、重复入账或重复抵扣。
- [ ] 兑换失败时能追加补偿流水或回滚积分扣减。
- [ ] 积分余额与 Relay 钱包余额不会双扣、漏扣或重复入账。

## 惩罚与冻结验收

- [ ] P0 事件只记录或轻微扣分，不影响正常路由。
- [ ] P1 事件降低权重并提高抽检率。
- [ ] P2 事件冻结待结算积分并暂停新流量。
- [ ] P3 事件需要证据链和人工复核。
- [ ] P4 事件可禁用贡献渠道并限制贡献者账户。
- [ ] 扣罚优先作用于 `pending_points`。
- [ ] 扣罚已释放积分时必须记录人工复核和操作人。
- [ ] 重复处罚请求不会重复扣罚。
- [ ] 处罚撤销后能追加恢复事件，不删除原事件。

## 申诉验收

- [ ] 贡献者能看到处罚摘要、时间、等级、影响积分和状态。
- [ ] 贡献者不能看到完整探针题和标准答案。
- [ ] 贡献者可以提交申诉说明。
- [ ] 运营能查看申诉关联的模型真实性抽检任务、请求摘要、质量事件和积分流水。
- [ ] 申诉通过能恢复质量分和释放冻结积分。
- [ ] 申诉失败能维持处罚并记录复核备注。
- [ ] 申诉处理全程有审计记录。

## 安全与隐私验收

- [ ] 用户提交 API Key 后不再明文展示。
- [ ] API Key 存储必须加密或使用现有安全凭证机制。
- [ ] 日志不得打印 API Key、完整 Authorization header 或完整探针私密 prompt。
- [ ] 贡献者撤回 API 后，渠道立即退出候选池。
- [ ] 撤回后运行时缓存失效。
- [ ] 贡献者只能访问自己的贡献收益和质量事件。
- [ ] 运营访问敏感数据需要权限控制和审计。

## 试点样本验收

准备 4 类样本：

1. **正常 API**
   - [ ] 能通过入驻验证。
   - [ ] 能正常承接流量。
   - [ ] 能获得 pending points 并释放。
2. **低质 API**
   - [ ] 高错误率或高延迟能降低质量分。
   - [ ] 能进入 `watch` 或 `degraded`。
   - [ ] 抽检频率提高。
3. **伪造 / 降级 API**
   - [ ] 能被模型能力探针或对照模型发现。
   - [ ] 能冻结待结算积分。
   - [ ] 严重时进入人工复核和处罚。
4. **间歇故障 API**
   - [ ] 不因单次失败立即封禁。
   - [ ] 连续异常后能降权。
   - [ ] 恢复稳定后质量分可逐步恢复。

## 成功指标

| 指标                    | 目标        |
| ----------------------- | ----------- |
| 明显无效 API 入驻拦截率 | >= 90%      |
| 劣质 API 降权/暂停时间  | 5-15 分钟内 |
| 积分重复入账            | 0           |
| 积分并发错账            | 0           |
| 探针题泄漏              | 0           |
| API Key 明文泄漏        | 0           |
| P3/P4 处罚证据链完整率  | 100%        |
| 申诉可追溯率            | 100%        |

## 推荐验证命令

当前文档阶段不要求新增代码命令。进入实现阶段后建议补充，避免用泛化 `Probe` 误跑现有 `ChannelProbe` 测试：

```bash
go test ./internal/server/biz -run 'Contributed|Authenticity|Quality|Point|Relay' -count=1
go test ./internal/server/api -run 'Contributed|Authenticity|Quality|Point|Relay' -count=1
pnpm --dir frontend lint
pnpm --dir frontend build
```

若本地工具链不满足 Go 版本要求，不要降级 `go.mod`，应记录准确错误并在 Go 1.26 环境补跑。

## 复盘记录模板

每轮试点结束后记录：

- 参与贡献者数量。
- 贡献 API 数量。
- 抽检次数与成本。
- 入驻失败数量与原因。
- 劣质 API 发现数量与发现时间。
- 误判数量与申诉结果。
- 积分发放、冻结、扣罚总量。
- 用户反馈与下一轮阈值调整。
