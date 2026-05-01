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

func TestRelayRuntimeService_ConcurrencyLimitAllowsLimitAndRejectsOverflow(t *testing.T) {
	svc := newRelayRuntimeConcurrencyTestService(map[int]RelayAuthContext{
		1: newRelayRuntimeConcurrencyRelay(101, 2),
	})

	first := requireRelayRuntimeAllowed(t, svc, 1)
	second := requireRelayRuntimeAllowed(t, svc, 1)
	requireRelayRuntimeConcurrencyDenied(t, svc, 1)

	requireRelayInflightRelease(t, first)()
	requireRelayInflightRelease(t, second)()
}

func TestRelayRuntimeService_ConcurrencyLimitReleaseAfterCompletionAllowsNext(t *testing.T) {
	svc := newRelayRuntimeConcurrencyTestService(map[int]RelayAuthContext{
		1: newRelayRuntimeConcurrencyRelay(102, 1),
	})

	first := requireRelayRuntimeAllowed(t, svc, 1)
	requireRelayRuntimeConcurrencyDenied(t, svc, 1)

	release := requireRelayInflightRelease(t, first)
	release()
	release()

	second := requireRelayRuntimeAllowed(t, svc, 1)
	requireRelayRuntimeConcurrencyDenied(t, svc, 1)
	requireRelayInflightRelease(t, second)()
}

func TestRelayRuntimeService_ConcurrencyLimitReleaseAfterCancelPathAllowsNext(t *testing.T) {
	svc := newRelayRuntimeConcurrencyTestService(map[int]RelayAuthContext{
		1: newRelayRuntimeConcurrencyRelay(103, 1),
	})

	runCanceledRequest := func() error {
		relay := requireRelayRuntimeAllowed(t, svc, 1)
		defer requireRelayInflightRelease(t, relay)()
		return context.Canceled
	}

	require.ErrorIs(t, runCanceledRequest(), context.Canceled)

	next := requireRelayRuntimeAllowed(t, svc, 1)
	requireRelayRuntimeConcurrencyDenied(t, svc, 1)
	requireRelayInflightRelease(t, next)()
}

func TestRelayRuntimeService_ConcurrencyLimitNonPositiveRejectsWithoutAcquiringSlot(t *testing.T) {
	relays := map[int]RelayAuthContext{
		1: newRelayRuntimeConcurrencyRelay(104, 0),
	}
	svc := newRelayRuntimeConcurrencyTestService(relays)

	requireRelayRuntimeConcurrencyDenied(t, svc, 1)

	relays[1] = newRelayRuntimeConcurrencyRelay(104, 1)
	first := requireRelayRuntimeAllowed(t, svc, 1)
	requireRelayRuntimeConcurrencyDenied(t, svc, 1)
	requireRelayInflightRelease(t, first)()
}

func TestRelayRuntimeService_ConcurrencyLimitSeparateRelayKeysDoNotShareCounters(t *testing.T) {
	svc := newRelayRuntimeConcurrencyTestService(map[int]RelayAuthContext{
		1: newRelayRuntimeConcurrencyRelay(105, 1),
		2: newRelayRuntimeConcurrencyRelay(205, 1),
	})

	firstKey := requireRelayRuntimeAllowed(t, svc, 1)
	secondKey := requireRelayRuntimeAllowed(t, svc, 2)
	requireRelayRuntimeConcurrencyDenied(t, svc, 1)
	requireRelayRuntimeConcurrencyDenied(t, svc, 2)

	requireRelayInflightRelease(t, firstKey)()
	firstKeyAgain := requireRelayRuntimeAllowed(t, svc, 1)
	requireRelayRuntimeConcurrencyDenied(t, svc, 2)

	requireRelayInflightRelease(t, firstKeyAgain)()
	requireRelayInflightRelease(t, secondKey)()
}

func TestRelayRuntimeService_ConcurrencyLimitSequentialRequestsUnderLimitPass(t *testing.T) {
	svc := newRelayRuntimeConcurrencyTestService(map[int]RelayAuthContext{
		1: newRelayRuntimeConcurrencyRelay(106, 1),
	})

	for i := 0; i < 5; i++ {
		relay := requireRelayRuntimeAllowed(t, svc, 1)
		requireRelayInflightRelease(t, relay)()
	}
}

type relayInflightReleaser interface {
	ReleaseInflight()
}

func newRelayRuntimeConcurrencyTestService(relays map[int]RelayAuthContext) *RelayRuntimeService {
	svc := NewRelayRuntimeService()
	svc.SetResolver(relayRuntimeResolverFunc(func(ctx context.Context, apiKey *ent.APIKey) (*RelayAuthContext, error) {
		if apiKey == nil {
			return nil, nil
		}
		relay, ok := relays[apiKey.ID]
		if !ok {
			return nil, nil
		}
		return &relay, nil
	}))
	svc.SetAccessChecker(NewRelayAccessService(RelayAccessServiceParams{}))
	return svc
}

func newRelayRuntimeConcurrencyRelay(relayKeyID int, limit int64) RelayAuthContext {
	return RelayAuthContext{
		RelayKeyID: relayKeyID,
		Status:     RelayKeyStatusActive,
		Quota: RelayKeyQuotaSnapshot{
			ConcurrencyLimit: &limit,
		},
	}
}

func requireRelayRuntimeAllowed(t *testing.T, svc *RelayRuntimeService, apiKeyID int) *RelayAuthContext {
	t.Helper()
	relay, decision, err := svc.ResolveAndCheckAccess(context.Background(), &ent.APIKey{ID: apiKeyID, ProjectID: 7000 + apiKeyID})
	require.NoError(t, err)
	require.NotNil(t, relay)
	require.NotNil(t, decision)
	require.True(t, decision.Allowed, "expected relay request to be allowed, got decision %#v", decision)
	return relay
}

func requireRelayRuntimeConcurrencyDenied(t *testing.T, svc *RelayRuntimeService, apiKeyID int) {
	t.Helper()
	relay, decision, err := svc.ResolveAndCheckAccess(context.Background(), &ent.APIKey{ID: apiKeyID, ProjectID: 7000 + apiKeyID})
	require.NoError(t, err)
	require.NotNil(t, relay)
	require.NotNil(t, decision)
	require.False(t, decision.Allowed, "expected relay request to be denied by ConcurrencyLimit")
	require.Equal(t, "relay_concurrency_quota_exceeded", decision.Code)
	require.ErrorIs(t, decision.ErrorOrNil(), ErrRelayQuotaExceeded)
}

func requireRelayInflightRelease(t *testing.T, relay *RelayAuthContext) func() {
	t.Helper()
	require.NotNil(t, relay)
	releaser, ok := any(relay).(relayInflightReleaser)
	require.True(t, ok, "allowed positive ConcurrencyLimit relay context should expose ReleaseInflight")
	return releaser.ReleaseInflight
}
