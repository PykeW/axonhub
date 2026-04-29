package biz

import (
	"context"
	"testing"

	"entgo.io/ent/dialect"
	"github.com/stretchr/testify/require"

	"github.com/looplj/axonhub/internal/authz"
	"github.com/looplj/axonhub/internal/ent/channel"
	"github.com/looplj/axonhub/internal/ent/enttest"
	"github.com/looplj/axonhub/internal/objects"
)

func TestRelayProductService_ValidateCreateRelayProductInput(t *testing.T) {
	svc := NewRelayProductService(RelayProductServiceParams{})

	valid := RelayProductCreateInput{
		Code:                  "relay-codex-shared",
		Name:                  "Codex Shared",
		ProviderType:          RelayProductProviderTypeCodex,
		AccessMode:            RelayProductAccessModeSharedCapacity,
		BillingMode:           RelayProductBillingModePrepaid,
		Status:                RelayProductStatusDraft,
		Currency:              RelayProductDefaultCurrency,
		AllowedModels:         []string{"gpt-4.1", "o3"},
		RequestTimeoutSeconds: RelayProductDefaultRequestTimeoutSeconds,
	}

	require.NoError(t, svc.ValidateCreateRelayProductInput(valid))

	t.Run("missing code", func(t *testing.T) {
		input := valid
		input.Code = "  "
		require.Error(t, svc.ValidateCreateRelayProductInput(input))
	})

	t.Run("unsupported provider type", func(t *testing.T) {
		input := valid
		input.ProviderType = RelayProductProviderType("other")
		require.Error(t, svc.ValidateCreateRelayProductInput(input))
	})

	t.Run("blank allowed model", func(t *testing.T) {
		input := valid
		input.AllowedModels = []string{"gpt-4.1", "   "}
		require.Error(t, svc.ValidateCreateRelayProductInput(input))
	})
}

func TestRelayProductService_DataFoundationContract(t *testing.T) {
	svc := NewRelayProductService(RelayProductServiceParams{})
	contract := svc.DataFoundationContract()

	require.ElementsMatch(t, []RelayProductDataEntity{
		{Table: RelayProductTableName, Description: "Sellable shared-capacity relay product catalog"},
		{Table: RelayProductChannelTableName, Description: "Product-to-upstream-channel pool bindings"},
	}, contract.Implemented)
	require.Contains(t, contract.Reused, RelayReusedAPIKeyTableName)
	require.Contains(t, contract.Reused, RelayReusedChannelTableName)
	require.Contains(t, contract.Reused, RelayReusedRequestTableName)
	require.Contains(t, contract.Deferred, RelayProductDataEntity{Table: RelayKeyTableName, Description: "Sub-Key business state bound to existing api_keys"})
	require.Contains(t, contract.Deferred, RelayProductDataEntity{Table: RelayWalletTableName, Description: "Fast balance snapshot for synchronous access checks"})
}

func TestRelayProductService_ValidateCreateRelayProductChannelBinding(t *testing.T) {
	client := enttest.Open(t, dialect.SQLite, "file:ent?mode=memory&_fk=1")
	defer client.Close()

	ctx := authz.WithTestBypass(context.Background())
	ch, err := client.Channel.Create().
		SetType(channel.TypeCodex).
		SetName("codex relay").
		SetCredentials(objects.ChannelCredentials{}).
		SetSupportedModels([]string{"o3"}).
		SetDefaultTestModel("o3").
		Save(ctx)
	require.NoError(t, err)

	svc := NewRelayProductService(RelayProductServiceParams{Ent: client})

	valid := RelayProductChannelBindingInput{
		ProductID: 1,
		ChannelID: ch.ID,
		Priority:  10,
		Weight:    RelayProductChannelDefaultWeight,
		Status:    RelayProductChannelStatusActive,
	}

	require.NoError(t, svc.ValidateCreateRelayProductChannelBinding(ctx, valid))

	t.Run("archived channel rejected", func(t *testing.T) {
		_, err := client.Channel.UpdateOneID(ch.ID).SetStatus(channel.StatusArchived).Save(ctx)
		require.NoError(t, err)
		require.Error(t, svc.ValidateCreateRelayProductChannelBinding(ctx, valid))
	})

	t.Run("non-positive weight rejected", func(t *testing.T) {
		_, err := client.Channel.UpdateOneID(ch.ID).SetStatus(channel.StatusDisabled).Save(ctx)
		require.NoError(t, err)
		invalid := valid
		invalid.Weight = 0
		require.Error(t, svc.ValidateCreateRelayProductChannelBinding(ctx, invalid))
	})
}

func TestRelayProductService_CRUDAndChannelBindingRawSQL(t *testing.T) {
	ctx := authz.WithTestBypass(context.Background())
	client := newRelayServicesTestClient(t)
	db := relayServicesTestDB(t, client)

	ch, err := client.Channel.Create().
		SetType(channel.TypeCodex).
		SetName("codex relay crud").
		SetStatus(channel.StatusEnabled).
		SetCredentials(objects.ChannelCredentials{}).
		SetSupportedModels([]string{"o3", "gpt-4.1"}).
		SetDefaultTestModel("o3").
		Save(ctx)
	require.NoError(t, err)

	svc := NewRelayProductService(RelayProductServiceParams{Ent: client})
	product, err := svc.CreateRelayProduct(ctx, RelayProductCreateInput{
		Code:                  "relay-codex-shared",
		Name:                  "Codex Shared",
		ProviderType:          RelayProductProviderTypeCodex,
		AllowedModels:         []string{"o3", "gpt-4.1"},
		RequestTimeoutSeconds: 120,
		ListPriceConfig: map[string]any{
			"unit": "request",
		},
	})
	require.NoError(t, err)
	require.NotZero(t, product.ID)
	require.Equal(t, RelayProductAccessModeSharedCapacity, product.AccessMode)
	require.Equal(t, RelayProductBillingModePrepaid, product.BillingMode)
	require.Equal(t, RelayProductStatusDraft, product.Status)
	require.Equal(t, RelayProductDefaultCurrency, product.Currency)
	require.Equal(t, []string{"o3", "gpt-4.1"}, product.AllowedModels)
	require.Equal(t, "request", product.ListPriceConfig["unit"])

	products, err := svc.ListRelayProducts(ctx, RelayProductListInput{ProviderType: ptrRelayProductProviderType(RelayProductProviderTypeCodex), Query: "shared"})
	require.NoError(t, err)
	require.Len(t, products, 1)
	require.Equal(t, product.ID, products[0].ID)

	active := RelayProductStatusActive
	name := "Codex Shared Active"
	updated, err := svc.UpdateRelayProduct(ctx, product.ID, RelayProductUpdateInput{
		Name:          &name,
		Status:        &active,
		AllowedModels: []string{"o3"},
	})
	require.NoError(t, err)
	require.Equal(t, name, updated.Name)
	require.Equal(t, RelayProductStatusActive, updated.Status)
	require.Equal(t, []string{"o3"}, updated.AllowedModels)

	allowFallback := false
	maxInflight := 4
	binding, err := svc.CreateRelayProductChannelBinding(ctx, RelayProductChannelBindingInput{
		ProductID:     product.ID,
		ChannelID:     ch.ID,
		Priority:      5,
		Weight:        50,
		Status:        RelayProductChannelStatusActive,
		AllowFallback: &allowFallback,
		ModelFilter:   map[string]any{"models": []any{"o3"}},
		MaxInflight:   &maxInflight,
	})
	require.NoError(t, err)
	require.Equal(t, product.ID, binding.ProductID)
	require.Equal(t, ch.ID, binding.ChannelID)
	require.False(t, binding.AllowFallback)
	require.Equal(t, maxInflight, *binding.MaxInflight)
	require.Contains(t, binding.ModelFilter, "models")

	paused := RelayProductChannelStatusPaused
	priority := 20
	updatedBinding, err := svc.UpdateRelayProductChannelBinding(ctx, binding.ID, RelayProductChannelBindingUpdateInput{
		Priority: &priority,
		Status:   &paused,
	})
	require.NoError(t, err)
	require.Equal(t, priority, updatedBinding.Priority)
	require.Equal(t, RelayProductChannelStatusPaused, updatedBinding.Status)

	require.NoError(t, svc.DeleteRelayProductChannelBinding(ctx, binding.ID))
	var bindingCount int
	require.NoError(t, db.QueryRowContext(ctx, "SELECT COUNT(*) FROM relay_product_channels WHERE id = ?", binding.ID).Scan(&bindingCount))
	require.Zero(t, bindingCount)
}

func TestRelayProductService_UniqueConstraintsRawSQL(t *testing.T) {
	ctx := authz.WithTestBypass(context.Background())
	client := newRelayServicesTestClient(t)
	ch, err := client.Channel.Create().
		SetType(channel.TypeCodex).
		SetName("codex relay unique").
		SetStatus(channel.StatusEnabled).
		SetCredentials(objects.ChannelCredentials{}).
		SetSupportedModels([]string{"o3"}).
		SetDefaultTestModel("o3").
		Save(ctx)
	require.NoError(t, err)

	svc := NewRelayProductService(RelayProductServiceParams{Ent: client})
	input := RelayProductCreateInput{
		Code:         "relay-codex-unique",
		Name:         "Codex Unique",
		ProviderType: RelayProductProviderTypeCodex,
	}
	product, err := svc.CreateRelayProduct(ctx, input)
	require.NoError(t, err)
	_, err = svc.CreateRelayProduct(ctx, input)
	require.Error(t, err)

	bindingInput := RelayProductChannelBindingInput{
		ProductID: product.ID,
		ChannelID: ch.ID,
		Weight:    RelayProductChannelDefaultWeight,
		Status:    RelayProductChannelStatusActive,
	}
	_, err = svc.CreateRelayProductChannelBinding(ctx, bindingInput)
	require.NoError(t, err)
	_, err = svc.CreateRelayProductChannelBinding(ctx, bindingInput)
	require.Error(t, err)
}

func ptrRelayProductProviderType(value RelayProductProviderType) *RelayProductProviderType {
	return &value
}
