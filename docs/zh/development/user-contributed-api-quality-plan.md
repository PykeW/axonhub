# 用户贡献 API：模型真实性验证、随机抽检与积分奖惩设计

## 背景

AxonHub 当前 Relay/Sub-Key MVP 的核心是“平台运营方自托管共享容量”：运营人员配置上游 `channels`，将多个渠道绑定到 `relay_products`，再给项目发放下游 `relay_keys`。该模式适合平台先跑通产品、渠道池、Sub-Key、钱包、请求排障和结算闭环。

下一阶段希望探索“用户共享自己的 API / 模型容量换取平台积分，再用积分兑换或抵扣 token 使用”的机制。这会把 AxonHub 从单一运营方容量池，扩展为小范围、多供给方的共享容量网络。为了避免劣质 API、伪造模型、转接弱模型冒充强模型、刷积分和用户体验下降，必须先建立模型真实性验证、随机抽检、质量评分和奖惩闭环。

本文定义前期小范围测试方案，目标是先验证信任与风控机制，不直接扩展成公开市场。

## 设计目标

1. 允许可信种子用户提交自己的上游 API Key / 兼容接口作为贡献渠道。
2. 平台能验证贡献渠道是否可用、是否基本符合声明的 provider / model 能力。
3. 平台能在不暴露探针题库的前提下进行随机抽检。
4. 平台能根据真实请求、抽检结果和用户反馈计算质量分。
5. 平台能对优质 API 给出积分奖励，对劣质或欺诈 API 降权、冻结、扣罚或下线。
6. 平台能控制小范围测试成本，并保留人工复核与申诉通道。
7. 所有积分变动必须可追溯、幂等、可回滚，不允许重复入账或异常扣罚。

## 非目标

前期试点不做以下能力：

- 不开放公开 API 市场或自由买卖上游 Key。
- 不承诺绝对证明“某 API 背后一定是真实官方模型”。
- 不做法币提现、公开分润、发票、支付网关或复杂财务结算。
- 不允许贡献者看到完整探针题、标准答案或反作弊细节。
- 不让未知用户直接把贡献 API 承接生产核心流量。
- 不对所有 provider 一次性泛化，先选择 1-2 类兼容协议做试点。

### OpenAI-compatible 试点适配边界

- 前期只对选定 provider / base_url 组合做严格校验，例如模型列表、usage 字段、错误码和声明模型一致性。
- 其他 OpenAI-compatible 兼容层先只做基础可用性、鉴权、响应格式和最小 usage 存在性校验，不把 provider 身份作为强承诺。
- usage 统计、错误码归一化、model list 解析和特殊能力边界需按 provider adapter 逐步扩展，不能用一套泛化规则覆盖所有兼容接口。

## 当前基础与缺口

### 可复用基础

| 现有对象 | 可复用点 |
| --- | --- |
| `channels` | 已承载上游 provider 凭证、模型能力、状态、错误信息和路由配置。 |
| `relay_product_channels` | 已表达产品与上游渠道池绑定，可扩展质量状态和权重策略。 |
| `requests` | 可作为下游请求事实与审计锚点。 |
| `request_executions` | 可记录实际命中的上游渠道，是贡献计量和质量分析的核心事实。 |
| `usage_logs` | 可记录 token、成本和模型使用事实，后续可用于积分结算。 |
| `provider_quota_status` | 可复用上游额度/可用性检测结果。 |
| `relay_wallet_ledger_entries` | 已体现不可变流水和 idempotency key 思路，可借鉴到积分账本。 |

### 主要缺口

| 缺口 | 影响 |
| --- | --- |
| 无贡献者模型 | 不知道某个上游 API 属于哪个用户，也无法给贡献者发放积分。 |
| 无贡献渠道质量状态 | 只能判断渠道启停或 quota，无法表达真实模型、劣质、欺诈、待观察。 |
| 无模型探针题库 | 无法系统性验证模型能力和声明一致性。 |
| 无抽检任务与结果 | 无法持续发现后期降级、替换、缓存回复或间歇故障。 |
| 无用户积分账本 | 无法把贡献收入和消费抵扣统一结算。 |
| 无奖惩事件 | 处罚、冻结、恢复、申诉缺少证据链。 |
| 无供给侧页面 | 用户无法提交 API、查看质量、收益、处罚和申诉。 |

## 角色与权限

| 角色 | 说明 | 允许动作 | 禁止动作 |
| --- | --- | --- | --- |
| 贡献者 | 提交自己 API / 模型容量的用户 | 提交 API、设置额度、查看质量分、查看积分、暂停共享、发起申诉 | 查看探针原题、查看他人渠道、修改评分规则 |
| 消费方 | 使用平台积分调用模型的用户或项目 | 使用平台 Sub-Key、查看消费积分、反馈质量问题 | 指定命中某贡献者 API、自行篡改结算 |
| 平台运营 | 管理试点和共享池的管理员 | 审核贡献渠道、查看抽检结果、调整权重、处理申诉 | 直接修改不可变积分流水 |
| 风控/客服 | 处理异常、投诉和处罚 | 冻结、解冻、备注、人工复核 | 绕过审计删除证据 |
| 系统任务 | 定时抽检、质量计算和积分释放 | 创建模型真实性抽检任务、计算质量分、生成幂等积分流水 | 使用普通用户上下文访问私密题库 |

## 模型真实性定义

黑盒 API 不能被绝对证明其背后真实模型，只能通过多信号提高置信度。因此平台应使用以下表述：

> `verified` 表示该贡献渠道通过了 AxonHub 当前模型可用性、声明一致性和质量抽检标准，不代表平台对上游 provider 身份或模型来源作绝对法律背书。

真实性分为四层：

1. **API 可用性真实**：Key 能鉴权成功，模型能返回有效响应，错误率和延迟符合阈值。
2. **声明一致性较高**：贡献者声明的 provider / model 与响应元数据、usage 结构、能力表现基本一致。
3. **质量达到阈值**：在探针、真实流量和用户反馈中达到平台质量分要求。
4. **未发现欺诈行为**：未发现模型替换、固定回复、缓存题库、伪造 usage、自刷积分、恶意中转等行为。

## 验证信号分层

| 层级 | 信号 | 说明 | 成本 | 可靠性 |
| --- | --- | --- | --- | --- |
| L1 元数据校验 | models endpoint、provider 字段、usage 结构、错误码 | 快速发现明显不匹配，但代理可伪造 | 低 | 中 |
| L2 能力探针 | 数学、逻辑、代码、多语言、格式遵循、长上下文 | 检查模型能力是否符合声明 | 中 | 中高 |
| L3 指纹对比 | 与可信基线渠道比较边界能力和输出风格 | 识别疑似降级或替换 | 中高 | 中 |
| L4 随机抽检 | 私有题库随机抽取，不提前公开 | 防止针对固定题作弊 | 中 | 高 |
| L5 真实流量质量 | 成功率、延迟、重试率、用户反馈、投诉 | 最贴近真实体验 | 低到中 | 高 |
| L6 对抗测试 | nonce、时间敏感题、反缓存题、陷阱题 | 发现固定回复和缓存 | 中 | 高 |

## 探针题库设计

### 题库分类

| 类别 | 目标 | 示例检查方式 | 是否可自动处罚 |
| --- | --- | --- | --- |
| 基础可用性题 | 检查 API 是否可请求、是否返回有效文本 | JSON schema、状态码、非空响应 | 可轻度处罚 |
| 模型能力题 | 区分强弱模型能力 | 数值答案、结构化答案、judge_model | 谨慎处罚 |
| 模型声明一致性题 | 判断是否明显不符合声明模型族 | 对照模型、能力边界、usage 格式 | 需多次命中 |
| 反缓存题 | 避免贡献者缓存固定答案 | nonce、随机变量、时间戳 | 可处罚 |
| 格式遵循题 | 检查严格输出格式 | JSON schema、regex | 可轻度处罚 |
| 安全与拒答题 | 发现严重不安全或异常回复 | 分类器、人工复核 | 需人工复核 |
| 真实业务影子题 | 贴近真实使用场景 | 脱敏题、judge_model、用户反馈 | 不单独处罚 |

### 题目字段

建议 `model_authenticity_probe_cases` 至少包含：

| 字段 | 说明 |
| --- | --- |
| `id` | 探针题 ID。 |
| `category` | 题目类别。 |
| `target_provider` | 适用 provider。 |
| `target_model_family` | 适用模型族，例如 GPT、Claude、Gemini、OpenAI compatible。 |
| `difficulty` | `easy` / `medium` / `hard`。 |
| `prompt_template` | 私有 prompt 模板，可包含变量。 |
| `expected_check_type` | `exact` / `regex` / `json_schema` / `judge_model` / `embedding_similarity` / `human_review`。 |
| `expected_payload` | 标准答案、schema 或评分提示。 |
| `anti_cache_nonce` | 是否注入 nonce。 |
| `cost_weight` | 预计成本权重。 |
| `sensitivity` | 是否允许自动处罚。 |
| `version` | 题目版本。 |
| `is_active` | 是否启用。 |

### 题库保密策略

- 探针原题和标准答案只允许系统任务和少数运营角色访问。
- 贡献者只看抽检结论、类别、摘要和影响，不看原题。
- 题库需要定期轮换，旧题降权或停用。
- 所有模型真实性探针请求注入 `authenticity_probe_run_id`、nonce 和 payload hash，防止复用标准答案。
- 生产环境日志不得打印完整探针 prompt。

## 判别方法

### 程序化校验

适合低成本、可确定答案的题：

- JSON schema。
- 正则匹配。
- 数字答案容差。
- 字段完整性。
- 是否引用 nonce。
- 是否满足最大/最小长度。

### 对照模型比较

对于开放题，可用平台可信渠道跑同一题作为基线，比较：

- 是否覆盖关键点。
- 是否遵循输出格式。
- 是否出现明显弱模型错误。
- 是否和声明模型族能力差距过大。

### LLM-as-judge

可用于开放题评分，但要有防护：

- 裁判模型必须来自平台可信渠道。
- 裁判结果只作为评分信号，不作为单次封禁依据。
- 严重处罚前必须结合多次探针、真实流量和人工复核。

### 人工复核

以下情况必须人工复核：

- 首次 P3 / P4 处罚。
- 大额积分扣罚。
- 贡献者申诉。
- 安全类或模型冒充类重大事件。
- 判别器给出 `inconclusive` 但系统准备下线渠道。

## 随机抽检机制

### 抽检触发

| 触发类型 | 触发条件 | 目标 |
| --- | --- | --- |
| 入驻验证 | 用户首次提交 API | 防止无效 API 入池。 |
| 定时抽检 | 活跃渠道每 6-24 小时 | 发现后期失效或降级。 |
| 流量抽检 | 每 N 次成功请求后 | 结合真实承载量动态抽查。 |
| 风险抽检 | 错误率、延迟、投诉、quota 异常 | 快速定位风险渠道。 |
| 奖励释放前抽检 | 待结算积分达到阈值 | 降低刷积分和短期欺诈。 |

### 小范围默认比例

| 场景 | 建议比例 |
| --- | --- |
| 入驻验证 | 100% 执行。 |
| 日常轻量抽检 | 每个活跃贡献渠道每日 1-3 次。 |
| 流量抽检 | 真实请求量的 0.5%-2%。 |
| 风险渠道 | 提升到 5%-10%，或直接暂停新流量。 |
| 稳定优质渠道 | 可降低频率，但不能为 0。 |

### 抽检隔离

- 抽检请求必须标记为 `probe`，不计入消费方账单。
- 抽检成本由平台试点预算承担。
- 抽检请求不能污染普通用户的请求列表，除非运营切换到质量治理视图。
- 抽检任务必须有独立 `idempotency_key`，避免重复处罚。
- 抽检失败只应先影响贡献渠道质量，不应直接影响消费方余额。

## 质量分模型

每个贡献渠道维护 `quality_score`，范围 0-100。

初期建议公式：

```text
quality_score =
  35% * availability_score
+ 25% * probe_score
+ 15% * latency_score
+ 15% * settlement_integrity_score
+ 10% * complaint_score
```

### 子分说明

| 子分 | 来源 | 说明 |
| --- | --- | --- |
| `availability_score` | 成功率、超时率、5xx、鉴权失败 | 衡量是否稳定可用。 |
| `probe_score` | 抽检通过率、严重失败次数 | 衡量真实性和能力一致性。 |
| `latency_score` | P50 / P95 / P99 延迟 | 衡量响应体验。 |
| `settlement_integrity_score` | usage 返回、token 统计、请求事实完整性 | 衡量结算是否可信。 |
| `complaint_score` | 用户反馈、人工投诉、退款事件 | 衡量真实体验。 |

### 状态阈值

| 分数 | 状态 | 路由策略 | 积分策略 |
| --- | --- | --- | --- |
| 90-100 | `excellent` | 可小幅优先分配 | 奖励倍率 1.05-1.10 |
| 75-89 | `good` | 正常参与路由 | 标准积分 |
| 60-74 | `watch` | 权重降低，提高抽检率 | 延迟释放或小幅折扣 |
| 40-59 | `degraded` | 暂停新流量或仅低优先级 | 冻结部分待结算积分 |
| < 40 | `blocked` | 不进入候选池 | 取消待结算积分，必要时扣罚 |

### 路由权重调整

```text
blocked/degraded -> 不进入正常候选池
watch -> 原始权重 * 0.3，并提高抽检率
verified/good -> 原始权重
excellent -> 原始权重 * 1.1，但仍受成本、额度和公平性限制
```

## 积分奖励机制

### 积分与 Relay 钱包的关系

试点期推荐把用户积分账本和 Relay 钱包账本保持独立：

- 用户积分使用独立 `user_point_accounts` / `user_point_ledger_entries`，表达贡献奖励、释放、冻结、扣罚、消费和过期。
- Relay 钱包继续使用 `relay_wallets` / `relay_wallet_ledger_entries`，表达 Sub-Key 或项目侧 token 使用、充值、扣费和退款。
- 积分用于兑换或抵扣 token 使用时，必须通过可追溯的转换/抵扣记录连接两套账本，不直接混写积分流水和钱包流水。

推荐兑换/抵扣流程：

1. 先扣减用户积分，写入 `user_point_ledger_entries`，`scene` 使用 `consume` 或 `point_redeem`，并带独立 `idempotency_key`。
2. 积分扣减成功后，再创建或关联 Relay 钱包侧充值/抵扣流水，避免钱包入账成功但积分未扣减。
3. 记录 `conversion_rate_snapshot`、`related_wallet_ledger_entry_id`、`related_relay_key_id`、`related_project_id` 等关联字段，或在策略配置中保存等价快照。
4. 若钱包侧入账或抵扣失败，必须追加补偿流水或回滚积分扣减，保证同一兑换请求可审计、可回滚、不会重复入账。

### 积分状态

| 状态 | 说明 |
| --- | --- |
| `pending_points` | 请求成功后生成，等待质量窗口结束。 |
| `available_points` | 已释放，可用于兑换或抵扣 token。 |
| `frozen_points` | 因投诉、抽检失败、申诉中等原因冻结。 |
| `spent_points` | 已消费积分。 |
| `expired_points` | 过期积分。 |
| `penalized_points` | 因确认违规而扣罚的积分。 |

### 奖励公式

```text
contribution_points_pending =
  base_token_value
* model_multiplier
* quality_multiplier
* availability_multiplier
* platform_adjustment
```

字段说明：

| 字段 | 说明 |
| --- | --- |
| `base_token_value` | 按真实承接 token 或请求价值计算的基础积分。 |
| `model_multiplier` | 高成本/高价值模型倍率更高。 |
| `quality_multiplier` | 由质量状态决定，优质略增，风险渠道折扣。 |
| `availability_multiplier` | 长期稳定可用可提高，间歇故障降低。 |
| `platform_adjustment` | 平台运营手动策略，例如试点期保护系数。 |

### 释放窗口

- 默认进入 `pending_points`。
- 经过 1-24 小时质量窗口后释放为 `available_points`。
- 若窗口内出现投诉、抽检失败、usage 异常，则冻结相关积分。
- 大额释放前触发 `pre_reward_release` 抽检。
- 释放和扣罚都必须通过不可变积分流水完成。

### 自刷与同主体消费规则

- 当贡献者 `contributor_user_id` / `contributor_project_id` 与消费方 `user_id` / `project_id` 判定为同主体时，默认不发放贡献奖励，或只按保守低倍率发放。
- 试点期先采用保守规则，例如同用户、同项目、同组织或明显关联项目直接判定为同主体。
- 后续再引入异常图谱、设备/IP/支付主体/调用模式等更复杂信号，避免过早依赖不可解释的自动风控。

## 惩罚机制

| 等级 | 场景 | 处理 | 是否需要人工复核 |
| --- | --- | --- | --- |
| P0 提示 | 单次超时、轻微格式错误 | 记录事件，轻微扣分或不扣分 | 否 |
| P1 降权 | 错误率升高、连续轻量探针失败 | 降低权重，提高抽检率 | 否 |
| P2 冻结 | 疑似降级、连续严重失败 | 暂停新流量，冻结待结算积分 | 建议 |
| P3 扣罚 | 证实伪造模型、恶意刷分、盗用 Key | 扣除相关待结算积分和信誉分 | 是 |
| P4 封禁 | 多次恶意或严重安全事件 | 禁用贡献渠道，限制贡献账户 | 是 |

### 扣罚原则

- 优先冻结 `pending_points`，不要直接扣可用积分。
- 只有证据链完整时才扣罚已释放积分。
- 扣罚必须绑定 `quality_event_id`、`authenticity_probe_run_id` 或 `request_id`。
- 每次扣罚必须有 `idempotency_key`。
- 人工调整必须记录 `operator_user_id` 和备注。

## 申诉机制

贡献者应能看到：

- 处罚时间。
- 处罚等级。
- 抽检类别。
- 摘要原因。
- 影响的积分。
- 当前状态。
- 可提交的补充说明。

贡献者不应看到：

- 完整探针 prompt。
- 标准答案。
- 反作弊判断细节。
- 其他贡献者数据。

申诉流程：

1. 贡献者提交申诉。
2. 系统将相关质量事件、模型真实性抽检任务、请求摘要、积分流水打包成复核上下文。
3. 风控/运营人工复核。
4. 复核通过：恢复质量分、释放冻结积分、记录误判原因。
5. 复核失败：维持处罚，提高后续抽检率。
6. 所有操作落 `channel_quality_events`。

## 建议数据模型

### 与现有 `ChannelProbe` 的区别

现有 `ChannelProbe` / `channel_probes` 用于渠道性能采样和趋势图，例如 total/success request count、tokens/sec、TTFT 等运行指标。本文新增的 `model_authenticity_probe_cases` 与 `model_authenticity_probe_runs` 只用于模型真实性/质量抽检，例如能力题、声明一致性、反缓存和判别结果。

两者都可以关联 `channels.id`，但不可共用表、字段语义或业务处理流程；性能探针不应被解释为模型真实性证据，模型真实性探针也不应污染渠道性能趋势统计。

### `contributed_channels`

用户贡献的 API / Channel 扩展信息，与现有 `channels` 一对一或一对多关联。

| 字段 | 说明 |
| --- | --- |
| `id` | 主键。 |
| `channel_id` | 关联现有 `channels.id`。 |
| `contributor_user_id` | 贡献者用户。 |
| `contributor_project_id` | 可选，贡献者所在项目。 |
| `declared_provider` | 用户声明 provider。 |
| `declared_models` | 用户声明可提供模型列表。 |
| `share_status` | `pending_verification` / `verification_timeout` / `active` / `watch` / `suspended` / `blocked` / `withdrawn`。 |
| `quality_score` | 0-100。 |
| `quality_status` | `unverified` / `verified` / `degraded` / `blocked`。 |
| `daily_token_limit` | 用户设置或平台限制的日 token 上限。 |
| `daily_request_limit` | 日请求上限。 |
| `max_inflight` | 最大并发。 |
| `last_verified_at` | 最近通过入驻验证时间。 |
| `last_probe_at` | 最近抽检时间。 |
| `penalty_active_until` | 惩罚生效截止时间。 |
| `created_at` / `updated_at` | 时间戳。 |

### `model_authenticity_probe_cases`

私有探针题库。

| 字段 | 说明 |
| --- | --- |
| `id` | 主键。 |
| `category` | 探针类别。 |
| `target_provider` | 适用 provider。 |
| `target_model_family` | 适用模型族。 |
| `difficulty` | 难度。 |
| `prompt_template` | 私有 prompt 模板。 |
| `expected_check_type` | 校验方式。 |
| `expected_payload` | 预期答案、schema 或 judge rubric。 |
| `cost_weight` | 成本权重。 |
| `sensitivity` | 是否允许自动处罚。 |
| `is_active` | 是否启用。 |
| `version` | 版本。 |

### `model_authenticity_probe_runs`

每一次抽检任务。

| 字段 | 说明 |
| --- | --- |
| `id` | 主键。 |
| `contributed_channel_id` | 被抽检贡献渠道。 |
| `channel_id` | 实际上游 channel。 |
| `probe_case_id` | 题目 ID。 |
| `trigger_type` | `onboarding` / `scheduled` / `traffic_sample` / `risk` / `pre_reward_release`。 |
| `requested_model` | 请求模型。 |
| `request_payload_hash` | 请求 payload hash。 |
| `response_hash` | 响应 hash。 |
| `status` | `passed` / `failed` / `inconclusive` / `timeout` / `error`。 |
| `score` | 本次分数。 |
| `latency_ms` | 延迟。 |
| `token_usage` | token 用量。 |
| `failure_reason` | 失败原因。 |
| `judge_metadata` | 判别器元数据。 |
| `idempotency_key` | 幂等键。 |
| `created_at` | 创建时间。 |

### `channel_quality_events`

质量分、奖惩、冻结、恢复、申诉证据链。

| 字段 | 说明 |
| --- | --- |
| `id` | 主键。 |
| `contributed_channel_id` | 关联贡献渠道。 |
| `event_type` | `score_change` / `reward_bonus` / `penalty` / `freeze` / `unfreeze` / `suspend` / `appeal_opened` / `appeal_resolved`。 |
| `severity` | `info` / `warning` / `critical`。 |
| `score_before` | 变更前分数。 |
| `score_after` | 变更后分数。 |
| `points_delta` | 影响积分。 |
| `related_authenticity_probe_run_id` | 关联抽检。 |
| `related_request_id` | 关联请求。 |
| `reason_code` | 原因编码。 |
| `reason_summary` | 展示给贡献者的摘要。 |
| `private_note` | 仅运营可见备注。 |
| `operator_user_id` | 人工操作人。 |
| `created_at` | 创建时间。 |

### `user_point_accounts` 与 `user_point_ledger_entries`

用户积分账户与不可变积分流水。

`user_point_accounts`：

- `user_id`
- `available_points`
- `pending_points`
- `frozen_points`
- `lifetime_earned`
- `lifetime_spent`
- `version`

`user_point_ledger_entries`：

- `user_id`
- `direction`: `credit` / `debit`
- `scene`: `contribution_pending` / `release` / `consume` / `point_redeem` / `freeze` / `unfreeze` / `penalty` / `adjustment` / `expire`
- `points`
- `balance_before`
- `balance_after`
- `idempotency_key`
- `related_channel_id`
- `related_request_id`
- `related_authenticity_probe_run_id`
- `related_wallet_ledger_entry_id`
- `related_relay_key_id`
- `related_project_id`
- `conversion_rate_snapshot`
- `settlement_status`
- `remark`

## 关键流程

### 流程 1：贡献者提交 API

1. 贡献者进入“贡献 API”页面。
2. 填写 provider、兼容 base URL、API Key、可贡献模型、每日额度、最大并发。
3. 平台加密保存凭证，创建 `channels` 与 `contributed_channels`，并生成入驻验证幂等键。
4. `share_status` 初始为 `pending_verification`，提交请求不阻塞等待完整探针完成。
5. 系统异步投递入驻验证后台任务，前端通过轮询、刷新或订阅展示验证进度。
6. 后台任务在超时窗口内执行验证，例如 5 分钟内完成 3-5 个轻量探针并写入抽检结果。
7. 验证通过后进入 `active` 或 `watch`；验证失败或超时则展示可解释原因，必要时转为 `verification_timeout`，且不进入路由候选池。

### 流程 2：入驻验证

1. 后台任务按入驻验证幂等键加载 `pending_verification` 渠道；重复投递只复用或更新同一任务状态。
2. 校验凭证是否可用。
3. 对试点内选定 provider / base_url 执行严格校验，包括模型列表、声明模型、usage、错误码和格式；其他 OpenAI-compatible 兼容层仅做基础可用性和响应格式校验。
4. 执行 3-5 个轻量探针。
5. 检查 usage、错误码、延迟和格式。
6. 生成 `model_authenticity_probe_runs`。
7. 计算初始 `quality_score`。
8. 写入 `channel_quality_events`，记录失败原因；超过 5 分钟未完成时转为可解释失败或 `verification_timeout`。

### 流程 3：真实流量命中贡献渠道

1. 消费方通过平台 Sub-Key 发起请求。
2. 路由层加载候选渠道，排除 `pending_verification`、`verification_timeout`、`blocked` 等不可用贡献渠道。
3. 候选过滤除了状态、quota、模型过滤外，还检查 `quality_status`。
4. 命中贡献渠道后写入 `request_executions`，记录消费方 user/project、`relay_key_id` 和命中 `channel_id`。
5. 请求成功后写入 `usage_logs`。
6. 积分结算任务根据 token、质量倍率和同主体消费规则生成 `pending_points`；同 user/project 或明显关联主体默认不奖励或降低奖励。
7. 释放窗口结束后转入 `available_points`。
8. 若消费方使用积分抵扣 token，先写 `user_point_ledger_entries` 扣减流水，再关联 Relay 钱包侧流水，并保留兑换率和项目/Sub-Key 快照。

### 流程 4：随机抽检

1. Scheduler 根据触发规则选择贡献渠道。
2. 从私有题库按 provider、模型族、成本预算和风险等级抽题。
3. 注入 nonce，生成 probe request。
4. 调用贡献渠道。
5. 用程序化校验、对照模型或 judge 模型评分。
6. 写 `model_authenticity_probe_runs`。
7. 触发质量分更新和必要的奖惩事件。

### 流程 5：劣质 API 处理

1. 系统发现连续失败、模型疑似不符或投诉升高。
2. 将渠道状态从 `active` 调整为 `watch` 或 `degraded`。
3. 路由权重降低或暂停新流量。
4. 冻结相关 `pending_points`。
5. 严重情况进入人工复核。
6. 确认违规后扣罚；误判则恢复质量分并释放积分。

## 小范围测试方案

### 测试对象

- 3-10 个可信种子用户。
- 每个用户最多贡献 1-3 个 API Key。
- 每个 API Key 每日贡献上限 50k-200k token。
- 仅支持明确选定的 1-2 个 provider / base_url 或 OpenAI-compatible 协议试点对象，其他兼容层只做基础可用性和格式校验。
- 禁止生产核心业务直接依赖试点容量。

### 成功指标

| 指标 | 目标 |
| --- | --- |
| 入驻验证发现明显无效 API | >= 90% |
| 劣质 API 降权/暂停时间 | 5-15 分钟内 |
| 误伤可复核率 | 100% 有证据链 |
| 积分重复入账 | 0 |
| 并发错账 | 0 |
| 探针题泄漏 | 0 |
| 用户 API Key 明文泄漏 | 0 |

### 试点节奏

1. **第 1 周：文档与规则冻结**
   - 固化试点边界、积分状态、质量分公式、处罚等级。
2. **第 2 周：数据模型与后台任务原型**
   - 实现贡献渠道、探针任务、质量事件、积分账本最小模型。
3. **第 3 周：内部假数据与模拟 API 测试**
   - 用正常、低质、伪造、间歇故障四类 API 样本验证判别能力。
4. **第 4 周：种子用户试点**
   - 开启真实用户贡献，但限制流量和每日预算。
5. **第 5 周：复盘与阈值调整**
   - 根据误判率、抽检成本和用户反馈调整规则。

## 风险与缓解

| 风险 | 影响 | 缓解 |
| --- | --- | --- |
| 无法绝对证明模型真实 | 高 | 使用“通过平台抽检标准”表述，多信号评分，持续抽检。 |
| 探针题库泄漏 | 高 | 私有题库、nonce、版本轮换，只展示结论。 |
| 误伤贡献者 | 高 | 先冻结待结算积分，严重处罚前人工复核，提供申诉。 |
| 贡献者刷积分 | 高 | 延迟释放、禁止自刷高额奖励、异常图谱检测、奖励前抽检。 |
| 抽检成本失控 | 中 | 每日预算、轻重探针分层、风险渠道才提高频率。 |
| 劣质 API 影响消费体验 | 高 | watch/degraded 降权，blocked 不入池，异常快速暂停。 |
| 账本并发错账 | 高 | 不可变流水、事务、版本号、idempotency key。 |
| API Key 泄露 | 高 | 加密存储、脱敏展示、访问审计、支持立即撤回。 |

## 与现有 Relay MVP 的关系

用户贡献 API 质量治理不是当前自托管 Relay MVP 的替代，而是后续实验层：

- 当前 MVP 继续优先跑通“平台运营配置共享容量 -> 发放 Sub-Key -> 请求结算/排障”。
- 用户贡献 API 机制复用 `channels`、`request_executions`、`usage_logs` 和渠道健康能力。
- 积分账本借鉴 `relay_wallet_ledger_entries` 的不可变流水和幂等设计。
- 在质量治理成熟前，不应把用户贡献渠道作为默认生产主池。

## 后续落地任务

1. 设计 `contributed_channels`、`model_authenticity_probe_cases`、`model_authenticity_probe_runs`、`channel_quality_events`、`user_point_accounts`、`user_point_ledger_entries`。
2. 实现贡献 API 入驻验证任务。
3. 实现随机抽检 scheduler。
4. 实现质量分计算与路由降权。
5. 实现积分待结算、释放、冻结、扣罚。
6. 实现贡献者页面与运营质量治理页面。
7. 按 QA 清单进行小范围试点。
