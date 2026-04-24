# Antigravity 集成指南

---

## 概览

AxonHub 支持将 Google Antigravity API 配置为渠道提供商，可通过 Google 的基础设施访问 Claude、Gemini 和 GPT-OSS 等模型。本指南介绍如何配置 Antigravity 渠道，并说明智能端点回退、按模型冷却追踪和双配额池等高级能力。

### 核心要点

- Antigravity 通过统一的 Google 基础设施提供 Claude、Gemini、GPT-OSS 等多类模型访问能力。
- AxonHub 会在 Daily、Autopush 和 Production 三类端点之间自动执行故障转移。
- 按“模型 + 端点”维度记录冷却状态，避免持续向受限端点发送无效请求。
- 支持 Antigravity 与 Gemini CLI 两套配额池，可通过命名约定最大化可用容量。

### 前置条件

- 已部署 AxonHub，且当前账号具备渠道管理权限。
- 拥有可访问 Antigravity 的 Google 账号。
- 已通过 Antigravity OAuth 流程获取凭据。

---

## 配置 Antigravity 渠道

### 获取 OAuth 凭据

1. 进入 AxonHub 管理界面的 **渠道** 页面。
2. 点击 **创建渠道**，并选择 **Antigravity** 作为渠道类型。
3. 点击 **启动 OAuth** 发起认证流程。
4. 系统会生成 Google OAuth URL。点击 **打开 OAuth 链接**，在浏览器中完成认证。
5. 认证成功后，浏览器会跳转到回调 URL。复制完整的回调 URL。
6. 将回调 URL 粘贴到 AxonHub 表单中，并点击 **交换并填充 API 密钥**。
7. 凭据会自动填充。继续完成渠道配置：
   - **名称**：便于识别的名称，例如 `Antigravity - Daily`。
   - **Base URL**：默认使用 `https://daily-cloudcode-pa.sandbox.googleapis.com`，推荐保留默认值。
   - **支持模型**：添加需要暴露给客户端的模型，参考下方模型列表。
8. 点击 **测试** 验证连接。
9. 测试成功后启用渠道。

### 可用模型

Antigravity 可访问多个模型家族：

**Claude 模型（Anthropic via Google）：**

- `claude-sonnet-4-5`
- `claude-sonnet-4-5-thinking`
- `claude-opus-4-5-thinking`

**Gemini 模型（Google）：**

- `gemini-2.5-pro`
- `gemini-2.5-flash`
- `gemini-2.5-flash-lite`
- `gemini-3-pro-low`
- `gemini-3-pro-high`
- `gemini-3-pro-medium`
- `gemini-3-flash`
- `gemini-3-pro-image`

**GPT-OSS 模型：**

- `gpt-oss-120b-medium`

---

## 端点回退与健康追踪

AxonHub 为 Antigravity 实现了智能端点管理，用于提升可用性并充分利用配额。

### 可用端点

Antigravity 使用三类端点：

1. **Daily** (`https://daily-cloudcode-pa.sandbox.googleapis.com`)
   - 通常包含最新功能和模型。
   - Antigravity 配额模型的首选端点。
2. **Autopush** (`https://autopush-cloudcode-pa.sandbox.googleapis.com`)
   - 预发布环境。
   - 作为回退端点使用。
3. **Production** (`https://cloudcode-pa.googleapis.com`)
   - 稳定性最高。
   - Gemini CLI 配额模型的首选端点。

### 自动故障转移

当请求遇到可重试错误（429 Rate Limit、403 Forbidden、404 Not Found、5xx Server Error）时，AxonHub 会自动执行以下步骤：

1. 记录失败，并将该端点放入 **60 秒冷却期**。
2. 使用下一个可用端点重试请求。
3. 返回第一个成功响应。

示例：

```text
Request for claude-sonnet-4-5:
  Daily -> 429 Rate Limit -> [60s cooldown]
  Autopush -> 200 OK

Next request (within 60 seconds):
  [Skip Daily - in cooldown]
  Autopush -> 200 OK  (faster, no wasted attempt)
```

### 按模型追踪冷却状态

冷却状态会按模型与端点组合分别记录：

- 如果 `claude-sonnet-4-5` 在 Daily 端点触发限流，只有这个模型与 Daily 的组合进入冷却。
- `gemini-2.5-pro` 仍可继续使用 Daily，因为它是不同模型。
- `claude-sonnet-4-5` 仍可继续使用 Autopush 或 Production，因为它们是不同端点。

这种隔离方式可以最大化整个渠道可用配额的利用率。

### 快速失败行为

如果某个模型的所有端点都处于冷却期，AxonHub 会立即返回错误：

```text
Error: all antigravity endpoints in cooldown for model claude-sonnet-4-5
```

这样上游重试逻辑、渠道故障转移或其他提供商可以继续接管请求，而不会被阻塞。

---

## 双配额池

Antigravity 支持两套独立配额池：**Antigravity** 和 **Gemini CLI**。可以通过特殊命名规则让同一模型使用不同配额池。

### 默认配额分配

**Antigravity 配额**（优先使用 Daily 端点）：

- 所有 Claude 模型（`claude-*`）
- 所有 GPT-OSS 模型（`gpt-*`）
- 图像生成模型（`*-image`、`*-imagen`）
- 旧版 Gemini 3 模型（`gemini-3-pro-low`、`gemini-3-flash` 等）

**Gemini CLI 配额**（优先使用 Production 端点）：

- 标准 Gemini 模型（`gemini-2.5-pro`、`gemini-2.5-flash`、`gemini-1.5-pro`）
- Preview 模型（`gemini-3-pro-preview`、`gemini-3-flash-preview`）

### 显式覆盖配额池

可以通过后缀写法覆盖默认配额池：

- `:antigravity`：强制使用 Antigravity 配额。
- `:gemini-cli`：强制使用 Gemini CLI 配额。

示例：

```text
gemini-2.5-pro              -> Gemini CLI quota (default)
gemini-2.5-pro:antigravity  -> Antigravity quota (override)
```

### 同时使用两套配额池

添加 `antigravity-` 前缀后，可以为同一模型访问另一套配额池：

```text
gemini-2.5-pro                  -> Gemini CLI quota only
antigravity-gemini-2.5-pro      -> Antigravity quota (separate pool)
```

使用场景：在支持模型列表中同时配置两个模型名称，以最大化可用配额：

```yaml
supported_models:
  - gemini-2.5-pro               # Uses Gemini CLI quota
  - antigravity-gemini-2.5-pro   # Uses Antigravity quota
```

当某个配额池耗尽后，AxonHub 可以通过渠道重试逻辑自动切换到另一套配额池。

---

## 模型路由示例

| 模型名称 | 配额池 | 初始端点 | 回退顺序 |
| -------- | ------ | -------- | -------- |
| `claude-sonnet-4-5` | Antigravity | Daily | Daily -> Autopush -> Prod |
| `gemini-2.5-pro` | Gemini CLI | Prod | Prod -> Daily -> Autopush |
| `gemini-2.5-pro:antigravity` | Antigravity | Daily | Daily -> Autopush -> Prod |
| `antigravity-gemini-2.5-pro` | Antigravity | Daily | Daily -> Autopush -> Prod |
| `gemini-3-flash` | Antigravity | Daily | Daily -> Autopush -> Prod |
| `gpt-oss-120b-medium` | Antigravity | Daily | Daily -> Autopush -> Prod |

---

## 最佳实践

### 最大化配额利用率

1. **使用双配额池**：同时配置标准模型名和带 `antigravity-` 前缀的模型名。

   ```yaml
   supported_models:
     - gemini-2.5-pro
     - antigravity-gemini-2.5-pro
   ```

2. **配合模型配置文件**：使用 AxonHub 模型配置文件在配额池之间自动路由。
   - 创建 `gemini-2.5-pro` 到 `antigravity-gemini-2.5-pro` 的回退映射。
   - 当 Gemini CLI 配额耗尽时，通过配置文件路由切换到 Antigravity 配额。
3. **配置多个渠道**：为不同端点创建独立渠道，手动控制优先级。
   - 渠道 1：Antigravity（Daily 端点），优先级 10。
   - 渠道 2：Antigravity（Production 端点），优先级 5。

### 性能优化

1. **关注冷却状态**：配置健康检查，及时发现端点是否进入冷却期。
2. **合理设置渠道优先级**：如果稳定性比新功能更重要，可提高 Production 端点优先级。
3. **使用负载均衡**：通过 AxonHub 自适应负载均衡，将请求分发到健康端点。

### 监控建议

建议在 AxonHub 追踪数据中关注以下指标：

- **端点失败次数**：每个端点出现 429 或 5xx 错误的频率。
- **冷却事件**：端点进入冷却期的次数。
- **回退成功率**：通过回退端点成功完成请求的比例。
- **按模型统计的配额耗尽情况**：哪些模型或配额池最容易触发限制。

---

## 故障排查

### 渠道测试失败

**现象**：OAuth 交换成功，但渠道测试失败。

**解决方法**：

- 确认凭据中的项目 ID 正确。
- 检查 AxonHub 实例是否能访问配置的 Base URL。
- 确认当前端点支持该模型名称。
- 尝试切换到其他端点（Daily、Autopush 或 Production）。

### 所有端点都处于冷却期

**现象**：请求返回 `all antigravity endpoints in cooldown`。

**解决方法**：

- 配置更多 Antigravity 渠道作为回退。
- 使用模型配置文件路由到其他提供商。
- 在客户端侧实现请求排队或限流。
- 如有需要，联系 Google 提升配额。

### 单个模型配额耗尽

**现象**：某个模型持续触发限流，而其他模型仍可正常使用。

**解决方法**：

- 添加带 `antigravity-` 前缀的模型名，使用另一套配额池。
- 使用模型配置文件路由到替代模型，例如 `gemini-2.5-pro` 到 `gemini-2.5-flash`。
- 将流量分散到多个模型变体。
- 检查请求是否可以批量处理或缓存。

### OAuth Token 过期

**现象**：渠道曾经可用，但之后请求返回 401 Unauthorized。

**解决方法**：

- 重新执行 OAuth 流程获取新凭据。
- 使用新的 OAuth 凭据更新渠道。
- 检查渠道设置中的 token 过期时间。
- 如果版本支持，启用自动刷新 token。

### 模型在端点上不可用

**现象**：请求返回 404 Not Found。

**解决方法**：

- 确认模型名称正确，参考上方支持模型列表。
- 尝试切换到其他端点，因为部分模型可能只存在于特定端点。
- 查看 Google Antigravity 文档确认模型可用性。
- 使用 `antigravity-` 前缀或 `:antigravity` 后缀尝试 Antigravity 配额池。

---

## 高级配置

### 自定义冷却时间

冷却时间默认固定为 60 秒。当前实现中该值是固定常量，如需修改可以：

1. Fork 仓库。
2. 修改 `llm/transformer/antigravity/health_tracker.go` 中的 `DefaultCooldownDuration`。
3. 重新构建 AxonHub。

### 健康追踪统计

健康追踪器会维护端点健康统计。自定义集成可以通过以下方式读取：

```go
// Example: Access health tracker stats (for custom integrations)
stats := healthTracker.Stats()
fmt.Printf("Total entries: %d\n", stats.TotalEntries)
fmt.Printf("In cooldown: %d\n", stats.InCooldown)
fmt.Printf("Expired: %d\n", stats.Expired)
```

### 内存管理

健康追踪器使用基于 TTL 的惰性清理：

- 条目在 **10 分钟** 无访问后过期。
- 内存使用量约为每个“模型 + 端点”组合 200 字节。
- 典型部署中，20 个模型乘以 3 个端点约占用 12 KB 内存。

系统不会执行后台清理；条目会在下次访问且发现过期时被移除。

---

## 相关文档

- [渠道管理指南](channel-management.md)
- [模型管理指南](model-management.md)
- [负载均衡指南](load-balance.md)
- [请求追踪指南](tracing.md)
- [OpenAI API](../api-reference/openai-api.md)
- [Anthropic API](../api-reference/anthropic-api.md)
- [Gemini API](../api-reference/gemini-api.md)

---

## FAQ

**Q: 端点会保持多久冷却状态？**  
A: 最近一次失败后会冷却 60 秒；成功请求会立即清除冷却状态。

**Q: 可以创建多个 Antigravity 渠道吗？**  
A: 可以。你可以为 Daily、Autopush 和 Production 分别创建渠道，并通过优先级或负载均衡控制路由。

**Q: 健康追踪状态会在重启后保留吗？**  
A: 不会。健康追踪状态只保存在内存中，重启后冷却状态会清空，这是为了让服务重启后恢复干净状态。

**Q: 使用无效模型名会怎样？**  
A: 请求会返回 404 Not Found，并触发端点回退。如果所有端点都返回 404，AxonHub 会将错误返回给客户端。

**Q: 可以关闭端点回退吗？**  
A: 当前没有直接开关；但你可以只为渠道配置一个端点，从而达到近似关闭回退的效果。

**Q: 如何知道请求最终由哪个端点处理？**  
A: 查看 AxonHub 追踪日志。回退成功时会记录 `antigravity request succeeded with fallback endpoint`。
