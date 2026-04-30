# Tasks: Workbench Program Views

## Programs List
- [ ] Add Programs route to sidebar navigation
- [ ] Create `programs-view.tsx` with gallery grid, search, status filter
- [ ] Create program card component with badges, health indicator (color + text)
- [ ] Add "New Program" button with admin-only visibility + tooltip for non-admins
- [ ] Add empty state and error state (PostgreSQL unavailable)
- [ ] Add `api/programs.ts` API client functions

## Create/Edit Program
- [ ] Create `program-form.tsx` modal dialog
- [ ] Wire POST /api/programs (create) and PUT /api/programs/{id} (edit)
- [ ] Policy multi-select populated from existing ClickHouse policies

## Program Detail
- [ ] Create `program-detail-view.tsx` with breadcrumb, header, tabs
- [ ] Overview tab: description list, policy links, recent runs
- [ ] Evidence tab: thin wrapper around `evidence-view.tsx` with program filter
- [ ] Commands tab: command bar + output panel
- [ ] Chat tab: reuse `chat-assistant.tsx` with program context + agent picker
- [ ] Add degraded-state messaging when ClickHouse unavailable in Evidence tab

## Command Bar
- [ ] Create `command-bar.tsx` with expandable category sections
- [ ] Create `command-output.tsx` with streaming markdown, tool indicators, quality gate badge
- [ ] Add `GET /api/commands` client call for command metadata
- [ ] Wire command execution via A2A proxy with SSE streaming
- [ ] Add progress indicator, disable button during execution, cancel support
- [ ] Add stream failure handling with retry button
- [ ] Add output truncation with download link for large outputs

## Dashboard Enhancement
- [ ] Add program health cards row to posture view
- [ ] Wire cards from `GET /api/programs`

## Accessibility
- [ ] aria-live on streaming output region
- [ ] Focus management after command execution
- [ ] Keyboard navigation through command sections
- [ ] Non-color health indicators (color + text label)
