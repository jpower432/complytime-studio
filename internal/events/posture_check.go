// SPDX-License-Identifier: Apache-2.0

package events

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"
)

// RateCache is a concurrency-safe cache of last-known pass rates per policy.
type RateCache struct {
	mu    sync.Mutex
	rates map[string]float64
}

// NewRateCache creates an empty RateCache.
func NewRateCache() *RateCache {
	return &RateCache{rates: make(map[string]float64)}
}

func (c *RateCache) get(policyID string) (float64, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	v, ok := c.rates[policyID]
	return v, ok
}

func (c *RateCache) set(policyID string, rate float64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.rates[policyID] = rate
}

// PostureDelta is the result of comparing posture before and after evidence arrival.
type PostureDelta struct {
	PolicyID        string  `json:"policy_id"`
	PreviousRate    float64 `json:"previous_pass_rate"`
	CurrentRate     float64 `json:"current_pass_rate"`
	NewFindingCount int     `json:"new_findings"`
}

// PostureQuerier fetches posture aggregates for a single policy.
type PostureQuerier interface {
	QueryPolicyPosture(ctx context.Context, policyID string) (total, passed, failed uint64, err error)
}

// NotificationWriter inserts notifications.
type NotificationWriter interface {
	InsertNotification(ctx context.Context, n Notification) error
}

// Notification mirrors store.Notification but lives here to avoid import cycles.
type Notification struct {
	NotificationID string    `json:"notification_id"`
	Type           string    `json:"type"`
	PolicyID       string    `json:"policy_id"`
	Payload        string    `json:"payload"`
	Read           bool      `json:"read"`
	CreatedAt      time.Time `json:"created_at"`
}

// PostureCheckHandler returns a debounce handler that queries posture, computes
// the delta, and inserts inbox notifications when changes exceed thresholds.
func PostureCheckHandler(
	ctx context.Context,
	querier PostureQuerier,
	notifier NotificationWriter,
	cache *RateCache,
) func(EvidenceEvent) {
	return func(evt EvidenceEvent) {
		total, passed, failed, err := querier.QueryPolicyPosture(ctx, evt.PolicyID)
		if err != nil {
			slog.Warn("posture query failed for event", "policy_id", evt.PolicyID, "error", err)
			return
		}

		var currentRate float64
		if total > 0 {
			currentRate = float64(passed) / float64(total) * 100
		}

		prevRate, hasPrev := cache.get(evt.PolicyID)
		cache.set(evt.PolicyID, currentRate)

		delta := PostureDelta{
			PolicyID:        evt.PolicyID,
			PreviousRate:    prevRate,
			CurrentRate:     currentRate,
			NewFindingCount: int(failed),
		}

		notifyEvidenceArrival(ctx, notifier, evt)

		rateDelta := prevRate - currentRate
		if rateDelta < 0 {
			rateDelta = -rateDelta
		}
		if hasPrev && rateDelta <= 2 && delta.NewFindingCount == 0 {
			slog.Debug("posture unchanged, skipping notification", "policy_id", evt.PolicyID)
			return
		}

		payload, _ := json.Marshal(delta)
		if err := notifier.InsertNotification(ctx, Notification{
			Type:     "posture_change",
			PolicyID: evt.PolicyID,
			Payload:  string(payload),
		}); err != nil {
			slog.Warn("failed to insert posture notification", "policy_id", evt.PolicyID, "error", err)
		} else {
			slog.Info("posture change notification created",
				"policy_id", evt.PolicyID,
				"prev_rate", fmt.Sprintf("%.1f%%", prevRate),
				"curr_rate", fmt.Sprintf("%.1f%%", currentRate),
			)
		}
	}
}

func notifyEvidenceArrival(ctx context.Context, notifier NotificationWriter, evt EvidenceEvent) {
	payload, _ := json.Marshal(map[string]any{
		"policy_id":    evt.PolicyID,
		"record_count": evt.RecordCount,
		"timestamp":    evt.Timestamp,
	})
	if err := notifier.InsertNotification(ctx, Notification{
		Type:     "evidence_arrival",
		PolicyID: evt.PolicyID,
		Payload:  string(payload),
	}); err != nil {
		slog.Warn("failed to insert evidence arrival notification", "policy_id", evt.PolicyID, "error", err)
	}
}
