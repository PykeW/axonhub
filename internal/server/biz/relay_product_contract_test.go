package biz

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRelayProductService_ContractMatchesMVPFixture(t *testing.T) {
	fixtureBytes, err := os.ReadFile("testdata/relay_product_contract.json")
	require.NoError(t, err)

	var expected RelayProductContract
	require.NoError(t, json.Unmarshal(fixtureBytes, &expected))

	actual := NewRelayProductService(RelayProductServiceParams{}).Contract()
	require.Equal(t, expected, actual)
}

func TestRelayProductService_ContractCoversFirstBackendMVPSlice(t *testing.T) {
	contract := NewRelayProductService(RelayProductServiceParams{}).Contract()

	require.Contains(t, contract.ProviderTypes, RelayProductProviderTypeClaudeCode)
	require.Contains(t, contract.ProviderTypes, RelayProductProviderTypeCodex)
	require.Contains(t, contract.AccessModes, RelayProductAccessModeSharedCapacity)
	require.Contains(t, contract.BillingModes, RelayProductBillingModePrepaid)
	require.Contains(t, contract.BillingModes, RelayProductBillingModeQuotaOnly)
	require.Contains(t, contract.Statuses, RelayProductStatusDraft)
	require.Contains(t, contract.Statuses, RelayProductStatusActive)
	require.Contains(t, contract.BindingStatus, RelayProductChannelStatusActive)
	require.Contains(t, contract.BindingStatus, RelayProductChannelStatusPaused)
	require.Equal(t, RelayProductDefaultCurrency, contract.Defaults.Currency)
	require.Equal(t, string(RelayProductAccessModeSharedCapacity), contract.Defaults.AccessMode)
	require.Equal(t, string(RelayProductBillingModePrepaid), contract.Defaults.BillingMode)
	require.Equal(t, string(RelayProductStatusDraft), contract.Defaults.Status)
	require.Equal(t, RelayProductDefaultRequestTimeoutSeconds, contract.Defaults.RequestTimeoutSeconds)
	require.Equal(t, RelayProductChannelDefaultWeight, contract.Defaults.BindingWeight)
}
