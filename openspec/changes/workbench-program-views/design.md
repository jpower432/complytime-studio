# Design: Workbench Program Views

## Navigation

Add to sidebar (`sidebar.tsx`):

| Section | Icon | Route | Existing? |
|:--|:--|:--|:--|
| Dashboard | grid | `/` | Exists (enhance) |
| Programs | folder | `/programs` | **New** |
| Policies | shield | `/policies` | Exists |
| Evidence | document | `/evidence` | Exists |
| Inbox | bell | `/inbox` | Exists |
| Audit History | clock | `/audit-history` | Exists |

Program detail is a sub-route: `/programs/{id}`.

## View 1: Programs List (`programs-view.tsx`)

### Layout

- Toolbar: search input + status filter dropdown + "New Program" button (admin only, disabled with tooltip for non-admins)
- Gallery grid of program cards (responsive: 1-3 columns)
- Empty state when no programs exist (prompt to create first program)
- Error state when PostgreSQL is unavailable (clear messaging, not blank page)

### Program Card

| Element | Source |
|:--|:--|
| Program name | `program.name` |
| Framework badge | `program.framework` (color-coded) |
| Status badge | `program.status` (`intake`, `active`, `monitoring`, `renewal`, `closed`) |
| Health indicator | `program.health` (green/yellow/red dot + text label, not color-only) |
| Owner | `program.owner` |
| Policy count | `program.policy_ids.length` |
| Last updated | `program.updated_at` relative time |

Click card → navigate to `/programs/{id}`.

### API

- `GET /api/programs` → `{ items: Program[], total: number }`
- `POST /api/programs` → create (admin only)
- `DELETE /api/programs/{id}` → soft delete (admin only)

## View 1a: Create/Edit Program (`program-form.tsx`)

Modal dialog triggered from "New Program" button or "Edit" button on detail view.

### Fields

| Field | Type | Required | Notes |
|:--|:--|:--|:--|
| Name | text | yes | |
| Framework | select | yes | Populated from known frameworks |
| Owner | text | no | |
| Description | textarea | no | |
| Policies | multi-select | no | Select from existing policies in ClickHouse |

Defaults populated where possible. Submit → `POST /api/programs` or `PUT /api/programs/{id}`.

## View 2: Program Detail (`program-detail-view.tsx`)

### Layout

- Breadcrumb: Programs > {program name}
- Header: name, framework badge, status badge, health, owner, "Edit" button
- Tabs: Overview | Evidence | Commands | Chat

### Tab: Overview

- Description list: framework, status, health, owner, created, updated
- Policy links: list of `policy_ids` linking to existing policy detail view
- Recent runs: last 5 runs with command name, status, duration, timestamp

### Tab: Evidence

- Reuse existing `evidence-view.tsx` component via thin wrapper (`ProgramEvidenceTab`)
- Pass `programId` as props → gateway resolves policy_ids → ClickHouse evidence query
- Stale evidence highlighted (existing posture logic)
- Degraded state messaging when ClickHouse is unavailable

### Tab: Commands

- Command bar (see View 3 below) scoped to this program
- Command output panel below the bar

### Tab: Chat

- Reuse existing `chat-assistant.tsx` component
- Context injection: program ID passed to A2A request so the agent has program context
- Agent picker: if sub-agents are registered, show dropdown to select which agent

### API

- `GET /api/programs/{id}` → single program
- `PUT /api/programs/{id}` → update (admin only, optimistic lock via `version`)
- `GET /api/runs?program_id={id}` → runs for this program

## View 3: Command Bar (`command-bar.tsx`)

### Layout

- Expandable sections grouped by command category
- Each command: name, description, argument form
- Execute button → SSE stream → output panel

### Command Source

Hybrid model: agent card `skills` field is source of truth for what's invokable. Gateway `GET /api/commands` (from ConfigMap) provides presentation metadata (categories, labels, argument schemas). CI check validates alignment between deployed agents and command catalog.

### Command Groups

| Group | Commands |
|:--|:--|
| Daily Operations | `daily-brief`, `program-status`, `due-this-week` |
| Evidence | `evidence-due`, `update-item` |
| Decisions | `log-decision`, `meeting-debrief`, `meeting-prep` |
| Communications | `draft-status-update` |
| Assessment | `control-assessment`, `auditor-view` |

### Command Execution Flow

1. User selects command, fills arguments (program select, text fields)
2. Workbench sends `POST /api/a2a/{agent-id}` with A2A task message containing command name + arguments
3. Agent pod receives, loads command spec, executes via LangGraph, streams SSE
4. Gateway proxies SSE back to workbench
5. Workbench renders streaming tokens in output panel with indeterminate progress on TTFB
6. On completion: quality gate results displayed as pass/fail badge

### UX for streaming

- Indeterminate progress indicator until first SSE token arrives
- Execute button disabled while stream is active (prevent duplicate submissions)
- On stream failure: "Command failed — Retry" button with run ID if available
- Cancel button sends abort signal to client reader
- Large outputs: truncate at 50k characters with "Download full output" link

### Command Output Panel (`command-output.tsx`)

- Streaming text with markdown rendering
- Tool use indicators (wrench icon + tool name) when agent calls MCP tools
- Quality gate result badge at end of output
- Expandable issues list when a gate fails
- Copy / download output buttons

### API

- Commands executed via A2A proxy, not a dedicated REST endpoint
- `GET /api/commands` → command metadata from ConfigMap (categories, labels, arg schemas)

## View 4: Dashboard Enhancement (`posture-view.tsx` modification)

Add to existing posture/dashboard view:

### Program Health Cards

- Row of cards: one per program (max 6 visible, scrollable)
- Each card: name, framework, health dot + text label, evidence coverage %, next deadline
- Click → navigate to `/programs/{id}`

Portfolio metrics (programs by status, cross-program stale evidence, upcoming deadlines) are **deferred** until the coordinator agent is validated in production.

### API

- `GET /api/programs` → program list for cards
- `GET /api/posture` → existing posture data
- Evidence coverage derived from posture data filtered by program's `policy_ids`

## State Management

Use existing Preact signals pattern:

```typescript
// store/programs.ts
const programs = signal<Program[]>([]);
const selectedProgram = signal<Program | null>(null);
const programRuns = signal<Run[]>([]);
```

API client functions in `api/` directory following existing patterns (`api/programs.ts`).

## Component Reuse

| Existing component | Reused in |
|:--|:--|
| `chat-assistant.tsx` | Program detail Chat tab |
| `evidence-view.tsx` | Program detail Evidence tab (thin wrapper with filter) |
| `filter-chip.tsx` | Programs list status filter |
| `header.tsx` | Unchanged |
| `sidebar.tsx` | Add Programs nav item |

## Accessibility

- `aria-live="polite"` on streaming output region
- Focus management: after "Execute," announce and move focus to output panel
- Health indicators: color + text label (not color-only)
- Keyboard navigation through expandable command sections
- Permission affordances: disabled buttons show tooltip explaining why
