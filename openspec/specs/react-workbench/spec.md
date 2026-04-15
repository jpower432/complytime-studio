## ADDED Requirements

### Requirement: React SPA replaces vanilla JS workbench
The workbench SHALL be a React single-page application built to `workbench/dist/` and embedded in the gateway binary via `go:embed`.

#### Scenario: SPA build and embed
- **WHEN** the React app is built (`npm run build` or equivalent)
- **THEN** static assets are output to `workbench/dist/`
- **THEN** the gateway embeds and serves them at `/` with SPA fallback routing

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
