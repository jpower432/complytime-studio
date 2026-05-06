# Known Issues: Posture Donut and Recommendation Engine

**Status:** Resolved
**Date:** 2026-05-05

## Issue 1: Posture Donut Always Shows 0%

**Symptom:** Program posture donut displayed 0% (or 90% threshold) regardless of actual evidence.

**Root cause:** `score_pct` was only updated reactively via NATS `EvidenceEvent`. If evidence existed before the subscriber started, or policies were attached after evidence ingestion, posture never recomputed.

**Resolution:**
- Added `PopulatePosture` startup backfill (async on gateway boot) — iterates programs with policies, computes real score from evidence
- Added posture recompute trigger on `PUT /programs/:id` when policy_ids change
- Migration `012_program_score_pct.sql` added `score_pct` column
- Frontend reads `score_pct` for the donut instead of `green_pct` threshold

**Files:** `internal/posture/posture.go`, `internal/store/handlers_programs.go`, `cmd/gateway/main.go`

## Issue 2: Recommendation Engine Not Working

**Symptom:** Recommendations tab returned empty for all programs.

**Root cause (actual):** The recommendation query joined `catalogs.policy_id` to find policies connected via mappings. But `catalogs.policy_id` is never populated during import — catalogs are stored standalone without policy linkage. The `controls` table (populated during policy import) correctly stores both `catalog_id` and `policy_id`.

Secondary issue: `guidance_catalog_id` auto-resolution used fuzzy `ILIKE` matching against catalog titles, which failed for "ISO 27001" (catalog_id is `iso27001-2022`).

**Resolution:**
- Changed recommendation query to join `controls` table (has `policy_id` set) instead of `catalogs` (always empty)
- Changed guidance resolution to exact-match `mapping_documents.framework`
- Added `predicted_score_pct` and `score_delta` optional fields on recommendation responses

**Files:** `internal/recommend/recommend.go`, `internal/postgres/programs.go`

## Residual Notes

- `enrichWithPredictedPosture` runs up to 10 sequential queries (one per candidate). Acceptable at current scale; batch if N grows.
- `PopulatePosture` runs async on boot. Partial failures are logged per-program at warn level.
- Recommendations query live (no cache). New imports appear on next tab view.
