# 自托管 Relay + Sub-Key 共享容量 MVP：QA 与测试清单

本文用于跟踪第一段后端 MVP 切片的可验证范围，优先覆盖低冲突测试、夹具和验收口径。核心实现仍以 `relay-subkey-mvp-backend-design.md` 和 `relay-subkey-mvp-page-flows.md` 为准。

## 当前可落地检查

- 产品契约：`RelayProductService.Contract()` 必须与 `internal/server/biz/testdata/relay_product_contract.json` 保持一致，避免前端选项、文档状态和后端常量漂移。
- 产品输入校验：产品编码、名称、providerType、billingMode、status、allowedModels、requestTimeoutSeconds 必须在进入 Ent CRUD 前被同步校验。
- 渠道池绑定校验：`product_id`、`channel_id`、`weight`、`status`、`max_inflight` 必须先做硬校验，归档渠道不能加入共享池。
- Runtime 薄契约：`RelayRuntimeService` 必须覆盖普通 API Key bypass、Sub-Key 默认访问判定、渠道池模型过滤和 usage 结算 recorder 调用。
- 文档引用：`docs/zh/development/development.md` 必须持续指向后端设计、页面流程和本 QA 清单。

## 契约夹具

后端契约测试会将面向运营配置的 Relay 产品常量固定到 `internal/server/biz/testdata/relay_product_contract.json`：

- Provider 类型：`claudecode`、`codex`、`openai_compatible`
- 访问模式：`shared_capacity`
- 计费模式：`prepaid`、`quota_only`
- 产品状态：`draft`、`active`、`archived`
- 渠道绑定状态：`active`、`paused`
- 默认值：`USD`、`shared_capacity`、`prepaid`、`draft`、超时 `600`、绑定权重 `100`

## 当前收口状态（2026-05-01 Relay 限额语义收口）

- 数据基础：6 个 Relay Ent schema 已存在（`RelayProduct`、`RelayProductChannel`、`RelayKey`、`RelayWallet`、`RelayWalletLedgerEntry`、`RelayDailyUsageSummary`），且对应生成资产已存在：`internal/ent/relayproduct*`、`internal/ent/relayproductchannel*`、`internal/ent/relaykey*`、`internal/ent/relaywallet*`、`internal/ent/relaywalletledgerentry*`、`internal/ent/relaydailyusagesummary*`。
- 工具链：当前环境 PATH 未找到 `go`/`gofmt`，因此本轮无法执行 `go fmt`/`go test`；**不要降级** `go.mod` 的 `go 1.26.0` 或 `tool` directive。后端 targeted tests 与 `gofmt` 必须在 Go 1.26 环境补跑（命令见末尾）。
- 产品/运营 REST 最小闭环：已落地 product-channel PATCH/DELETE 路由与 contract、归档/未启用渠道拦截、`CreateKey` 一次性 `plaintextKey`、`RechargeWallet` finite + exhausted 恢复；剩余 `RelayProductService` 是否完全替换 raw SQL 至 Ent 路径需在补跑时复核。
- Runtime 接入：`RelayRuntimeService` Fx wiring + `AuthenticateRelayAPIKey -> ResolveAndCheckAccess` + `UsageLogService -> RecordRelayUsage -> RelaySettlementService` 已接入；orchestrator `select_candidates` 已叠加产品池、绑定状态、模型过滤、渠道状态、`ProviderQuotaStatus.ready` 过滤；结算 idempotency 在 unique conflict 后回滚钱包。
- 前端 REST 接入：`VITE_RELAY_SUBKEYS_API_MODE=rest` 时 401/403/contract 错误进入错误态、不 fallback；REST 模式缺 projectId 抛错而非 mock；`filterKeysByProject` 不再回退 `project-alpha`；项目侧详情页 keyId 不存在不再回退首条 key；route permission 已对齐后端 all-scopes 语义；admin Relay 路由 `scopeLevel='system'` 显式标注；运营详情页写操作（产品激活/绑定/Pause/Resume/Remove/Suspend/Archive/Increase concurrency/Recharge）已按 `write_channels`/`write_api_keys` 在组件级隐藏。
- 已决策限制 / 已知语义：
  - `MonthlyCostLimit` 当前语义为请求前 soft/preflight guard，基于 `relay_daily_usage_summaries` 月度聚合判断；它降低明显超限请求，但不承诺并发下的严格财务 hard cap。
  - 正向 `ConcurrencyLimit` 当前语义为 preview/config-only，不强制阻塞超额 in-flight；`<= 0` 仍可表达拒绝/停用语义，Post-MVP 通过 inflight Begin/Release tracker 才能变成硬阻塞。
- 仍需关注的已知风险：
  - `PromptTokens` / `CompletionTokens` 字段语义不完美：后端目前只有 totals 摘要，前端聚合 token 总量可用但分项语义不准。

## 后端数据基础合并后补测

- Ent schema：确认 6 张 Relay 新表字段、枚举、索引、边和隐私策略与设计文档一致。
- 迁移生成：运行 `make generate` 后检查生成代码与迁移可在 SQLite 内存库创建表。
- CRUD 行为：补充 `CreateRelayProduct`、`ListRelayProducts`、`UpdateRelayProduct`、`CreateRelayProductChannelBinding`、`UpdateRelayProductChannelBinding`、`DeleteRelayProductChannelBinding` 的成功路径和唯一索引冲突测试。
- Key/钱包行为：补充 Relay Key 创建、钱包快照创建、充值/冻结/恢复、ledger 幂等扣费和 daily summary 增量 upsert 测试。
- 查询排序：验证渠道池按 `(product_id, status, priority)` 查询时只返回 active 绑定，并能保留 priority/weight 语义。

## 运行时/API 合并后补测

- 鉴权后钩子：普通 API Key 不应被 Relay 逻辑误拦截；绑定 `relay_keys` 的 Sub-Key 必须校验 status、expires_at、余额、日级硬限额和月度 soft/preflight guard。
- 路由前钩子：候选渠道必须受产品池、绑定状态、模型过滤、渠道状态和 provider quota 状态共同约束；带 `ProviderQuotaStatus.ready=false` 的渠道必须被排除，未绑定 quota status 或 `ready=true` 的渠道可保留。
- 结算钩子：基于 `usage_log_id` 的扣费必须幂等，重复调用不得重复写账本或重复扣余额；唯一冲突必须回滚同事务内钱包扣减。
- 项目侧隔离：`ProjectOverview` 只能返回当前项目持有 Key 对应产品的基础元数据，不得暴露全局 channelPool 或全局产品请求/token/cost 统计。
- 前端 REST 模式：`VITE_RELAY_SUBKEYS_API_MODE=rest` 时 401/403/API 错误必须进入错误态，不得 fallback 到 mock；未知项目 mock 数据不得回退到 `project-alpha`。
- 权限契约：前端 route-permission 对 Relay 页面使用与后端 `RequireScopes`/`RequireProjectScopes` 一致的 all-scopes 语义。
- 失败分层：余额不足、Key 暂停、Key 归档、产品池不可用、上游失败和结算失败需要返回可区分错误，方便页面展示。

## 轻量验证命令

```bash
gofmt -w internal/ent/schema/relay_*.go internal/server/biz/relay_product.go internal/server/biz/relay_runtime.go internal/server/biz/relay_services.go internal/server/biz/relay_admin.go internal/server/middleware/relay_authz.go
GOTOOLCHAIN=local go test ./internal/server/biz -run 'Relay(Product|Runtime|Access|Settlement)' -count=1
GOTOOLCHAIN=local go generate ./internal/server/gql
```

如果本地 Go 版本低于 `go.mod` 要求，预期可能阻塞在 `go.mod:234: unknown directive: tool` 或 toolchain 下载；记录准确输出，不要在当前工作树里降级 `go.mod` 或强制下载工具链。

后端/runtime 合并完成后再追加（Phase 2-3 已完成，待 Go 1.26 补跑）：

```bash
go test ./internal/server/biz -run 'Relay(Product|Key|Wallet|Admin|Runtime|Access|Router|Settlement|Auth)' -count=1
go test ./internal/server/api -run Relay -count=1
go test ./internal/server/middleware -run 'Relay|RequireProjectScopes' -count=1
go test ./internal/server/orchestrator -count=1
```

前端页面接入完成后追加（Phase 4-5 已完成）：

```bash
pnpm --dir frontend lint
pnpm --dir frontend build
```

注意：仓库存在历史 lint 噪音，关注新增/修改文件的 ESLint 是否 0 报错；本轮 `frontend/src/hooks/useRoutePermissions.ts` 与 `frontend/src/features/relay-subkeys/pages.tsx` 经 `pnpm exec eslint` 检查 0 报错；`pnpm --dir frontend build` exit 0。

## Post-MVP 验收 backlog（仅在 Go 1.26 环境可执行）

- 并发同 `usageLog.ID` 结算只扣一次（验证 `RelaySettlementService.SettleUsage` 的 unique conflict 回滚 + idempotency key）。
- `RelayAdminService.GetOverview(ctx, projectID)` 不暴露其他项目/全局产品/全局 channelPool 统计。
- Post-MVP hard monthly cap：通过请求预授权/额度保留，或 settlement 事务内 serial lock/atomic check + debit 实现；补并发对照测试，并评估上游已完成后 settlement 拒绝的体验与对账风险。
- Post-MVP inflight tracker：实现 request-scoped Begin/Release tracker 后，正向 `ConcurrencyLimit` 才能阻塞超额 in-flight；补正常释放、异常释放和并发拒绝测试。
- `ProviderQuotaStatus.ready=false` 渠道必须从 `RelayRouterService.listActivePool` 候选中排除；`ready=true` 与无 status 必须保留。
- 前端 REST 模式 401/403 不 fallback、contract drift/unwrap 失败抛 `relayContractError`、缺 projectId 不返回 mock；新增 `requireEntityResponse` 对 primitive bare 响应的严格性。
- `routePathMatches('/project/relay-subkeys/keys/$keyId', '/project/relay-subkeys/keys/abc')` 必须 true；`ProjectRelaySubkeysKeyDetailPage` keyId 不存在显示 not-found 而非首条 key。
- `RouteGuard` `scopeLevel='any'` 不允许跨级别拼接（不允许 system `read_api_keys` + project `read_requests` 同时满足 `['read_api_keys','read_requests']`）。
- `RequireProjectScopes` 允许 owner、全量系统 scopes、全量项目 scopes 三种语义；不允许跨级别拼接。
- 运营详情页写按钮（产品激活/绑定/Pause/Resume/Remove/Suspend/Archive/Increase concurrency/Recharge）在仅持有 `read_channels` / `read_api_keys` 系统 scope 时不渲染。

## Runtime 协作约束

- data/QA 切片避免直接改 runtime hook 文件，除非 runtime 队友明确协调。
- 产品/渠道池常量需要与后端设计文档和页面流程文档同步更新。
- 契约夹具漂移应视为前端选项列表和 API 文档的 review blocker。
