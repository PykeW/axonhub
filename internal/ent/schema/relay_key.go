package schema

import (
	"entgo.io/contrib/entgql"
	"entgo.io/ent"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	"github.com/looplj/axonhub/internal/ent/schema/schematype"
	"github.com/looplj/axonhub/internal/scopes"
)

type RelayKey struct {
	ent.Schema
}

func (RelayKey) Mixin() []ent.Mixin {
	return []ent.Mixin{
		TimeMixin{},
		schematype.SoftDeleteMixin{},
	}
}

func (RelayKey) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("api_key_id").
			StorageKey("relay_keys_by_api_key_id").
			Unique(),
		index.Fields("project_id", "status").
			StorageKey("relay_keys_by_project_status"),
		index.Fields("product_id", "status").
			StorageKey("relay_keys_by_product_status"),
		index.Fields("expires_at").
			StorageKey("relay_keys_by_expires_at"),
	}
}

func (RelayKey) Fields() []ent.Field {
	return []ent.Field{
		field.Int("api_key_id").
			Immutable().
			Comment("Existing API key that authenticates this downstream Relay Sub-Key"),
		field.Int("project_id").
			Immutable().
			Comment("Project scope for buyer isolation"),
		field.Int("product_id").
			Comment("Relay product sold to this Sub-Key"),
		field.Int("owner_user_id").
			Optional().
			Nillable().
			Comment("Optional project user that owns or operates this Sub-Key"),
		field.String("display_name").
			Comment("Operator/project facing display name"),
		field.Enum("status").
			Values("active", "suspended", "exhausted", "archived").
			Default("active").
			Comment("Persistent Relay key lifecycle state").
			Annotations(entgql.OrderField("STATUS")),
		field.Enum("balance_mode").
			Values("prepaid", "quota_only").
			Default("prepaid").
			Comment("Whether access requires wallet balance or only hard quotas"),
		field.Int64("daily_request_limit").
			Optional().
			Nillable().
			Comment("Optional per-day request hard limit"),
		field.Int64("daily_token_limit").
			Optional().
			Nillable().
			Comment("Optional per-day token hard limit"),
		field.String("monthly_cost_limit").
			Optional().
			Nillable().
			Comment("Optional monthly charge limit stored as a decimal string"),
		field.Int64("concurrency_limit").
			Optional().
			Nillable().
			Comment("Optional concurrent request limit"),
		field.Time("expires_at").
			Optional().
			Nillable().
			Comment("Optional expiration time after which access is denied"),
		field.Time("last_used_at").
			Optional().
			Nillable().
			Comment("Last successful runtime access timestamp"),
	}
}

func (RelayKey) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("api_key", APIKey.Type).
			Ref("relay_key").
			Field("api_key_id").
			Required().
			Immutable().
			Unique(),
		edge.From("project", Project.Type).
			Ref("relay_keys").
			Field("project_id").
			Required().
			Immutable().
			Unique(),
		edge.From("product", RelayProduct.Type).
			Ref("relay_keys").
			Field("product_id").
			Required().
			Unique(),
		edge.From("owner_user", User.Type).
			Ref("relay_keys").
			Field("owner_user_id").
			Unique(),
		edge.To("wallet", RelayWallet.Type).
			Unique().
			Annotations(entgql.Skip(entgql.SkipMutationCreateInput, entgql.SkipMutationUpdateInput)),
		edge.To("ledger_entries", RelayWalletLedgerEntry.Type).
			Annotations(entgql.Skip(entgql.SkipMutationCreateInput, entgql.SkipMutationUpdateInput), entgql.RelayConnection()),
		edge.To("daily_usage_summaries", RelayDailyUsageSummary.Type).
			Annotations(entgql.Skip(entgql.SkipMutationCreateInput, entgql.SkipMutationUpdateInput), entgql.RelayConnection()),
	}
}

func (RelayKey) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entgql.RelayConnection(),
	}
}

func (RelayKey) Policy() ent.Policy {
	return scopes.Policy{
		Query: scopes.QueryPolicy{
			scopes.UserProjectScopeReadRule(scopes.ScopeReadAPIKeys),
			scopes.OwnerRule(),
		},
		Mutation: scopes.MutationPolicy{
			scopes.UserProjectScopeWriteRule(scopes.ScopeWriteAPIKeys),
			scopes.OwnerRule(),
		},
	}
}
