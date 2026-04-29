package schema

import (
	"entgo.io/contrib/entgql"
	"entgo.io/ent"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	"github.com/looplj/axonhub/internal/scopes"
)

type RelayWallet struct {
	ent.Schema
}

func (RelayWallet) Mixin() []ent.Mixin {
	return []ent.Mixin{
		TimeMixin{},
	}
}

func (RelayWallet) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("relay_key_id").
			StorageKey("relay_wallets_by_relay_key_id").
			Unique(),
		index.Fields("project_id").
			StorageKey("relay_wallets_by_project_id"),
	}
}

func (RelayWallet) Fields() []ent.Field {
	return []ent.Field{
		field.Int("relay_key_id").
			Immutable().
			Comment("Relay key that owns this balance snapshot"),
		field.Int("project_id").
			Immutable().
			Comment("Project scope copied from the owning Relay key for privacy filters"),
		field.String("currency").
			Default("USD").
			Comment("Wallet currency"),
		field.String("available_amount").
			Default("0").
			Comment("Spendable balance stored as a decimal string"),
		field.String("frozen_amount").
			Default("0").
			Comment("Frozen balance stored as a decimal string"),
		field.String("overdraft_limit").
			Default("0").
			Comment("Allowed overdraft stored as a decimal string"),
		field.Int64("version").
			Default(1).
			Comment("Optimistic-lock version for wallet snapshot updates"),
	}
}

func (RelayWallet) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("relay_key", RelayKey.Type).
			Ref("wallet").
			Field("relay_key_id").
			Required().
			Immutable().
			Unique(),
	}
}

func (RelayWallet) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entgql.RelayConnection(),
	}
}

func (RelayWallet) Policy() ent.Policy {
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
