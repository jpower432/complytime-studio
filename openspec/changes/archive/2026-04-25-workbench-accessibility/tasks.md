## 1. Classification Rename (Blind -> No Evidence)

- [x] 1.1 Update `CLASSIFICATIONS` array in `workbench/src/components/requirement-matrix-view.tsx`: replace `"Blind"` with `"No Evidence"`
- [x] 1.2 Update `ClassificationBadge` CSS class derivation to handle "No Evidence" (e.g., `no-evidence` class)
- [x] 1.3 Update ClickHouse CASE expressions in `internal/clickhouse/client.go` that produce "Blind" to produce "No Evidence"
- [x] 1.4 Update classification references in `internal/store/store.go`
- [x] 1.5 Update `skills/posture-check/SKILL.md` five-state classification table: rename Blind to No Evidence
- [x] 1.6 Update `skills/studio-audit/SKILL.md` if it references Blind classification
- [x] 1.7 Update `agents/assistant/prompt.md` classification references
- [x] 1.8 Run `make sync-prompts` to sync chart prompt copy

## 2. Semantic Landmarks

- [x] 2.1 Wrap `PostureView` root in `<section>` with `<h2>` heading; wrap each `PostureCard` in `<article>`
- [x] 2.2 Convert evidence view results from `<div>` rows to `<table>` / `<thead>` / `<th>` / `<td>`
- [x] 2.3 Convert audit history log list from `<div>` cards to `<table>` with `<th>` headers
- [x] 2.4 Convert draft review list from `<div>` cards to semantic `<article>` or `<table>` as appropriate
- [x] 2.5 Verify app shell uses `<header>` for top bar (update `header.tsx` if using `<div>`)
- [x] 2.6 Verify `<main>` wraps the content area in `app.tsx` (already present — confirm)

## 3. Agent DOM Contract

- [x] 3.1 Add `data-view={view}` to the `<main>` element in `app.tsx`
- [x] 3.2 Add `data-policy-id={row.policy_id}` to posture card `<article>` elements
- [x] 3.3 Add `data-classification={value}` to `ClassificationBadge` span elements
- [x] 3.4 Add `data-requirement-id={row.requirement_id}` to matrix `<tr>` elements
- [x] 3.5 Add `data-evidence-id={ev.evidence_id}` to evidence `<tr>` elements
- [x] 3.6 Add `data-expanded="true|false"` to expandable requirement matrix rows
- [x] 3.7 Add `data-policy-id` to policy filter `<select>` elements in requirement matrix and evidence view
- [x] 3.8 Document the agent DOM contract in `docs/design/architecture.md` under a new section

## 4. Chat Accessibility

- [x] 4.1 Add `aria-live="polite"` to the `.chat-overlay-messages` container in `chat-assistant.tsx`
- [x] 4.2 Add `aria-expanded={open}` to the chat FAB button
- [x] 4.3 Add `role="log"` to the messages container so screen readers treat it as a log region

## 5. Keyboard Navigation

- [x] 5.1 Add skip-to-main link as first child of `.app-shell` in `app.tsx` (visually hidden, visible on focus)
- [x] 5.2 Add `:focus-visible` outline styles to `global.css` for buttons, links, inputs, selects, and table rows
- [x] 5.3 Remove any existing `outline: none` declarations in `global.css`
- [x] 5.4 Add `aria-expanded` to expandable requirement matrix rows and collapsible filter panels
- [x] 5.5 Ensure chat input `tabindex` follows logical order (after main content, before nothing)

## 6. Contrast Verification

- [x] 6.1 Audit light theme CSS variables (`--text` on `--bg`, `--text-muted` on `--bg-surface`, etc.) against WCAG AA 4.5:1
- [x] 6.2 Audit dark theme CSS variables against WCAG AA 4.5:1
- [x] 6.3 Verify classification badge colors meet 3:1 contrast against card backgrounds in both themes
- [x] 6.4 Fix any failing contrast pairs by adjusting CSS custom properties

## 7. Verification

- [x] 7.1 Tab through the full app: skip link -> header -> sidebar -> main content -> chat FAB — confirm logical order
- [x] 7.2 Verify requirement matrix rows announce `data-expanded` and `aria-expanded` state changes
- [x] 7.3 Verify chat streaming responses are announced by screen reader via `aria-live`
- [x] 7.4 Verify "No Evidence" appears in classification filter, badges, and agent prompt — zero references to "Blind" remain
- [x] 7.5 Verify `data-view`, `data-policy-id`, `data-classification`, `data-requirement-id`, `data-evidence-id` attributes render correctly in DOM
