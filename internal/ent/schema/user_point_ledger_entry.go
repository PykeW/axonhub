package schema

import (
	"entgo.io/contrib/entgql"
	"entgo.io/ent"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	"github.com/looplj/axonhub/internal/scopes"
)

type UserPointLedgerEntry struct {
	ent.Schema
}

func (UserPointLedgerEntry) Mixin() []ent.Mixin {
	return []ent.Mixin{
		TimeMixin{},
	}
}

func (UserPointLedgerEntry) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("idempotency_key").
			StorageKey("user_point_ledger_entries_by_idempotency_key").
			Unique(),
		index.Fields("user_id", "created_at").
			StorageKey("user_point_ledger_entries_by_user_created_at"),
		index.Fields("related_request_id").
			StorageKey("user_point_ledger_entries_by_request_id"),
		index.Fields("related_usage_log_id").
			StorageKey("user_point_ledger_entries_by_usage_log_id"),
		index.Fields("related_channel_id").
			StorageKey("user_point_ledger_entries_by_channel_id"),
	}
}

func (UserPointLedgerEntry) Fields() []ent.Field {
	return []ent.Field{
		field.Int("user_id").
			Immutable().
			Comment("Owner user id for this point ledger row"),
		field.Enum("direction").
			Values("credit", "debit").
			Immutable().
			Comment("Point movement direction"),
		field.Enum("scene").
			Values("contribution_pending", "contribution_reward", "consume", "adjustment").
			Immutable().
			Comment("Business reason for this ledger row"),
		field.String("points").
			Immutable().
			Comment("Points delta stored as a decimal string"),
		field.String("balance_before").
			Immutable().
			Comment("Available balance before this entry"),
		field.String("balance_after").
			Immutable().
			Comment("Available balance after this entry"),
		field.String("idempotency_key").
			Immutable().
			Comment("Unique key preventing duplicate point settlement"),
		field.Int("related_channel_id").
			Optional().
			Nillable().
			Immutable().
			Comment("Optional shared channel id anchor for settlement"),
		field.Int("related_request_id").
			Optional().
			Nillable().
			Immutable().
			Comment("Optional request id anchor for settlement"),
		field.Int("related_usage_log_id").
			Optional().
			Nillable().
			Immutable().
			Comment("Optional usage log id anchor for idempotent settlement"),
		field.Int("related_api_key_id").
			Optional().
			Nillable().
			Immutable().
			Comment("Optional caller API key id used for this settlement"),
		field.Int("related_project_id").
			Optional().
			Nillable().
			Immutable().
			Comment("Optional project id used for this settlement"),
		field.String("conversion_rate_snapshot").
			Optional().
			Nillable().
			Immutable().
			Comment("Opaque cost-to-points conversion snapshot"),
		field.Enum("settlement_status").
			Values("posted").
			Default("posted").
			Immutable().
			Comment("Settlement lifecycle status"),
		field.String("remark").
			Optional().
			Nillable().
			Immutable().
			Comment("Settlement remark"),
	}
}

func (UserPointLedgerEntry) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entgql.Skip(entgql.SkipAll),
	}
}

func (UserPointLedgerEntry) Policy() ent.Policy {
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
