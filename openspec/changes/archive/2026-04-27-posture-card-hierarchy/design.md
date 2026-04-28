# Posture card hierarchy — design

## Context

`PostureCard` currently renders: title + risk badge + version → inventory stats (targets, controls, owner) → pass/fail/other counts → evidence freshness → "View Details" button. The compliance manager's eye scans top-to-bottom, but the most actionable info (freshness, risk) is at the bottom.

## Decision 1: Reorder, not redesign

**Choice:** Rearrange existing elements within the card. Do not change the card's overall shape, size, or component structure.

**Rationale:** The data is already fetched and rendered. This is a layout change, not an architecture change. Minimal risk, maximum impact on usability.

**Consequences:** CSS adjustments only. No API changes for the reorder itself.

## Decision 2: Readiness dot computed client-side

**Choice:** Compute a simple red/yellow/green classification in the `PostureCard` component based on data already available:
- **Green**: `latest_evidence_at` is within 7 days AND risk severity is not Critical or High
- **Yellow**: `latest_evidence_at` is 7–30 days old OR risk severity is Medium
- **Red**: `latest_evidence_at` is older than 30 days OR missing OR risk severity is Critical/High

**Rationale:** All inputs (`latest_evidence_at`, risk severity from `riskMap`) are already fetched by `PostureView`. No new API call needed. Thresholds can be hardcoded initially, made configurable later.

**Consequences:** Thresholds are opinionated. Document them in code. If users disagree with the 7/30 day cutoffs, a future change can make them configurable per policy.

## Decision 3: Delta from posture history (deferred)

**Choice:** The pass rate delta (e.g., "95% → 82%") requires knowing the previous pass rate. The Inbox already computes this server-side for `posture_change` notifications. For the card, defer the delta to a follow-up change that adds a `GET /api/posture/history?policy_id=X&limit=2` endpoint returning the last two snapshots.

**Rationale:** Adding a new endpoint is out of scope for a card reorder. The readiness dot and reordered layout deliver most of the value. Delta is additive.

**Consequences:** The delta indicator is not included in this change. Noted as a follow-up in tasks.

## Related documents

- `openspec/changes/posture-card-clickable/` — card click behavior (can be combined in implementation)
