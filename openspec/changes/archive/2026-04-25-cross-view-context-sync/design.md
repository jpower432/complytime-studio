## Context

The workbench uses Preact signals in `app.tsx` for shared state: `currentView`, `selectedPolicyId`, `selectedTimeRange`, `selectedControlId`, `selectedRequirementId`, `selectedEvalResult`, and `viewInvalidation`. Views read these signals to pre-scope queries and the chat assistant injects them into the first message of each agent task.

Current wiring gaps:
- `selectedPolicyId`: set by posture card buttons and requirement matrix `useEffect`. Not set by evidence, audit history, or draft review policy selects.
- `selectedTimeRange`: never set by any view. Defined as `signal<{ start: string; end: string } | null>(null)`.
- `selectedControlId`: never set. Defined as `signal<string | null>(null)`.
- `selectedEvalResult`: never set. Defined as `signal<string | null>(null)`.
- Requirement matrix subscribes to `policyId` changes but not `viewInvalidation`.

## Goals / Non-Goals

**Goals:**
- Every policy selector writes `selectedPolicyId` on change
- Date range inputs in audit history and requirement matrix write `selectedTimeRange`
- Requirement matrix control family filter writes `selectedControlId`
- Evidence view result filter writes `selectedEvalResult` (if one exists — currently eval_result is shown in badges but not filterable as a standalone dropdown)
- Views pre-fill their local filter state from shared signals on mount
- Requirement matrix refetches when `viewInvalidation` changes
- URL hash encodes active filters for deep linking

**Non-Goals:**
- Server-side persistence of filter state (signals are ephemeral)
- Changing the agent prompt or A2A protocol
- Adding new views or API endpoints

## Decisions

**1. Signal writes at the filter onChange handler**

Each view's policy/date/control `onChange` handler writes to the shared signal in addition to local state. This is a one-line addition per handler.

Alternative considered: centralized signal-driven state where views don't have local state. Rejected — too invasive, and local state is needed for uncommitted filter changes (user may change a dropdown without clicking Search).

Compromise: write to shared signal on **search/apply**, not on every keystroke. This prevents partial state from leaking across views.

**2. Pre-fill on mount via useEffect**

Each view reads shared signals in a `useEffect([], ...)` hook on mount and sets local filter state if the signal is non-null. This runs once per navigation.

**3. Deep link hash format**

Extend the current `#/<view>` format to `#/<view>?policy=X&start=Y&end=Z&req=R`. Parse on load, write to signals. The `?` separator is non-standard for hash routing but widely supported and avoids conflicting with the real query string.

**4. viewInvalidation subscription in requirement matrix**

Add `viewInvalidation.value` to the dependency array of the `useEffect` that calls `fetchMatrix`. Guard with `if (policyId)` to avoid fetching without a policy.

## Risks / Trade-offs

**[Signal write timing]** Writing to shared signals on "Search" click means navigating away before clicking Search loses the filter context. Mitigation: acceptable — uncommitted filters are intentionally not shared.

**[Hash complexity]** Deep links add URL parsing complexity. Mitigation: a single `parseHashParams` utility centralizes parsing. Views read params on mount only.

**[Stale signals]** Navigating to a view with a stale `selectedPolicyId` that no longer exists in the API. Mitigation: views already handle empty/error API responses gracefully. The pre-filled dropdown simply won't match, showing the default "Select a policy..." option.
