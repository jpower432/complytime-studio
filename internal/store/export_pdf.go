// SPDX-License-Identifier: Apache-2.0

// PDF export renders a compact auditor-facing pack. Truncation rules:
//   - Requirement table: at most maxPDFRequirementPrintRows rows; if matrix is
//     larger, a footer states the omitted count (full data: Excel/CSV).
//   - Gap list: at most maxPDFGapPrintRows rows; same footer convention.
//   - Agent narrative (audit_logs.summary): at most maxPDFNarrativeRunes runes;
//     remainder replaced with "… (truncated)".
package store

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/complytime/complytime-studio/internal/consts"
	"github.com/go-pdf/fpdf"
)

const (
	maxPDFRequirementPrintRows = 800
	maxPDFGapPrintRows         = 400
	maxPDFNarrativeRunes       = 4000
)

func exportPDFHandler(rs RequirementStore, ps PolicyStore, als AuditLogStore) http.HandlerFunc {
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
			slog.Error("export pdf: matrix query failed", "error", err)
			http.Error(w, "query failed", http.StatusInternalServerError)
			return
		}

		auditLog, err := SelectExportAuditLog(ctx, als, policyID, auditID, f.Start, f.End)
		if err != nil {
			if errors.Is(err, ErrExportAuditPolicyMismatch) {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			slog.Error("export pdf: audit log lookup failed", "error", err)
			http.Error(w, "audit log not found", http.StatusNotFound)
			return
		}

		agg := BuildExportSummaryAgg(matrix, title, version)
		gaps := filterGapRows(matrix)

		pdfBytes, err := buildPDFDocument(policyID, f.Start, f.End, agg, matrix, gaps, auditLog)
		if err != nil {
			slog.Error("export pdf: build failed", "error", err)
			http.Error(w, "export failed", http.StatusInternalServerError)
			return
		}

		if ctx.Err() != nil {
			http.Error(w, "export timed out", http.StatusServiceUnavailable)
			return
		}

		fname := SanitizeExportFilename(policyID, ".pdf")
		w.Header().Set("Content-Type", "application/pdf")
		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, fname))
		w.Header().Set("Cache-Control", "no-store")
		_, _ = w.Write(pdfBytes)

		slog.Info("pdf export complete",
			"policy_id", policyID,
			"matrix_rows", len(matrix),
			"gap_rows", len(gaps),
			"duration_ms", time.Since(started).Milliseconds(),
		)
	}
}

func pdfSafe(s string) string {
	return strings.Map(func(r rune) rune {
		if r > 127 {
			return '?'
		}
		if r < 32 && r != '\n' {
			return ' '
		}
		return r
	}, s)
}

func truncatePDFNarrative(s string) string {
	if utf8.RuneCountInString(s) <= maxPDFNarrativeRunes {
		return s
	}
	runes := []rune(s)
	return string(runes[:maxPDFNarrativeRunes]) + "… (truncated)"
}

func buildPDFDocument(policyID string, winStart, winEnd time.Time, agg ExportSummaryAgg, matrix, gaps []RequirementRow, audit *AuditLog) ([]byte, error) {
	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.SetTitle(pdfSafe("ComplyTime audit export"), false)
	pdf.SetAuthor("ComplyTime Studio", false)

	// Cover
	pdf.AddPage()
	pdf.SetFont("Helvetica", "B", 18)
	pdf.CellFormat(0, 12, pdfSafe("ComplyTime audit export"), "", 1, "C", false, 0, "")
	pdf.Ln(6)
	pdf.SetFont("Helvetica", "", 11)
	pdf.MultiCell(0, 6, pdfSafe(fmt.Sprintf("Policy ID: %s", policyID)), "", "L", false)
	if agg.PolicyTitle != "" {
		pdf.MultiCell(0, 6, pdfSafe(fmt.Sprintf("Title: %s", agg.PolicyTitle)), "", "L", false)
	}
	if agg.PolicyVersion != "" {
		pdf.MultiCell(0, 6, pdfSafe(fmt.Sprintf("Version: %s", agg.PolicyVersion)), "", "L", false)
	}
	pdf.MultiCell(0, 6, pdfSafe("Audit window: "+formatExportTime(winStart)+" — "+formatExportTime(winEnd)), "", "L", false)
	pdf.MultiCell(0, 6, pdfSafe("Generated (UTC): "+time.Now().UTC().Format(time.RFC3339)), "", "L", false)

	// Summary
	pdf.AddPage()
	pdf.SetFont("Helvetica", "B", 14)
	pdf.Cell(0, 8, pdfSafe("Summary"))
	pdf.Ln(10)
	pdf.SetFont("Helvetica", "", 10)
	lines := []string{
		fmt.Sprintf("Total requirements: %d", agg.TotalRequirements),
		fmt.Sprintf("Requirements with evidence: %d", agg.RequirementsWithEvidence),
		fmt.Sprintf("Total evidence pieces (sum): %d", agg.TotalEvidencePieces),
		fmt.Sprintf("Gap requirements: %d", len(gaps)),
	}
	for _, ln := range lines {
		pdf.MultiCell(0, 5, pdfSafe(ln), "", "L", false)
	}
	pdf.Ln(4)
	pdf.SetFont("Helvetica", "B", 10)
	pdf.Cell(0, 6, pdfSafe("By classification"))
	pdf.Ln(6)
	pdf.SetFont("Helvetica", "", 10)
	for cls, n := range agg.ByClassification {
		pdf.MultiCell(0, 5, pdfSafe(fmt.Sprintf("%s: %d", cls, n)), "", "L", false)
	}

	if audit != nil && audit.Summary != "" {
		pdf.Ln(6)
		pdf.SetFont("Helvetica", "B", 10)
		pdf.MultiCell(0, 5, pdfSafe(AgentNarrativeLabel), "", "L", false)
		pdf.SetFont("Helvetica", "", 9)
		pdf.MultiCell(0, 4, pdfSafe(truncatePDFNarrative(audit.Summary)), "", "L", false)
	}

	// Requirements table
	pdf.AddPage()
	pdf.SetFont("Helvetica", "B", 14)
	pdf.Cell(0, 8, pdfSafe("Requirement detail"))
	pdf.Ln(6)
	reqShown := len(matrix)
	omittedReq := 0
	if reqShown > maxPDFRequirementPrintRows {
		omittedReq = reqShown - maxPDFRequirementPrintRows
		reqShown = maxPDFRequirementPrintRows
	}
	pdf.SetFont("Helvetica", "", 8)
	if len(matrix) == 0 {
		pdf.SetFont("Helvetica", "", 10)
		pdf.MultiCell(0, 5, pdfSafe("No requirement rows in this window."), "", "L", false)
	} else {
		drawReqTable(pdf, matrix[:reqShown])
	}
	if omittedReq > 0 {
		pdf.Ln(4)
		pdf.SetFont("Helvetica", "I", 9)
		pdf.MultiCell(0, 5, pdfSafe(fmt.Sprintf(
			"Table truncated: %d rows omitted (PDF limit %d). Use Excel or CSV for the full matrix.",
			omittedReq, maxPDFRequirementPrintRows,
		)), "", "L", false)
	}

	// Gaps
	pdf.AddPage()
	pdf.SetFont("Helvetica", "B", 14)
	pdf.Cell(0, 8, pdfSafe("Gap list"))
	pdf.Ln(6)
	gapShown := len(gaps)
	omittedGap := 0
	if gapShown > maxPDFGapPrintRows {
		omittedGap = gapShown - maxPDFGapPrintRows
		gapShown = maxPDFGapPrintRows
	}
	pdf.SetFont("Helvetica", "", 8)
	if gapShown > 0 {
		drawReqTable(pdf, gaps[:gapShown])
	} else {
		pdf.SetFont("Helvetica", "", 10)
		pdf.MultiCell(0, 5, pdfSafe("No gap rows in this window."), "", "L", false)
	}
	if omittedGap > 0 {
		pdf.Ln(4)
		pdf.SetFont("Helvetica", "I", 9)
		pdf.MultiCell(0, 5, pdfSafe(fmt.Sprintf(
			"Gap list truncated: %d rows omitted (PDF limit %d). Use Excel or CSV.",
			omittedGap, maxPDFGapPrintRows,
		)), "", "L", false)
	}

	var buf bytes.Buffer
	err := pdf.Output(&buf)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func drawReqTable(pdf *fpdf.Fpdf, rows []RequirementRow) {
	if len(rows) == 0 {
		return
	}
	headers := []string{"Control", "Requirement", "Evid#", "Class"}
	ws := []float64{28, 95, 18, 45}
	pdf.SetFont("Helvetica", "B", 8)
	for i, h := range headers {
		ln := 0
		if i == len(headers)-1 {
			ln = 1
		}
		pdf.CellFormat(ws[i], 6, pdfSafe(h), "1", ln, "L", false, 0, "")
	}
	pdf.SetFont("Helvetica", "", 7)
	for _, row := range rows {
		line := []string{
			pdfSafe(trunc(row.ControlID, 24)),
			pdfSafe(trunc(row.RequirementID+": "+row.RequirementText, 120)),
			pdfSafe(strconv.FormatUint(row.EvidenceCount, 10)),
			pdfSafe(trunc(row.Classification, 28)),
		}
		for i := range line {
			ln := 0
			if i == len(line)-1 {
				ln = 1
			}
			pdf.CellFormat(ws[i], 6, line[i], "1", ln, "L", false, 0, "")
		}
	}
}

func trunc(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "…"
}
