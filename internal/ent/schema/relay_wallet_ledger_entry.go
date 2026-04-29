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

type RelayWalletLedgerEntry struct {
	ent.Schema
}

func (RelayWalletLedgerEntry) Mixin() []ent.Mixin {
	return []ent.Mixin{
		TimeMixin{},
	}
}

func (RelayWalletLedgerEntry) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("idempotency_key").
			StorageKey("relay_wallet_ledger_entries_by_idempotency_key").
			Unique(),
		index.Fields("relay_key_id", "created_at").
			StorageKey("relay_wallet_ledger_entries_by_relay_key_created_at"),
		index.Fields("project_id", "created_at").
			StorageKey("relay_wallet_ledger_entries_by_project_created_at"),
		index.Fields("request_id").
			StorageKey("relay_wallet_ledger_entries_by_request_id"),
		index.Fields("usage_log_id").
			StorageKey("relay_wallet_ledger_entries_by_usage_log_id"),
	}
}

func (RelayWalletLedgerEntry) Fields() []ent.Field {
	return []ent.Field{
		field.Int("relay_key_id").
			Immutable().
			Comment("Relay key whose wallet changed"),
		field.Int("project_id").
			Immutable().
			Comment("Project scope copied from the owning Relay key for privacy filters"),
		field.Int("request_id").
			Optional().
			Nillable().
			Immutable().
			Comment("Optional request anchor for consumption ledger rows"),
		field.Int("usage_log_id").
			Optional().
			Nillable().
			Immutable().
			Comment("Optional usage log anchor for idempotent settlement"),
		field.Enum("direction").
			Values("credit", "debit").
			Immutable().
			Comment("Balance movement direction"),
		field.Enum("scene").
			Values("recharge", "consume", "refund", "freeze", "unfreeze", "manual_adjust").
			Immutable().
			Comment("Business reason for this ledger row"),
		field.String("amount").
			Immutable().
			Comment("Ledger amount stored as a decimal string"),
		field.String("balance_before").
			Immutable().
			Comment("Available balance before this entry"),
		field.String("balance_after").
			Immutable().
			Comment("Available balance after this entry"),
		field.String("upstream_cost").
			Optional().
			Nillable().
			Immutable().
			Comment("Optional upstream cost snapshot stored as a decimal string"),
		field.JSON("price_snapshot", map[string]any{}).
			Default(map[string]any{}).
			Optional().
			Immutable().
			Comment("Opaque downstream price snapshot used for this entry"),
		field.String("idempotency_key").
			Immutable().
			Comment("Unique key preventing duplicate wallet settlement"),
		field.Int("operator_user_id").
			Optional().
			Nillable().
			Immutable().
			Comment("Optional operator that created a manual ledger row"),
		field.String("remark").
			Optional().
			Nillable().
			Immutable().
			Comment("Operator or settlement remark"),
	}
}

func (RelayWalletLedgerEntry) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("relay_key", RelayKey.Type).
			Ref("ledger_entries").
			Field("relay_key_id").
			Required().
			Immutable().
			Unique(),
	}
}

func (RelayWalletLedgerEntry) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entgql.RelayConnection(),
	}
}

func (RelayWalletLedgerEntry) Policy() ent.Policy {
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
