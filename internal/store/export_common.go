// SPDX-License-Identifier: Apache-2.0

package store

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/complytime/complytime-studio/internal/consts"
)

// ErrExportAuditPolicyMismatch is returned when audit_id refers to another policy.
var ErrExportAuditPolicyMismatch = errors.New("audit_id does not match policy_id")

// ErrExportRowLimit is returned when an export would exceed configured row caps.
var ErrExportRowLimit = errors.New("export row limit exceeded")

// ParseExportQuery validates policy_id and optional audit window (aligned with
// GET /api/requirements and CSV export).
func ParseExportQuery(q url.Values) (policyID, auditID string, f RequirementFilter, err error) {
	policyID = strings.TrimSpace(q.Get("policy_id"))
	if policyID == "" {
		return "", "", f, fmt.Errorf("policy_id required")
	}
	auditID = strings.TrimSpace(q.Get("audit_id"))

	if v := q.Get("audit_start"); v != "" {
		t, e1 := time.Parse(time.RFC3339, v)
		if e1 != nil {
			t, e1 = time.Parse("2006-01-02", v)
		}
		if e1 != nil {
			return "", "", f, fmt.Errorf("invalid audit_start format")
		}
		f.Start = t
	}
	if v := q.Get("audit_end"); v != "" {
		t, e1 := time.Parse(time.RFC3339, v)
		if e1 != nil {
			t, e1 = time.Parse("2006-01-02", v)
		}
		if e1 != nil {
			return "", "", f, fmt.Errorf("invalid audit_end format")
		}
		f.End = t
	}
	if !f.Start.IsZero() && !f.End.IsZero() && f.End.Before(f.Start) {
		return "", "", f, fmt.Errorf("audit_end must be >= audit_start")
	}

	f.PolicyID = policyID
	return policyID, auditID, f, nil
}

// SanitizeExportFilenamePart strips characters unsafe in Content-Disposition filenames.
func SanitizeExportFilenamePart(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch r {
		case '/', '\\', ':', '*', '?', '"', '<', '>', '|', ';', '\n', '\r':
			continue
		case ' ':
			b.WriteRune('_')
		default:
			if r < 32 {
				continue
			}
			b.WriteRune(r)
		}
	}
	return b.String()
}

// SanitizeExportFilename builds attachment filename: complytime-export_<policy>_<ts>.<ext>
func SanitizeExportFilename(policyID, ext string) string {
	base := SanitizeExportFilenamePart(policyID)
	if base == "" {
		base = "export"
	}
	if len(base) > 32 {
		base = base[:32]
	}
	return fmt.Sprintf("complytime-export_%s_%s%s", base, time.Now().UTC().Format("20060102-150405"), ext)
}

// ExportSummaryAgg holds executive-summary aggregates derived from matrix rows
// (same underlying data as the requirement matrix / posture views for the window).
type ExportSummaryAgg struct {
	PolicyTitle              string
	PolicyVersion            string
	TotalRequirements        int
	RequirementsWithEvidence int
	TotalEvidencePieces      uint64
	ByClassification         map[string]int
}

// BuildExportSummaryAgg computes executive-summary fields from requirement matrix rows.
func BuildExportSummaryAgg(matrix []RequirementRow, policyTitle, policyVersion string) ExportSummaryAgg {
	agg := ExportSummaryAgg{
		PolicyTitle:       policyTitle,
		PolicyVersion:     policyVersion,
		ByClassification:  make(map[string]int),
		TotalRequirements: len(matrix),
	}
	for _, row := range matrix {
		cls := row.Classification
		if cls == "" {
			cls = "Unassessed"
		}
		agg.ByClassification[cls]++
		if row.EvidenceCount > 0 {
			agg.RequirementsWithEvidence++
			agg.TotalEvidencePieces += row.EvidenceCount
		}
	}
	return agg
}

// IsGapRow identifies requirement rows for the Gap List sheet.
func IsGapRow(row RequirementRow) bool {
	if row.Classification == "No Evidence" {
		return true
	}
	return row.EvidenceCount == 0
}

// SelectExportAuditLog returns the audit log for optional agent narrative: explicit
// audit_id, else latest ListAuditLogs row for policy_id within the audit window
// (matching audit_start / audit_end overlap filter on the store).
func SelectExportAuditLog(ctx context.Context, als AuditLogStore, policyID, auditID string, windowStart, windowEnd time.Time) (*AuditLog, error) {
	if als == nil {
		return nil, nil
	}
	if auditID != "" {
		a, err := als.GetAuditLog(ctx, auditID)
		if err != nil {
			return nil, err
		}
		if a.PolicyID != policyID {
			return nil, ErrExportAuditPolicyMismatch
		}
		return a, nil
	}
	logs, err := als.ListAuditLogs(ctx, policyID, windowStart, windowEnd, 1)
	if err != nil {
		return nil, err
	}
	if len(logs) == 0 {
		return nil, nil
	}
	return &logs[0], nil
}

func policyDisplayMeta(ctx context.Context, ps PolicyStore, policyID string) (title, version string) {
	if ps == nil {
		return "", ""
	}
	p, err := ps.GetPolicy(ctx, policyID)
	if err != nil || p == nil {
		return "", ""
	}
	return p.Title, p.Version
}

// LoadExportMatrix fetches matrix rows enforcing MaxExportRequirementRows (+1 probe).
func LoadExportMatrix(ctx context.Context, rs RequirementStore, f RequirementFilter) ([]RequirementRow, error) {
	f.Limit = consts.MaxExportRequirementRows + 1
	rows, err := rs.ListRequirementMatrix(ctx, f)
	if err != nil {
		return nil, err
	}
	if len(rows) > consts.MaxExportRequirementRows {
		return nil, fmt.Errorf("%w: max %d requirement rows", ErrExportRowLimit, consts.MaxExportRequirementRows)
	}
	return rows, nil
}

// LoadExportEvidence fetches evidence for the inventory sheet with row cap.
func LoadExportEvidence(ctx context.Context, es EvidenceStore, policyID string, start, end time.Time) ([]EvidenceRecord, error) {
	if es == nil {
		return nil, nil
	}
	f := EvidenceFilter{
		PolicyID: policyID,
		Start:    start,
		End:      end,
		Limit:    consts.MaxExportEvidenceRows + 1,
	}
	rows, err := es.QueryEvidence(ctx, f)
	if err != nil {
		return nil, err
	}
	if len(rows) > consts.MaxExportEvidenceRows {
		return nil, fmt.Errorf("%w: max %d evidence rows", ErrExportRowLimit, consts.MaxExportEvidenceRows)
	}
	return rows, nil
}

// AgentNarrativeLabel is shown next to audit_logs.summary-derived content.
const AgentNarrativeLabel = "Agent-generated summary (from audit_logs; not independently verified)"
