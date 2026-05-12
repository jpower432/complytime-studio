## MODIFIED Requirements

### Requirement: Job state machine
The system SHALL manage jobs through a defined state machine: `submitted → working → input-required → ready → accepted`. The `input-required` state SHALL be entered when the graph's validation gate passes and the `interrupt_before` on `publish_draft` fires. The `cancelled` state SHALL be reachable from `submitted`, `working`, `input-required`, and `ready`. The `failed` state SHALL be set when the agent reports failure or the halt node is reached.

#### Scenario: Validation gate passes — job enters input-required
- **WHEN** the `validate_draft` node sets `validation_result.valid` to `true`
- **AND** the graph reaches the `interrupt_before` on `publish_draft`
- **THEN** the A2A stream SHALL emit a `TaskStatusUpdateEvent` with `state: "input-required"`
- **THEN** the workbench SHALL display the validated draft YAML and enable the reply input with "Approve" / "Reject" options

#### Scenario: Human approves at publish gate
- **WHEN** the user sends a reply indicating approval (e.g., "approve", "yes", "publish")
- **THEN** the graph SHALL resume from the `publish_draft` node
- **THEN** the job status SHALL transition to `working` during publish, then `ready` on completion

#### Scenario: Human rejects at publish gate
- **WHEN** the user sends a reply indicating rejection (e.g., "reject", "no", "redo")
- **THEN** the graph SHALL resume and route back to the `agent` node with rejection context
- **THEN** the job status SHALL transition back to `working`

#### Scenario: Halt node reached after retry exhaustion
- **WHEN** the graph routes to the `halt` node (3 validation failures or infrastructure error)
- **THEN** the A2A stream SHALL emit a `TaskStatusUpdateEvent` with `state: "failed"`
- **THEN** the job status SHALL transition to `failed` with error details from `validation_result.errors`
