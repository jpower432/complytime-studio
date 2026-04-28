## Why

The posture card's information hierarchy serves a developer mental model (pass/fail counts, pass rate) rather than a compliance manager's decision flow. The compliance manager's first question is "do I need to act?" — answered by evidence freshness and risk severity, not raw counts. Both signals exist on the card today but are buried at the bottom.

The QE review flagged this: "Information hierarchy doesn't match the compliance manager's decision flow."

## What Changes

- **Reorder the posture card** to surface decision-driving signals first:
  1. **Top**: Title + risk severity badge (existing) + evidence freshness ("Last evidence: 3d ago" moved from bottom)
  2. **Middle**: Pass/fail/other counts + pass rate (existing, repositioned)
  3. **Bottom**: Inventory stats (targets, controls) + owner (existing, repositioned)
- **Add a readiness indicator** — simple red/yellow/green dot next to the title based on: green = all evidence fresh + no critical/high risk; yellow = stale evidence or medium risk; red = missing evidence or critical/high risk.
- **Add delta indicator on card** — show pass rate change since last evidence batch (e.g., "95% → 82%") directly on the card, not just in Inbox notifications.

## Capabilities

### New Capabilities
- `posture-readiness-indicator`: At-a-glance red/yellow/green dot computed from evidence freshness + risk severity.
- `posture-delta-on-card`: Pass rate delta shown directly on the posture card.

### Modified Capabilities
- `posture-card-layout`: Reordered information hierarchy — decision signals first, details second.

## Impact

- **Workbench**: `posture-view.tsx` `PostureCard` — reorder JSX, add readiness dot, add delta display.
- **API**: May need a lightweight endpoint or extension to return previous pass rate for delta computation. Alternatively, compute client-side from cached previous fetch.
- **CSS**: New `.readiness-dot` styles (3 color variants). Adjust card spacing for reordered layout.

## Constitution Alignment

### III. Observable Quality

**Assessment**: PASS

Surfaces the most actionable signals first. The compliance manager can scan 20 policy cards and immediately identify which need attention.
