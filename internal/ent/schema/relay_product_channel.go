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
		field.Int("product_id").
			Immutable().
			Comment("Owning relay product id"),
		field.Int("channel_id").
			Immutable().
			Comment("Bound upstream channel id"),
		field.Int("priority").
			Default(0).
			Comment("Lower values are selected first during pool candidate ordering"),
		field.Int("weight").
			Default(100).
			Comment("Relative weight when multiple bindings share the same priority"),
		field.Enum("status").
			Values("active", "paused").
			Default("active").
			Comment("Binding-level availability inside the product pool").
			Annotations(entgql.OrderField("STATUS")),
		field.Bool("allow_fallback").
			Default(true).
			Comment("Whether routing may continue to lower-priority bindings when this binding is unsuitable"),
		field.Any("model_filter").
			Optional().
			Annotations(entgql.Type("Any")).
			Comment("Opaque per-binding model filter rules for future router matching"),
		field.Int("max_inflight").
			Optional().
			Nillable().
			Comment("Optional per-binding inflight cap before routing falls back to another candidate"),
	}
}

func (RelayProductChannel) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("product", RelayProduct.Type).
			Ref("channel_bindings").
			Field("product_id").
			Required().
			Immutable().
			Unique(),
		edge.From("channel", Channel.Type).
			Ref("relay_product_bindings").
			Field("channel_id").
			Required().
			Immutable().
			Unique().
			Annotations(
				entgql.Skip(entgql.SkipMutationCreateInput, entgql.SkipMutationUpdateInput),
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
