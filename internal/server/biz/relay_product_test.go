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
