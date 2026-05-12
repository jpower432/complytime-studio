## ADDED Requirements

### Requirement: Deterministic validation node in graph
The LangGraph graph SHALL include a `validate_draft` node that executes CUE schema validation and evidence reference verification as a deterministic Python function (no LLM invocation).

#### Scenario: Draft passes schema and evidence checks
- **WHEN** the `validate_draft` node receives state with `draft_yaml` containing a valid AuditLog and all `evidence_refs` exist in the `evidence` table within the audit window
- **THEN** the node SHALL set `validation_result` to `{"valid": true, "errors": []}`
- **THEN** the graph SHALL route to the `publish_draft` node

#### Scenario: Draft fails CUE schema validation
- **WHEN** the `validate_draft` node receives state with `draft_yaml` that fails `validate_gemara_artifact` with `definition: "#AuditLog"`
- **THEN** the node SHALL set `validation_result` to `{"valid": false, "errors": [<schema errors>]}`
- **THEN** the node SHALL increment `validation_attempts`
- **THEN** the graph SHALL route back to the `agent` node with error context

#### Scenario: Draft references non-existent evidence
- **WHEN** the `validate_draft` node finds `evidence_refs` entries that do not exist in the `evidence` table
- **THEN** the node SHALL append `"Missing evidence refs: [<ids>]"` to `validation_result.errors`
- **THEN** `validation_result.valid` SHALL be `false`

#### Scenario: Evidence exists but outside audit window
- **WHEN** the `validate_draft` node finds evidence records whose `collected_at` falls outside the declared audit window in the draft's `scope`
- **THEN** the node SHALL append `"Evidence outside audit window: [<ids>]"` to `validation_result.errors`
- **THEN** `validation_result.valid` SHALL be `false`

### Requirement: Retry budget enforcement
The graph SHALL enforce a maximum of 3 validation attempts per draft. After 3 failed attempts the graph SHALL route to a `halt` terminal node.

#### Scenario: Third validation attempt fails
- **WHEN** `validation_attempts` equals 3 and `validation_result.valid` is `false`
- **THEN** the graph SHALL route to the `halt` node
- **THEN** the `halt` node SHALL emit a message listing all accumulated errors and stating "Validation failed after 3 attempts. Human intervention required."

#### Scenario: Validation succeeds on retry
- **WHEN** `validation_attempts` is less than 3 and `validation_result.valid` is `true`
- **THEN** the graph SHALL route to `publish_draft` regardless of previous failures

### Requirement: Human approval interrupt before publish
The graph SHALL use `interrupt_before` on the `publish_draft` node to checkpoint state and signal `input-required` via A2A before persisting the draft.

#### Scenario: Validation passes, awaiting human approval
- **WHEN** the graph reaches the `publish_draft` node after successful validation
- **THEN** the graph SHALL interrupt execution and emit an `input-required` task status
- **THEN** the workbench SHALL display the validated draft YAML for human review
- **THEN** the graph SHALL resume only after the human sends a reply via A2A `message/stream`

#### Scenario: Human rejects draft
- **WHEN** the human reply indicates rejection (e.g., "reject", "no", "redo")
- **THEN** the graph SHALL route back to the `agent` node with rejection context
- **THEN** `validation_attempts` SHALL reset to 0

### Requirement: Validation calls MCP directly
The `validate_draft` node SHALL call `validate_gemara_artifact` via the MCP client directly (Streamable HTTP transport), not via LLM tool binding. The LLM SHALL NOT have `validate_gemara_artifact` available as a callable tool in the audit production subgraph.

#### Scenario: MCP server unavailable during validation
- **WHEN** the `validate_draft` node cannot reach `studio-gemara-mcp`
- **THEN** the node SHALL set `validation_result` to `{"valid": false, "errors": ["Gemara MCP unavailable: <error>"]}`
- **THEN** the graph SHALL route to `halt` (do not retry on infrastructure failure)
