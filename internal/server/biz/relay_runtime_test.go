package biz

import (
	"context"
	"errors"
	"net/http"
	"sync"
	"sync/atomic"
	"testing"
	"time"

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

func TestRelayInflightTracker_BeginReleaseEnforcesLimit(t *testing.T) {
	tracker := NewRelayInflightTracker()
	limit := int64(2)
	relay := &RelayAuthContext{RelayKeyID: 77, Quota: RelayKeyQuotaSnapshot{ConcurrencyLimit: &limit}}

	first, decision := tracker.Begin(context.Background(), relay)
	require.Nil(t, decision)
	require.NotNil(t, first)
	second, decision := tracker.Begin(context.Background(), relay)
	require.Nil(t, decision)
	require.NotNil(t, second)
	require.Equal(t, int64(2), tracker.Current(relay.RelayKeyID))

	third, decision := tracker.Begin(context.Background(), relay)
	require.Nil(t, third)
	require.NotNil(t, decision)
	require.False(t, decision.Allowed)
	require.ErrorIs(t, decision.ErrorOrNil(), ErrRelayQuotaExceeded)
	require.Equal(t, int64(2), tracker.Current(relay.RelayKeyID))

	otherRelay := &RelayAuthContext{RelayKeyID: 78, Quota: RelayKeyQuotaSnapshot{ConcurrencyLimit: &limit}}
	otherLease, decision := tracker.Begin(context.Background(), otherRelay)
	require.Nil(t, decision)
	require.NotNil(t, otherLease)
	require.Equal(t, int64(1), tracker.Current(otherRelay.RelayKeyID))

	first.Release()
	first.Release()
	require.Equal(t, int64(1), tracker.Current(relay.RelayKeyID))
	second.Release()
	otherLease.Release()
	require.Equal(t, int64(0), tracker.Current(relay.RelayKeyID))
	require.Equal(t, int64(0), tracker.Current(otherRelay.RelayKeyID))
}

func TestRelayInflightTracker_BeginReleasesOnContextCancel(t *testing.T) {
	tracker := NewRelayInflightTracker()
	limit := int64(1)
	relay := &RelayAuthContext{RelayKeyID: 88, Quota: RelayKeyQuotaSnapshot{ConcurrencyLimit: &limit}}
	ctx, cancel := context.WithCancel(context.Background())

	lease, decision := tracker.Begin(ctx, relay)
	require.Nil(t, decision)
	require.NotNil(t, lease)
	require.Equal(t, int64(1), tracker.Current(relay.RelayKeyID))

	cancel()
	require.Eventually(t, func() bool {
		return tracker.Current(relay.RelayKeyID) == 0
	}, time.Second, 10*time.Millisecond)
}

func TestRelayRuntimeService_ResolveAndCheckAccess_EnforcesConcurrencyLimit(t *testing.T) {
	tracker := NewRelayInflightTracker()
	svc := NewRelayRuntimeService()
	svc.SetInflightTracker(tracker)
	apiKey := &ent.APIKey{ID: 12, ProjectID: 34}
	limit := int64(1)
	svc.SetResolver(relayRuntimeResolverFunc(func(ctx context.Context, got *ent.APIKey) (*RelayAuthContext, error) {
		require.Same(t, apiKey, got)
		return &RelayAuthContext{RelayKeyID: 99, Quota: RelayKeyQuotaSnapshot{ConcurrencyLimit: &limit}}, nil
	}))
	svc.SetAccessChecker(relayRuntimeAccessFunc(func(ctx context.Context, relay *RelayAuthContext, input RelayAccessCheckInput) (*RelayAccessDecision, error) {
		return allowRelayAccess(), nil
	}))

	relay, decision, err := svc.ResolveAndCheckAccess(context.Background(), apiKey)
	require.NoError(t, err)
	require.True(t, decision.Allowed)
	require.Equal(t, int64(1), tracker.Current(relay.RelayKeyID))

	blockedRelay, decision, err := svc.ResolveAndCheckAccess(context.Background(), apiKey)
	require.NoError(t, err)
	require.NotNil(t, blockedRelay)
	require.False(t, decision.Allowed)
	require.Equal(t, "relay_concurrency_quota_exceeded", decision.Code)
	require.ErrorIs(t, decision.ErrorOrNil(), ErrRelayQuotaExceeded)
	require.Equal(t, int64(1), tracker.Current(relay.RelayKeyID))

	relay.ReleaseInflight()
	relay.ReleaseInflight()
	require.Equal(t, int64(0), tracker.Current(relay.RelayKeyID))

	relay, decision, err = svc.ResolveAndCheckAccess(context.Background(), apiKey)
	require.NoError(t, err)
	require.True(t, decision.Allowed)
	relay.ReleaseInflight()
}

func TestRelayRuntimeService_ResolveAndCheckAccess_DoesNotBeginOnDeniedAccess(t *testing.T) {
	tracker := NewRelayInflightTracker()
	svc := NewRelayRuntimeService()
	svc.SetInflightTracker(tracker)
	limit := int64(1)
	svc.SetResolver(relayRuntimeResolverFunc(func(ctx context.Context, got *ent.APIKey) (*RelayAuthContext, error) {
		return &RelayAuthContext{RelayKeyID: 101, Status: RelayKeyStatusSuspended, Quota: RelayKeyQuotaSnapshot{ConcurrencyLimit: &limit}}, nil
	}))

	relay, decision, err := svc.ResolveAndCheckAccess(context.Background(), &ent.APIKey{ID: 1, ProjectID: 2})
	require.NoError(t, err)
	require.NotNil(t, relay)
	require.False(t, decision.Allowed)
	require.Equal(t, int64(0), tracker.Current(relay.RelayKeyID))
}

func TestRelayRuntimeService_ResolveAndCheckAccess_NonPositiveConcurrencyDoesNotBegin(t *testing.T) {
	tracker := NewRelayInflightTracker()
	svc := NewRelayRuntimeService()
	svc.SetInflightTracker(tracker)
	limit := int64(0)
	svc.SetResolver(relayRuntimeResolverFunc(func(ctx context.Context, got *ent.APIKey) (*RelayAuthContext, error) {
		return &RelayAuthContext{RelayKeyID: 102, Status: RelayKeyStatusActive, Quota: RelayKeyQuotaSnapshot{ConcurrencyLimit: &limit}}, nil
	}))

	relay, decision, err := svc.ResolveAndCheckAccess(context.Background(), &ent.APIKey{ID: 1, ProjectID: 2})
	require.NoError(t, err)
	require.NotNil(t, relay)
	require.False(t, decision.Allowed)
	require.Equal(t, "relay_concurrency_quota_exceeded", decision.Code)
	require.ErrorIs(t, decision.ErrorOrNil(), ErrRelayQuotaExceeded)
	require.Equal(t, int64(0), tracker.Current(relay.RelayKeyID))
}

func TestRelayInflightTracker_BeginIsThreadSafe(t *testing.T) {
	tracker := NewRelayInflightTracker()
	const workers = 32
	const limit = int64(3)
	relay := &RelayAuthContext{RelayKeyID: 103, Quota: RelayKeyQuotaSnapshot{ConcurrencyLimit: ptrInt64(limit)}}

	start := make(chan struct{})
	releases := make(chan *RelayInflightLease, workers)
	var allowed int64
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			lease, decision := tracker.Begin(context.Background(), relay)
			if decision == nil && lease != nil {
				atomic.AddInt64(&allowed, 1)
				releases <- lease
			}
		}()
	}
	close(start)
	wg.Wait()
	close(releases)

	require.Equal(t, limit, allowed)
	require.Equal(t, limit, tracker.Current(relay.RelayKeyID))
	for release := range releases {
		release.Release()
		release.Release()
	}
	require.Equal(t, int64(0), tracker.Current(relay.RelayKeyID))
}

func ptrInt64(value int64) *int64 {
	return &value
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
