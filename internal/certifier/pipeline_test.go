// SPDX-License-Identifier: Apache-2.0

package certifier

import (
	"context"
	"testing"
)

type stubCertifier struct {
	name    string
	version string
	verdict Verdict
	reason  string
}

func (s *stubCertifier) Name() string    { return s.name }
func (s *stubCertifier) Version() string { return s.version }
func (s *stubCertifier) Certify(_ context.Context, _ EvidenceRow) CertResult {
	return CertResult{
		Certifier: s.name,
		Version:   s.version,
		Verdict:   s.verdict,
		Reason:    s.reason,
	}
}

func TestPipelineRunsAllCertifiers(t *testing.T) {
	p := NewPipeline(
		&stubCertifier{name: "a", version: "1", verdict: VerdictFail, reason: "bad"},
		&stubCertifier{name: "b", version: "1", verdict: VerdictPass, reason: "ok"},
		&stubCertifier{name: "c", version: "1", verdict: VerdictSkip, reason: "n/a"},
		&stubCertifier{name: "d", version: "1", verdict: VerdictError, reason: "timeout"},
	)
	results := p.Run(context.Background(), EvidenceRow{})
	if len(results) != 4 {
		t.Fatalf("expected 4 results, got %d", len(results))
	}
	expected := []Verdict{VerdictFail, VerdictPass, VerdictSkip, VerdictError}
	for i, r := range results {
		if r.Verdict != expected[i] {
			t.Errorf("result[%d]: expected %s, got %s", i, expected[i], r.Verdict)
		}
	}
}

func TestIsCertifiedAllPass(t *testing.T) {
	results := []CertResult{
		{Verdict: VerdictPass},
		{Verdict: VerdictPass},
		{Verdict: VerdictSkip},
	}
	if !IsCertified(results) {
		t.Error("expected certified=true for [pass, pass, skip]")
	}
}

func TestIsCertifiedAnyFail(t *testing.T) {
	results := []CertResult{
		{Verdict: VerdictPass},
		{Verdict: VerdictFail},
		{Verdict: VerdictPass},
	}
	if IsCertified(results) {
		t.Error("expected certified=false when any fail exists")
	}
}

func TestIsCertifiedOnlySkipAndError(t *testing.T) {
	results := []CertResult{
		{Verdict: VerdictSkip},
		{Verdict: VerdictError},
		{Verdict: VerdictSkip},
	}
	if IsCertified(results) {
		t.Error("expected certified=false when no pass exists")
	}
}

func TestIsCertifiedEmpty(t *testing.T) {
	if IsCertified(nil) {
		t.Error("expected certified=false for nil results")
	}
}
