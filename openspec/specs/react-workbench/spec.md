## Requirements

### Requirement: React SPA replaces vanilla JS workbench
The workbench SHALL be a React single-page application built to `workbench/dist/` and embedded in the gateway binary via `go:embed`. All view components SHALL use semantic HTML elements (`<section>`, `<article>`, `<table>`, `<header>`) and include `data-*` attributes as defined in the agent DOM contract.

#### Scenario: SPA build and embed
- **WHEN** the React app is built (`npm run build` or equivalent)
- **THEN** static assets are output to `workbench/dist/`
- **THEN** the gateway embeds and serves them at `/` with SPA fallback routing

#### Scenario: Semantic structure in all views
- **WHEN** any view component renders
- **THEN** the root element is a `<section>` with an `<h2>` heading
- **THEN** tabular data uses `<table>` with `<thead>` and `<th>` headers
- **THEN** card groups use `<article>` elements

### Requirement: Dashboard displays agent cards
The workbench SHALL display a dashboard view showing one card per available agent, sourced from `GET /api/agents`.

#### Scenario: Agent card rendering
- **WHEN** the dashboard loads
- **THEN** it fetches `GET /api/agents` and renders a card for each agent
- **THEN** each card shows the agent name, description, and skill tags
- **THEN** each card has a button to open a chat session with that agent

### Requirement: Chat view supports A2A streaming
The workbench SHALL provide a chat interface per agent that sends messages via `POST /api/a2a/{agent-name}` and renders streamed responses in real time.

#### Scenario: Streaming conversation
- **WHEN** the user sends a message in the chat view
- **THEN** the frontend sends an A2A SendStreamingMessage request to the gateway
- **THEN** the response stream is rendered incrementally as the agent produces output
- **THEN** artifact YAML blocks in agent responses are syntax-highlighted

### Requirement: Artifact editor with live validation
The workbench SHALL provide a YAML editor (Monaco or CodeMirror) that validates content against Gemara schemas via `POST /api/validate` on change.

#### Scenario: Live validation feedback
- **WHEN** the user edits YAML in the artifact editor
- **THEN** the editor debounces and sends content to `POST /api/validate`
- **THEN** validation errors are displayed inline in the editor

### Requirement: Publish panel
The workbench SHALL provide a publish panel where the user selects artifacts from the workspace, sets a registry target and tag, and triggers `POST /api/publish`.

#### Scenario: Publish workflow
- **WHEN** the user selects artifacts from the workspace and clicks Publish
- **THEN** the frontend sends `POST /api/publish` with the artifact YAML, target, tag, and optional sign flag
- **THEN** the result (reference, digest, tag) is displayed to the user

### Requirement: Browser-side workspace state
All workspace state (artifacts, chat history, editor content) SHALL be stored in browser localStorage or IndexedDB. The gateway does not persist workspace state.

#### Scenario: State persistence
- **WHEN** the user saves an artifact to the workspace
- **THEN** it is stored in browser storage
- **WHEN** the user refreshes the page
- **THEN** workspace state is restored from browser storage

### Requirement: Authentication-aware UI
The workbench SHALL check authentication state on load and redirect to `/auth/login` if the user is not authenticated.

#### Scenario: Unauthenticated access
- **WHEN** an unauthenticated user loads the workbench
- **THEN** the frontend calls `GET /auth/me`
- **THEN** on HTTP 401, the frontend redirects to `/auth/login`

#### Scenario: Authenticated user display
- **WHEN** an authenticated user loads the workbench
- **THEN** the user's GitHub avatar and login name are displayed in the UI header

### Requirement: All policy selectors write selectedPolicyId on search
Every view with a policy dropdown SHALL write `selectedPolicyId` when the user triggers a search or applies filters.

#### Scenario: Evidence view sets policy signal
- **WHEN** the user selects a policy in evidence view and clicks Search
- **THEN** `selectedPolicyId` is updated to the selected value

#### Scenario: Audit history sets policy signal
- **WHEN** the user selects a policy in audit history and clicks Search
- **THEN** `selectedPolicyId` is updated to the selected value

### Requirement: Date range filters write selectedTimeRange
Views with start/end date inputs SHALL write `selectedTimeRange` when the user triggers a search.

#### Scenario: Audit history sets time range
- **WHEN** the user sets start and end dates in audit history and clicks Search
- **THEN** `selectedTimeRange` is updated with `{ start, end }`

#### Scenario: Requirement matrix sets time range
- **WHEN** the user sets audit start/end in requirement matrix and clicks Search
- **THEN** `selectedTimeRange` is updated with `{ start, end }`

### Requirement: Control family filter writes selectedControlId
The requirement matrix SHALL write `selectedControlId` when a control family filter is applied.

#### Scenario: Filter by control family
- **WHEN** the user selects control family "AC" in the requirement matrix and clicks Search
- **THEN** `selectedControlId` is updated to "AC"

#### Scenario: Clear control family filter
- **WHEN** the user selects "All control families" and clicks Search
- **THEN** `selectedControlId` is set to null

### Requirement: Views pre-fill filters from shared signals on mount
Each view SHALL read shared signals on mount and pre-fill local filter state if the signal is non-null.

#### Scenario: Navigate from posture to evidence
- **WHEN** the user clicks a posture card (setting `selectedPolicyId`) then navigates to evidence
- **THEN** the evidence view policy dropdown is pre-filled with the selected policy

#### Scenario: Navigate from audit history to requirements
- **WHEN** the user searches audit history with dates (setting `selectedTimeRange`) then navigates to requirements
- **THEN** the requirement matrix date inputs are pre-filled

### Requirement: Requirement matrix refetches on viewInvalidation
The requirement matrix SHALL refetch data when `viewInvalidation` changes, provided a policy is selected.

#### Scenario: Agent produces artifact
- **WHEN** the agent produces an AuditLog artifact that triggers `invalidateViews()`
- **THEN** the requirement matrix refetches if a policy is currently selected

### Requirement: Draft Review auto-saves reviewer edits
The Draft Review UI SHALL auto-save reviewer edits (type overrides and notes) to the server via `PATCH /api/draft-audit-logs/{id}` with a 1-second debounce after each change. A "Saving..." / "Saved" indicator SHALL be displayed.

#### Scenario: Type override triggers auto-save
- **WHEN** the reviewer changes a result type from "Finding" to "Strength"
- **THEN** the UI debounces for 1 second and sends a PATCH with the updated `reviewer_edits`
- **THEN** a "Saved" indicator appears after successful save

#### Scenario: Note input triggers auto-save
- **WHEN** the reviewer types a note on a result
- **THEN** the UI debounces for 1 second and sends a PATCH with the updated `reviewer_edits`

### Requirement: Draft Review loads persisted edits on open
When the reviewer opens a draft detail, the UI SHALL read `reviewer_edits` from the GET response and pre-fill type overrides and notes for each result card.

#### Scenario: Reopen draft with saved edits
- **WHEN** the reviewer navigates away and returns to a draft with saved edits
- **THEN** the type override dropdowns and notes reflect the previously saved values

#### Scenario: Open draft with no edits
- **WHEN** the reviewer opens a draft that has no reviewer edits
- **THEN** all result cards show the original agent classification with empty notes

### Requirement: Save indicator shows auto-save state
The Draft Review detail panel SHALL display a save indicator with three states: idle (hidden), saving ("Saving..."), saved ("Saved").

#### Scenario: Save lifecycle
- **WHEN** an edit triggers auto-save
- **THEN** the indicator shows "Saving..." during the PATCH request
- **THEN** the indicator shows "Saved" after a successful response
- **THEN** the indicator fades after 2 seconds

### Requirement: Sidebar navigation items
The sidebar SHALL display the following navigation items in order: Posture, Policies, Evidence, Inbox (with unread badge). The sidebar SHALL NOT include standalone "Audit History" or "Review" items.

#### Scenario: Sidebar shows four items
- **WHEN** the workbench renders
- **THEN** the sidebar displays exactly four nav items: Posture, Policies, Evidence, Inbox

#### Scenario: Inbox badge shows unread count
- **WHEN** the inbox has 3 unread notifications
- **THEN** the Inbox nav item displays a badge with "3"

### Requirement: View routing supports nested paths
The router SHALL support nested hash paths for policy detail drill-down: `#/posture/{policy_id}?tab=requirements|evidence|history`. The `View` type SHALL include `"posture-detail"` as a valid view.

#### Scenario: Nested posture route renders policy detail
- **WHEN** the URL hash is `#/posture/ampel-branch-protection?tab=requirements`
- **THEN** the app renders `PolicyDetailView` with the Requirements tab active

#### Scenario: Legacy audit-history route redirects
- **WHEN** the URL hash is `#/audit-history`
- **THEN** the app redirects to `#/posture`
