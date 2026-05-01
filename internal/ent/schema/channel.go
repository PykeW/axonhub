package schema

import (
	"entgo.io/contrib/entgql"
	"entgo.io/ent"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	"github.com/looplj/axonhub/internal/ent/schema/schematype"
	"github.com/looplj/axonhub/internal/objects"
	"github.com/looplj/axonhub/internal/scopes"
)

type Channel struct {
	ent.Schema
}

func (Channel) Mixin() []ent.Mixin {
	return []ent.Mixin{
		TimeMixin{},
		schematype.SoftDeleteMixin{},
	}
}

func (Channel) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("name", "deleted_at").
			StorageKey("channels_by_name").
			Unique(),
		index.Fields("owner_user_id", "visibility", "status").
			StorageKey("channels_by_owner_visibility_status"),
		index.Fields("visibility", "status").
			StorageKey("channels_by_visibility_status"),
	}
}

func (Channel) Fields() []ent.Field {
	return []ent.Field{
		field.Enum("type").
			Values(
				"openai",
				"openai_responses",
				"codex",
				"vercel",
				"anthropic",
				"anthropic_aws",
				"anthropic_gcp",
				"gemini_openai",
				"gemini",
				"gemini_vertex",
				"deepseek",
				"deepseek_anthropic",
				"deepinfra",
				"fireworks",
				"doubao",
				"doubao_anthropic",
				"moonshot",
				"moonshot_anthropic",
				"zhipu",
				"zai",
				"zhipu_anthropic",
				"zai_anthropic",
				"anthropic_fake",
				"openai_fake",
				"openrouter",
				"xiaomi",
				"xai",
				"ppio",
				"siliconflow",
				"volcengine",
				"longcat",
				"longcat_anthropic",
				"minimax",
				"minimax_anthropic",
				"aihubmix",
				"burncloud",
				"modelscope",
				"bailian",
				"jina",
				"github",
				"github_copilot",
				"claudecode",
				"cerebras",
				"antigravity",
				"nanogpt",
				"nanogpt_responses",
			).
			Annotations(
				entgql.OrderField("TYPE"),
			),
		field.String("base_url").Optional(),
		field.String("name").
			Annotations(
				entgql.OrderField("NAME"),
			),
		field.Enum("status").Values("enabled", "disabled", "archived").Default("disabled").
			Annotations(
				entgql.Skip(entgql.SkipMutationCreateInput),
				entgql.OrderField("STATUS"),
			),
		field.Int("owner_user_id").
			Optional().
			Comment("User ID that owns this channel. Zero or null means legacy/system-owned channel.").
			Annotations(
				entgql.Skip(entgql.SkipMutationCreateInput, entgql.SkipMutationUpdateInput),
			),
		field.Enum("visibility").Values("private", "shared").Default("private").
			Comment("Share/Use visibility: private channels are owner-only; shared channels can be used by other users.").
			Annotations(
				entgql.OrderField("VISIBILITY"),
			),
		field.Int("share_refresh_window_seconds").
			Default(5 * 60 * 60).
			Comment("Share quota refresh window in seconds. Default is 5 hours."),
		field.Int("share_refresh_quota").
			Default(1).
			Comment("Maximum shared-use quota restored each refresh window."),
		field.Time("share_next_refresh_at").
			Optional().
			Nillable().
			Comment("Next time the shared-use quota can be lazily refreshed.").
			Annotations(
				entgql.Skip(entgql.SkipMutationCreateInput, entgql.SkipMutationUpdateInput),
			),
		field.Int("share_remaining_quota").
			Default(1).
			Comment("Remaining shared-use quota in the current refresh window.").
			Annotations(
				entgql.Skip(entgql.SkipMutationCreateInput, entgql.SkipMutationUpdateInput),
			),
		field.Int("share_recent_error_count").
			Default(0).
			Comment("Recent shared-use routing error count for health-aware sorting.").
			Annotations(
				entgql.Skip(entgql.SkipMutationCreateInput, entgql.SkipMutationUpdateInput),
			),
		field.JSON("credentials", objects.ChannelCredentials{}).Sensitive(),
		field.JSON("disabled_api_keys", []objects.DisabledAPIKey{}).
			Default([]objects.DisabledAPIKey{}).
			Optional().
			Sensitive().
			Comment("Disabled API keys with metadata (sensitive; requires channel write permission)").
			Annotations(
				entgql.Skip(entgql.SkipMutationCreateInput, entgql.SkipMutationUpdateInput),
			),
		field.Strings("supported_models"),
		field.Strings("manual_models").Optional().Default([]string{}),
		field.Bool("auto_sync_supported_models").Default(false),
		field.String("auto_sync_model_pattern").Optional().Default("").
			Comment("Regex pattern to filter models during auto-sync. Empty string means no filtering."),
		field.Strings("tags").Optional().Default([]string{}),
		field.String("default_test_model"),
		field.JSON("policies", objects.ChannelPolicies{}).
			Default(objects.ChannelPolicies{
				Stream: objects.CapabilityPolicyUnlimited,
			}).
			Annotations(
				entgql.Directives(forceResolver()),
			).
			Optional(),
		field.JSON("settings", &objects.ChannelSettings{}).
			Default(&objects.ChannelSettings{
				ModelMappings: []objects.ModelMapping{},
			}).Optional().Annotations(),
		field.Int("ordering_weight").Default(0).Comment("Ordering weight for display sorting").
			Annotations(
				entgql.OrderField("ORDERING_WEIGHT"),
			),
		field.String("error_message").
			Optional().Nillable().
			Annotations(
				entgql.Skip(entgql.SkipMutationCreateInput),
			),
		field.String("remark").
			Optional().Nillable().
			Comment("User-defined remark or note for the channel"),
	}
}

func (Channel) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("owner_user", User.Type).
			Ref("channels").
			Field("owner_user_id").
			Unique().
			Annotations(
				entgql.Skip(entgql.SkipMutationCreateInput, entgql.SkipMutationUpdateInput),
			),
		edge.To("requests", Request.Type).
			Annotations(
				entgql.Skip(entgql.SkipMutationCreateInput, entgql.SkipMutationUpdateInput),
				entgql.RelayConnection(),
			),
		edge.To("executions", RequestExecution.Type).
			Annotations(
				entgql.Skip(entgql.SkipMutationCreateInput, entgql.SkipMutationUpdateInput),
				entgql.RelayConnection(),
			),
		edge.To("usage_logs", UsageLog.Type).
			Annotations(
				entgql.Skip(entgql.SkipMutationCreateInput, entgql.SkipMutationUpdateInput),
				entgql.RelayConnection(),
			),
		edge.To("channel_probes", ChannelProbe.Type).
			Annotations(
				entgql.Skip(entgql.SkipMutationCreateInput, entgql.SkipMutationUpdateInput),
			),
		edge.To("channel_model_prices", ChannelModelPrice.Type).
			Annotations(
				entgql.Skip(entgql.SkipMutationCreateInput, entgql.SkipMutationUpdateInput),
			),
		edge.To("relay_product_bindings", RelayProductChannel.Type).
			Annotations(
				entgql.Skip(entgql.SkipMutationCreateInput, entgql.SkipMutationUpdateInput),
				entgql.RelayConnection(),
			),
		edge.To("provider_quota_status", ProviderQuotaStatus.Type).
			Unique().
			Annotations(
				entgql.Directives(forceResolver()),
				entgql.Skip(entgql.SkipMutationCreateInput, entgql.SkipMutationUpdateInput),
			),
	}
}

func (Channel) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entgql.QueryField(),
		entgql.RelayConnection(),
		entgql.Mutations(entgql.MutationCreate(), entgql.MutationUpdate()),
	}
}

// Policy 定义 Channel 的权限策略.
func (Channel) Policy() ent.Policy {
	return scopes.Policy{
		Query: scopes.QueryPolicy{
			scopes.APIKeyScopeQueryRule(scopes.ScopeReadChannels),
			scopes.OwnerRule(),              // owner 用户可以访问所有渠道
			scopes.UserReadScopeRule(scopes.ScopeReadChannels), // 需要 channels 读取权限
		},
		Mutation: scopes.MutationPolicy{
			scopes.OwnerRule(),               // owner 用户可以修改所有渠道
			scopes.UserWriteScopeRule(scopes.ScopeWriteChannels), // 需要 channels 写入权限
		},
	}
}
