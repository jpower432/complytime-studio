---
name: cypress-demo
description: >-
  Generate a Cypress demo video spec for a completed OpenSpec change. Produces a
  self-contained .cy.ts file with synthetic cursor, click ripples, and captions.
  Use when recording feature demos, creating PR walkthrough videos, or generating
  visual proof-of-work after a change is implemented.
---

# Cypress Demo Skill

Generate a polished Cypress demo spec that records a human-paced video walkthrough of a completed UI change in ComplyTime Studio.

## When to use

After completing an OpenSpec change implementation, invoke this skill to produce a `demo/cypress/e2e/<change-name>-demo.cy.ts` spec that:

- Demonstrates the happy path end-to-end
- Renders with a synthetic cursor, click ripples, and caption overlay
- Is self-contained — no external imports needed
- Outputs an `.mp4` video via `cypress run --no-runner-ui`

## Inputs

| Input | Source | Required |
|-------|--------|----------|
| Change name | OpenSpec change being demoed | Yes |
| Task list | Completed tasks from the change | Yes |
| UI routes | Hash routes used by the new feature (`#/posture`, `#/audit`, etc.) | Yes |
| Key selectors | CSS classes/data-testid of new UI elements | Yes |
| Demo narrative | What each step should convey to the audience | Infer from tasks |

## Workflow

### 1. Read context

Read the following files to understand what was built:
- `demo/cypress/e2e/_template.cy.ts` — base structure to copy
- `demo/cypress/support/demo-helpers.ts` — helper reference (inline in spec)
- The relevant workbench component files for correct selectors
- `demo/prompts.md` — existing demo narrative patterns

### 2. Determine the demo flow

Map each major task group to a demo step. Typical structure:

| Step | Pattern |
|------|---------|
| Orient | Navigate to the view, establish context with a caption |
| Primary Action | Click the new UI element, show the result |
| Edge / Detail | Drill into a specific scenario if applicable |
| Artifact / Output | Show the final artifact, save it if applicable |

Aim for **~2 minutes** total. Adjust `LONG`/`PAUSE`/`SHORT` constants if needed.

### 3. Generate the spec file

Create `demo/cypress/e2e/<change-name>-demo.cy.ts` by:

1. Copy the helper block verbatim from `_template.cy.ts` (keep specs self-contained)
2. Name the `describe()` block: `"<Change Name>: <one-line summary>"`
3. Use `caption()` before every significant action
4. Use `cursorClick()` for all interactive elements
5. Use `cy.wait(LONG)` after agent responses to let content settle on camera
6. Pin `cy.visit("/")` or the specific hash route for the feature
7. Guard against auth: check for `STUDIO_API_TOKEN` env var

### 4. Selector rules

| Pattern | Rule |
|---------|------|
| Sidebar navigation | `.sidebar-item` — use `.first()` or filter by text |
| Chat assistant open | `.chat-fab` |
| Chat input | `.chat-overlay-input textarea` or `input[type=text]` |
| Send button | `.chat-overlay-input .btn-primary` |
| Agent thinking | `.chat-thinking` — wait for `not.exist` to confirm response |
| Artifact save | `.chat-artifact-card .btn-primary` |
| New views | Inspect the component file; prefer class selectors over text |

### 5. Verify prerequisites comment

Always include a prerequisites comment at the top of the spec:

```typescript
// Prerequisites:
//   - Stack running: make compose-up (or port-forward to cluster)
//   - Demo data seeded: make seed (if applicable)
//   - STUDIO_API_TOKEN env var if auth is enabled
//
// Run:
//   cd demo && npx cypress run --no-runner-ui --spec 'cypress/e2e/<change-name>-demo.cy.ts'
```

### 6. Run and verify

```bash
cd demo
npx cypress run --no-runner-ui --spec "cypress/e2e/<change-name>-demo.cy.ts"
```

Video output: `demo/cypress/videos/<change-name>-demo.cy.ts.mp4`

If the spec fails:
- Check that the stack is running and seeded
- Inspect `demo/cypress/screenshots/` for failure snapshots
- Adjust timeouts in `waitForAgentResponse()` for slow LLM responses

## File layout

```
demo/
├── cypress/
│   ├── e2e/
│   │   ├── _template.cy.ts              # Base template
│   │   ├── soc2-gap-analysis.cy.ts      # Existing baseline demo
│   │   └── <change-name>-demo.cy.ts    # Generated per change
│   ├── support/
│   │   └── demo-helpers.ts              # Helper reference (inline in specs)
│   ├── videos/                          # .mp4 outputs (gitignored)
│   └── screenshots/                     # Failure screenshots (gitignored)
├── cypress.config.js
├── package.json
├── seed.sh
├── prompts.md
├── policy.json
├── mapping-soc2.json
└── evidence.json
```

## Caption style guide

| Do | Don't |
|----|-------|
| `"Step 2: Filtering evidence by scan date"` | `"Clicking the filter button"` |
| `"CC8.1: Not fully covered · complyctl: Clean ✓"` | `"Result shown"` |
| `"Authoring Gemara #AuditLog — 3 targets, 5 criteria"` | `"Generating..."` |

Captions narrate **what is meaningful**, not what is mechanically happening. Write for a stakeholder watching a 2-minute recording.

## Timing reference

| Constant | Value | Use when |
|----------|-------|----------|
| `LONG` | 1800ms | After agent response, before next step |
| `PAUSE` | 900ms | Before clicking, after navigating |
| `SHORT` | 400ms | After cursor movement |
| `TYPE_DELAY` | 40ms | Between keystrokes for human feel |
| `waitForAgentResponse` timeout | 90000ms | Streaming LLM responses (increase to 120000 for artifact generation) |
