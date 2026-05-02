# 权限管理指南

## 概述

AxonHub 使用基于角色的访问控制（RBAC）来控制后台页面、API 资源和项目数据的可见性。当前实现同时区分 system-level 和 project-level scopes：前者用于系统级资源，后者用于某个 Project 内的资源。

是否能访问某个页面，取决于两部分：

1. 后端是否定义了对应的 scope 以及它支持的层级
2. 前端路由是否把该页面声明为需要这些 scope

虽然每个用户默认可以查看和管理自己的资源，但涉及他人资源、系统配置或跨项目数据时，仍然需要显式授权。

## 权限模型

### 角色、作用域与 Owner

| 概念          | 说明                                                 |
| ------------- | ---------------------------------------------------- |
| Global Role   | 绑定系统级权限，在所有 Projects 中生效               |
| Project Role  | 绑定项目内权限，只在指定 Project 中生效              |
| Scope         | 细粒度权限点，例如 `read_channels`、`write_api_keys` |
| Global Owner  | 拥有所有系统级与项目级权限                           |
| Project Owner | 拥有当前 Project 内所有权限                          |
| API Key       | 属于某个用户和项目，可继承对应角色作用域             |

从当前权限实现与 ERD 文档来看，权限检查的大致顺序是：

1. 先判断当前用户是否为 Owner
2. 再检查系统级角色与 scopes
3. 最后检查当前 Project 下的角色与 scopes

### Scope 层级

| 层级               | 说明                                    | 典型 scope                                                               |
| ------------------ | --------------------------------------- | ------------------------------------------------------------------------ |
| `system`           | 管理系统全局资源                        | `read_channels`、`write_channels`、`read_projects`、`read_data_storages` |
| `project`          | 管理当前 Project 中的业务资源           | `read_api_keys`、`write_api_keys`、`read_requests`                       |
| `system + project` | 同一个 scope 可被系统角色或项目角色授予 | `read_users`、`read_roles`、`read_prompts`、`read_requests`              |

当前代码里，以下 scope 仅支持 system-level：

- `read_dashboard`
- `read_settings` / `write_settings`
- `read_channels` / `write_channels`
- `read_data_storages` / `write_data_storages`
- `read_projects` / `write_projects`

以下 scope 同时支持 system-level 和 project-level：

- `read_users` / `write_users`
- `read_roles` / `write_roles`
- `read_api_keys` / `write_api_keys`
- `read_requests` / `write_requests`
- `read_prompts` / `write_prompts`

## 页面访问与常见入口

下表总结了当前前端路由里的几类常见入口：

| 页面 / 入口                                                | 需要的 scope    | 层级说明          | 备注                                                                 |
| ---------------------------------------------------------- | --------------- | ----------------- | -------------------------------------------------------------------- |
| `/channels`、`/models`、`/prompt-protection-rules`         | `read_channels` | 仅 system-level   | 渠道和模型属于系统级管理面                                           |
| `/share`                                                   | `read_channels` | 仅 system-level   | 当前仍是 Share / Use 指南包装页，不是独立 CRUD                       |
| `/use`、`/project/api-keys`                                | `read_api_keys` | system 或 project | `/use` 当前支持创建 key，也支持通过 `apiKeyId` 深链编辑既有 user key |
| `/project/requests`、`/project/traces`、`/project/threads` | `read_requests` | system 或 project | 查看请求、链路和线程数据                                             |
| `/project/users`                                           | `read_users`    | system 或 project | 项目成员管理                                                         |
| `/project/roles`                                           | `read_roles`    | system 或 project | 项目角色管理                                                         |

如果需要写操作，通常还要额外具备对应的 `write_*` scope，例如：

- 创建或编辑渠道：`write_channels`
- 创建或编辑 API Key：`write_api_keys`
- 创建或编辑角色：`write_roles`

## Share / Use 当前权限口径

Share / Use 方向现在处于“文档 + 最小可用表单”阶段，权限口径最好与当前实现一起理解：

- `/share` 入口属于 `Share/Use` 路由组，但实际仍复用渠道读权限；没有 `read_channels` 就无法进入。
- 由于 `read_channels` 当前只支持 system-level，所以 `/share` 目前更接近“有渠道管理权限的用户查看 Share 指南页”。
- `/use` 入口依赖 `read_api_keys`；它既可以由 system-level 角色授权，也可以由 project-level 角色授权。
- `/use` 的保存目标是当前 user API Key 的 active profile；因此在实际分配权限时，通常还要同时考虑谁可以创建、编辑或轮换 API Key。

## 常见授权策略

### 1. 系统管理员 / 渠道管理员

适合维护全局渠道、模型和系统配置：

- `read_channels` / `write_channels`
- `read_projects`
- `read_data_storages` / `write_data_storages`
- 按需要补充 `read_users` / `read_roles`

### 2. Project 开发者

适合在项目内自助创建 Key、查看请求和调试路由：

- `read_api_keys` / `write_api_keys`
- `read_requests`
- `read_prompts` / `write_prompts`（如果需要提示词功能）

### 3. 审计 / 只读排障角色

适合只看数据、不改配置：

- `read_requests`
- `read_api_keys`
- `read_users`
- `read_roles`

## 最佳实践

- 只授予完成工作所需的最小权限，避免把 `write_channels` 或 `write_roles` 直接给普通调用方。
- 把“系统级资源管理”和“项目内业务协作”分开授权，减少误操作范围。
- 为自动化流水线单独创建服务账号，并使用最小 scope 的 API Key。
- 定期轮换 API Key，回收不再使用的角色和项目成员关系。
- 在落地 Share / Use 能力时，优先确认 `/share` 与 `/use` 的访问入口是否符合目标用户画像，避免文档和实际授权口径脱节。

## 相关文档

- [共享/使用 MVP 指南](share-use-mvp.md) - 查看 Share / Use 的当前能力边界
- [API Key Profile 指南](api-key-profiles.md) - 了解 `/use` 页面实际写入的配置
- [渠道配置指南](channel-management.md) - 了解渠道管理与 Share 语义的当前关系
- [实体关系图](../development/erd.md) - 查看 Role、Scope、API Key、Project 关系
- [授权编码规范](../development/authz-coding-guidelines.md) - 查看开发侧权限实现约定
