// SPDX-License-Identifier: Apache-2.0

package events

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/nats-io/nats.go"
)

// SubjectPrefix is the NATS subject namespace for studio events.
const SubjectPrefix = "studio.evidence"

// EvidenceEvent is published after evidence is ingested for a policy.
type EvidenceEvent struct {
	PolicyID    string    `json:"policy_id"`
	RecordCount int       `json:"record_count"`
	Timestamp   time.Time `json:"timestamp"`
}

// Bus wraps a NATS connection for studio event publishing and subscribing.
type Bus struct {
	conn *nats.Conn
}

// Connect creates a new Bus connected to the given NATS URL.
// Returns (nil, nil) if natsURL is empty (NATS disabled).
func Connect(natsURL string) (*Bus, error) {
	if natsURL == "" {
		return nil, nil
	}
	nc, err := nats.Connect(natsURL,
		nats.RetryOnFailedConnect(true),
		nats.MaxReconnects(-1),
		nats.ReconnectWait(2*time.Second),
		nats.DisconnectErrHandler(func(_ *nats.Conn, err error) {
			slog.Warn("nats disconnected", "error", err)
		}),
		nats.ReconnectHandler(func(_ *nats.Conn) {
			slog.Info("nats reconnected")
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("nats connect: %w", err)
	}
	slog.Info("nats connected", "url", natsURL)
	return &Bus{conn: nc}, nil
}

// PublishEvidence publishes an evidence event. Errors are logged, never returned
// — callers must not block ingestion on NATS availability.
func (b *Bus) PublishEvidence(policyID string, recordCount int) {
	if b == nil || b.conn == nil {
		return
	}
	evt := EvidenceEvent{
		PolicyID:    policyID,
		RecordCount: recordCount,
		Timestamp:   time.Now().UTC(),
	}
	data, err := json.Marshal(evt)
	if err != nil {
		slog.Warn("nats marshal failed", "error", err)
		return
	}
	subject := SubjectPrefix + "." + policyID
	if err := b.conn.Publish(subject, data); err != nil {
		slog.Warn("nats publish failed", "subject", subject, "error", err)
	}
}

// SubscribeEvidence subscribes to all evidence events (studio.evidence.>).
// The handler receives decoded events. Returns the subscription for lifecycle management.
func (b *Bus) SubscribeEvidence(handler func(EvidenceEvent)) (*nats.Subscription, error) {
	if b == nil || b.conn == nil {
		return nil, nil
	}
	return b.conn.Subscribe(SubjectPrefix+".>", func(msg *nats.Msg) {
		var evt EvidenceEvent
		if err := json.Unmarshal(msg.Data, &evt); err != nil {
			slog.Warn("nats unmarshal failed", "error", err)
			return
		}
		handler(evt)
	})
}

// Close drains and closes the NATS connection.
func (b *Bus) Close() {
	if b == nil || b.conn == nil {
		return
	}
	_ = b.conn.Drain()
}
