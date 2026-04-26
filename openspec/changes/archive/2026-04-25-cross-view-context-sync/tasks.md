## 1. Signal Writes on Search

- [x] 1.1 Evidence view: write `selectedPolicyId.value = policyId` in the `search()` function (after params are built)
- [x] 1.2 Audit history: write `selectedPolicyId.value = policyId` in `fetchLogs()` when policyId is non-empty
- [x] 1.3 Audit history: write `selectedTimeRange.value = { start: startFilter, end: endFilter }` in `fetchLogs()` when either date is non-empty, null otherwise
- [x] 1.4 Requirement matrix: write `selectedTimeRange.value = { start: startFilter, end: endFilter }` in `fetchMatrix()` when either date is non-empty
- [x] 1.5 Requirement matrix: write `selectedControlId.value = familyFilter || null` in `fetchMatrix()`
- [x] 1.6 Import `selectedTimeRange`, `selectedControlId` in evidence view and requirement matrix (add to existing import from `../app`)

## 2. Signal Pre-fill on Mount

- [x] 2.1 Evidence view: add `useEffect` that reads `selectedPolicyId.value` and sets local `policyId` state on mount
- [x] 2.2 Audit history: add `useEffect` that reads `selectedPolicyId.value` and `selectedTimeRange.value`, sets local state on mount
- [x] 2.3 Requirement matrix: existing `useEffect` for `selectedPolicyId` is present — add pre-fill for `selectedTimeRange.value` into `startFilter`/`endFilter`
- [x] 2.4 Draft review: no policy filter to pre-fill (status-only) — skip

## 3. Requirement Matrix Invalidation

- [x] 3.1 Add `viewInvalidation.value` to the dependency array of the `useEffect` that calls `fetchMatrix`
- [x] 3.2 Guard the fetch with `if (policyId)` to avoid fetching without a selected policy
- [x] 3.3 Import `viewInvalidation` in requirement matrix (add to existing import)

## 4. Deep Link Routing

- [x] 4.1 Create `parseHashParams()` utility in `app.tsx` that extracts `?key=value` pairs from the hash after the view name
- [x] 4.2 Create `updateHash()` utility that builds the hash string from current view + non-null signal values
- [x] 4.3 Call `updateHash()` in the `navigate()` function and after each signal write in search functions
- [x] 4.4 In `syncFromHash()`, call `parseHashParams()` and write parsed values to shared signals
- [x] 4.5 Handle `policy`, `start`, `end`, `control`, `req` parameters in the parser

## 5. Verification

- [x] 5.1 Navigate posture -> requirements via card button: verify policy pre-fills in requirement matrix
- [x] 5.2 Set date range in audit history, navigate to requirements: verify dates pre-fill
- [x] 5.3 Set policy in evidence view, navigate to requirements: verify policy pre-fills
- [x] 5.4 Trigger `invalidateViews()` via agent artifact: verify requirement matrix refetches
- [x] 5.5 Copy URL with hash params, open in new tab: verify view loads with correct filters
- [x] 5.6 Verify agent chat context includes populated `policy_id` and `time_range_start`/`end` after cross-view navigation
