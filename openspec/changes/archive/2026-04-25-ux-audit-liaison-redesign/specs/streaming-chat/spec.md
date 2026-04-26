## MODIFIED Requirements

### Requirement: Chat pre-loads active policy context
The chat assistant SHALL pre-load the active policy context when the user opens the chat from a policy detail view. The first message SHALL include the policy ID, active tab, and any selected filters as context for the agent.

#### Scenario: Chat opened from posture detail
- **WHEN** the user opens the chat FAB while viewing policy "ampel-branch-protection" on the Requirements tab
- **THEN** the chat sends the first user message with context: `{"policy_id":"ampel-branch-protection","view":"requirements","filters":{...}}`

#### Scenario: Chat opened from posture grid
- **WHEN** the user opens the chat FAB from the top-level posture view (no policy selected)
- **THEN** the chat sends no pre-loaded context and the agent asks which policy to work with
