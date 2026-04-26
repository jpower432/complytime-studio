// SPDX-License-Identifier: Apache-2.0

package store

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/complytime/complytime-studio/internal/consts"
	"github.com/xuri/excelize/v2"
)

func exportExcelHandler(rs RequirementStore, es EvidenceStore, ps PolicyStore, als AuditLogStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		policyID, auditID, f, err := ParseExportQuery(r.URL.Query())
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), consts.ExportHandlerTimeout)
		defer cancel()

		title, version := policyDisplayMeta(ctx, ps, policyID)

		matrix, err := LoadExportMatrix(ctx, rs, f)
		if err != nil {
			if errors.Is(err, ErrExportRowLimit) {
				http.Error(w, err.Error(), http.StatusRequestEntityTooLarge)
				return
			}
			slog.Error("export excel: matrix query failed", "error", err)
			http.Error(w, "query failed", http.StatusInternalServerError)
			return
		}

		evidence, err := LoadExportEvidence(ctx, es, policyID, f.Start, f.End)
		if err != nil {
			if errors.Is(err, ErrExportRowLimit) {
				http.Error(w, err.Error(), http.StatusRequestEntityTooLarge)
				return
			}
			slog.Error("export excel: evidence query failed", "error", err)
			http.Error(w, "query failed", http.StatusInternalServerError)
			return
		}

		auditLog, err := SelectExportAuditLog(ctx, als, policyID, auditID, f.Start, f.End)
		if err != nil {
			if errors.Is(err, ErrExportAuditPolicyMismatch) {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			slog.Error("export excel: audit log lookup failed", "error", err)
			http.Error(w, "audit log not found", http.StatusNotFound)
			return
		}

		agg := BuildExportSummaryAgg(matrix, title, version)
		gaps := filterGapRows(matrix)

		buf, err := buildExcelWorkbook(policyID, f.Start, f.End, agg, matrix, evidence, gaps, auditLog)
		if err != nil {
			slog.Error("export excel: build workbook failed", "error", err)
			http.Error(w, "export failed", http.StatusInternalServerError)
			return
		}

		if ctx.Err() != nil {
			http.Error(w, "export timed out", http.StatusServiceUnavailable)
			return
		}

		fname := SanitizeExportFilename(policyID, ".xlsx")
		w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, fname))
		w.Header().Set("Cache-Control", "no-store")
		_, _ = w.Write(buf.Bytes())

		slog.Info("excel export complete",
			"policy_id", policyID,
			"matrix_rows", len(matrix),
			"evidence_rows", len(evidence),
			"gap_rows", len(gaps),
			"duration_ms", time.Since(started).Milliseconds(),
		)
	}
}

func filterGapRows(matrix []RequirementRow) []RequirementRow {
	var out []RequirementRow
	for _, row := range matrix {
		if IsGapRow(row) {
			out = append(out, row)
		}
	}
	return out
}

func buildExcelWorkbook(policyID string, winStart, winEnd time.Time, agg ExportSummaryAgg, matrix []RequirementRow, evidence []EvidenceRecord, gaps []RequirementRow, audit *AuditLog) (*bytes.Buffer, error) {
	f := excelize.NewFile()
	defer func() { _ = f.Close() }()

	headerStyle, err := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"#DDDDDD"}, Pattern: 1},
	})
	if err != nil {
		return nil, err
	}

	summarySheet := "Executive Summary"
	detailSheet := "Requirement Detail"
	evSheet := "Evidence Inventory"
	gapSheet := "Gap List"

	_, _ = f.NewSheet(summarySheet)
	_, _ = f.NewSheet(detailSheet)
	_, _ = f.NewSheet(evSheet)
	_, _ = f.NewSheet(gapSheet)
	_ = f.DeleteSheet("Sheet1")

	set := func(sheet string, col, rn int, val string) {
		cell, _ := excelize.CoordinatesToCellName(col, rn)
		_ = f.SetCellStr(sheet, cell, val)
	}
	styleHeaderRow := func(sheet string, col1, col2, rn int) {
		c1, _ := excelize.CoordinatesToCellName(col1, rn)
		c2, _ := excelize.CoordinatesToCellName(col2, rn)
		_ = f.SetCellStyle(sheet, c1, c2, headerStyle)
	}

	rn := 1
	set(summarySheet, 1, rn, "Policy ID")
	set(summarySheet, 2, rn, policyID)
	rn++
	set(summarySheet, 1, rn, "Policy title")
	set(summarySheet, 2, rn, agg.PolicyTitle)
	rn++
	set(summarySheet, 1, rn, "Policy version")
	set(summarySheet, 2, rn, agg.PolicyVersion)
	rn++
	set(summarySheet, 1, rn, "Audit window start")
	set(summarySheet, 2, rn, formatExportTime(winStart))
	rn++
	set(summarySheet, 1, rn, "Audit window end")
	set(summarySheet, 2, rn, formatExportTime(winEnd))
	rn++
	set(summarySheet, 1, rn, "Generated (UTC)")
	set(summarySheet, 2, rn, time.Now().UTC().Format(time.RFC3339))
	rn += 2

	set(summarySheet, 1, rn, "Metric")
	set(summarySheet, 2, rn, "Value")
	styleHeaderRow(summarySheet, 1, 2, rn)
	rn++
	set(summarySheet, 1, rn, "Total requirements")
	set(summarySheet, 2, rn, strconv.Itoa(agg.TotalRequirements))
	rn++
	set(summarySheet, 1, rn, "Requirements with evidence")
	set(summarySheet, 2, rn, strconv.Itoa(agg.RequirementsWithEvidence))
	rn++
	set(summarySheet, 1, rn, "Total evidence pieces (sum of counts)")
	set(summarySheet, 2, rn, strconv.FormatUint(agg.TotalEvidencePieces, 10))
	rn++
	set(summarySheet, 1, rn, "Gap requirements (sheet: Gap List)")
	set(summarySheet, 2, rn, strconv.Itoa(len(gaps)))
	rn += 2

	set(summarySheet, 1, rn, "Classification")
	set(summarySheet, 2, rn, "Requirement rows")
	styleHeaderRow(summarySheet, 1, 2, rn)
	rn++
	for cls, n := range agg.ByClassification {
		set(summarySheet, 1, rn, cls)
		set(summarySheet, 2, rn, strconv.Itoa(n))
		rn++
	}

	if audit != nil && audit.Summary != "" {
		rn += 2
		set(summarySheet, 1, rn, AgentNarrativeLabel)
		styleHeaderRow(summarySheet, 1, 1, rn)
		rn++
		set(summarySheet, 1, rn, audit.Summary)
	}

	dh := []string{
		"catalog_id", "control_id", "control_title",
		"requirement_id", "requirement_text",
		"evidence_count", "latest_evidence", "classification",
	}
	for i, h := range dh {
		set(detailSheet, i+1, 1, h)
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		_ = f.SetCellStyle(detailSheet, cell, cell, headerStyle)
	}
	for ri, rowObj := range matrix {
		r := ri + 2
		vals := []string{
			rowObj.CatalogID, rowObj.ControlID, rowObj.ControlTitle,
			rowObj.RequirementID, rowObj.RequirementText,
			strconv.FormatUint(rowObj.EvidenceCount, 10), rowObj.LatestEvidence, rowObj.Classification,
		}
		for ci, v := range vals {
			set(detailSheet, ci+1, r, v)
		}
	}

	eh := []string{
		"evidence_id", "policy_id", "target_id", "rule_id",
		"control_id", "requirement_id", "eval_result", "collected_at",
	}
	for i, h := range eh {
		set(evSheet, i+1, 1, h)
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		_ = f.SetCellStyle(evSheet, cell, cell, headerStyle)
	}
	for ri, e := range evidence {
		r := ri + 2
		vals := []string{
			e.EvidenceID, e.PolicyID, e.TargetID, e.RuleID,
			e.ControlID, e.RequirementID, e.EvalResult, e.CollectedAt.UTC().Format(time.RFC3339),
		}
		for ci, v := range vals {
			set(evSheet, ci+1, r, v)
		}
	}

	for i, h := range dh {
		set(gapSheet, i+1, 1, h)
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		_ = f.SetCellStyle(gapSheet, cell, cell, headerStyle)
	}
	for ri, rowObj := range gaps {
		r := ri + 2
		vals := []string{
			rowObj.CatalogID, rowObj.ControlID, rowObj.ControlTitle,
			rowObj.RequirementID, rowObj.RequirementText,
			strconv.FormatUint(rowObj.EvidenceCount, 10), rowObj.LatestEvidence, rowObj.Classification,
		}
		for ci, v := range vals {
			set(gapSheet, ci+1, r, v)
		}
	}

	return f.WriteToBuffer()
}

func formatExportTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}
