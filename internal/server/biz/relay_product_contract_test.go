package biz

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRelayProductService_ContractMatchesFixture(t *testing.T) {
	fixture, err := os.ReadFile("testdata/relay_product_contract.json")
	require.NoError(t, err)

	var expected RelayProductContract
	require.NoError(t, json.Unmarshal(fixture, &expected))

	actual := NewRelayProductService(RelayProductServiceParams{}).Contract()
	require.Equal(t, expected, actual)
}
