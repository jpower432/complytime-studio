## Context

The workbench is a Preact SPA with six views, a sidebar, and a floating chat overlay. Current state:

- Requirement matrix already uses `<table>` — no change needed there
- Posture cards use `<div class="posture-card">` — not parsable as structured data
- Evidence view tables use `<div>` rows — screen readers cannot navigate
- Chat overlay has no live region — streaming agent responses are invisible to assistive tech
- No `data-*` attributes — the agent cannot read UI state from the DOM
- No visible focus styles — keyboard users cannot track position
- "Blind" classification used in skill, backend, and frontend — rename to "No Evidence"
- CSS uses `--text` / `--text-muted` and `--bg-surface` tokens — need to verify WCAG AA contrast ratios

## Goals / Non-Goals

**Goals:**
- Semantic HTML across all views so screen readers and agents parse structure natively
- `data-*` attribute contract on interactive elements for agent DOM awareness
- `aria-live` on chat streaming region; `aria-expanded` on expandable rows/panels
- Visible focus styles and logical tab order on all interactive elements
- Skip-to-main keyboard shortcut
- Rename "Blind" -> "No Evidence" everywhere
- WCAG AA contrast compliance for all text/background combinations

**Non-Goals:**
- Full WCAG AAA compliance (AA is the target)
- Rewriting component architecture or state management
- Adding new views or features
- Supporting right-to-left (RTL) localization

## Decisions

**1. Semantic elements over ARIA**

Use native HTML elements (`<table>`, `<article>`, `<section>`, `<nav>`, `<header>`) instead of `<div>` + ARIA roles. ARIA is only used where no HTML equivalent exists: `aria-live` for SSE streaming, `aria-expanded` for expand/collapse.

Alternative considered: ARIA roles on existing `<div>` markup. Rejected — more code, less maintainable, same net effect.

**2. Data attribute contract for agent parsability**

Define a stable set of `data-*` attributes the agent can rely on:

| Attribute | Elements | Purpose |
|-----------|----------|---------|
| `data-view` | `<main>` | Current active view name |
| `data-policy-id` | Posture cards, filter selects, table rows | Active policy context |
| `data-classification` | Classification badges | Machine-readable classification value |
| `data-requirement-id` | Matrix rows | Requirement identifier |
| `data-evidence-id` | Evidence rows | Evidence identifier |
| `data-eval-result` | Result badges | Evaluation result value |
| `data-expanded` | Expandable rows | Boolean expand state |

Alternative considered: custom elements or shadow DOM. Rejected — Preact does not use shadow DOM, and data attributes are universally supported.

**3. Focus management strategy**

Add CSS `outline` on `:focus-visible` for all interactive elements using a utility class. Use `:focus-visible` (not `:focus`) to avoid showing outlines on mouse clicks.

Skip link as the first focusable element in `App`, targeting `<main>`.

**4. Classification rename scope**

"Blind" -> "No Evidence" touches:

| Layer | Files |
|-------|-------|
| Frontend | `requirement-matrix-view.tsx` (CLASSIFICATIONS array, ClassificationBadge) |
| Backend | `internal/store/store.go`, `internal/clickhouse/client.go` (classification logic) |
| Skills | `skills/posture-check/SKILL.md`, `skills/studio-audit/SKILL.md` |
| Prompt | `agents/assistant/prompt.md` |
| Chart prompt | `charts/complytime-studio/agents/assistant/prompt.md` |

Backend classification is computed in ClickHouse queries — the string literal appears in SQL CASE expressions.

## Risks / Trade-offs

**[CSS specificity conflicts]** Adding `:focus-visible` styles may conflict with existing component styles that use `outline: none`. Mitigation: audit and remove `outline: none` declarations; apply focus styles at the `:root` level.

**[Data attribute contract stability]** Agents may depend on `data-*` attributes. Changing attribute names becomes a breaking change for agent prompts. Mitigation: document the contract in `docs/design/architecture.md` and treat attribute renames as breaking.

**[Classification rename migration]** Existing ClickHouse data may contain "Blind" in stored classification values. Mitigation: classification is computed at query time (CASE expression), not stored — no data migration needed. Verify no materialized views cache the old value.
