package biz

import (
	"context"
	"net/http"
	"sync"
)

type RelayInflightTracker struct {
	mu     sync.Mutex
	counts map[int]int64
}

type RelayInflightLease struct {
	tracker    *RelayInflightTracker
	relayKeyID int
	once       sync.Once
}

func NewRelayInflightTracker() *RelayInflightTracker {
	return &RelayInflightTracker{counts: make(map[int]int64)}
}

func (t *RelayInflightTracker) Begin(ctx context.Context, relay *RelayAuthContext) (*RelayInflightLease, *RelayAccessDecision) {
	if t == nil || relay == nil || relay.Quota.ConcurrencyLimit == nil || relay.RelayKeyID <= 0 {
		return nil, nil
	}

	limit := *relay.Quota.ConcurrencyLimit
	if limit <= 0 {
		return nil, denyRelayAccess(http.StatusForbidden, "relay_concurrency_quota_exceeded", "relay key concurrency quota exceeded")
	}

	t.mu.Lock()
	if t.counts == nil {
		t.counts = make(map[int]int64)
	}
	current := t.counts[relay.RelayKeyID]
	if current >= limit {
		t.mu.Unlock()
		return nil, denyRelayAccess(http.StatusForbidden, "relay_concurrency_quota_exceeded", "relay key concurrency quota exceeded")
	}
	t.counts[relay.RelayKeyID] = current + 1
	t.mu.Unlock()

	lease := &RelayInflightLease{tracker: t, relayKeyID: relay.RelayKeyID}
	if ctx != nil && ctx.Done() != nil {
		go func() {
			<-ctx.Done()
			lease.Release()
		}()
	}

	return lease, nil
}

func (t *RelayInflightTracker) Current(relayKeyID int) int64 {
	if t == nil || relayKeyID <= 0 {
		return 0
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	return t.counts[relayKeyID]
}

func (l *RelayInflightLease) Release() {
	if l == nil || l.tracker == nil || l.relayKeyID <= 0 {
		return
	}

	l.once.Do(func() {
		l.tracker.mu.Lock()
		defer l.tracker.mu.Unlock()

		current := l.tracker.counts[l.relayKeyID]
		if current <= 1 {
			delete(l.tracker.counts, l.relayKeyID)
			return
		}
		l.tracker.counts[l.relayKeyID] = current - 1
	})
}
