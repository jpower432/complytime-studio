## Why

Shared dashboard signals (`selectedPolicyId`, `selectedTimeRange`, `selectedControlId`, `selectedEvalResult`) are defined in `app.tsx` but only partially wired. `selectedPolicyId` updates from posture shortcuts and requirement expansion. The other three signals are never set by any view. This means:

- Analyst selects a policy in posture, navigates to evidence — filter is empty, must re-select
- Auditor sets date range in audit history, navigates to requirements — dates not carried over
- Agent receives dashboard context with null time range and control, reducing response relevance
- Requirement matrix does not refetch when `viewInvalidation` fires after agent updates

The agent's injected context (`buildInjectedContext`) reads all five signals. Every null signal is a missed opportunity for the agent to scope its response correctly.

## What Changes

- Wire `selectedPolicyId` from every view that has a policy selector (evidence, audit history, draft review) — currently only posture and requirements set it
- Wire `selectedTimeRange` from audit history date filters and requirement matrix date filters
- Wire `selectedControlId` from requirement matrix when a control family is filtered
- Wire `selectedEvalResult` from evidence view when a result filter is applied
- Pre-fill filter inputs from shared signals when navigating into a view
- Refetch requirement matrix on `viewInvalidation` (currently only refetches on `policyId` change)
- Add deep link support: encode policy, time range, and requirement in the URL hash

## Capabilities

### New Capabilities
- `deep-link-routing`: URL hash encodes view + active filters, enabling shareable links and browser back/forward

### Modified Capabilities
- `react-workbench`: Signal propagation across views and pre-fill on navigation
- `streaming-chat`: Agent context injection benefits from populated signals (no code change, but behavior improves)

## Impact

- **Frontend:** `app.tsx` (signal writes), `posture-view.tsx`, `evidence-view.tsx`, `audit-history-view.tsx`, `draft-review-view.tsx`, `requirement-matrix-view.tsx` (filter pre-fill + signal writes), `chat-assistant.tsx` (context injection already reads signals — no change needed)
- **No backend changes** — signals are client-side Preact signals
- **No new dependencies**
