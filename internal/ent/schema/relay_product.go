package schema

import (
	"entgo.io/contrib/entgql"
	"entgo.io/ent"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	"github.com/looplj/axonhub/internal/ent/schema/schematype"
)

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
		index.Fields("status").
			StorageKey("relay_products_by_status"),
		index.Fields("provider_type").
			StorageKey("relay_products_by_provider_type"),
	}
}

func (RelayProduct) Fields() []ent.Field {
	return []ent.Field{
		field.String("code").
			Immutable().
			Comment("Operator-facing unique relay product code").
			Annotations(entgql.OrderField("CODE")),
		field.String("name").
			Comment("Display name for the sellable shared-capacity product").
			Annotations(entgql.OrderField("NAME")),
		field.Enum("provider_type").
			Values("claudecode", "codex", "openai_compatible").
			Immutable().
			Comment("High-level provider family exposed by this product").
			Annotations(entgql.OrderField("PROVIDER_TYPE")),
		field.Enum("access_mode").
			Values("shared_capacity").
			Default("shared_capacity").
			Immutable().
			Comment("MVP keeps all relay products on shared upstream capacity"),
		field.Enum("billing_mode").
			Values("prepaid", "quota_only").
			Default("prepaid").
			Comment("Whether the product uses prepaid balance or quota-only access"),
		field.Enum("status").
			Values("draft", "active", "archived").
			Default("draft").
			Comment("Lifecycle status for operator product management").
			Annotations(entgql.OrderField("STATUS")),
		field.String("currency").
			Default("USD").
			Comment("Settlement currency used by downstream billing presentation"),
		field.JSON("list_price_config", map[string]any{}).
			Default(map[string]any{}).
			Optional().
			Comment("Opaque downstream pricing rules kept separate from upstream usage cost facts"),
		field.Strings("allowed_models").
			Optional().
			Default([]string{}).
			Comment("Models exposed by this relay product at the catalog layer"),
		field.Int("request_timeout_seconds").
			Default(600).
			Comment("Default request timeout applied to downstream calls for this product"),
	}
}

func (RelayProduct) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("channel_bindings", RelayProductChannel.Type).
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
