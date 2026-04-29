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

## 当前阻塞项（2026-04-28 codegen 复核）

- 数据基础：6 个 Relay Ent schema 已存在（`RelayProduct`、`RelayProductChannel`、`RelayKey`、`RelayWallet`、`RelayWalletLedgerEntry`、`RelayDailyUsageSummary`），但生成资产仍缺失：`internal/ent/relayproduct*`、`internal/ent/relaykey*`、`internal/ent/relaywallet*` 等目录/文件均未生成。
- 工具链：`go.mod` 要求 `go 1.26.0` 且使用 `tool github.com/99designs/gqlgen`；本机 `GOTOOLCHAIN=local` 为 `go1.22.12`，执行 `go test`/`go generate` 失败于 `go.mod:234: unknown directive: tool`。
- 离线 codegen：`GOTOOLCHAIN=auto` + `GOPROXY=off` 会尝试下载 `go1.26.0` 并失败为 `toolchain not available`；网络代理超时时不能离线生成 Ent/GQL 资产。
- 产品 CRUD：`RelayProductService` 已用最小 raw SQL 临时实现产品列表/创建/更新与渠道绑定创建/更新/删除；生成代码合并后仍需复核并替换为 Ent 路径。
- Runtime 接入：`NewRelayRuntimeService`、鉴权后 resolver/checker、路由前渠道池过滤、`UsageLogService` 后置结算 hook 必须在 runtime/API 切片合并后复核。
- 前端完成度：运营侧和项目侧页面若仍标注 skeleton/no API wiring，必须保留为未完成项；上线前至少需要 query/mock fallback、loading/empty/error 和项目侧只读隔离。

## 后端数据基础合并后补测

- Ent schema：确认 6 张 Relay 新表字段、枚举、索引、边和隐私策略与设计文档一致。
- 迁移生成：运行 `make generate` 后检查生成代码与迁移可在 SQLite 内存库创建表。
- CRUD 行为：补充 `CreateRelayProduct`、`ListRelayProducts`、`UpdateRelayProduct`、`CreateRelayProductChannelBinding`、`UpdateRelayProductChannelBinding`、`DeleteRelayProductChannelBinding` 的成功路径和唯一索引冲突测试。
- Key/钱包行为：补充 Relay Key 创建、钱包快照创建、充值/冻结/恢复、ledger 幂等扣费和 daily summary 增量 upsert 测试。
- 查询排序：验证渠道池按 `(product_id, status, priority)` 查询时只返回 active 绑定，并能保留 priority/weight 语义。

## 运行时/API 合并后补测

- 鉴权后钩子：普通 API Key 不应被 Relay 逻辑误拦截；绑定 `relay_keys` 的 Sub-Key 必须校验 status、expires_at、余额和硬限额。
- 路由前钩子：候选渠道必须受产品池、绑定状态、模型过滤、渠道状态和 provider quota 状态共同约束。
- 结算钩子：基于 `usage_log_id` 的扣费必须幂等，重复调用不得重复写账本或重复扣余额。
- 失败分层：余额不足、Key 暂停、Key 归档、产品池不可用、上游失败和结算失败需要返回可区分错误，方便页面展示。

## 轻量验证命令

```bash
gofmt -w internal/ent/schema/relay_*.go internal/server/biz/relay_product.go internal/server/biz/relay_runtime.go internal/server/biz/relay_services.go
GOTOOLCHAIN=local go test ./internal/server/biz -run 'Relay(Product|Runtime)' -count=1
GOTOOLCHAIN=local go generate ./internal/server/gql
```

如果本地 Go 版本低于 `go.mod` 要求，预期可能阻塞在 `go.mod:234: unknown directive: tool` 或 toolchain 下载；记录准确输出，不要在当前工作树里降级 `go.mod` 或强制下载工具链。

后端/runtime 合并完成后再追加：

```bash
go test ./internal/server/biz -run 'Relay(Product|Key|Wallet|Access|Router|Settlement|Runtime)'
go test ./internal/server/api -run Relay
```

前端页面接入完成后追加：

```bash
cd frontend
pnpm lint
pnpm build
```

## Runtime 协作约束

- data/QA 切片避免直接改 runtime hook 文件，除非 runtime 队友明确协调。
- 产品/渠道池常量需要与后端设计文档和页面流程文档同步更新。
- 契约夹具漂移应视为前端选项列表和 API 文档的 review blocker。
