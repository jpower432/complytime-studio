// SPDX-License-Identifier: Apache-2.0

package events

import (
	"context"
	"errors"
	"sync"
	"testing"
)

func TestRateCache_GetSetConcurrent(t *testing.T) {
	t.Parallel()

	c := NewRateCache()
	const n = 100
	var wg sync.WaitGroup
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func(id int) {
			defer wg.Done()
			key := "policy"
			if id%2 == 0 {
				c.set(key, float64(id))
			} else {
				_, _ = c.get(key)
			}
		}(i)
	}
	wg.Wait()
}

func TestRateCache_DefaultZero(t *testing.T) {
	t.Parallel()

	c := NewRateCache()
	v, ok := c.get("unknown")
	if ok {
		t.Fatal("want ok false for unknown key")
	}
	if v != 0 {
		t.Fatalf("want zero value, got %v", v)
	}
}

type mockQuerier struct {
	total, passed, failed uint64
	err                   error
}

func (m *mockQuerier) QueryPolicyPosture(_ context.Context, _ string) (uint64, uint64, uint64, error) {
	if m.err != nil {
		return 0, 0, 0, m.err
	}
	return m.total, m.passed, m.failed, nil
}

type notifCall struct {
	Type     string
	PolicyID string
	Payload  string
}

type mockNotifier struct {
	mu    sync.Mutex
	calls []notifCall
}

func (m *mockNotifier) InsertNotification(_ context.Context, notifType, policyID, payload string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls = append(m.calls, notifCall{Type: notifType, PolicyID: policyID, Payload: payload})
	return nil
}

func (m *mockNotifier) notifications() []notifCall {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]notifCall, len(m.calls))
	copy(out, m.calls)
	return out
}

func countType(calls []notifCall, typ string) int {
	n := 0
	for _, c := range calls {
		if c.Type == typ {
			n++
		}
	}
	return n
}

func TestPostureCheckHandler_DeltaTriggersNotification(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	cache := NewRateCache()
	cache.set("pol-1", 80)

	q := &mockQuerier{total: 100, passed: 30, failed: 0}
	n := &mockNotifier{}

	h := PostureCheckHandler(ctx, q, n, cache)
	h(EvidenceEvent{PolicyID: "pol-1", RecordCount: 1})

	calls := n.notifications()
	if countType(calls, "posture_change") != 1 {
		t.Fatalf("want 1 posture_change, got %#v", calls)
	}
	if countType(calls, "evidence_arrival") != 1 {
		t.Fatalf("want 1 evidence_arrival, got %#v", calls)
	}
}

func TestPostureCheckHandler_NoDeltaSkipsNotification(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	cache := NewRateCache()
	cache.set("pol-2", 50)

	q := &mockQuerier{total: 100, passed: 50, failed: 0}
	n := &mockNotifier{}

	h := PostureCheckHandler(ctx, q, n, cache)
	h(EvidenceEvent{PolicyID: "pol-2", RecordCount: 1})

	calls := n.notifications()
	if countType(calls, "posture_change") != 0 {
		t.Fatalf("want no posture_change, got %#v", calls)
	}
	if countType(calls, "evidence_arrival") != 1 {
		t.Fatalf("want 1 evidence_arrival, got %#v", calls)
	}
}

func TestPostureCheckHandler_QueryError(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	cache := NewRateCache()
	q := &mockQuerier{err: errors.New("db down")}
	n := &mockNotifier{}

	h := PostureCheckHandler(ctx, q, n, cache)

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("unexpected panic: %v", r)
		}
	}()

	h(EvidenceEvent{PolicyID: "pol-err", RecordCount: 1})

	if len(n.notifications()) != 0 {
		t.Fatalf("want no notifications on query error, got %#v", n.notifications())
	}
}
