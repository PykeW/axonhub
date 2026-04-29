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

type RelayDailyUsageSummary struct {
	ent.Schema
}

func (RelayDailyUsageSummary) Mixin() []ent.Mixin {
	return []ent.Mixin{
		TimeMixin{},
	}
}

func (RelayDailyUsageSummary) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("relay_key_id", "stat_date").
			StorageKey("relay_daily_usage_summaries_by_relay_key_stat_date").
			Unique(),
		index.Fields("project_id", "stat_date").
			StorageKey("relay_daily_usage_summaries_by_project_stat_date"),
		index.Fields("stat_date", "total_charge").
			StorageKey("relay_daily_usage_summaries_by_stat_date_total_charge"),
	}
}

func (RelayDailyUsageSummary) Fields() []ent.Field {
	return []ent.Field{
		field.Int("relay_key_id").
			Immutable().
			Comment("Relay key summarized by this daily aggregate"),
		field.Int("project_id").
			Immutable().
			Comment("Project scope copied from the owning Relay key for privacy filters"),
		field.Time("stat_date").
			Immutable().
			Comment("UTC date bucket for the aggregate"),
		field.Int64("request_count").
			Default(0).
			Comment("Number of settled requests in this day"),
		field.Int64("total_tokens").
			Default(0).
			Comment("Total settled tokens in this day"),
		field.String("total_charge").
			Default("0").
			Comment("Downstream charge total stored as a decimal string"),
		field.String("total_upstream_cost").
			Default("0").
			Comment("Upstream cost total stored as a decimal string"),
		field.Int("last_request_id").
			Optional().
			Nillable().
			Comment("Most recent request included in this aggregate"),
	}
}

func (RelayDailyUsageSummary) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("relay_key", RelayKey.Type).
			Ref("daily_usage_summaries").
			Field("relay_key_id").
			Required().
			Immutable().
			Unique(),
	}
}

func (RelayDailyUsageSummary) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entgql.RelayConnection(),
	}
}

func (RelayDailyUsageSummary) Policy() ent.Policy {
	return scopes.Policy{
		Query: scopes.QueryPolicy{
			scopes.UserProjectScopeReadRule(scopes.ScopeReadRequests),
			scopes.OwnerRule(),
		},
		Mutation: scopes.MutationPolicy{
			scopes.UserProjectScopeWriteRule(scopes.ScopeWriteRequests),
			scopes.OwnerRule(),
		},
	}
}
