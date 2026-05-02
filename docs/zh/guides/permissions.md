# Share / Use 权限指南

本指南不再从泛化的 RBAC 教材角度展开，而是只回答当前产品主线里最重要的问题：**谁能上传自己的渠道到 Share，谁能在 Use 里选择如何消费渠道，谁又能排查这条分发链路。**

## 先看两个入口

当前前端把 `/share` 和 `/use` 放在同一个路由组里，但它们依赖的权限并不相同：

| 入口     | 当前依赖的 scope | scope 层级            | 主要作用                                    |
| -------- | ---------------- | --------------------- | ------------------------------------------- |
| `/share` | `read_channels`  | 仅 `system`           | 查看 Share 说明，并进入现有渠道管理能力     |
| `/use`   | `read_api_keys`  | `system` 或 `project` | 查看和维护当前用户的 Use API Key / 使用策略 |

如果要做写操作，还需要额外的写权限：

| 操作                                             | 需要的 scope     |
| ------------------------------------------------ | ---------------- |
| 创建、编辑、测试、启用渠道                       | `write_channels` |
| 创建、编辑、轮换 Use API Key 或其 active profile | `write_api_keys` |
| 查看请求与追踪数据                               | `read_requests`  |

## 当前最小权限模型

### 1. 渠道提供者

如果一个用户要把自己的上游渠道贡献给平台，当前最小权限组合通常是：

- `read_channels`
- `write_channels`

如果还要自己验证渠道是否真的参与了分发，通常再补：

- `read_requests`

需要注意的是，`read_channels` / `write_channels` 目前只支持 `system` 层级，所以**现在的 Share 上传者更接近“被授予渠道管理能力的用户”，而不是纯 Project 内的普通成员。**

### 2. 渠道使用者

如果一个用户只想在项目流量进入平台后，决定优先用自己的渠道还是别人的共享渠道，最小权限通常是：

- `read_api_keys`

如果他还需要创建、编辑或切换自己的 Use 配置，则再补：

- `write_api_keys`

如果还要自己排查为什么某次请求没有命中预期渠道，则再补：

- `read_requests`

### 3. 平台运维 / 授权管理员

如果一个用户负责给别人分配入口、查看权限差异或维护角色，才需要进一步补充：

- `read_roles` / `write_roles`
- `read_users` / `write_users`

这类权限不属于 Share / Use 的最少必要集合，不应该默认发给普通调用者。

## 权限和共享分发的边界

权限决定的是“能不能进入页面、能不能改配置”，不是“能不能直接拿到别人的原始渠道凭据”。

在当前主线里要分清 3 件事：

1. `shared` 只是说明一个渠道**可以进入其他用户的共享候选池**。
2. 它不意味着其他用户获得了这个渠道的后台管理权。
3. 它也不意味着其他用户可以直接看到 owner 的原始 API Key。

也就是说：

- **渠道所有权** 仍归 owner
- **平台分发权** 由 Share / Use 路由和结算逻辑控制
- **后台编辑权** 仍由 `write_channels` / `write_api_keys` 这类 scope 控制

## 与共享池规则的关系

Share / Use 文档已经约定：

- `OwnPool(U)` 表示用户自己的渠道池
- `SharedPool(U)` 表示所有 `shared` 且 `owner != U` 的渠道池

这条规则和权限模型是正交的：

- 你有没有 `read_api_keys`，决定你能不能进入 `/use`
- 你有没有 `read_channels` / `write_channels`，决定你能不能维护供给侧渠道
- 你是否会命中别人的共享渠道，取决于 `useStrategy`、模型暴露、健康状态和共享池过滤规则

换句话说，**权限只打开入口，不直接决定某次请求会路由到哪条渠道。**

## 当前实现口径

当前文档需要和代码里的真实配置保持一致：

- 后端 `internal/scopes/scopes.go` 中，`read_channels` / `write_channels` 只支持 `system`。
- 后端 `internal/scopes/scopes.go` 中，`read_api_keys` / `write_api_keys` / `read_requests` 同时支持 `system` 和 `project`。
- 前端 `frontend/src/config/route-permission.ts` 中，`/share` 依赖 `read_channels`，`/use` 依赖 `read_api_keys`。
- 虽然 `Share/Use` 路由组的 `scopeLevel` 是 `any`，但 `/share` 本身仍绑定到 system-only 的 `read_channels`，所以 project-only 用户当前并不能真正进入 Share 侧入口。

## 推荐授权方式

### 只负责上传渠道

适合愿意提供供给、但不需要管理项目成员或角色的人：

- `read_channels`
- `write_channels`
- 可选 `read_requests`

### 只负责消费渠道

适合只想配置自己如何用渠道的人：

- `read_api_keys`
- 可选 `write_api_keys`
- 可选 `read_requests`

### 同时负责上传与消费

适合既维护自己渠道，又要调 Use 策略的人：

- `read_channels`
- `write_channels`
- `read_api_keys`
- `write_api_keys`
- 可选 `read_requests`

## 最佳实践

- 只给普通调用者发完成当前工作所需的最小权限，不要默认附带 `write_roles` 或 `write_users`。
- 把“上传渠道的人”和“消费渠道的人”拆开授权，避免所有人都持有渠道编辑权。
- 如果某个成员只需要调试为什么没命中共享池，优先补 `read_requests`，而不是直接给 `write_channels`。
- 现阶段 `/share` 仍复用系统级渠道能力；如果后续产品把 Share 做成真正的用户自助入口，文档也要跟着 scope 设计一起更新。

## 相关文档

- [共享/使用 MVP 指南](share-use-mvp.md) - 查看共享池、策略值和结算边界
- [渠道配置指南](channel-management.md) - 查看供给侧渠道上传与配置方式
- [模型管理指南](model-management.md) - 查看模型暴露与路由映射如何影响候选渠道
- [请求处理流程指南](../getting-started/request-processing.md) - 查看权限之外真正影响分发顺序的链路步骤
