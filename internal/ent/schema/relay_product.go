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

// RelayProduct models a sellable shared-capacity relay product.
type RelayProduct struct {
	ent.Schema
}

func (RelayProduct) Mixin() []ent.Mixin {
	return []ent.Mixin{
		TimeMixin{},
		schematype.SoftDeleteMixin{},
	}
}

func (RelayProduct) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("code", "deleted_at").
			StorageKey("relay_products_by_code").
			Unique(),
		index.Fields("status").StorageKey("relay_products_by_status"),
		index.Fields("provider_type").StorageKey("relay_products_by_provider_type"),
	}
}

func (RelayProduct) Fields() []ent.Field {
	return []ent.Field{
		field.String("code").
			Annotations(entgql.OrderField("CODE")),
		field.String("name").
			Annotations(entgql.OrderField("NAME")),
		field.Enum("provider_type").
			Values("claudecode", "codex", "openai_compatible").
			Annotations(entgql.OrderField("PROVIDER_TYPE")),
		field.Enum("access_mode").
			Values("shared_capacity").
			Default("shared_capacity"),
		field.Enum("billing_mode").
			Values("prepaid", "quota_only").
			Default("prepaid"),
		field.Enum("status").
			Values("draft", "active", "archived").
			Default("draft").
			Annotations(entgql.OrderField("STATUS")),
		field.String("currency").Default("USD"),
		field.JSON("list_price_config", map[string]any{}).
			Optional().
			Default(map[string]any{}),
		field.Strings("allowed_models").
			Optional().
			Default([]string{}),
		field.Int("request_timeout_seconds").Default(0),
	}
}

func (RelayProduct) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("product_channels", RelayProductChannel.Type).
			Annotations(
				entgql.Skip(entgql.SkipMutationCreateInput, entgql.SkipMutationUpdateInput),
				entgql.RelayConnection(),
			),
	}
}

func (RelayProduct) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entgql.QueryField(),
		entgql.RelayConnection(),
		entgql.Mutations(entgql.MutationCreate(), entgql.MutationUpdate()),
	}
}

func (RelayProduct) Policy() ent.Policy {
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
