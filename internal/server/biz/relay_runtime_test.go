package biz

import (
	"context"
	"errors"
	"net/http"
	"sync"
	"sync/atomic"
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

func TestRelayRuntimeService_ResolveAndCheckAccess_PositiveConcurrencyLimit(t *testing.T) {
	limit := int64(1)
	svc := NewRelayRuntimeService()
	svc.SetResolver(relayRuntimeResolverFunc(func(ctx context.Context, apiKey *ent.APIKey) (*RelayAuthContext, error) {
		return &RelayAuthContext{
			RelayKeyID:  77,
			Status:      RelayKeyStatusActive,
			BalanceMode: RelayKeyBalanceModeQuotaOnly,
			Quota:       RelayKeyQuotaSnapshot{ConcurrencyLimit: &limit},
		}, nil
	}))

	first, decision, err := svc.ResolveAndCheckAccess(context.Background(), &ent.APIKey{ID: 12, ProjectID: 34})
	require.NoError(t, err)
	require.True(t, decision.Allowed)
	require.Equal(t, int64(1), svc.inflight.activeCount(77))

	second, decision, err := svc.ResolveAndCheckAccess(context.Background(), &ent.APIKey{ID: 12, ProjectID: 34})
	require.NoError(t, err)
	require.False(t, decision.Allowed)
	require.Equal(t, "relay_concurrency_quota_exceeded", decision.Code)
	require.Nil(t, second.inflight)
	require.Equal(t, int64(1), svc.inflight.activeCount(77))

	first.ReleaseInflight()
	first.ReleaseInflight()
	require.Equal(t, int64(0), svc.inflight.activeCount(77))

	third, decision, err := svc.ResolveAndCheckAccess(context.Background(), &ent.APIKey{ID: 12, ProjectID: 34})
	require.NoError(t, err)
	require.True(t, decision.Allowed)
	require.Equal(t, int64(1), svc.inflight.activeCount(77))
	third.ReleaseInflight()
}

func TestRelayRuntimeService_ResolveAndCheckAccess_NonPositiveConcurrencyDoesNotAcquire(t *testing.T) {
	limit := int64(0)
	svc := NewRelayRuntimeService()
	svc.SetResolver(relayRuntimeResolverFunc(func(ctx context.Context, apiKey *ent.APIKey) (*RelayAuthContext, error) {
		return &RelayAuthContext{
			RelayKeyID:  88,
			Status:      RelayKeyStatusActive,
			BalanceMode: RelayKeyBalanceModeQuotaOnly,
			Quota:       RelayKeyQuotaSnapshot{ConcurrencyLimit: &limit},
		}, nil
	}))

	relay, decision, err := svc.ResolveAndCheckAccess(context.Background(), &ent.APIKey{ID: 12, ProjectID: 34})
	require.NoError(t, err)
	require.False(t, decision.Allowed)
	require.Equal(t, "relay_concurrency_quota_exceeded", decision.Code)
	require.Nil(t, relay.inflight)
	require.Equal(t, int64(0), svc.inflight.activeCount(88))
}

func TestRelayInflightTrackerBeginIsThreadSafe(t *testing.T) {
	tracker := NewRelayInflightTracker()
	const workers = 32
	const limit = int64(3)

	start := make(chan struct{})
	releases := make(chan func(), workers)
	var allowed int64
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			release, ok := tracker.Begin(99, limit)
			if ok {
				atomic.AddInt64(&allowed, 1)
				releases <- release
			}
		}()
	}
	close(start)
	wg.Wait()
	close(releases)

	require.Equal(t, limit, allowed)
	require.Equal(t, limit, tracker.activeCount(99))
	for release := range releases {
		release()
		release()
	}
	require.Equal(t, int64(0), tracker.activeCount(99))
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
