package biz

import "sync"

type RelayInflightTracker struct {
	mu     sync.Mutex
	active map[int]int64
}

func NewRelayInflightTracker() *RelayInflightTracker {
	return &RelayInflightTracker{active: make(map[int]int64)}
}

func (t *RelayInflightTracker) Begin(relayKeyID int, limit int64) (func(), bool) {
	if t == nil || relayKeyID <= 0 || limit <= 0 {
		return nil, true
	}

	t.mu.Lock()
	if t.active == nil {
		t.active = make(map[int]int64)
	}
	if t.active[relayKeyID] >= limit {
		t.mu.Unlock()
		return nil, false
	}
	t.active[relayKeyID]++
	t.mu.Unlock()

	var once sync.Once
	return func() {
		once.Do(func() {
			t.mu.Lock()
			defer t.mu.Unlock()

			current := t.active[relayKeyID]
			if current <= 1 {
				delete(t.active, relayKeyID)
				return
			}
			t.active[relayKeyID] = current - 1
		})
	}, true
}

func (t *RelayInflightTracker) activeCount(relayKeyID int) int64 {
	if t == nil || relayKeyID <= 0 {
		return 0
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	return t.active[relayKeyID]
}

type relayInflightLease struct {
	once    sync.Once
	release func()
}

func newRelayInflightLease(release func()) *relayInflightLease {
	if release == nil {
		return nil
	}
	return &relayInflightLease{release: release}
}

func (l *relayInflightLease) Release() {
	if l == nil || l.release == nil {
		return
	}
	l.once.Do(l.release)
}
