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

| 角色 | 典型身份 | 主要页面 | 核心动作 | 不在 MVP 的动作 |
| --- | --- | --- | --- | --- |
| 平台运营 | Relay 部署者、管理员 | 产品页、渠道池页、Sub-Key 列表、账务页、请求排障页 | 创建产品、绑定渠道、发放/冻结/归档 Key、充值、退款、查看请求明细 | 自动定价实验、代理分润 |
| 项目管理员 | 买方团队 owner | 项目下的产品接入页、Sub-Key 详情、用量与账单页 | 查看可用产品、复制接入凭证、设置显示名、查看余额/限额/过期时间 | 自助绑定上游 provider 凭证 |
| 开发者 | 调用 Relay API 的工程师 | 接入说明、Key 详情、请求日志 | 复制 Base URL / API Key、排查最近失败请求 | 修改计费策略、管理共享池 |
| 风控/客服 | 运营支持角色 | Key 详情、流水页、请求跟踪页 | 人工冻结、备注、退款、解释失败原因 | 独立审批流 |

实现建议：权限层面优先复用现有 `project_id` 隔离与后台角色能力，不单独设计新 ACL，只增加页面级动作开关。

## Key 状态模型

### 持久化状态

`relay_keys.status` 建议直接使用后端设计中的 4 个持久化状态：

| 状态 | 页面展示 | 请求行为 | 允许动作 |
| --- | --- | --- | --- |
| `active` | 正常、可调用 | 允许通过同步校验后继续转发 | 充值、改名、暂停、归档 |
| `suspended` | 已暂停 | 直接拒绝请求 | 恢复、归档、备注 |
| `exhausted` | 已耗尽 | 余额不足或硬配额达到上限时拒绝请求 | 充值后恢复、调整限额、归档 |
| `archived` | 已归档 | 永久拒绝请求，不再出现在默认列表 | 仅查看历史 |

### 运行时派生状态

以下状态不一定单独落库，但必须在页面上有清晰 badge：

| 派生状态 | 判定来源 | 页面用途 |
| --- | --- | --- |
| `expired` | `expires_at < now()` | 告知 Key 已过期，需要续期或重发 |
| `low_balance` | `relay_wallets.available_amount` 低于阈值 | 在列表和详情页提前预警 |
| `quota_reached` | `relay_daily_usage_summaries` 或月度聚合超过限制 | 标记为什么进入 `exhausted` |
| `concurrency_blocked` | 当前并发超过 `concurrency_limit` | 请求失败时给出可解释错误 |
| `upstream_pool_degraded` | 关联产品的候选渠道不足或全部 unhealthy | 提示是共享池问题，而不是单 Key 问题 |

实现建议：列表页展示“持久化状态 + 派生 badge”双层信息，避免把上游池故障误判成用户余额问题。

## 页面清单

### 运营侧页面

| 页面 | 建议路由 | 主要角色 | 依赖数据 | 核心动作 |
| --- | --- | --- | --- | --- |
| 共享容量产品列表 | `/console/relay/products` | 平台运营 | `relay_products` | 查看产品状态、上下架、进入详情 |
| 产品详情与渠道池配置 | `/console/relay/products/:id` | 平台运营 | `relay_products`、`relay_product_channels`、`channels`、`provider_quota_status` | 编辑产品信息、绑定/解绑渠道、调整优先级/权重、限制模型 |
| Sub-Key 列表 | `/console/relay/keys` | 平台运营、客服 | `relay_keys`、`relay_wallets`、项目信息 | 筛选状态、搜索项目、查看低余额与过期 Key |
| Sub-Key 详情 | `/console/relay/keys/:id` | 平台运营、客服 | `relay_keys`、`relay_wallets`、`relay_daily_usage_summaries`、最近 `requests` | 暂停、恢复、归档、改名、调整到期时间 |
| 充值/流水页 | `/console/relay/keys/:id/billing` | 平台运营、客服 | `relay_wallets`、`relay_wallet_ledger_entries` | 充值、退款、人工调整、查看账务凭证 |
| 请求跟踪页 | `/console/relay/requests` | 平台运营、客服 | `requests`、`request_executions`、`usage_logs` | 按 Key / 产品 / 渠道排查失败、查看扣费结果 |
| 渠道健康看板 | `/console/relay/channel-pool-health` | 平台运营 | `relay_product_channels`、`channels`、`provider_quota_status` | 判断某产品是否存在上游容量风险 |

### 买方项目侧页面

| 页面 | 建议路由 | 主要角色 | 依赖数据 | 核心动作 |
| --- | --- | --- | --- | --- |
| 产品接入页 | `/projects/:projectId/relay/products` | 项目管理员 | 项目可见产品、产品说明、模型范围 | 了解可购买/可使用的产品，进入 Key 列表 |
| 项目 Sub-Key 列表 | `/projects/:projectId/relay/keys` | 项目管理员 | `relay_keys`、`relay_wallets` | 查看项目内可用 Key、复制接入信息 |
| 项目 Sub-Key 详情 | `/projects/:projectId/relay/keys/:id` | 项目管理员、开发者 | `relay_keys`、钱包快照、最近请求、SDK 示例 | 复制 API Key / Base URL、查看状态、最近失败原因 |
| 用量与账单页 | `/projects/:projectId/relay/usage` | 项目管理员 | `relay_daily_usage_summaries`、`relay_wallet_ledger_entries`、`usage_logs` 汇总 | 查看余额、消耗趋势、最近扣费 |
| 接入说明页 | `/projects/:projectId/relay/get-started` | 开发者 | 产品允许模型、示例请求、错误码说明 | 快速接入并理解共享容量模型 |

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

| 触发动作 | 迁移前 | 迁移后 | 页面入口 |
| --- | --- | --- | --- |
| 创建 Key | 无 | `active` 或 `suspended` | 新建 Sub-Key 弹窗 |
| 余额耗尽 | `active` | `exhausted` | 自动触发，无需人工页面 |
| 充值恢复 | `exhausted` | `active` | 余额与流水 Tab |
| 人工暂停 | `active` / `exhausted` | `suspended` | Key 详情页 |
| 恢复使用 | `suspended` | `active` | Key 详情页 |
| 归档 | 任意非归档状态 | `archived` | Key 详情页 |
| 到期后续期 | `active` + `expired` badge | `active` | Key 详情页修改有效期 |

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

文档需要明确区分两层概念：
- `URL 路由树 / 信息架构`：面向用户的导航与职责划分。
- `file-based route 命名`：挂在现有 `__root`、`_auth`、operator layout、project layout 之下的实现方式。

可参考的实现层命名示例：

```text
frontend/src/routes/
  __root.tsx
  _auth.operator.relay-subkeys.products.index.tsx
  _auth.operator.relay-subkeys.products.create.tsx
  _auth.operator.relay-subkeys.products.$productId.index.tsx
  _auth.operator.relay-subkeys.keys.index.tsx
  _auth.operator.relay-subkeys.keys.create.tsx
  _auth.operator.relay-subkeys.keys.$keyId.index.tsx
  _auth.operator.relay-subkeys.keys.$keyId.billing.tsx
  _auth.operator.relay-subkeys.requests.tsx
  _auth.operator.relay-subkeys.channel-pool-health.tsx

  _auth.projects.$projectId.relay-subkey.overview.tsx
  _auth.projects.$projectId.relay-subkey.products.tsx
  _auth.projects.$projectId.relay-subkey.keys.tsx
  _auth.projects.$projectId.relay-subkey.keys.$keyId.tsx
  _auth.projects.$projectId.relay-subkey.usage.tsx
  _auth.projects.$projectId.relay-subkey.get-started.tsx
  _auth.projects.$projectId.relay-subkey.verify.tsx
```

### 页面职责与 Flow 映射

#### 运营侧页面

| 页面 | 路由 | 对应 Flow | 核心职责 |
| --- | --- | --- | --- |
| 产品列表 | `/operator/relay-subkeys/products` | 运营创建共享容量产品 | 查看状态、筛选、上下架、进入详情 |
| 产品新建 | `/operator/relay-subkeys/products/create` | 运营创建共享容量产品 | 填写产品编码、名称、provider 类型、模型范围 |
| 产品详情 | `/operator/relay-subkeys/products/:productId` | 产品详情与渠道池配置 | 配置渠道池、优先级、权重、模型过滤 |
| Sub-Key 列表 | `/operator/relay-subkeys/keys` | 运营为项目发放 Sub-Key | 查看所有 Key、筛选状态、识别低余额和过期 Key |
| Sub-Key 新建 | `/operator/relay-subkeys/keys/create` | 运营为项目发放 Sub-Key | 选择项目、产品、余额模式、有效期、硬限额 |
| Sub-Key 详情 | `/operator/relay-subkeys/keys/:keyId` | Key 管理、暂停、恢复、归档 | 查看概览、最近失败原因、执行管理动作 |
| 账务页 | `/operator/relay-subkeys/keys/:keyId/billing` | 充值/流水页 | 充值、退款、人工调整、查看账务凭证 |
| 请求排障页 | `/operator/relay-subkeys/requests` | 共享池故障排查 | 按 Key / 产品 / 渠道检索失败请求 |
| 渠道健康看板 | `/operator/relay-subkeys/channel-pool-health` | 共享池故障排查 | 查看产品池健康、识别上游容量风险 |

#### 项目侧页面

| 页面 | 路由 | 对应 Flow | 核心职责 |
| --- | --- | --- | --- |
| 总览页 | `/projects/:projectId/relay-subkey/overview` | 项目管理员查看并接入 Key | 汇总产品、Key、余额和最近失败 |
| 产品接入页 | `/projects/:projectId/relay-subkey/products` | 项目管理员查看产品范围 | 查看项目可见产品和模型说明 |
| Key 列表 | `/projects/:projectId/relay-subkey/keys` | 项目管理员查看并接入 Key | 查看项目内可用 Key、复制接入信息 |
| Key 详情 | `/projects/:projectId/relay-subkey/keys/:keyId` | 项目管理员查看并接入 Key | 复制 API Key / Base URL、查看状态和最近失败 |
| 用量页 | `/projects/:projectId/relay-subkey/usage` | 请求成功并完成扣费 | 查看余额、消耗趋势、最近扣费 |
| 接入说明页 | `/projects/:projectId/relay-subkey/get-started` | 项目管理员查看并接入 Key | 提供 SDK 示例、错误码说明、接入提醒 |
| 验证页 | `/projects/:projectId/relay-subkey/verify` | 接入验证 | 执行简单联通校验并展示最近验证结果 |

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
