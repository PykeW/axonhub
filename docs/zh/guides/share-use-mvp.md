# 共享/使用 MVP 指南

本文记录简化版“共享/使用”方案的产品边界、数据约定、调度规则、当前实现状态与后续验收标准。它既是产品计划，也是当前仓库实现与后续工作之间的对照入口。
当前仓库也分成两层状态：已提交的小切片已经打通 `APIKeyProfile.useStrategy` 的 GraphQL / schema 保存链路，并在对象层提供 `ChannelSettings.Share` 元数据与默认值；当前工作树又在其上恢复了 `/use` 最小可用表单。`/share` 仍是指南包装页，真正的共享路由、刷新窗口强约束和更完整的 Use 配置体验仍是后续工作。

## 范围

### 页面

| 页面 | 用户目标                            | 目标 MVP 能力（后续验收）                                                 |
| ---- | ----------------------------------- | ------------------------------------------------------------------------- |
| 共享 | 把自己的上游账号贡献给 AxonHub 使用 | 创建/编辑/归档自己的渠道，设置 `private` 或 `shared`，查看 5 小时刷新窗口 |
| 使用 | 用简单 API Key 调用可用模型         | 创建或编辑 API Key，选择 `modelIDs`，选择 `useStrategy`                   |

### 当前已落地切片

- 前端：`/share` 仍是指南包装路由；`/use` 已恢复为最小可用表单，支持创建 API Key、选择 `modelIDs`、选择 `useStrategy`、保存 profile 并复制生成结果；当前也支持通过 `/use?apiKeyId=<id>` 直接加载并编辑既有 user API Key 的当前生效 profile。
- 前端联动：API Keys 列表行操作已新增 `Use 配置` 入口，可直接深链到对应 `/use` 编辑页。
- 后端 / 数据契约：`APIKeyProfile.useStrategy` 已贯通 GraphQL schema、前端 Zod schema 和 update mutation；`ChannelSettings.Share` 元数据 / 默认值已存在于对象层，但渠道 schema / 前端表单尚未暴露 share 字段。
- 兼容性：现有渠道管理、API Key Profile、配额、模型映射、负载均衡和重试行为保持原状。

### 尚未实现的完整行为

- Share 页面真实创建 / 编辑 / 归档表单、所有者权限、敏感字段脱敏、5 小时刷新窗口与并发刷新控制。
- 渠道 schema / API / 前端表单尚未暴露 `ChannelSettings.Share`、owner/visibility、`nextRefreshAt` 等真实字段，因此 Share 语义仍未接入实体 CRUD。
- 请求路由尚未按 `useStrategy` 做自有 / 共享候选过滤、own-first、soonest-refresh-first、配额强制和可观测日志。
- `/use` 已支持通过 `/use?apiKeyId=<id>` 深链加载既有 user API Key，并编辑当前 active profile 的 `modelIDs/useStrategy`；下一步再补更完整的高级 Profile 字段（如 quota / channelTags / modelMappings）编辑体验。
- 面向共享容量的端到端测试、前端 E2E 和完整后端 selector / orchestrator 回归仍需补齐。

### 不做的事

- 不引入复杂资源池、市场、结算或多租户套餐模型。
- 不要求普通用户理解 API Key Profile、渠道标签或完整负载均衡配置。
- 不改变现有管理员渠道管理、模型映射、配额和重试能力；MVP 只在其上增加用户视角。

## 下一步开发计划

当前最小闭环已经能创建 / 编辑 Use API Key 的 `modelIDs/useStrategy`。下一步应先把 Share 的实体语义落到数据契约和权限层，再接入请求路由；否则前端即使先做完整表单，也无法保证共享渠道的安全边界、刷新窗口和调度行为一致。

### 推荐实施顺序

| 优先级 | 阶段                       | 目标                                                                   | 主要落点                                                          | 完成标准                                                                                              |
| ------ | -------------------------- | ---------------------------------------------------------------------- | ----------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------- |
| P0     | 稳定当前 Use 闭环          | 给已完成的 `/use` 创建 / 编辑能力补回归保护                            | `/use` 页面、API Keys 行操作、API Key Profile mutation            | 创建模式、`apiKeyId` 编辑模式、无模型保存失败、无权限提示均有测试或手工验收记录                       |
| P1     | Share 数据契约实体化       | 在渠道 schema / API 中暴露 owner、visibility、刷新窗口字段             | Channel schema、GraphQL input/output、迁移默认值                  | 历史渠道有默认 owner/visibility 策略；`private/shared` 非法值被拒绝；敏感字段默认不外泄               |
| P2     | Share 服务端权限与刷新窗口 | 实现普通用户只管理自有渠道、shared 脱敏读取、5 小时刷新冷却和并发保护  | Channel service、permission scope、refresh mutation               | 非所有者无法读写 private；shared 只暴露可路由信息；同渠道并发刷新只有一次成功                         |
| P3     | Share 前端真实页面         | 用真实列表和表单替换 `/share` 指南包装页                               | `/share` route、渠道表单、状态提示、i18n                          | 可创建 / 编辑 / 归档自有渠道；可见性、支持模型、默认测试模型、刷新冷却状态可见                        |
| P4     | Use 策略接入请求路由       | 按 `useStrategy`、owner/visibility 和 `nextRefreshAt` 过滤排序候选渠道 | request processing、selector/orchestrator、load balancer 前置过滤 | `only_own` 不使用共享；`prefer_own` 自有优先且可回退；`allow_shared` 保留共享候选；旧配额和重试不回归 |
| P5     | Use 高级 Profile 编辑      | 在最小表单之上补 quota、channelTags、modelMappings 等高级字段          | `/use` 页面、API Key Profile UI、schema 校验                      | 默认模式仍简单；高级模式只覆盖用户明确编辑的字段，不破坏既有 profile                                  |
| P6     | 可观测与端到端验收         | 补齐共享容量链路的日志、指标、E2E 和后端回归                           | orchestrator logs、frontend e2e、Go tests                         | 日志包含策略和候选桶；Share/Use 主流程有 e2e；测试矩阵中的目标项可稳定运行                            |

### 最近两个开发切片

1. **P0：当前 Use 闭环回归**

   - 给 `/use` 创建模式补前端测试：选择模型、保存 `prefer_own`、复制生成 key。
   - 给 `/use?apiKeyId=<id>` 编辑模式补测试：加载既有 user key，仅更新 active profile 的 `modelIDs/useStrategy`，保留 `quota/channelTags/modelMappings`。
   - 给 API Keys 行操作补断言：只有 user key 显示 `Use 配置`，点击后带 `apiKeyId` 进入 `/use`。
   - 验证命令优先使用 `cd frontend && npm run build`，如有 E2E 基础数据再补 `cd frontend && npm run test:e2e -- apikeys/share-use.spec.ts`。

2. **P1/P2：Share 字段与权限后端切片**
   - 明确历史渠道的兼容默认值：管理员渠道默认 `private` 且 owner 为空时只保留管理员可见，还是迁移到创建者 / 默认项目所有者。
   - 在渠道对象和 GraphQL 层增加 `ownerUserID`、`visibility`、`supportedModels`、`defaultTestModel`、`lastRefreshedAt`、`nextRefreshAt` 的读写契约。
   - 增加服务端校验：普通用户不能改 owner；`supportedModels` 不能为空；`defaultTestModel` 必须属于 `supportedModels`；刷新冷却按服务端时间判断。
   - 先做后端单元测试，再接 `/share` 前端表单，避免 UI 先行导致权限和脱敏语义返工。

### 关键依赖和阻塞点

- **权限模型**：必须先确认“渠道所有者”到底绑定 user、project，还是二者都要记录；这会影响 private/shared 的可见范围。
- **历史数据迁移**：没有 owner/visibility 的渠道必须有明确默认值，否则路由过滤会出现灰区。
- **刷新窗口一致性**：5 小时窗口应以服务端成功刷新时间为准，并在事务或锁内更新 `nextRefreshAt`。
- **调度层插入点**：own-first 与 soonest-refresh-first 应放在现有负载均衡和重试之前，但不能绕过现有模型关联、配额、健康、熔断规则。
- **前端降级**：在后端字段未全量发布前，`/share` 和 `/use` 需要能显示“暂不可配置”的明确状态，而不是静默保存不完整数据。

## 共享页面

共享页面管理“我上传的渠道”。每个渠道仍然是 AxonHub 渠道，但必须带有所有者和可见性语义。

### 字段约定

| 字段               | 要求                                              |
| ------------------ | ------------------------------------------------- |
| `ownerUserID`      | 渠道所有者；普通用户只能管理自己的渠道            |
| `visibility`       | `private` 或 `shared`；默认建议为 `private`       |
| `supportedModels`  | 该渠道可承载的模型列表，必须至少包含一个模型      |
| `defaultTestModel` | 测试连接默认模型，必须属于 `supportedModels`      |
| `lastRefreshedAt`  | 最近一次用户主动刷新时间                          |
| `nextRefreshAt`    | 下一次允许刷新时间，等于最近一次成功刷新后 5 小时 |

### 可见性

| 可见性    | 路由范围                                 | 凭据可见性                                   | 可编辑者       |
| --------- | ---------------------------------------- | -------------------------------------------- | -------------- |
| `private` | 仅渠道所有者自己的 Use API Key 可用      | 仅所有者和有权限的管理员可见                 | 所有者或管理员 |
| `shared`  | 所有允许使用共享容量的用户可作为候选渠道 | 非所有者不得看到凭据、完整错误细节或敏感备注 | 所有者或管理员 |

### 5 小时刷新窗口

- 每个渠道独立计算刷新窗口。
- 第一次刷新应允许立即执行。
- 刷新成功后，`nextRefreshAt = refreshedAt + 5h`。
- 在 `nextRefreshAt` 之前再次刷新必须被拒绝，并返回清晰的下一次可刷新时间。
- 到达或超过 `nextRefreshAt` 后，下一次刷新应允许执行。
- 并发刷新只能有一个成功，不能双花刷新额度。

## 使用页面

使用页面面向“我要调用模型”。用户不需要直接理解多个 Profile，MVP 可以使用单一默认 Profile 存储配置。

### API Key 配置

| 字段            | 要求                                                 |
| --------------- | ---------------------------------------------------- |
| `modelIDs`      | 用户显式选择的模型列表；请求模型不在列表内时必须拒绝 |
| `useStrategy`   | 使用策略：`prefer_own`、`only_own`、`allow_shared`   |
| `quota`         | 可沿用 API Key Profile 配额；不是 MVP 必填项         |
| `modelMappings` | 默认空；高级兼容场景可继续沿用现有 Profile 能力      |

示例：

```json
{
  "activeProfile": "default",
  "profiles": [
    {
      "name": "default",
      "modelIDs": ["gpt-4o", "claude-3-5-sonnet"],
      "useStrategy": "prefer_own",
      "modelMappings": []
    }
  ]
}
```

### 使用策略

| `useStrategy`  | 含义                                                   | 无匹配自有渠道时     |
| -------------- | ------------------------------------------------------ | -------------------- |
| `prefer_own`   | 优先使用自己的渠道，自己的渠道不可用时允许使用共享渠道 | 回退到共享渠道       |
| `only_own`     | 只使用自己的渠道，不使用其他人的共享渠道               | 返回无可用渠道错误   |
| `allow_shared` | 自有和共享渠道都可以作为候选                           | 在所有候选中继续排序 |

建议默认值为 `prefer_own`，因为它兼顾用户自有资源优先和共享兜底。

## 调度规则

调度规则必须保持简单、可解释，并且不绕过现有模型访问、配额、健康检查和重试逻辑。

### 候选过滤

请求进入路由前应依次过滤：

1. API Key 有效，且当前 Profile 存在。
2. 请求模型命中 `modelIDs`；未命中直接拒绝。
3. 渠道状态为启用，且支持请求模型。
4. `private` 渠道只保留所有者自己的渠道。
5. `shared` 渠道可被其他用户使用，但不得暴露敏感字段。
6. 继续应用现有项目/Profile 的渠道 ID、渠道标签、模型关联、配额和重试规则。

### 排序顺序

在候选过滤后执行排序：

1. **own-first**：在 `prefer_own` 下，自有渠道排在共享渠道前；`only_own` 不保留共享渠道；`allow_shared` 不强制自有优先，除非产品决定也沿用 own-first。
2. **soonest-refresh-first**：同一优先级桶内，`nextRefreshAt` 越早的渠道越靠前；没有刷新时间的渠道按“可立即刷新/最早”处理。
3. **现有负载均衡**：在前两层仍无法区分时，继续使用现有负载均衡、健康、限流、熔断和重试排序。

推荐伪流程：

```text
请求模型 -> 校验 modelIDs -> 找支持模型的启用渠道 -> 应用 private/shared 可见性
  -> 按 useStrategy 分桶 -> own-first -> soonest-refresh-first -> 现有 LoadBalancer/Retry
```

## 验收标准

本节区分“当前切片可验收”和“目标 MVP 后续验收”。除当前切片条目外，下列产品、后端和前端验收标准均表示后续完成项，不代表当前合并已经具备完整行为。

### 当前切片可验收

- 用户能在导航中看到 **共享** 和 **使用** 两个入口；其中 `/share` 仍是指南包装页，`/use` 已恢复为最小可用表单。
- `/use` 会强制至少选择一个 `modelIDs`，并把 `useStrategy` 保存到单一 `APIKeyProfile`；创建成功后可立即展示并复制生成的 key。
- `/use` 已支持通过 `/use?apiKeyId=<id>` 加载既有 user API Key，并仅更新当前 active profile 的 `modelIDs/useStrategy`。
- API Keys 列表行操作已提供直达 `/use` 编辑入口。
- GraphQL schema 与前端校验都只接受 `prefer_own`、`only_own`、`allow_shared`；空值按 `prefer_own` 处理。
- 后端对象层已具备 `ChannelSettings.Share` 元数据 / 默认值，但实际请求路由与渠道 CRUD 仍沿用现有链路，不影响当前渠道管理、API Key 管理、模型映射、配额、负载均衡和重试能力。

### 目标产品验收（后续完成）

- 共享页面只能展示和管理当前用户自己的渠道；管理员视角不破坏现有渠道管理能力。
- 用户创建 / 编辑渠道时必须选择 `private` 或 `shared`；未选择时使用默认 `private`。
- 共享渠道可被其他用户路由使用，但其他用户无法读取凭据、敏感备注和完整错误细节。
- 刷新按钮展示当前状态：可刷新、冷却中、下一次可刷新时间。
- 在现有深链编辑能力基础上，进一步补齐高级 Profile 字段（如 quota / channelTags / modelMappings）的完整编辑体验。
- 请求未选择模型时不能保存 Use 配置；请求未授权模型时返回明确错误。
- `prefer_own` 有自有候选时优先自有，无自有候选时可回退共享。
- `only_own` 不使用共享渠道，且无自有候选时返回明确无可用渠道错误。
- `allow_shared` 可以使用自有和共享候选，并继续遵守模型、配额、健康、重试规则。

### 目标后端验收（后续完成）

- 渠道 schema / API 能暴露所有者、可见性、刷新窗口与下一次可刷新时间，并给历史渠道明确兼容策略。
- 普通用户不能读取或修改其他用户的 private 渠道。
- 非所有者读取 shared 渠道时敏感字段被脱敏。
- 服务端继续只接受 `prefer_own`、`only_own`、`allow_shared`。
- `modelIDs` 为空时 Use API Key 创建 / 保存被服务端拒绝，除非明确进入管理员高级模式。
- 5 小时刷新窗口使用服务端时间计算，并处理并发请求。
- 路由选择输出可观测日志，至少包含策略、候选数量、自有 / 共享桶数量和最终渠道 ID。
- 现有 API Key Profile 的 `modelIDs`、`quota`、`loadBalanceStrategy` 行为保持兼容。

### 目标前端验收（Share 页面与 Use 页增强后）

- 第一阶段目标已完成：`/use` 已恢复为最小可用表单，支持名称、模型多选、`useStrategy` 选择和保存反馈。
- 第二阶段的最小编辑闭环也已完成：可通过 `/use?apiKeyId=<id>` 或 API Keys 列表的 `Use 配置` 入口加载既有 user API Key，并保存当前 active profile 的 `modelIDs/useStrategy`。
- 共享页面真实表单包含渠道名称、类型、Base URL、API Key、支持模型、默认测试模型、可见性。
- 使用页面后续继续补高级 Profile 字段编辑、更加细粒度的保存状态和更完整的错误提示。
- 中英文文案清楚区分“私有”“共享”“优先自有”“仅自有”“允许共享”。
- 保存失败、刷新冷却、无可用渠道、模型未授权都有明确错误提示。
- 页面可在无后端新增字段时优雅降级，不阻塞现有渠道 / API Key 管理页面。

## 测试矩阵

以下矩阵覆盖目标 MVP。当前仓库应至少验证文档、`/share` 包装页、`/use` 最小创建流程，以及 `useStrategy` / `ChannelSettings.Share` 相关 schema 与保存链路；完整共享路由、刷新窗口和真实 Share 表单测试需在对应实现补齐后启用。

| 模块        | 场景                                            | 预期                              | 建议命令/位置                                                                |
| ----------- | ----------------------------------------------- | --------------------------------- | ---------------------------------------------------------------------------- |
| Schema/验证 | `visibility=private/shared`                     | 合法值通过，非法值拒绝            | `go test ./internal/server/biz -run Channel`                                 |
| Schema/验证 | `useStrategy` 非法值                            | 保存 API Key Profile 失败         | `go test ./internal/server/biz -run APIKey`                                  |
| Schema/验证 | `modelIDs` 为空                                 | Use API Key 创建/保存失败         | `go test ./internal/server/biz -run APIKey`                                  |
| 权限        | 用户 A 读取用户 B 的 private 渠道               | 拒绝或返回空                      | `go test ./internal/scopes ./internal/server/biz -run Channel`               |
| 权限        | 用户 A 读取用户 B 的 shared 渠道                | 可作为候选，但凭据脱敏            | `go test ./internal/server/gql ./internal/server/biz -run Channel`           |
| 刷新窗口    | 首次刷新                                        | 成功并写入 `nextRefreshAt`        | `go test ./internal/server/biz -run Refresh`                                 |
| 刷新窗口    | 5 小时内重复刷新                                | 拒绝并返回下一次时间              | `go test ./internal/server/biz -run Refresh`                                 |
| 刷新窗口    | 正好到达 5 小时                                 | 允许刷新                          | `go test ./internal/server/biz -run Refresh`                                 |
| 刷新窗口    | 并发刷新同一渠道                                | 只有一个成功                      | `go test ./internal/server/biz -run Refresh -count=20`                       |
| 模型访问    | 请求模型不在 `modelIDs`                         | 返回模型无权限错误                | `go test ./internal/server/orchestrator -run TestCheckApiKeyModelAccess`     |
| 路由        | `only_own` 有自有候选                           | 只返回自有渠道                    | `go test ./internal/server/orchestrator -run 'Selector\|Share\|Use'`         |
| 路由        | `only_own` 只有共享候选                         | 返回无可用渠道                    | `go test ./internal/server/orchestrator -run 'Selector\|Share\|Use'`         |
| 路由        | `prefer_own` 同时有自有和共享                   | 自有渠道排在共享前                | `go test ./internal/server/orchestrator -run 'Selector\|LoadBalanced'`       |
| 路由        | `prefer_own` 无自有候选                         | 回退共享渠道                      | `go test ./internal/server/orchestrator -run 'Selector\|LoadBalanced'`       |
| 路由        | 同桶多个候选                                    | `nextRefreshAt` 最早的排前        | `go test ./internal/server/orchestrator -run 'Refresh\|Selector'`            |
| 配额        | API Key Profile quota 命中                      | 请求被拒绝，错误为 quota exceeded | `go test ./internal/server/orchestrator -run Quota`                          |
| 回归        | APIKeyProfile `channelIDs/channelTags/modelIDs` | 旧行为不变                        | `go test ./internal/server/biz -run TestModelService_ListEnabledModels`      |
| 回归        | Profile `loadBalanceStrategy`                   | 旧策略派生不变                    | `go test ./internal/server/orchestrator -run TestDeriveLoadBalancerStrategy` |
| 前端        | 共享页创建/编辑/归档                            | 表单可用，列表更新                | `cd frontend && pnpm test:e2e -- channels.spec.ts` 加新增 share spec         |
| 前端        | 使用页创建 API Key                              | 模型和策略保存成功                | `cd frontend && pnpm test:e2e -- apikeys/share-use.spec.ts`                  |
| 前端        | 基础质量                                        | lint/build 通过                   | `cd frontend && pnpm lint && pnpm build`                                     |

## 建议验证命令

目标 MVP 完成前建议至少运行：

```bash
go test ./internal/server/biz -run 'TestAPIKeyService_UpdateAPIKeyProfiles|TestValidateProfileQuota|TestQuotaService|TestQuotaWindow|TestModelService_ListEnabledModels'
go test ./internal/server/orchestrator -run 'TestCheckApiKeyModelAccess|TestDeriveLoadBalancerStrategy|TestLoadBalancedSelector|TestDefaultChannelSelector|TestChatCompletionOrchestrator_Process_MinuteQuotaExceeded'
go test ./internal/scopes ./internal/ent -run 'TestAPIKey|TestChannel|Scope'
cd frontend && pnpm lint
cd frontend && pnpm test:e2e -- channels.spec.ts models.spec.ts
```

完整回归：

```bash
make test-backend-all
make build
```

## 文档落点

- 本文作为 MVP 验收和联调入口：`docs/zh/guides/share-use-mvp.md`。
- `docs/zh/guides/api-key-profiles.md` 已补充 `/use` 最小表单、`modelIDs/useStrategy` 以及通过 `apiKeyId` 深链编辑既有 key 的关系。
- `docs/zh/guides/channel-management.md` 已补充当前 `/share` 仍为包装页，以及 `private/shared` 与 5 小时刷新窗口的现阶段关系。
- `docs/zh/guides/permissions.md` 已补充 `/share` 依赖 `read_channels`、`/use` 依赖 `read_api_keys` 的访问前提。
- `docs/zh/getting-started/request-processing.md` 已补充 own-first 与 soonest-refresh-first 在请求链路中的位置。

## 风险和待确认

- 5 小时刷新窗口需确认是滚动窗口还是固定窗口；本文按“每渠道成功刷新后滚动 5 小时”描述。
- `useStrategy` 建议独立于现有 `loadBalanceStrategy`，避免把“共享使用策略”和“负载均衡算法”混在同一字段。
- soonest-refresh-first 与现有 TraceAware、ErrorAware、LatencyAware、RateLimitAware、CircuitBreaker 的优先级必须固定，否则测试会不稳定。
- shared 渠道失败时是否允许系统自动禁用或影响所有者渠道，需要产品和后端确认。
- 历史渠道没有 owner/visibility 时需要迁移默认值；否则权限和路由可能出现灰区。
