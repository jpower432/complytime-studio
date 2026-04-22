### Requirement: Server-side conversation storage
The gateway SHALL store conversation turns per-user in memory, keyed by the authenticated user's email. Each user SHALL have at most one active conversation session.

#### Scenario: Turn stored after agent response
- **WHEN** the agent completes a response for user `alice@acme.com`
- **THEN** the client SHALL PUT the full message array and current taskId to `/api/chat/history`
- **THEN** the gateway SHALL store the state keyed by `alice@acme.com`

#### Scenario: Concurrent users
- **WHEN** `alice@acme.com` and `bob@acme.com` both have active conversations
- **THEN** each user's conversation state SHALL be independent

### Requirement: Load conversation on mount
The chat panel SHALL load the user's conversation state from the server on component mount. If a prior session exists, messages and taskId SHALL be restored.

#### Scenario: Page refresh with existing session
- **WHEN** a user refreshes the page and a server-side conversation exists
- **THEN** the chat panel SHALL display the prior messages
- **THEN** `taskIdRef` SHALL be restored so `streamReply` resumes the A2A task

#### Scenario: Page load with no prior session
- **WHEN** a user loads the page and no server-side conversation exists
- **THEN** the chat panel SHALL display an empty message list
- **THEN** the next message SHALL use `streamMessage` (new task)

### Requirement: GET /api/chat/history endpoint
The gateway SHALL expose `GET /api/chat/history` which returns the authenticated user's conversation state.

#### Scenario: User has conversation state
- **WHEN** an authenticated user requests `GET /api/chat/history` and stored messages exist
- **THEN** the gateway SHALL return `{"messages": [...], "taskId": "<id>"}` with HTTP 200

#### Scenario: User has no conversation state
- **WHEN** an authenticated user requests `GET /api/chat/history` and no stored state exists
- **THEN** the gateway SHALL return `{"messages": [], "taskId": null}` with HTTP 200

#### Scenario: Unauthenticated request
- **WHEN** a request to `GET /api/chat/history` has no valid session
- **THEN** the gateway SHALL return HTTP 401

### Requirement: PUT /api/chat/history endpoint
The gateway SHALL expose `PUT /api/chat/history` which replaces the authenticated user's conversation state.

#### Scenario: Save conversation state
- **WHEN** an authenticated user sends `PUT /api/chat/history` with `{"messages": [...], "taskId": "<id>"}`
- **THEN** the gateway SHALL store the state, replacing any prior state for that user
- **THEN** the gateway SHALL return HTTP 204

#### Scenario: Clear conversation state
- **WHEN** an authenticated user sends `PUT /api/chat/history` with `{"messages": [], "taskId": null}`
- **THEN** the gateway SHALL clear the stored state for that user
- **THEN** the gateway SHALL return HTTP 204

### Requirement: Conversation TTL matches auth session
Stored conversation state SHALL expire after 8 hours (matching `sessionMaxAge`). Expired state SHALL be treated as nonexistent.

#### Scenario: State expires after TTL
- **WHEN** a user's conversation state was last updated more than 8 hours ago
- **THEN** `GET /api/chat/history` SHALL return `{"messages": [], "taskId": null}`

### Requirement: New Session clears server state
Clicking "New Session" SHALL clear the server-side conversation state in addition to resetting the client.

#### Scenario: User clicks New Session
- **WHEN** the user clicks "New Session"
- **THEN** the client SHALL send `PUT /api/chat/history` with empty messages and null taskId
- **THEN** the message list SHALL be cleared
- **THEN** `taskIdRef` SHALL be set to null

