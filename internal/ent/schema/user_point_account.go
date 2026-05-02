package schema

import (
	"entgo.io/contrib/entgql"
	"entgo.io/ent"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	"github.com/looplj/axonhub/internal/scopes"
)

type UserPointAccount struct {
	ent.Schema
}

func (UserPointAccount) Mixin() []ent.Mixin {
	return []ent.Mixin{
		TimeMixin{},
	}
}

func (UserPointAccount) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("user_id").
			StorageKey("user_point_accounts_by_user_id").
			Unique(),
	}
}

func (UserPointAccount) Fields() []ent.Field {
	return []ent.Field{
		field.Int("user_id").
			Immutable().
			Comment("Owner user id for this point account"),
		field.String("available_points").
			Default("0").
			Comment("Spendable points stored as a decimal string"),
		field.String("pending_points").
			Default("0").
			Comment("Pending points stored as a decimal string"),
		field.String("frozen_points").
			Default("0").
			Comment("Frozen points stored as a decimal string"),
		field.String("lifetime_earned").
			Default("0").
			Comment("Lifetime earned points stored as a decimal string"),
		field.String("lifetime_spent").
			Default("0").
			Comment("Lifetime spent points stored as a decimal string"),
		field.Int64("version").
			Default(1).
			Comment("Optimistic-lock version for point account updates"),
	}
}

func (UserPointAccount) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entgql.Skip(entgql.SkipAll),
	}
}

func (UserPointAccount) Policy() ent.Policy {
	return scopes.Policy{
		Query: scopes.QueryPolicy{
			scopes.OwnerRule(),
			scopes.UserOwnedQueryRule(),
		},
		Mutation: scopes.MutationPolicy{
			scopes.OwnerRule(),
			scopes.UserOwnedMutationRule(),
		},
	}
}
