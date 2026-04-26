## Why

The workbench uses `<div>` elements for data tables, cards, and layout regions. Screen readers cannot parse structure, keyboard users cannot navigate, and the agent cannot reliably read UI state from the DOM. Accessibility is a prerequisite for the agent to become an assistive interface — an auditor using a screen reader should be able to ask the agent "what's failing?" instead of tabbing through unstructured markup.

## What Changes

- Replace `<div>`-based tables with semantic `<table>` / `<thead>` / `<th>` / `<td>` across all views
- Add structural landmarks: `<section>`, `<article>`, `<header>` for cards and view regions
- Add `data-*` attributes on interactive elements so the agent can read policy ID, classification, view state, and requirement context from the DOM
- Add `aria-live="polite"` on the chat assistant streaming region (the only custom widget requiring ARIA)
- Add `aria-expanded` on expandable requirement rows and collapsible filter panels
- Add visible focus styles and logical tab order on all interactive elements
- Add skip-to-main link for keyboard navigation
- Rename "Blind" classification to "No Evidence" across skill, backend, and frontend
- Ensure color is never the sole indicator — all badges already have text labels, verify contrast ratios meet WCAG AA (4.5:1 normal text, 3:1 large text)

## Capabilities

### New Capabilities
- `semantic-landmarks`: Semantic HTML structure and landmark elements across all workbench views
- `agent-dom-contract`: Data attribute contract (`data-policy-id`, `data-classification`, `data-view`, etc.) enabling agent DOM parsing
- `keyboard-navigation`: Focus management, skip links, visible focus states, logical tab order
- `classification-rename`: Rename "Blind" to "No Evidence" in classification system (skill, backend, frontend)

### Modified Capabilities
- `react-workbench`: Component markup changes from `<div>` to semantic elements
- `streaming-chat`: Add `aria-live` region for SSE streaming responses
- `posture-check-skill`: Update classification label from "Blind" to "No Evidence"

## Impact

- **Frontend:** Every view component in `workbench/src/components/` changes markup
- **CSS:** `workbench/src/global.css` needs focus styles, skip link styles, contrast adjustments
- **Backend:** `internal/store/` queries and handlers that reference "Blind" classification
- **Skills:** `skills/posture-check/SKILL.md` and `skills/studio-audit/SKILL.md` classification tables
- **Agent prompt:** `agents/assistant/prompt.md` classification references
- **No new dependencies** — semantic HTML and data attributes are native platform features
