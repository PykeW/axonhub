# 渠道配置指南

本指南介绍如何在 AxonHub 中配置 AI 服务提供商（如 OpenAI、Anthropic、DeepSeek 等）。

## 什么是渠道？

**渠道**是 AxonHub 连接 AI 提供商的通道。你可以把渠道理解为"供应商连接线"——每个渠道对应一个 AI 服务商（如 OpenAI、Claude、DeepSeek）。

通过渠道，你可以：

- 同时连接多个 AI 服务商
- 设置模型名称转换规则
- 启用或暂停某个服务商
- 配置多个 API Key 实现负载均衡

## 渠道模型映射在请求流程中的位置

渠道模型映射是三层流水线中的**最后一步**。完整说明请参阅 [请求处理流程](../getting-started/request-processing.md#核心概念三层模型设置)。

简单来说：**API Key Profile 改模型名 → 模型关联选渠道 → 渠道改模型名 → 发给上游**

## 创建渠道

### 基本步骤

1. 进入 AxonHub 管理界面 → **渠道管理**
2. 点击 **新建渠道**
3. 填写基本信息：
   - **名称**：给渠道起个名字（如"OpenAI 主账号"、"DeepSeek 国内"）
   - **类型**：选择服务商类型（OpenAI、Anthropic、DeepSeek 等）
   - **Base URL**：API 地址（一般使用默认值即可）
   - **API Key**：服务商提供的密钥

### 配置示例

**OpenAI 渠道：**

| 字段     | 值                         |
| -------- | -------------------------- |
| 名称     | OpenAI 主账号              |
| 类型     | openai                     |
| Base URL | https://api.openai.com/v1  |
| API Key  | sk-your-openai-key         |
| 支持模型 | gpt-4o, gpt-4o-mini, gpt-5 |

**DeepSeek 渠道：**

| 字段     | 值                               |
| -------- | -------------------------------- |
| 名称     | DeepSeek 国内                    |
| 类型     | deepseek                         |
| Base URL | https://api.deepseek.com/v1      |
| API Key  | sk-your-deepseek-key             |
| 支持模型 | deepseek-chat, deepseek-reasoner |

## 配置多个 API Key

当一个账号有多个 API Key 时，可以都配置到同一个渠道中，AxonHub 会自动轮流使用，提高稳定性。

在渠道编辑界面的 **API Key** 区域，逐行添加多个 Key 即可，例如：

- `sk-key-1`
- `sk-key-2`
- `sk-key-3`

### 负载均衡说明

- 相同的 Trace ID 会始终使用同一个 Key（保证会话一致性）
- 不同请求会随机选择可用的 Key
- 某个 Key 出错时，系统会自动切换到其他 Key

## 模型映射配置

**什么时候需要模型映射？**

当你想让客户端用一个名称请求，但实际发给上游的是另一个名称时。

**常见场景：**

1. **客户端用简化的名称**：客户端请求 `gpt-4`，实际发给 OpenAI 的是 `gpt-4o`
2. **统一不同渠道的模型名**：让 `claude-sonnet` 和 `gpt-4` 都指向同一个实际模型
3. **旧版兼容**：客户端请求旧版模型名，自动映射到新版

### 配置方法

在渠道的 **Settings** 中的模型映射区域添加：

| 客户端请求的模型名 (from) | 实际发给上游的模型名 (to) |
| ------------------------- | ------------------------- |
| gpt-4o-mini               | gpt-4o                    |
| claude-3-sonnet           | claude-3.5-sonnet         |

**注意**：目标模型（to）必须在 `supported_models` 列表中。

## 测试和启用渠道

### 测试连接

在启用渠道前，建议先测试连接：

1. 在渠道列表中找到刚创建的渠道
2. 点击 **测试** 按钮
3. 等待测试结果
4. 如果显示成功，说明配置正确

### 启用渠道

测试通过后，点击 **启用** 按钮，渠道状态变为 **活跃**，即可开始接收请求。

## Share / Use 场景补充

如果你正在按 [共享/使用 MVP 指南](share-use-mvp.md) 规划“共享自己的渠道给其他用户使用”，需要注意当前实现仍是过渡状态：

- `/share` 页面目前是指南和导航包装页，真实的渠道创建、编辑、启用、测试仍在现有 **渠道管理** 页面完成。
- 后端对象层已经有 `ChannelSettings.Share` 默认值和 `private/shared` 可见性枚举，但渠道 GraphQL schema、表单和列表还没有完整暴露这些字段。
- 因此，本文档中的 Share 语义更适合作为当前操作说明 + 目标能力说明，而不是“已完整上线的独立 Share CRUD 页面”。

### `private` / `shared` 的目标语义

| 可见性    | 目标含义                                           | 当前实现状态                                                 |
| --------- | -------------------------------------------------- | ------------------------------------------------------------ |
| `private` | 仅渠道所有者自己的 Use API Key 可以使用            | 还未在渠道表单中直接配置，当前仍主要靠现有权限和管理约束实现 |
| `shared`  | 渠道可进入共享候选池，供允许共享容量的用户路由使用 | 语义已写入文档与对象层，真实路由与脱敏读取仍待后续开发       |

### 5 小时刷新窗口（目标规则）

Share / Use 方案里的刷新窗口约定如下：

- 每个共享渠道独立计算刷新窗口
- 第一次刷新应允许立即执行
- 刷新成功后，下一次允许刷新时间应顺延 5 小时
- 并发刷新同一渠道时，应该只有一个成功

当前渠道管理页还没有直接展示 `nextRefreshAt` 或刷新冷却状态，因此现阶段不要把它当成已经可用的后台表单能力；需要以 [共享/使用 MVP 指南](share-use-mvp.md) 的目标验收为准。

### 当前推荐操作流程

在 Share 独立页面和字段完全落地前，推荐按下面顺序操作：

1. 在 **渠道管理** 页面完成渠道基础信息、Base URL、API Key 和支持模型配置。
2. 在 **模型管理** 页面确认模型暴露是否正确。
3. 用 **测试** / **启用** 和请求追踪验证渠道是否稳定可用。
4. 如果要为后续共享能力做准备，优先保证 `supported_models`、默认测试模型和敏感凭据管理清晰，不要依赖尚未开放的 Share 专用字段。

## Base URL 特殊配置

### 默认地址

| 服务商    | 默认 Base URL                                      |
| --------- | -------------------------------------------------- |
| OpenAI    | `https://api.openai.com/v1`                        |
| Anthropic | `https://api.anthropic.com`                        |
| DeepSeek  | `https://api.deepseek.com/v1`                      |
| Gemini    | `https://generativelanguage.googleapis.com/v1beta` |

### 自定义地址

如果使用代理或私有化部署，可以修改 Base URL。

**禁用版本号自动追加**：在 URL 末尾加 `#`

```
https://custom-proxy.example.com/api#
# 实际请求: /api/messages（不会自动加 /v1）
```

**完全原始模式**：在 URL 末尾加 `##`

```
https://custom-gateway.example.com/api##
# 实际请求: /api（不会加版本号和端点路径）
```

## 常见问题

### Q: 测试连接失败怎么办？

- 检查 API Key 是否正确（复制时是否有多余空格）
- 确认 Base URL 是否可访问
- 检查服务商账户是否有余额/额度

### Q: 请求时提示"模型未找到"？

- 确认模型已在渠道的 `supported_models` 中
- 检查模型映射配置是否正确
- 确认渠道已启用

### Q: 如何设置多个 API Key？

在 `credentials.api_keys` 中列出所有 Key，系统会自动轮询使用。

### Q: API Key 被禁用了怎么恢复？

进入渠道详情，在 **禁用列表** 中找到该 Key，点击 **恢复**。

## 相关文档

- [模型管理指南](model-management.md) - 配置模型与渠道的关联关系
- [API Key Profile 指南](api-key-profiles.md) - 配置模型映射和访问权限
- [请求处理流程](../getting-started/request-processing.md) - 了解完整请求链路
