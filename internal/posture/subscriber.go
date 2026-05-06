// SPDX-License-Identifier: Apache-2.0

package posture

import (
	"context"
	"fmt"
	"log/slog"
	"slices"

	"github.com/complytime/complytime-studio/internal/events"
	"github.com/complytime/complytime-studio/internal/store"
)

type Subscriber struct {
	engine   *Engine
	programs store.ProgramStore
	bus      *events.Bus
	notifier func(ctx context.Context, msg, severity string) error
}

func NewSubscriber(
	engine *Engine,
	programs store.ProgramStore,
	bus *events.Bus,
	notifier func(ctx context.Context, msg, severity string) error,
) *Subscriber {
	return &Subscriber{
		engine:   engine,
		programs: programs,
		bus:      bus,
		notifier: notifier,
	}
}

func (s *Subscriber) Start(ctx context.Context) error {
	if s == nil {
		return fmt.Errorf("posture subscriber: nil")
	}
	if s.bus == nil {
		return fmt.Errorf("posture subscriber: nil bus")
	}
	sub, err := s.bus.SubscribeEvidence(func(evt events.EvidenceEvent) {
		s.onEvidence(ctx, evt)
	})
	if err != nil {
		return fmt.Errorf("posture subscriber subscribe: %w", err)
	}
	if sub == nil {
		return fmt.Errorf("posture subscriber: nil subscription")
	}
	defer func() { _ = sub.Unsubscribe() }()
	<-ctx.Done()
	return ctx.Err()
}

func (s *Subscriber) onEvidence(ctx context.Context, evt events.EvidenceEvent) {
	if s.engine == nil || s.programs == nil {
		return
	}
	if ctx.Err() != nil {
		return
	}
	if evt.PolicyID == "" {
		slog.Debug("posture subscriber: skip empty policy_id")
		return
	}
	programs, err := s.programs.ListPrograms(ctx)
	if err != nil {
		slog.Warn("posture subscriber list programs failed", "error", err)
		return
	}
	for _, p := range programs {
		if !slices.Contains(p.PolicyIDs, evt.PolicyID) {
			continue
		}
		s.recomputeProgram(ctx, p)
	}
}

func (s *Subscriber) recomputeProgram(ctx context.Context, p store.Program) {
	if ctx.Err() != nil {
		return
	}
	prev := healthString(p.Health)
	greenPct := p.GreenPct
	redPct := p.RedPct
	summary, err := s.engine.ComputeAndStore(ctx, p.ID, p.PolicyIDs, greenPct, redPct)
	if err != nil {
		slog.Warn("posture subscriber recompute failed",
			"program_id", p.ID, "error", err)
		return
	}
	if prev == summary.Health {
		return
	}
	if s.notifier == nil {
		return
	}
	msg := fmt.Sprintf("Program %q (%s) health changed from %q to %q",
		p.Name, p.ID, prev, summary.Health)
	sev := healthSeverity(summary.Health)
	if err := s.notifier(ctx, msg, sev); err != nil {
		slog.Warn("posture subscriber notifier failed", "program_id", p.ID, "error", err)
	}
}

func healthString(h *string) string {
	if h == nil {
		return ""
	}
	return *h
}

func healthSeverity(health string) string {
	switch health {
	case "red":
		return "error"
	case "yellow":
		return "warning"
	default:
		return "info"
	}
}
