# 共享/使用 MVP 指南

本文记录简化版“共享/使用”方案的产品边界、数据约定、调度规则、验收标准和测试矩阵。目标是用最少架构完成用户自助接入：用户在 **共享** 页面上传自己的渠道，在 **使用** 页面创建只选择模型和使用策略的 API Key。

## 范围

### 页面

| 页面 | 用户目标 | MVP 能力 |
|------|----------|----------|
| 共享 | 把自己的上游账号贡献给 AxonHub 使用 | 创建/编辑/归档自己的渠道，设置 `private` 或 `shared`，查看 5 小时刷新窗口 |
| 使用 | 用简单 API Key 调用可用模型 | 创建 API Key，选择 `modelIDs`，选择 `useStrategy` |

### 不做的事

- 不引入复杂资源池、市场、结算或多租户套餐模型。
- 不要求普通用户理解 API Key Profile、渠道标签或完整负载均衡配置。
- 不改变现有管理员渠道管理、模型映射、配额和重试能力；MVP 只在其上增加用户视角。

## 共享页面

共享页面管理“我上传的渠道”。每个渠道仍然是 AxonHub 渠道，但必须带有所有者和可见性语义。

### 字段约定

| 字段 | 要求 |
|------|------|
| `ownerUserID` | 渠道所有者；普通用户只能管理自己的渠道 |
| `visibility` | `private` 或 `shared`；默认建议为 `private` |
| `supportedModels` | 该渠道可承载的模型列表，必须至少包含一个模型 |
| `defaultTestModel` | 测试连接默认模型，必须属于 `supportedModels` |
| `lastRefreshedAt` | 最近一次用户主动刷新时间 |
| `nextRefreshAt` | 下一次允许刷新时间，等于最近一次成功刷新后 5 小时 |

### 可见性

| 可见性 | 路由范围 | 凭据可见性 | 可编辑者 |
|--------|----------|------------|----------|
| `private` | 仅渠道所有者自己的 Use API Key 可用 | 仅所有者和有权限的管理员可见 | 所有者或管理员 |
| `shared` | 所有允许使用共享容量的用户可作为候选渠道 | 非所有者不得看到凭据、完整错误细节或敏感备注 | 所有者或管理员 |

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

| 字段 | 要求 |
|------|------|
| `modelIDs` | 用户显式选择的模型列表；请求模型不在列表内时必须拒绝 |
| `useStrategy` | 使用策略：`prefer_own`、`only_own`、`allow_shared` |
| `quota` | 可沿用 API Key Profile 配额；不是 MVP 必填项 |
| `modelMappings` | 默认空；高级兼容场景可继续沿用现有 Profile 能力 |

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

| `useStrategy` | 含义 | 无匹配自有渠道时 |
|---------------|------|------------------|
| `prefer_own` | 优先使用自己的渠道，自己的渠道不可用时允许使用共享渠道 | 回退到共享渠道 |
| `only_own` | 只使用自己的渠道，不使用其他人的共享渠道 | 返回无可用渠道错误 |
| `allow_shared` | 自有和共享渠道都可以作为候选 | 在所有候选中继续排序 |

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

### 产品验收

- 用户能在导航中看到 **共享** 和 **使用** 两个入口。
- 共享页面只能展示和管理当前用户自己的渠道；管理员视角不破坏现有渠道管理能力。
- 用户创建渠道时必须选择 `private` 或 `shared`；未选择时使用默认 `private`。
- 共享渠道可被其他用户路由使用，但其他用户无法读取凭据、敏感备注和完整错误细节。
- 刷新按钮展示当前状态：可刷新、冷却中、下一次可刷新时间。
- 5 小时窗口内重复刷新被拒绝，到点后刷新成功。
- 使用页面能创建 API Key，并把模型选择写入 `modelIDs`。
- 使用页面能选择 `prefer_own`、`only_own`、`allow_shared`，保存后请求按策略路由。
- 请求未选择模型时不能创建 Use API Key；请求未授权模型时返回明确错误。
- `prefer_own` 有自有候选时优先自有，无自有候选时可回退共享。
- `only_own` 不使用共享渠道，且无自有候选时返回明确无可用渠道错误。
- `allow_shared` 可以使用自有和共享候选，并继续遵守模型、配额、健康、重试规则。

### 后端验收

- 渠道有所有者和可见性默认值；历史渠道迁移或默认行为有明确兼容策略。
- 普通用户不能读取或修改其他用户的 private 渠道。
- 非所有者读取 shared 渠道时敏感字段被脱敏。
- `useStrategy` 只接受 `prefer_own`、`only_own`、`allow_shared`。
- `modelIDs` 为空时 Use API Key 创建/保存被拒绝，除非明确进入管理员高级模式。
- 5 小时刷新窗口使用服务端时间计算，并处理并发请求。
- 路由选择输出可观测日志，至少包含策略、候选数量、自有/共享桶数量和最终渠道 ID。
- 现有 API Key Profile 的 `modelIDs`、`quota`、`loadBalanceStrategy` 行为保持兼容。

### 前端验收

- 共享页面表单包含渠道名称、类型、Base URL、API Key、支持模型、默认测试模型、可见性。
- 使用页面表单包含名称、模型多选、使用策略、生成后的 API Key 展示/复制。
- 中英文文案清楚区分“私有”“共享”“优先自有”“仅自有”“允许共享”。
- 保存失败、刷新冷却、无可用渠道、模型未授权都有明确错误提示。
- 页面可在无后端新增字段时优雅降级，不阻塞现有渠道/API Key 管理页面。

## 测试矩阵

| 模块 | 场景 | 预期 | 建议命令/位置 |
|------|------|------|---------------|
| Schema/验证 | `visibility=private/shared` | 合法值通过，非法值拒绝 | `go test ./internal/server/biz -run Channel` |
| Schema/验证 | `useStrategy` 非法值 | 保存 API Key Profile 失败 | `go test ./internal/server/biz -run APIKey` |
| Schema/验证 | `modelIDs` 为空 | Use API Key 创建/保存失败 | `go test ./internal/server/biz -run APIKey` |
| 权限 | 用户 A 读取用户 B 的 private 渠道 | 拒绝或返回空 | `go test ./internal/scopes ./internal/server/biz -run Channel` |
| 权限 | 用户 A 读取用户 B 的 shared 渠道 | 可作为候选，但凭据脱敏 | `go test ./internal/server/gql ./internal/server/biz -run Channel` |
| 刷新窗口 | 首次刷新 | 成功并写入 `nextRefreshAt` | `go test ./internal/server/biz -run Refresh` |
| 刷新窗口 | 5 小时内重复刷新 | 拒绝并返回下一次时间 | `go test ./internal/server/biz -run Refresh` |
| 刷新窗口 | 正好到达 5 小时 | 允许刷新 | `go test ./internal/server/biz -run Refresh` |
| 刷新窗口 | 并发刷新同一渠道 | 只有一个成功 | `go test ./internal/server/biz -run Refresh -count=20` |
| 模型访问 | 请求模型不在 `modelIDs` | 返回模型无权限错误 | `go test ./internal/server/orchestrator -run TestCheckApiKeyModelAccess` |
| 路由 | `only_own` 有自有候选 | 只返回自有渠道 | `go test ./internal/server/orchestrator -run 'Selector|Share|Use'` |
| 路由 | `only_own` 只有共享候选 | 返回无可用渠道 | `go test ./internal/server/orchestrator -run 'Selector|Share|Use'` |
| 路由 | `prefer_own` 同时有自有和共享 | 自有渠道排在共享前 | `go test ./internal/server/orchestrator -run 'Selector|LoadBalanced'` |
| 路由 | `prefer_own` 无自有候选 | 回退共享渠道 | `go test ./internal/server/orchestrator -run 'Selector|LoadBalanced'` |
| 路由 | 同桶多个候选 | `nextRefreshAt` 最早的排前 | `go test ./internal/server/orchestrator -run 'Refresh|Selector'` |
| 配额 | API Key Profile quota 命中 | 请求被拒绝，错误为 quota exceeded | `go test ./internal/server/orchestrator -run Quota` |
| 回归 | APIKeyProfile `channelIDs/channelTags/modelIDs` | 旧行为不变 | `go test ./internal/server/biz -run TestModelService_ListEnabledModels` |
| 回归 | Profile `loadBalanceStrategy` | 旧策略派生不变 | `go test ./internal/server/orchestrator -run TestDeriveLoadBalancerStrategy` |
| 前端 | 共享页创建/编辑/归档 | 表单可用，列表更新 | `cd frontend && pnpm test:e2e -- channels.spec.ts` 加新增 share spec |
| 前端 | 使用页创建 API Key | 模型和策略保存成功 | `cd frontend && pnpm test:e2e -- apikeys/share-use.spec.ts` |
| 前端 | 基础质量 | lint/build 通过 | `cd frontend && pnpm lint && pnpm build` |

## 建议验证命令

MVP 合并前建议至少运行：

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
- `docs/zh/guides/api-key-profiles.md` 后续可补充 Use 页面与 `modelIDs/useStrategy` 的关系。
- `docs/zh/guides/channel-management.md` 后续可补充 Share 页面与 `private/shared` 的关系。
- `docs/zh/getting-started/request-processing.md` 后续可补充 own-first 与 soonest-refresh-first 在请求链路中的位置。

## 风险和待确认

- 5 小时刷新窗口需确认是滚动窗口还是固定窗口；本文按“每渠道成功刷新后滚动 5 小时”描述。
- `useStrategy` 建议独立于现有 `loadBalanceStrategy`，避免把“共享使用策略”和“负载均衡算法”混在同一字段。
- soonest-refresh-first 与现有 TraceAware、ErrorAware、LatencyAware、RateLimitAware、CircuitBreaker 的优先级必须固定，否则测试会不稳定。
- shared 渠道失败时是否允许系统自动禁用或影响所有者渠道，需要产品和后端确认。
- 历史渠道没有 owner/visibility 时需要迁移默认值；否则权限和路由可能出现灰区。
