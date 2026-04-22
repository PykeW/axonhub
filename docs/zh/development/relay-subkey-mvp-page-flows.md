# 自托管 Relay + Sub-Key 共享容量 MVP：页面清单与用户流程

## 背景

AxonHub 已经具备 API 网关、渠道路由、请求审计与 API Key 鉴权能力。针对“自托管 Relay + Sub-Key 共享容量”方案，MVP 的重点不是重做协议层，而是在现有控制台上补齐一套面向运营与买方项目的产品页、Sub-Key 管理页、账务页与请求排障页。

与后端设计文档对应，这套页面需要围绕以下对象展开：

- `relay_products`：对外售卖的共享容量产品。
- `relay_product_channels`：产品绑定的上游渠道池。
- `relay_keys`：对外发放的下游 Sub-Key 业务实体。
- `relay_wallets` / `relay_wallet_ledger_entries`：余额快照与账务流水。
- `relay_daily_usage_summaries`：限额检查与运营看板聚合。
- `requests` / `request_executions` / `usage_logs`：请求事实、命中渠道与结算事实。

## MVP 目标

1. 让运营人员可以手工完成“建产品、绑渠道、发 Key、充值、冻结、排障”闭环。
2. 让买方项目管理员可以自助查看产品范围、创建或接收可用的 Sub-Key、跟踪余额与限额。
3. 让开发者沿用现有 OpenAI / Anthropic / Codex 兼容调用方式，不需要新的接入协议。
4. 页面只覆盖 MVP 必需动作，不引入自动支付、代理分销、多卖家市场或复杂工单系统。

## 角色与权限边界

| 角色       | 典型身份                | 主要页面                                           | 核心动作                                                         | 不在 MVP 的动作            |
| ---------- | ----------------------- | -------------------------------------------------- | ---------------------------------------------------------------- | -------------------------- |
| 平台运营   | Relay 部署者、管理员    | 产品页、渠道池页、Sub-Key 列表、账务页、请求排障页 | 创建产品、绑定渠道、发放/冻结/归档 Key、充值、退款、查看请求明细 | 自动定价实验、代理分润     |
| 项目管理员 | 买方团队 owner          | 项目下的产品接入页、Sub-Key 详情、用量与账单页     | 查看可用产品、复制接入凭证、设置显示名、查看余额/限额/过期时间   | 自助绑定上游 provider 凭证 |
| 开发者     | 调用 Relay API 的工程师 | 接入说明、Key 详情、请求日志                       | 复制 Base URL / API Key、排查最近失败请求                        | 修改计费策略、管理共享池   |
| 风控/客服  | 运营支持角色            | Key 详情、流水页、请求跟踪页                       | 人工冻结、备注、退款、解释失败原因                               | 独立审批流                 |

实现建议：权限层面优先复用现有 `project_id` 隔离与后台角色能力，不单独设计新 ACL，只增加页面级动作开关。

## Key 状态模型

### 持久化状态

`relay_keys.status` 建议直接使用后端设计中的 4 个持久化状态：

| 状态        | 页面展示     | 请求行为                           | 允许动作                   |
| ----------- | ------------ | ---------------------------------- | -------------------------- |
| `active`    | 正常、可调用 | 允许通过同步校验后继续转发         | 充值、改名、暂停、归档     |
| `suspended` | 已暂停       | 直接拒绝请求                       | 恢复、归档、备注           |
| `exhausted` | 已耗尽       | 余额不足或硬配额达到上限时拒绝请求 | 充值后恢复、调整限额、归档 |
| `archived`  | 已归档       | 永久拒绝请求，不再出现在默认列表   | 仅查看历史                 |

### 运行时派生状态

以下状态不一定单独落库，但必须在页面上有清晰 badge：

| 派生状态                 | 判定来源                                         | 页面用途                            |
| ------------------------ | ------------------------------------------------ | ----------------------------------- |
| `expired`                | `expires_at < now()`                             | 告知 Key 已过期，需要续期或重发     |
| `low_balance`            | `relay_wallets.available_amount` 低于阈值        | 在列表和详情页提前预警              |
| `quota_reached`          | `relay_daily_usage_summaries` 或月度聚合超过限制 | 标记为什么进入 `exhausted`          |
| `concurrency_blocked`    | 当前并发超过 `concurrency_limit`                 | 请求失败时给出可解释错误            |
| `upstream_pool_degraded` | 关联产品的候选渠道不足或全部 unhealthy           | 提示是共享池问题，而不是单 Key 问题 |

实现建议：列表页展示“持久化状态 + 派生 badge”双层信息，避免把上游池故障误判成用户余额问题。

## 页面清单

### 运营侧页面

| 页面                 | 建议路由                             | 主要角色       | 依赖数据                                                                        | 核心动作                                               |
| -------------------- | ------------------------------------ | -------------- | ------------------------------------------------------------------------------- | ------------------------------------------------------ |
| 共享容量产品列表     | `/console/relay/products`            | 平台运营       | `relay_products`                                                                | 查看产品状态、上下架、进入详情                         |
| 产品详情与渠道池配置 | `/console/relay/products/:id`        | 平台运营       | `relay_products`、`relay_product_channels`、`channels`、`provider_quota_status` | 编辑产品信息、绑定/解绑渠道、调整优先级/权重、限制模型 |
| Sub-Key 列表         | `/console/relay/keys`                | 平台运营、客服 | `relay_keys`、`relay_wallets`、项目信息                                         | 筛选状态、搜索项目、查看低余额与过期 Key               |
| Sub-Key 详情         | `/console/relay/keys/:id`            | 平台运营、客服 | `relay_keys`、`relay_wallets`、`relay_daily_usage_summaries`、最近 `requests`   | 暂停、恢复、归档、改名、调整到期时间                   |
| 充值/流水页          | `/console/relay/keys/:id/billing`    | 平台运营、客服 | `relay_wallets`、`relay_wallet_ledger_entries`                                  | 充值、退款、人工调整、查看账务凭证                     |
| 请求跟踪页           | `/console/relay/requests`            | 平台运营、客服 | `requests`、`request_executions`、`usage_logs`                                  | 按 Key / 产品 / 渠道排查失败、查看扣费结果             |
| 渠道健康看板         | `/console/relay/channel-pool-health` | 平台运营       | `relay_product_channels`、`channels`、`provider_quota_status`                   | 判断某产品是否存在上游容量风险                         |

### 买方项目侧页面

| 页面              | 建议路由                                 | 主要角色           | 依赖数据                                                                        | 核心动作                                        |
| ----------------- | ---------------------------------------- | ------------------ | ------------------------------------------------------------------------------- | ----------------------------------------------- |
| 产品接入页        | `/projects/:projectId/relay/products`    | 项目管理员         | 项目可见产品、产品说明、模型范围                                                | 了解可购买/可使用的产品，进入 Key 列表          |
| 项目 Sub-Key 列表 | `/projects/:projectId/relay/keys`        | 项目管理员         | `relay_keys`、`relay_wallets`                                                   | 查看项目内可用 Key、复制接入信息                |
| 项目 Sub-Key 详情 | `/projects/:projectId/relay/keys/:id`    | 项目管理员、开发者 | `relay_keys`、钱包快照、最近请求、SDK 示例                                      | 复制 API Key / Base URL、查看状态、最近失败原因 |
| 用量与账单页      | `/projects/:projectId/relay/usage`       | 项目管理员         | `relay_daily_usage_summaries`、`relay_wallet_ledger_entries`、`usage_logs` 汇总 | 查看余额、消耗趋势、最近扣费                    |
| 接入说明页        | `/projects/:projectId/relay/get-started` | 开发者             | 产品允许模型、示例请求、错误码说明                                              | 快速接入并理解共享容量模型                      |

### 页面组合建议

为减少前端路由数量，MVP 可以采用“列表页 + 详情页多 Tab”结构：

- 产品详情页内包含 `基本信息`、`渠道池`、`可用模型`、`关联 Key` 四个 Tab。
- Key 详情页内包含 `概览`、`余额与流水`、`请求记录`、`限额设置` 四个 Tab。
- 项目侧页面可尽量复用运营侧组件，只隐藏高权限动作按钮。

## 页面字段重点

### 1. 产品详情页必须展示的字段

- 产品编码、名称、provider 类型、状态。
- 允许模型列表、默认超时、计费模式。
- 已绑定渠道数、健康渠道数、当前不可用原因。
- 每个渠道的优先级、权重、模型过滤规则、最近 quota 状态。

### 2. Key 列表与详情页必须展示的字段

- Key 显示名、所属项目、绑定产品、当前状态。
- 掩码后的 API Key、创建时间、过期时间、最后使用时间。
- 可用余额、冻结金额、当日请求数、当日 token、月度成本。
- 最近失败原因摘要，例如余额不足、达到日限额、上游池不可用。

### 3. 请求排障页必须展示的字段

- 请求时间、项目、Key、产品、请求模型。
- 命中的 `channel_id`、上游 provider、执行延迟、响应状态码。
- 是否扣费、扣费金额、对应 `usage_log_id` / ledger entry。
- 失败阶段：鉴权失败、Key 校验失败、路由失败、上游执行失败、结算失败。

## 端到端用户流程

### 流程 1：运营创建共享容量产品

1. 运营进入“共享容量产品列表”，点击“新建产品”。
2. 填写产品编码、名称、provider 类型、计费模式、允许模型、默认超时。
3. 保存后进入产品详情页，在“渠道池”Tab 中绑定一个或多个 `channels`。
4. 为每个绑定关系设置 `priority`、`weight`、`model_filter`、`allow_fallback`。
5. 页面实时显示候选池健康情况；如果所有渠道都不可用，则禁止把产品切到 `active`。
6. 产品激活后，项目管理员即可在项目侧看到该产品或由运营继续发放 Key。

实现要点：产品页保存时只写 `relay_products`；渠道池配置通过独立请求写 `relay_product_channels`，方便后续做拖拽排序和局部更新。

### 流程 2：运营为项目发放 Sub-Key

1. 运营进入“Sub-Key 列表”，点击“创建 Sub-Key”。
2. 选择目标 `project_id`、绑定产品、填写显示名、有效期、余额模式与硬限额。
3. 后端创建 `api_keys` 记录，并同步创建 `relay_keys`、`relay_wallets`。
4. 创建成功后返回 Key 详情页，显示一次性可复制的明文 Key。
5. 若是预付费模式，运营可立即在“余额与流水”Tab 完成首充。

实现要点：创建完成后必须立即返回“明文 Key 只显示一次”提醒；后续页面只显示掩码值。

### 流程 3：项目管理员查看并接入 Key

1. 项目管理员在“项目 Sub-Key 列表”中找到已发放的 Key。
2. 进入详情页后查看 Base URL、支持的模型、SDK 调用示例和状态提示。
3. 将 Key 配置到现有 OpenAI / Anthropic / Codex 客户端。
4. 首次调用成功后，页面更新 `last_used_at`，并在最近请求区域看到成功记录。

实现要点：接入说明必须明确“这是 AxonHub 下游 Key，不是上游 provider 原生 Key”，避免用户误把 Key 用在 provider 原站点。

### 流程 4：请求成功并完成扣费

1. 客户端携带 AxonHub Sub-Key 访问兼容接口。
2. 鉴权中间件先校验 `api_keys`，随后加载 `relay_keys` 与 `relay_wallets`。
3. 如果 Key 状态正常、余额与硬限额通过，则根据 `product_id` 加载渠道池。
4. 路由层过滤不可用渠道，选出最终命中的 `channel_id`，并写入 `requests` 与 `request_executions`。
5. 上游响应成功后，结算层依据 `usage_logs` 生成账务流水并更新钱包快照。
6. 项目侧详情页与运营侧请求页都能看到这次调用，包括模型、token、成本和命中渠道。

实现要点：页面显示的“请求成功”和“扣费成功”要拆成两个状态位，避免上游成功但结算延迟时让用户误解为未记录。

### 流程 5：余额不足或硬限额触发耗尽

1. 请求进入后，系统检测到 `available_amount <= 0`，或日限额/月限额已达上限。
2. 后端把 Key 标记为 `exhausted`，并返回可解释错误信息。
3. Key 列表页显示 `exhausted`，同时带上 `low_balance` 或 `quota_reached` badge。
4. 项目管理员在详情页看到失败原因和最近一次触发时间。
5. 运营在账务页完成充值或调整限额后，Key 恢复为 `active`。

实现要点：`exhausted` 应是可恢复状态，不要与 `suspended` 共用文案；页面操作按钮也要区分“充值恢复”和“人工解封”。

### 流程 6：运营主动暂停或归档 Key

1. 运营或客服在 Key 详情页点击“暂停 Key”。
2. 系统把 `relay_keys.status` 改成 `suspended`，并要求填写备注。
3. 后续请求直接拒绝，请求页显示失败阶段为“Key 校验失败”。
4. 如果是永久下线，则继续执行“归档”，Key 从默认列表移除，仅保留历史查询能力。

实现要点：暂停动作必须要求备注，方便客服在项目侧展示明确原因，而不是仅显示通用 403。

### 流程 7：共享池故障排查

1. 多个项目反馈同一产品持续报错。
2. 运营进入“渠道健康看板”先看该产品绑定渠道是否都处于不可用状态。
3. 再进入“请求跟踪页”，按产品和时间范围筛选失败请求。
4. 对比 `request_executions` 与 `provider_quota_status`，确认是 quota 用尽、渠道禁用还是上游接口异常。
5. 临时处理方式包括：暂停故障渠道、提高健康渠道优先级、限制特定模型、或将产品整体切回 `draft` / `paused` 状态。

实现要点：页面需要同时提供“按 Key 看问题”和“按产品/渠道池看问题”两种入口，否则共享池故障很容易被误以为是单用户问题。

## 状态迁移建议

| 触发动作   | 迁移前                     | 迁移后                  | 页面入口               |
| ---------- | -------------------------- | ----------------------- | ---------------------- |
| 创建 Key   | 无                         | `active` 或 `suspended` | 新建 Sub-Key 弹窗      |
| 余额耗尽   | `active`                   | `exhausted`             | 自动触发，无需人工页面 |
| 充值恢复   | `exhausted`                | `active`                | 余额与流水 Tab         |
| 人工暂停   | `active` / `exhausted`     | `suspended`             | Key 详情页             |
| 恢复使用   | `suspended`                | `active`                | Key 详情页             |
| 归档       | 任意非归档状态             | `archived`              | Key 详情页             |
| 到期后续期 | `active` + `expired` badge | `active`                | Key 详情页修改有效期   |

## MVP 不做的页面

以下内容建议明确不进入首版，避免页面范围失控：

- 在线支付、订单中心、发票下载。
- 自助购买流程与购物车。
- 上游 provider 凭证市场或卖家中心。
- 复杂审批流、客服工单系统、对账导出中心。
- 细粒度代理商层级与分润结算页。

## 实施优先级建议

### Phase 1: 可上线最小闭环

1. 产品列表 + 产品详情（含渠道池配置）。
2. Sub-Key 列表 + Key 详情。
3. 充值/流水 Tab。
4. 项目侧 Key 列表 + 接入说明。

### Phase 2: 运营稳定性补强

1. 请求跟踪页。
2. 渠道健康看板。
3. 更细的状态 badge 与失败原因透出。

按这个顺序推进，可以先跑通“卖出共享容量并可被真实调用”的最小闭环，再补强排障与运营效率。

## 前端路由树与页面拆解

### 设计目标

- 将页面信息架构拆为“运营侧”和“项目侧”两条主线，降低单页状态机复杂度。
- 保持 URL 语义稳定，便于后续从 MVP 扩展到更完整的运营、治理与排障流程。
- 实现层继续兼容 AxonHub 现有 TanStack Router 与 file-based routing 风格，但不要求 URL 与最终文件名完全一致。
- MVP 首发优先保证“产品创建 -> 渠道池绑定 -> Sub-Key 发放 -> 项目接入 -> 请求排障”闭环，其它能力先以内嵌 Tab 或 Section 承载。

### 路由树建议

建议把可见 URL 信息架构组织为两条主线：

```text
/operator/relay-subkeys
  /products
  /products/create
  /products/:productId
  /products/:productId/edit
  /keys
  /keys/create
  /keys/:keyId
  /keys/:keyId/billing
  /keys/:keyId/requests
  /channel-pool-health
  /requests

/projects/:projectId/relay-subkey
  /overview
  /products
  /keys
  /keys/:keyId
  /usage
  /get-started
  /verify
```

说明：

- `operator` 主线承载平台运营、客服和风控支持视角。
- `projects/:projectId/relay-subkey` 主线承载买方项目管理员和开发者视角。
- 若 MVP 需要进一步收敛，可先只落 `products`、`keys`、`overview`、`get-started` 等最小页面，其余能力以内嵌 Tab 过渡。

### File-Based Route 命名建议

当前仓库更接近“目录 + `route.tsx` / `index.tsx`”的 TanStack Router file-based style，而不是把整条路径压成一个点分文件名。文档里建议明确区分两层概念：

- `URL 路由树 / 信息架构`：面向用户的导航与职责划分。
- `frontend/src/routes` 实际落地：优先贴合现有 `__root`、`_authenticated`、`_authenticated/project` 结构。
- 如果项目上下文继续由 `ProjectGuard` 和当前选中项目提供，项目侧页面可以先挂在 `_authenticated/project/relay-subkeys/**` 下，不必为了 MVP 先引入 `$projectId` 级路径段。

可参考的实现层命名示例：

```text
frontend/src/routes/
  __root.tsx
  _authenticated/
    relay-subkeys/
      route.tsx
      products/
        index.tsx
        create.tsx
        $productId.tsx
      keys/
        index.tsx
        create.tsx
        $keyId.tsx
        $keyId.billing.tsx
      requests/
        index.tsx
      channel-pool-health/
        index.tsx

    project/
      relay-subkeys/
        route.tsx
        overview.tsx
        products.tsx
        keys/
          index.tsx
          $keyId.tsx
        usage.tsx
        get-started.tsx
        verify.tsx
```

### 页面职责与 Flow 映射

#### 运营侧页面

| 页面         | 路由                                          | 对应 Flow                  | 核心职责                                     |
| ------------ | --------------------------------------------- | -------------------------- | -------------------------------------------- |
| 产品列表     | `/operator/relay-subkeys/products`            | 运营创建共享容量产品       | 查看状态、筛选、上下架、进入详情             |
| 产品新建     | `/operator/relay-subkeys/products/create`     | 运营创建共享容量产品       | 填写产品编码、名称、provider 类型、模型范围  |
| 产品详情     | `/operator/relay-subkeys/products/:productId` | 产品详情与渠道池配置       | 配置渠道池、优先级、权重、模型过滤           |
| Sub-Key 列表 | `/operator/relay-subkeys/keys`                | 运营为项目发放 Sub-Key     | 查看所有 Key、筛选状态、识别低余额和过期 Key |
| Sub-Key 新建 | `/operator/relay-subkeys/keys/create`         | 运营为项目发放 Sub-Key     | 选择项目、产品、余额模式、有效期、硬限额     |
| Sub-Key 详情 | `/operator/relay-subkeys/keys/:keyId`         | Key 管理、暂停、恢复、归档 | 查看概览、最近失败原因、执行管理动作         |
| 账务页       | `/operator/relay-subkeys/keys/:keyId/billing` | 充值/流水页                | 充值、退款、人工调整、查看账务凭证           |
| 请求排障页   | `/operator/relay-subkeys/requests`            | 共享池故障排查             | 按 Key / 产品 / 渠道检索失败请求             |
| 渠道健康看板 | `/operator/relay-subkeys/channel-pool-health` | 共享池故障排查             | 查看产品池健康、识别上游容量风险             |

#### 项目侧页面

| 页面       | 路由                                            | 对应 Flow                | 核心职责                                    |
| ---------- | ----------------------------------------------- | ------------------------ | ------------------------------------------- |
| 总览页     | `/projects/:projectId/relay-subkey/overview`    | 项目管理员查看并接入 Key | 汇总产品、Key、余额和最近失败               |
| 产品接入页 | `/projects/:projectId/relay-subkey/products`    | 项目管理员查看产品范围   | 查看项目可见产品和模型说明                  |
| Key 列表   | `/projects/:projectId/relay-subkey/keys`        | 项目管理员查看并接入 Key | 查看项目内可用 Key、复制接入信息            |
| Key 详情   | `/projects/:projectId/relay-subkey/keys/:keyId` | 项目管理员查看并接入 Key | 复制 API Key / Base URL、查看状态和最近失败 |
| 用量页     | `/projects/:projectId/relay-subkey/usage`       | 请求成功并完成扣费       | 查看余额、消耗趋势、最近扣费                |
| 接入说明页 | `/projects/:projectId/relay-subkey/get-started` | 项目管理员查看并接入 Key | 提供 SDK 示例、错误码说明、接入提醒         |
| 验证页     | `/projects/:projectId/relay-subkey/verify`      | 接入验证                 | 执行简单联通校验并展示最近验证结果          |

### 页面与组件拆分建议

#### 页面级组件

运营侧：

- `RelayProductListPage`
- `RelayProductCreatePage`
- `RelayProductDetailPage`
- `RelayKeyListPage`
- `RelayKeyCreatePage`
- `RelayKeyDetailPage`
- `RelayKeyBillingPage`
- `RelayRequestTracePage`
- `RelayChannelPoolHealthPage`

项目侧：

- `ProjectRelayOverviewPage`
- `ProjectRelayProductAccessPage`
- `ProjectRelayKeyListPage`
- `ProjectRelayKeyDetailPage`
- `ProjectRelayUsagePage`
- `ProjectRelayGetStartedPage`
- `ProjectRelayVerifyPage`

#### 共享业务组件

产品相关：

- `RelayProductTable`
- `RelayProductStatusBadge`
- `RelayProductForm`
- `RelayChannelPoolTable`
- `RelayChannelPoolEditor`
- `RelayChannelHealthSummary`
- `AllowedModelList`

Key 相关：

- `RelayKeyTable`
- `RelayKeyStatusBadge`
- `RelayKeyCreateForm`
- `RelayKeyOverviewCard`
- `RelayKeyMaskedSecretCard`
- `RelayKeyLimitPanel`
- `RelayKeyDerivedStateBadges`

账务与用量相关：

- `RelayWalletSummaryCard`
- `RelayLedgerTable`
- `RelayRechargeDialog`
- `RelayUsageSummaryChart`
- `RelayUsageFilters`
- `RelayChargeResultBadge`

请求排障相关：

- `RelayRequestTable`
- `RelayRequestFilters`
- `RelayExecutionTimeline`
- `RelayFailureStageBadge`
- `RelayRequestDetailDrawer`

项目接入相关：

- `RelaySdkExampleCard`
- `RelayBaseUrlCard`
- `RelayVerifyPanel`
- `RelayErrorGuide`

#### 页面内部推荐拆分

- `RelayProductDetailPage`
  - `RelayProductHeader`
  - `RelayProductOverviewTab`
  - `RelayProductChannelPoolTab`
  - `RelayProductAllowedModelsTab`
  - `RelayProductAssignedKeysTab`
- `RelayKeyDetailPage`
  - `RelayKeyHeader`
  - `RelayKeyOverviewTab`
  - `RelayKeyBillingTab`
  - `RelayKeyRequestTab`
  - `RelayKeyLimitTab`
- `ProjectRelayKeyDetailPage`
  - `ProjectRelayKeyHeader`
  - `ProjectRelayKeyAccessTab`
  - `ProjectRelayKeyUsageTab`
  - `ProjectRelayKeyRecentFailuresTab`

### 前端落地跟进建议

#### 建议的 `frontend/src/routes` 文件映射

| 页面          | 建议 route file                                                            | 说明                                |
| ------------- | -------------------------------------------------------------------------- | ----------------------------------- |
| 运营产品列表  | `frontend/src/routes/_authenticated/relay-subkeys/products/index.tsx`      | 列表筛选、状态切换入口              |
| 运营产品详情  | `frontend/src/routes/_authenticated/relay-subkeys/products/$productId.tsx` | 产品头部 + 渠道池/模型/关联 Key Tab |
| 运营 Key 列表 | `frontend/src/routes/_authenticated/relay-subkeys/keys/index.tsx`          | 统一承载状态、余额、过期筛选        |
| 运营 Key 详情 | `frontend/src/routes/_authenticated/relay-subkeys/keys/$keyId.tsx`         | 概览、限额、最近失败与管理动作      |
| 运营账务页    | `frontend/src/routes/_authenticated/relay-subkeys/keys/$keyId.billing.tsx` | 充值、退款、流水明细                |
| 运营请求排障  | `frontend/src/routes/_authenticated/relay-subkeys/requests/index.tsx`      | 按产品/Key/渠道筛选请求             |
| 项目总览      | `frontend/src/routes/_authenticated/project/relay-subkeys/overview.tsx`    | 汇总产品、Key、余额与失败摘要       |
| 项目 Key 详情 | `frontend/src/routes/_authenticated/project/relay-subkeys/keys/$keyId.tsx` | 接入信息、最近调用、失败原因        |

#### 页面级 Query / Mutation 矩阵

| 页面                 | 主要 Query                                                 | 主要 Mutation                                                                                                             |
| -------------------- | ---------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------- |
| 产品列表 / 新建      | `useRelayProductsQuery`                                    | `useCreateRelayProductMutation`                                                                                           |
| 产品详情             | `useRelayProductDetailQuery`、`useRelayChannelPoolQuery`   | `useUpdateRelayProductMutation`、`useBindRelayChannelMutation`                                                            |
| Key 列表 / 新建      | `useRelayKeysQuery`                                        | `useCreateRelayKeyMutation`                                                                                               |
| Key 详情             | `useRelayKeyDetailQuery`、`useRelayWalletQuery`            | `useSuspendRelayKeyMutation`、`useResumeRelayKeyMutation`、`useArchiveRelayKeyMutation`、`useAdjustRelayKeyLimitMutation` |
| Key 账务页           | `useRelayWalletQuery`、`useRelayLedgerEntriesQuery`        | `useRechargeRelayWalletMutation`                                                                                          |
| 请求排障页           | `useRelayRequestTraceQuery`                                | `-`                                                                                                                       |
| 渠道健康页           | `useRelayChannelPoolHealthQuery`                           | `-`                                                                                                                       |
| 项目总览 / 产品页    | `useProjectRelayOverviewQuery`、`listProjectRelayProducts` | `-`                                                                                                                       |
| 项目 Key 列表 / 详情 | `listProjectRelayKeys`、`getProjectRelayKeyDetail`         | `-`                                                                                                                       |
| 项目用量 / 接入页    | `getProjectRelayUsageSummary`                              | `-`                                                                                                                       |

#### 与当前 TanStack Router 风格对齐的实现说明

- `route.tsx` 负责区段级 layout、`AuthGuard` / `ProjectGuard` 包装和共享页面壳，叶子 route 保持薄包装，只 import 对应 feature 页面。
- 列表筛选、Tab、分页优先放在 `validateSearch` + `Route.useSearch()`，与现有 `frontend/src/routes/_authenticated/system/index.tsx` 的写法保持一致。
- 项目侧优先复用现有 `/project/*` 上下文模式；只有当仓库先引入显式项目 URL 标识时，再把文档里的 `:projectId` 语义升级为真实路径段。
- Mutation 成功后只失效紧邻的 list/detail query；一次性明文 Key 建议通过创建成功后的本地页面状态或 dialog 透出，不写入长驻 store。
- 真实页面实现建议放在 `frontend/src/features/relay-subkeys/**`，route 文件只承担守卫、search 参数解析和页面挂载。

### 数据加载与状态管理建议

建议优先使用：

- 页面级数据：TanStack Query
- 少量跨页 UI 状态：Zustand
- 表单：React Hook Form + Zod
- 表格筛选参数：URL search params

建议的 Query hooks：

- `useRelayProductsQuery`
- `useRelayProductDetailQuery`
- `useRelayChannelPoolQuery`
- `useRelayKeysQuery`
- `useRelayKeyDetailQuery`
- `useRelayWalletQuery`
- `useRelayLedgerEntriesQuery`
- `useRelayRequestTraceQuery`
- `useRelayChannelPoolHealthQuery`
- `useProjectRelayOverviewQuery`
- `useProjectRelayUsageQuery`

建议的 Mutation hooks：

- `useCreateRelayProductMutation`
- `useUpdateRelayProductMutation`
- `useBindRelayChannelMutation`
- `useCreateRelayKeyMutation`
- `useSuspendRelayKeyMutation`
- `useResumeRelayKeyMutation`
- `useArchiveRelayKeyMutation`
- `useRechargeRelayWalletMutation`
- `useAdjustRelayKeyLimitMutation`

建议的 Zustand stores：

- `useRelayProductFilterStore`
- `useRelayKeyFilterStore`
- `useRelayRequestTraceFilterStore`
- `useRelayUiStore`

说明：业务真相应来自 Query，store 仅保存 UI 状态和轻量筛选状态，不缓存完整服务端实体。

### GraphQL / API 边界建议

建议 Relay 管理台继续走 GraphQL 主线，边界可按以下领域组织：

产品域：

- `listRelayProducts`
- `getRelayProduct`
- `createRelayProduct`
- `updateRelayProduct`
- `archiveRelayProduct`

渠道池域：

- `listRelayProductChannels`
- `bindRelayProductChannel`
- `unbindRelayProductChannel`
- `reorderRelayProductChannels`

Sub-Key 域：

- `listRelayKeys`
- `getRelayKey`
- `createRelayKey`
- `suspendRelayKey`
- `resumeRelayKey`
- `archiveRelayKey`
- `renewRelayKey`

钱包域：

- `getRelayWallet`
- `listRelayLedgerEntries`
- `rechargeRelayWallet`
- `refundRelayWallet`
- `adjustRelayWallet`

请求排障域：

- `listRelayRequests`
- `getRelayRequestDetail`
- `listRelayChannelPoolHealth`

项目侧域：

- `getProjectRelayOverview`
- `listProjectRelayProducts`
- `listProjectRelayKeys`
- `getProjectRelayKeyDetail`
- `getProjectRelayUsageSummary`

### MVP 路由边界

首版建议只把以下页面做成独立 route，其余先以内嵌 Tab 或 Drawer 承载：

运营侧：

- `/operator/relay-subkeys/products`
- `/operator/relay-subkeys/products/:productId`
- `/operator/relay-subkeys/keys`
- `/operator/relay-subkeys/keys/:keyId`

项目侧：

- `/projects/:projectId/relay-subkey/overview`
- `/projects/:projectId/relay-subkey/keys/:keyId`
- `/projects/:projectId/relay-subkey/get-started`

这样做的好处：

- 路由数量少。
- 权限判断简单。
- 更适合快速跑通 MVP。
- 后续再把 `billing`、`requests`、`channel-pool-health` 拆为独立页面，也不会推翻 URL 语义。

### Post-MVP 扩展方向

后续可以再拆为独立 route 的能力：

- 更完整的请求排障页。
- 更完整的渠道池健康看板。
- 产品审批流 / 发布流。
- Key 批量管理页。
- 项目级验证中心。
- 自定义通知与告警页。

### Page 级 loader / action / search 参数草案

#### 运营产品列表页

- 目标 route file：`frontend/src/routes/_authenticated/relay-subkeys/products/index.tsx`
- `validateSearch`
  - `status?: 'draft' | 'active' | 'archived'`
  - `providerType?: string`
  - `keyword?: string`
  - `page?: number`
  - `pageSize?: number`
- loader
  - 解析 search 参数并预取 `useRelayProductsQuery` 对应的列表数据。
  - 返回当前筛选条件、默认分页配置、是否允许创建产品的权限信息。
- action / mutation 触发点
  - 页面级不定义 TanStack action，统一走按钮触发的 `useCreateRelayProductMutation` 或状态切换 mutation。
  - 对批量上/下架建议保留为后续能力，MVP 只支持单条操作。

#### 运营产品详情页

- 目标 route file：`frontend/src/routes/_authenticated/relay-subkeys/products/$productId.tsx`
- `validateSearch`
  - `tab?: 'overview' | 'pool' | 'models' | 'keys'`
  - `channelStatus?: 'active' | 'paused' | 'all'`
- loader
  - 按 `productId` 预取 `useRelayProductDetailQuery` 与 `useRelayChannelPoolQuery`。
  - 如果 `tab === 'keys'`，可额外预取关联 Key 列表。
- action / mutation 触发点
  - `useUpdateRelayProductMutation`
  - `useBindRelayChannelMutation`
  - 渠道优先级调整可走独立排序 mutation，但首版也可先只支持保存整表。

#### 运营 Key 列表页

- 目标 route file：`frontend/src/routes/_authenticated/relay-subkeys/keys/index.tsx`
- `validateSearch`
  - `status?: 'active' | 'suspended' | 'exhausted' | 'archived'`
  - `productId?: string`
  - `projectId?: string`
  - `lowBalanceOnly?: boolean`
  - `expiredOnly?: boolean`
  - `page?: number`
  - `pageSize?: number`
- loader
  - 预取 `useRelayKeysQuery`，并把 search 条件标准化到页面。
  - 可同步预取产品下拉与项目筛选项。
- action / mutation 触发点
  - 页面按钮触发 `useCreateRelayKeyMutation`。
  - 批量冻结/恢复留待后续，不放首版。

#### 运营 Key 详情页

- 目标 route file：`frontend/src/routes/_authenticated/relay-subkeys/keys/$keyId.tsx`
- `validateSearch`
  - `tab?: 'overview' | 'limits' | 'failures'`
  - `range?: '24h' | '7d' | '30d'`
- loader
  - 预取 `useRelayKeyDetailQuery` 与 `useRelayWalletQuery`。
  - 当 `tab === 'failures'` 时可预取最近失败请求摘要。
- action / mutation 触发点
  - `useSuspendRelayKeyMutation`
  - `useResumeRelayKeyMutation`
  - `useArchiveRelayKeyMutation`
  - `useAdjustRelayKeyLimitMutation`
- 说明
  - 一次性明文 Key 不应由 loader 返回，而应由“创建成功后的本地状态或 dialog”处理。

#### 运营账务页

- 目标 route file：`frontend/src/routes/_authenticated/relay-subkeys/keys/$keyId.billing.tsx`
- `validateSearch`
  - `scene?: 'all' | 'recharge' | 'consume' | 'refund' | 'manual_adjust'`
  - `page?: number`
  - `pageSize?: number`
- loader
  - 预取 `useRelayWalletQuery` 与 `useRelayLedgerEntriesQuery`。
- action / mutation 触发点
  - `useRechargeRelayWalletMutation`
  - 退款和人工调整如 MVP 已开放，则各自作为独立 mutation。

#### 运营请求排障页

- 目标 route file：`frontend/src/routes/_authenticated/relay-subkeys/requests/index.tsx`
- `validateSearch`
  - `productId?: string`
  - `keyId?: string`
  - `channelId?: string`
  - `status?: 'failed' | 'completed' | 'all'`
  - `timeRange?: '1h' | '24h' | '7d'`
  - `page?: number`
- loader
  - 预取 `useRelayRequestTraceQuery`。
  - 根据筛选条件预取产品、Key、渠道下拉数据。
- action / mutation 触发点
  - 无首版页面级 action；详情 Drawer 中如需重试/冻结，建议跳转到对应 Key 或产品页面处理。

#### 项目总览页

- 目标 route file：`frontend/src/routes/_authenticated/project/relay-subkeys/overview.tsx`
- `validateSearch`
  - `tab?: 'overview' | 'products' | 'keys'`
  - `range?: '24h' | '7d' | '30d'`
- loader
  - 在 `ProjectGuard` 通过后预取 `useProjectRelayOverviewQuery`。
  - 如当前项目上下文由上层 route 提供，则直接读取 project context，不在 URL 中重复携带项目标识。
- action / mutation 触发点
  - 无；该页以只读汇总为主。

#### 项目 Key 详情页

- 目标 route file：`frontend/src/routes/_authenticated/project/relay-subkeys/keys/$keyId.tsx`
- `validateSearch`
  - `tab?: 'access' | 'usage' | 'failures'`
  - `range?: '24h' | '7d' | '30d'`
- loader
  - 预取 `getProjectRelayKeyDetail`。
  - 若 `tab === 'usage'`，可追加预取最近账务或 token 聚合。
- action / mutation 触发点
  - 项目侧首版不建议开放写操作；复制 Key、复制 Base URL、验证接入由本地 UI 行为完成。

#### 项目接入说明页

- 目标 route file：`frontend/src/routes/_authenticated/project/relay-subkeys/get-started.tsx`
- `validateSearch`
  - `provider?: 'openai' | 'anthropic' | 'codex'`
  - `keyId?: string`
- loader
  - 预取项目侧可见产品、默认推荐 Key、SDK 示例所需元数据。
- action / mutation 触发点
  - 无；如提供“验证接入”按钮，应跳到 verify 页或触发一次轻量校验 mutation。

#### 统一实现说明

- 所有 search 参数优先使用 `validateSearch` 标准化，并通过 `Route.useSearch()` 驱动列表筛选、Tab 和分页。
- 需要权限判断的 route，仍由 `route.tsx`、`AuthGuard`、`ProjectGuard` 控制，loader 不重复实现鉴权逻辑。
- loader 只负责“首屏所需数据”和 search 参数归一化；真正的刷新、失效与重取仍交给 TanStack Query。
- 若后续引入 TanStack Router action，可优先用于简单表单提交；当前 MVP 保持 mutation hook 驱动更贴近现有实现。

### UI 状态与空态建议

#### 列表页状态

- `loading`：列表表格使用 skeleton 行，顶部筛选栏保持可见，避免用户误判页面空白。
- `empty`：在无数据时给出明确 CTA，例如“创建产品”或“联系运营发放 Key”。
- `filtered-empty`：当筛选后无结果时，保留筛选条件并提供“一键清空筛选”。
- `error`：保留上次 search 参数，展示可重试提示，不自动清空当前筛选。

#### 详情页状态

- `loading`：头部卡片先渲染基础骨架，Tab 内容延迟加载。
- `not-found`：产品或 Key 不存在时，跳转到列表页并带 toast 提示。
- `forbidden`：权限不足时显示受限提示，而不是伪装成 404。
- `partial-degraded`：当详情页主实体可读但关联 query 失败时，仅在对应区块内展示错误态，不阻塞整页。

#### 操作反馈

- `success`：创建、暂停、恢复、充值等操作统一使用 toast + 轻量局部刷新。
- `submitting`：按钮进入 loading 态，同时禁用重复点击。
- `failure`：优先展示后端返回的可解释错误，如余额不足、限额冲突、共享池不可用。

### Query Invalidation 与刷新建议

#### 产品域

- 创建产品成功后：失效 `useRelayProductsQuery`。
- 更新产品成功后：失效 `useRelayProductDetailQuery` 与 `useRelayProductsQuery`。
- 渠道池绑定变更后：失效 `useRelayChannelPoolQuery`，必要时同时失效产品详情 query。

#### Key 域

- 创建 Key 成功后：失效 `useRelayKeysQuery`；如果创建在产品详情页内触发，同时失效产品关联 Key 列表。
- 暂停 / 恢复 / 归档成功后：失效 `useRelayKeyDetailQuery` 与 `useRelayKeysQuery`。
- 调整限额后：仅失效当前 Key 详情 query；如列表页展示限额摘要，可追加失效列表 query。

#### 钱包与账务域

- 充值 / 退款 / 人工调整成功后：失效 `useRelayWalletQuery`、`useRelayLedgerEntriesQuery`，并按需失效 Key 详情 query。
- 若项目总览页展示余额聚合：额外失效 `useProjectRelayOverviewQuery`。

#### 项目侧只读域

- 项目总览与用量页默认按页面进入时加载，不做高频自动轮询。
- 仅在用户停留于“最近请求 / 最近失败”Tab 时，才为该 Tab 局部启用短周期刷新。

### 页面验收标准草案

#### 运营产品列表页

- 支持按状态、providerType、关键字筛选。
- 列表数据与 URL search 参数保持一致，刷新页面后条件不丢失。
- 创建产品成功后，返回列表可立即看到新产品。

#### 运营产品详情页

- 支持切换 `overview/pool/models/keys` Tab。
- 渠道池绑定变更后，详情页与健康摘要同步刷新。
- 当上游池完全不可用时，页面能明确提示不能切到 `active`。

#### 运营 Key 列表与详情页

- 能区分 `active/suspended/exhausted/archived` 四类持久化状态。
- 能展示 `expired/low_balance/quota_reached/upstream_pool_degraded` 等派生 badge。
- 暂停、恢复、归档操作后，列表与详情状态一致。

#### 运营账务页

- 余额卡片、流水表格、充值动作三者保持一致刷新。
- 充值成功后无需手动刷新即可看到新余额与最新流水。
- 失败时能展示后端返回原因，不出现静默失败。

#### 项目总览 / Key 详情 / 接入说明页

- 项目侧页面不暴露运营动作按钮。
- 项目侧能稳定展示 Base URL、掩码 Key、推荐模型与最近失败原因。
- 接入说明页切换 provider 示例时，URL search 参数和代码示例同步变化。

### 文档与实现同步提醒

- 若后续前端真实路由改为显式 `projects/:projectId/*` 结构，需要同步修正文档中的“项目上下文复用”说明。
- 若后续全面引入 TanStack Router loader/action，应回补本节，把当前“mutation hook 驱动”的假设改成真实实现。
- 若项目决定增加审批流或申请流，应把本篇中的资源中心式路由树拆分为申请、审批、绑定三条业务流并重写验收标准。

### UI copy 文案库

#### 空态文案

| 场景             | 标题                 | 辅助文案                                           | 主按钮       |
| ---------------- | -------------------- | -------------------------------------------------- | ------------ |
| 产品列表为空     | 还没有共享容量产品   | 先创建一个产品，再绑定上游渠道池和可售 Key。       | 创建产品     |
| Key 列表为空     | 还没有可用的 Sub-Key | 创建 Key 后，项目方才能看到可接入的共享容量入口。  | 创建 Sub-Key |
| 项目侧无可见产品 | 当前项目暂无可用产品 | 请联系平台运营分配产品或确认项目权限范围。         | 联系运营     |
| 筛选后无结果     | 没有匹配的结果       | 当前筛选条件下没有找到数据，可尝试清空筛选后重试。 | 清空筛选     |
| 最近请求为空     | 暂无请求记录         | 首次调用成功后，这里会展示最近请求与失败摘要。     | 查看接入说明 |

#### Toast 文案

| 场景          | 成功文案             | 失败文案                                |
| ------------- | -------------------- | --------------------------------------- |
| 创建产品      | 产品已创建           | 创建产品失败，请稍后重试                |
| 更新产品      | 产品配置已保存       | 保存产品配置失败，请检查输入后重试      |
| 绑定渠道池    | 渠道已加入共享池     | 绑定渠道失败，请确认渠道状态与权限      |
| 创建 Key      | Sub-Key 已创建       | 创建 Sub-Key 失败，请检查项目和产品配置 |
| 暂停 Key      | Key 已暂停           | 暂停 Key 失败，请稍后重试               |
| 恢复 Key      | Key 已恢复           | 恢复 Key 失败，请检查当前状态           |
| 归档 Key      | Key 已归档           | 归档 Key 失败，请稍后重试               |
| 充值          | 充值成功，余额已更新 | 充值失败，请检查金额或稍后重试          |
| 复制 Base URL | 已复制 Base URL      | 复制失败，请手动复制                    |
| 复制 Key      | 已复制 Key           | 复制失败，请手动复制                    |

#### 错误提示文案

| 错误类型     | 建议文案                                            |
| ------------ | --------------------------------------------------- |
| 余额不足     | 当前 Key 余额不足，请联系运营充值后再试。           |
| 达到日限额   | 当前 Key 已达到今日限额，请明日再试或切换其他 Key。 |
| Key 已暂停   | 当前 Key 已被暂停，请联系平台运营确认原因。         |
| Key 已归档   | 当前 Key 已归档，不再支持新的请求。                 |
| 产品池不可用 | 当前共享池暂时不可用，请稍后重试或联系运营排查。    |
| 权限不足     | 您当前没有访问此页面或执行该操作的权限。            |
| 数据不存在   | 请求的产品或 Key 不存在，可能已被删除或归档。       |
| 加载失败     | 数据加载失败，请稍后重试。                          |

#### 危险操作确认文案

| 操作     | 标题               | 确认文案                                                   | 确认按钮 |
| -------- | ------------------ | ---------------------------------------------------------- | -------- |
| 暂停 Key | 确认暂停当前 Key？ | 暂停后，新请求会立即被拒绝，但历史数据仍可查看。           | 确认暂停 |
| 恢复 Key | 确认恢复当前 Key？ | 恢复后，该 Key 会重新参与请求校验与路由。                  | 确认恢复 |
| 归档 Key | 确认归档当前 Key？ | 归档后，Key 将从默认列表中移除，且不再接受新请求。         | 确认归档 |
| 移除渠道 | 确认移除该渠道？   | 移除后，产品可用容量可能下降，必要时请先确认其他渠道可用。 | 确认移除 |

#### Skeleton / Loading 文案建议

- 产品列表：保留筛选栏，仅将表格区切为 skeleton 行。
- 产品详情：先展示头部卡片骨架，再按 Tab 延迟加载下方内容。
- Key 详情：顶部状态卡片先出，账务与失败记录区域独立 loading。
- 项目接入说明页：代码示例区域可显示“正在生成示例配置...”。

#### Empty / Error 交互约定

- 空态优先给下一步操作，不只说“暂无数据”。
- Error 态优先保留 search 参数与用户输入，避免用户重复填写。
- 禁止把 `forbidden` 和 `not-found` 用同一套文案处理。
- 对于共享池故障，优先说明“是池状态问题，不一定是当前 Key 自身问题”。

### 按钮文案与禁用条件矩阵

#### 主操作按钮

| 按钮         | 默认文案     | loading 文案  | 禁用条件                                     |
| ------------ | ------------ | ------------- | -------------------------------------------- |
| 创建产品     | 创建产品     | 正在创建...   | 表单未通过校验、必填项缺失、当前用户无写权限 |
| 保存产品配置 | 保存配置     | 正在保存...   | 无变更、表单校验失败、渠道池配置不合法       |
| 创建 Sub-Key | 创建 Sub-Key | 正在创建...   | 未选择项目、未绑定产品、关键限额字段非法     |
| 保存限额     | 保存限额     | 正在保存...   | 限额未变化、值超出允许范围                   |
| 立即充值     | 立即充值     | 正在充值...   | 金额为空、金额小于最小值、当前用户无账务权限 |
| 重新加载     | 重新加载     | 重新加载中... | 当前已有进行中的同类请求                     |

#### 危险操作按钮

| 按钮     | 默认文案 | loading 文案 | 禁用条件                                   |
| -------- | -------- | ------------ | ------------------------------------------ |
| 暂停 Key | 暂停 Key | 正在暂停...  | Key 当前已是 `suspended` 或 `archived`     |
| 恢复 Key | 恢复 Key | 正在恢复...  | Key 当前不是 `suspended` / `exhausted`     |
| 归档 Key | 归档 Key | 正在归档...  | Key 当前已是 `archived`                    |
| 移除渠道 | 移除渠道 | 正在移除...  | 当前产品仅剩最后一个可用渠道且没有替代渠道 |

#### 项目侧只读按钮

| 按钮          | 默认文案      | loading 文案 | 禁用条件                           |
| ------------- | ------------- | ------------ | ---------------------------------- |
| 复制 Base URL | 复制 Base URL | 正在复制...  | Base URL 为空                      |
| 复制 Key      | 复制 Key      | 正在复制...  | 当前页面不允许显示或复制 Key       |
| 查看接入说明  | 查看接入说明  | 打开中...    | 无                                 |
| 验证接入      | 验证接入      | 正在验证...  | 当前没有可用 Key、共享池状态不可用 |

#### 禁用态展示约定

- 按钮禁用时优先保留按钮位置，不要因禁用而隐藏，避免用户误以为功能不存在。
- 对于权限导致的禁用，优先展示 tooltip：`您没有执行该操作的权限`。
- 对于状态导致的禁用，tooltip 应尽量复用具体原因，例如：`当前 Key 已归档`、`当前没有可用渠道可移除`。
- 对于表单未完成导致的禁用，不建议只写“不可用”，应明确指出缺失项，例如：`请先选择项目和产品`。

#### 次级按钮文案建议

- 返回列表：`返回列表`
- 查看详情：`查看详情`
- 清空筛选：`清空筛选`
- 展开全部：`展开全部`
- 收起：`收起`
- 查看最近失败：`查看最近失败`
- 查看账务流水：`查看账务流水`

### 表格列标题与 tooltip 文案库

#### 产品列表表格

| 列标题        | Tooltip 文案                                                                   |
| ------------- | ------------------------------------------------------------------------------ |
| 产品名称      | 用于区分共享容量产品的展示名称，面向运营与项目管理员可见。                     |
| Provider 类型 | 标识该产品面向的 provider / 协议类型，例如 Codex、Claude Code 或 OpenAI 兼容。 |
| 状态          | 当前产品是否处于草稿、启用或归档状态。                                         |
| 允许模型数    | 该产品当前允许访问的模型数量，用于快速判断产品覆盖范围。                       |
| 渠道池健康度  | 当前产品绑定渠道池的整体可用情况，优先用于排查共享池风险。                     |
| 最近更新时间  | 最近一次修改产品配置的时间。                                                   |

#### Key 列表表格

| 列标题       | Tooltip 文案                                                          |
| ------------ | --------------------------------------------------------------------- |
| Key 名称     | 该 Sub-Key 的展示名称，便于按项目或用途识别。                         |
| 所属项目     | 当前 Key 归属的项目上下文。                                           |
| 绑定产品     | 当前 Key 对应的共享容量产品。                                         |
| 状态         | 持久化状态，如 `active`、`suspended`、`exhausted`、`archived`。       |
| 可用余额     | 当前可用于请求结算的余额，不含冻结金额。                              |
| 到期时间     | Key 的失效时间，到期后应触发过期提示或续期动作。                      |
| 最近使用时间 | 最近一次成功或进入主链路请求的时间。                                  |
| 派生状态     | 运行时 badge，如 `expired`、`low_balance`、`upstream_pool_degraded`。 |

#### 账务流水表格

| 列标题     | Tooltip 文案                                         |
| ---------- | ---------------------------------------------------- |
| 流水时间   | 本条账务流水写入时间。                               |
| 类型       | 表示是充值、消费、退款还是人工调整。                 |
| 金额       | 本次流水变化的金额，正负方向需结合类型理解。         |
| 余额变更后 | 该流水落库后的余额快照。                             |
| 关联请求   | 若本条流水来自请求结算，可通过该字段跳转到对应请求。 |
| 操作人     | 发起人工调整或账务动作的运营账号。                   |
| 备注       | 对该流水的人工说明或系统生成说明。                   |

#### 请求排障表格

| 列标题   | Tooltip 文案                                         |
| -------- | ---------------------------------------------------- |
| 请求时间 | 请求进入平台主链路的时间。                           |
| 产品     | 当前请求关联的共享容量产品。                         |
| Key      | 当前请求使用的 Sub-Key。                             |
| 目标模型 | 用户请求的模型标识。                                 |
| 命中渠道 | 实际执行请求的上游渠道。                             |
| 响应状态 | 请求最终状态，如成功、失败或被拒绝。                 |
| 失败阶段 | 若失败，标识失败发生在鉴权、路由、执行还是结算阶段。 |
| 扣费结果 | 当前请求是否成功完成扣费。                           |
| Trace ID | 用于串联请求追踪、排障与日志检索的唯一标识。         |

#### 项目侧用量表格

| 列标题       | Tooltip 文案                                     |
| ------------ | ------------------------------------------------ |
| 日期         | 当前聚合数据的统计日期。                         |
| 请求数       | 该时间粒度下的总请求次数。                       |
| Token 消耗   | 已结算的总 token 数，用于观察消耗趋势。          |
| 余额变化     | 该时间粒度内的余额净变化。                       |
| 主要失败原因 | 当前周期内最常见的失败摘要，用于辅助项目侧排障。 |

#### Tooltip 使用约定

- Tooltip 优先解释“字段语义”，不重复列标题本身。
- 若列值已经非常直白，Tooltip 可省略，避免悬停噪音过多。
- 对状态类列，Tooltip 优先解释“状态含义 + 对用户的影响”。
- 对账务类列，Tooltip 优先解释“金额口径”和“是否已包含冻结/结算后结果”。

### SDK 接入反馈文案

#### 复制反馈文案

| 场景                | 成功文案                  | 失败文案                           |
| ------------------- | ------------------------- | ---------------------------------- |
| 复制 Base URL       | 已复制 Base URL           | Base URL 复制失败，请手动复制      |
| 复制 Sub-Key        | 已复制 Sub-Key            | Sub-Key 复制失败，请手动复制       |
| 复制当前示例        | 已复制当前示例代码        | 当前示例复制失败，请手动复制       |
| 复制 OpenAI 示例    | 已复制 OpenAI 示例代码    | OpenAI 示例复制失败，请手动复制    |
| 复制 Anthropic 示例 | 已复制 Anthropic 示例代码 | Anthropic 示例复制失败，请手动复制 |
| 复制 Codex 示例     | 已复制 Codex 示例代码     | Codex 示例复制失败，请手动复制     |

#### 接入验证反馈文案

| 场景                       | 文案                                                                    |
| -------------------------- | ----------------------------------------------------------------------- |
| 测试请求进行中             | 正在发起测试请求并验证当前接入配置，请稍候...                           |
| 验证成功                   | 接入验证成功，当前 Sub-Key 可正常请求共享容量服务。                     |
| 验证成功（降级池）         | 接入验证成功：当前 Relay 池处于降级状态，已回退到降级池，响应可能略慢。 |
| 验证失败（缺少配置）       | 接入验证失败：请先补全 Base URL、Sub-Key 与模型名称后再试。             |
| 验证失败（鉴权）           | 接入验证失败：请检查 Sub-Key 是否正确、是否已过期或已被停用。           |
| 验证失败（权限不足）       | 接入验证失败：当前 Sub-Key 无权访问该产品或模型，请联系运营开通权限。   |
| 验证失败（共享池不可用）   | 接入验证失败：当前共享池暂时不可用，请稍后重试。                        |
| 验证失败（模型不可用）     | 接入验证失败：当前产品未开放所选模型，请切换模型后重试。                |
| 验证失败（网络）           | 接入验证失败：网络请求未完成，请检查网络或稍后重试。                    |
| 验证失败（降级池也不可用） | 接入验证失败：主 Relay 池与降级池均不可用，请稍后重试或联系运营。       |

#### 接入提示文案

- 请使用 AxonHub 下游 Sub-Key，而不是上游 provider 原生密钥。
- 若复制后仍无法请求，请优先检查 Base URL、Sub-Key 和模型名称是否与页面示例一致。
- 若测试请求提示权限不足，优先确认当前项目是否已绑定目标产品、模型或套餐额度。
- 当 Relay 池处于降级状态时，页面应明确提示“已切换到降级池”，避免用户误判为 Key 整体不可用。
- 若长时间验证失败，请联系平台运营并提供最近一次请求时间与 Trace ID。

#### 示例切换文案

- OpenAI 示例
- Anthropic 示例
- Codex 示例
- 复制当前示例
- 已切换到 OpenAI 示例
- 已切换到 Anthropic 示例
- 已切换到 Codex 示例
- 当前 provider 暂无专属示例，已切换到默认 OpenAI 示例

#### SDK 页空态 / 错误态建议

- 缺少接入配置：请先选择产品并生成 Sub-Key，系统才能展示可复制的接入信息。
- 无可用 Sub-Key：当前项目暂无可用于接入的 Sub-Key，请联系运营分配。
- 无可用模型：当前产品暂无可用模型，请稍后刷新或联系运营确认。
- 权限不足：当前项目暂无 SDK 接入权限，请联系运营开通后重试。
- Relay 池降级：当前共享池处于降级状态，已展示可用的降级接入方式，建议先完成联调。
- 示例生成失败：示例配置加载失败，请稍后重试。
- 验证记录为空：完成首次验证后，这里会展示最近一次验证结果。
