## ADDED Requirements

### Requirement: Delegate node invokes BYO agent via A2A
The graph SHALL include a `delegate` node that sends a structured A2A request to a registered BYO agent and writes the response into `worker_data` State.

#### Scenario: Successful delegation
- **WHEN** the graph routes to the `delegate` node
- **THEN** the node SHALL resolve the target agent's URL from `GET {GATEWAY_URL}/api/agents`
- **THEN** the node SHALL POST a JSON-RPC `message/send` request to the agent's A2A endpoint via AgentGateway
- **THEN** the node SHALL write the response into `state.worker_data` keyed by agent ID
- **THEN** the graph SHALL route to the next node (agent LLM) with worker data available in State

#### Scenario: Agent not found in directory
- **WHEN** the `delegate` node cannot find the target agent in the directory
- **THEN** the node SHALL write `{"error": "Agent '<id>' not found in directory"}` to `state.worker_data`
- **THEN** the graph SHALL route to the agent node (LLM reports issue to user)

#### Scenario: Agent unreachable
- **WHEN** the HTTP request to the BYO agent fails (timeout, connection refused, 5xx)
- **THEN** the node SHALL write `{"error": "Agent '<id>' unavailable: <detail>"}` to `state.worker_data`
- **THEN** the graph SHALL route to the agent node (LLM reports issue to user)

#### Scenario: Agent returns failure status
- **WHEN** the BYO agent returns a Task with `status.state: "failed"`
- **THEN** the node SHALL write `{"error": "Agent '<id>' reported failure: <message>"}` to `state.worker_data`

### Requirement: Conditional edge triggers delegation
The graph SHALL include a conditional edge that routes to the `delegate` node when the workflow requires domain-specific data from the BYO agent. The condition SHALL be deterministic, based on policy metadata or a `needs_delegation` flag in State.

#### Scenario: Policy requires BYO agent data
- **WHEN** the policy metadata indicates it requires domain-specific data (e.g., a tag, a field, or a known policy_id pattern)
- **THEN** the graph SHALL route through the `delegate` node before the classification phase

#### Scenario: Policy does not require BYO data
- **WHEN** the policy metadata does not indicate a need for domain-specific data
- **THEN** the graph SHALL skip the `delegate` node and proceed directly to classification

#### Scenario: Agent node sets needs_delegation flag
- **WHEN** the LLM determines during evidence assembly that BYO data is needed and sets `state.needs_delegation = true`
- **THEN** the conditional edge SHALL route to the `delegate` node on the next transition

### Requirement: Delegate node sets identity header
The `delegate` node SHALL set `X-Agent-ID: studio-assistant` on all A2A requests.

#### Scenario: Header present on delegation call
- **WHEN** the delegate node sends an A2A request
- **THEN** the HTTP request SHALL include `X-Agent-ID: studio-assistant`

### Requirement: Delegate node enforces timeout and size limits
The `delegate` node SHALL enforce a 30-second timeout and 1MB response size cap.

#### Scenario: Agent exceeds timeout
- **WHEN** the BYO agent does not respond within 30 seconds
- **THEN** the node SHALL abort the request and write `{"error": "Agent '<id>' timed out after 30s"}` to `state.worker_data`

#### Scenario: Agent returns oversized response
- **WHEN** the BYO agent's response body exceeds 1MB
- **THEN** the node SHALL truncate the response and append `[TRUNCATED — response exceeded 1MB]`

### Requirement: Worker data persisted in State
The `delegate` node SHALL write responses to a `worker_data` field in State (type: `dict`). This field SHALL be checkpointed and survive message window truncation.

#### Scenario: Worker data available after graph resume
- **WHEN** the graph resumes from an interrupt after delegation occurred
- **THEN** `state.worker_data` SHALL contain the BYO agent's response from the prior step

#### Scenario: Worker data available to LLM
- **WHEN** the agent node runs after the delegate node
- **THEN** the system prompt or injected context SHALL include `state.worker_data` content so the LLM can reference it
