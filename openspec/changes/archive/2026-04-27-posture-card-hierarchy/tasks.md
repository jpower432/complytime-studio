# Tasks — posture card hierarchy

## Card reorder

- [x] In `PostureCard` (`posture-view.tsx`), reorder JSX:
  1. Header: title + readiness dot + risk badge + version
  2. Evidence freshness line (`Last evidence: Xd ago`) — moved up from bottom
  3. Pass/fail/other counts + pass rate
  4. Inventory stats (targets, controls, owner) — moved to bottom

## Readiness indicator

- [x] Add `readinessLevel(row, riskSeverity)` helper function returning `"green"` | `"yellow"` | `"red"` based on:
  - Green: `latest_evidence_at` within 7 days AND severity not Critical/High
  - Yellow: `latest_evidence_at` 7–30 days OR severity Medium
  - Red: `latest_evidence_at` > 30 days OR missing OR severity Critical/High
- [x] Render a `.readiness-dot` span next to the title in the card header.
- [x] Add CSS for `.readiness-dot`: 8px circle, inline with title, three color variants.

## CSS adjustments

- [x] Adjust `.posture-card` internal spacing for new element order.
- [x] Ensure evidence freshness line has appropriate font size/color when positioned higher.

## Tests

- [ ] Verify readiness dot is green when evidence is fresh and risk is low.
- [ ] Verify readiness dot is red when evidence is missing.
- [ ] Verify readiness dot is yellow when evidence is stale (8–30 days).
- [ ] Visual check: card scans top-to-bottom with actionable info first.

## Follow-up (out of scope)

- [ ] Delta indicator on card — requires `GET /api/posture/history` endpoint. Track as separate change.
