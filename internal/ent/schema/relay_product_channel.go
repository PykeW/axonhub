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

// RelayProductChannel binds a relay product to a concrete upstream channel.
type RelayProductChannel struct {
	ent.Schema
}

func (RelayProductChannel) Mixin() []ent.Mixin {
	return []ent.Mixin{
		TimeMixin{},
	}
}

func (RelayProductChannel) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("product_id", "channel_id").
			StorageKey("relay_product_channels_by_product_channel").
			Unique(),
		index.Fields("product_id", "status", "priority").
			StorageKey("relay_product_channels_by_product_status_priority"),
	}
}

func (RelayProductChannel) Fields() []ent.Field {
	return []ent.Field{
		field.Int("product_id").Immutable(),
		field.Int("channel_id").Immutable(),
		field.Int("priority").Default(100),
		field.Int("weight").Default(100),
		field.Enum("status").
			Values("active", "paused").
			Default("active"),
		field.Bool("allow_fallback").Default(true),
		field.JSON("model_filter", map[string]any{}).
			Optional().
			Default(map[string]any{}),
		field.Int("max_inflight").Default(0),
	}
}

func (RelayProductChannel) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("product", RelayProduct.Type).
			Ref("product_channels").
			Field("product_id").
			Required().
			Immutable().
			Unique(),
		edge.From("channel", Channel.Type).
			Ref("relay_product_channels").
			Field("channel_id").
			Required().
			Immutable().
			Unique().
			Annotations(
				entgql.Directives(forceResolver()),
			),
	}
}

func (RelayProductChannel) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entgql.QueryField(),
		entgql.RelayConnection(),
		entgql.Mutations(entgql.MutationCreate(), entgql.MutationUpdate()),
	}
}

func (RelayProductChannel) Policy() ent.Policy {
	return scopes.Policy{
		Query: scopes.QueryPolicy{
			scopes.OwnerRule(),
			scopes.UserReadScopeRule(scopes.ScopeReadChannels),
		},
		Mutation: scopes.MutationPolicy{
			scopes.OwnerRule(),
			scopes.UserWriteScopeRule(scopes.ScopeWriteChannels),
		},
	}
}
