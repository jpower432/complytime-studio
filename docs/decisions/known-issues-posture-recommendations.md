# Known Issues: Posture Donut and Recommendation Engine

**Status:** Known — tracking
**Date:** 2026-05-05

## Issue 1: Posture Donut Always Shows 90%

**Symptom:** The program detail coverage donut displays 90% pass regardless of actual posture data.

**Root cause:** `programs.green_pct` defaults to `90` in both the schema and the Go insert fallback:

```sql
-- 005_programs.sql
green_pct INT NOT NULL DEFAULT 90,
red_pct   INT NOT NULL DEFAULT 50,
```

```go
// internal/postgres/programs.go
if greenPct == 0 {
    greenPct = 90
}
```

The posture engine (`internal/posture/subscriber.go`) should recompute and store real percentages via `ComputeAndStore`, but this only fires when posture check events are processed. Until the engine runs against live evidence data, programs display the hardcoded defaults.

**Fix path:** Either:
- Default to `0` and show an "awaiting data" state in the donut when `green_pct == 0`
- Trigger an initial posture computation on program creation or evidence ingestion

## Issue 2: Recommendation Engine Not Working

**Symptom:** The Recommendations tab in Program Detail shows no recommendations or fails to load.

**Endpoint:** `GET /api/programs/{id}/recommendations`

**Root cause:** Recommendations depend on:
1. Mapping documents loaded and entries populated (`PopulateMappingEntries`)
2. Mapping strength scores computed between program policies and available catalogs
3. Evidence counts per policy

If any of these prerequisites are missing (common in fresh deployments or when `make seed` hasn't fully completed), the recommendation engine returns empty results or errors.

**Fix path:** Ensure the recommendation handler gracefully degrades and surfaces which prerequisites are missing, rather than returning empty results silently.
