# 自托管 Relay + Sub-Key 共享容量 MVP：QA 与测试清单

本文用于跟踪第一段后端 MVP 切片的可验证范围，优先覆盖低冲突测试、夹具和验收口径。核心实现仍以 `relay-subkey-mvp-backend-design.md` 和 `relay-subkey-mvp-page-flows.md` 为准。

## 当前可落地检查

- 产品契约：`RelayProductService.Contract()` 必须与 `internal/server/biz/testdata/relay_product_contract.json` 保持一致，避免前端选项、文档状态和后端常量漂移。
- 产品输入校验：产品编码、名称、providerType、billingMode、status、allowedModels、requestTimeoutSeconds 必须在进入 Ent CRUD 前被同步校验。
- 渠道池绑定校验：`product_id`、`channel_id`、`weight`、`status`、`max_inflight` 必须先做硬校验，归档渠道不能加入共享池。
- 文档引用：`docs/zh/development/development.md` 必须持续指向后端设计、页面流程和本 QA 清单。

## 后端数据基础合并后补测

- Ent schema：确认 `relay_products` 与 `relay_product_channels` 字段、枚举、索引、边和隐私策略与设计文档一致。
- 迁移生成：运行 `make generate` 后检查生成代码与迁移可在 SQLite 内存库创建表。
- CRUD 行为：补充 `CreateRelayProduct`、`ListRelayProducts`、`UpdateRelayProduct`、`CreateRelayProductChannelBinding`、`UpdateRelayProductChannelBinding`、`DeleteRelayProductChannelBinding` 的成功路径和唯一索引冲突测试。
- 查询排序：验证渠道池按 `(product_id, status, priority)` 查询时只返回 active 绑定，并能保留 priority/weight 语义。

## 运行时/API 合并后补测

- 鉴权后钩子：普通 API Key 不应被 Relay 逻辑误拦截；绑定 `relay_keys` 的 Sub-Key 必须校验 status、expires_at、余额和硬限额。
- 路由前钩子：候选渠道必须受产品池、绑定状态、模型过滤、渠道状态和 provider quota 状态共同约束。
- 结算钩子：基于 `usage_log_id` 的扣费必须幂等，重复调用不得重复写账本或重复扣余额。
- 失败分层：余额不足、Key 暂停、Key 归档、产品池不可用、上游失败和结算失败需要返回可区分错误，方便页面展示。

## 轻量验证命令

```bash
go test ./internal/server/biz -run RelayProduct
```

后端/runtime 合并完成后再追加：

```bash
go test ./internal/server/biz -run 'Relay(Product|Key|Wallet|Access|Router|Settlement)'
go test ./internal/server/api -run Relay
```
