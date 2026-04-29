package biz

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/looplj/axonhub/internal/ent"
)

type relayRuntimeResolverFunc func(context.Context, *ent.APIKey) (*RelayAuthContext, error)

func (f relayRuntimeResolverFunc) ResolveRelayAuth(ctx context.Context, apiKey *ent.APIKey) (*RelayAuthContext, error) {
	return f(ctx, apiKey)
}

type relayRuntimeAccessFunc func(context.Context, *RelayAuthContext, RelayAccessCheckInput) (*RelayAccessDecision, error)

func (f relayRuntimeAccessFunc) CheckRelayAccess(ctx context.Context, relay *RelayAuthContext, input RelayAccessCheckInput) (*RelayAccessDecision, error) {
	return f(ctx, relay, input)
}

func TestRelayRuntimeService_ResolveAndCheckAccess_FillsAPIKeyScope(t *testing.T) {
	svc := NewRelayRuntimeService()
	apiKey := &ent.APIKey{ID: 12, ProjectID: 34}
	resolved := &RelayAuthContext{RelayKeyID: 56}

	var accessInput RelayAccessCheckInput
	svc.SetResolver(relayRuntimeResolverFunc(func(ctx context.Context, got *ent.APIKey) (*RelayAuthContext, error) {
		require.Same(t, apiKey, got)
		return resolved, nil
	}))
	svc.SetAccessChecker(relayRuntimeAccessFunc(func(ctx context.Context, relay *RelayAuthContext, input RelayAccessCheckInput) (*RelayAccessDecision, error) {
		require.Same(t, resolved, relay)
		accessInput = input
		return allowRelayAccess(), nil
	}))

	relay, decision, err := svc.ResolveAndCheckAccess(context.Background(), apiKey)

	require.NoError(t, err)
	require.Same(t, resolved, relay)
	require.True(t, decision.Allowed)
	require.Equal(t, apiKey.ID, relay.APIKeyID)
	require.Equal(t, apiKey.ProjectID, relay.ProjectID)
	require.Same(t, apiKey, accessInput.APIKey)
	require.False(t, accessInput.Now.IsZero())
}

func TestRelayAuthErrorPreservesDecisionDiagnostics(t *testing.T) {
	decision := denyRelayAccess(http.StatusPaymentRequired, "relay_balance_exhausted", "relay key balance exhausted")

	relayErr := NewRelayAuthError(decision)

	require.NotNil(t, relayErr)
	require.Equal(t, http.StatusPaymentRequired, relayErr.HTTPStatus())
	require.Equal(t, "relay_balance_exhausted", relayErr.ErrorCode())
	require.Equal(t, "relay key balance exhausted", relayErr.ErrorMessage())
	require.Contains(t, relayErr.Error(), "relay_balance_exhausted")
	require.True(t, errors.Is(relayErr, ErrRelayInsufficientBalance))
}
