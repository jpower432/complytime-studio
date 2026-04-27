// SPDX-License-Identifier: Apache-2.0

// Package certifier provides post-ingest trust checks for evidence rows.
// Certifiers run sequentially in a Pipeline without short-circuiting;
// they annotate evidence with pass/fail/skip/error verdicts rather than
// gating ingestion. The aggregate verdict drives the denormalized
// evidence.certified column for fast UI reads.
package certifier

import (
	"context"
	"time"
)

// Verdict represents the outcome of a single certifier check.
type Verdict string

const (
	VerdictPass  Verdict = "pass"
	VerdictFail  Verdict = "fail"
	VerdictSkip  Verdict = "skip"
	VerdictError Verdict = "error"
)

// CertResult holds the outcome of a single certifier run against one evidence row.
type CertResult struct {
	Certifier string
	Version   string
	Verdict   Verdict
	Reason    string
}

// EvidenceRow is a lightweight projection of an evidence record used by
// certifiers. Fields are optional (pointer or empty) depending on evidence
// completeness.
type EvidenceRow struct {
	EvidenceID       string
	TargetID         string
	RuleID           string
	EvalResult       string
	ComplianceStatus string
	EngineName       string
	SourceRegistry   string
	AttestationRef   string
	EnrichmentStatus string
	CollectedAt      time.Time
}

// Certifier is the interface every certifier must implement.
type Certifier interface {
	Name() string
	Version() string
	Certify(ctx context.Context, row EvidenceRow) CertResult
}
