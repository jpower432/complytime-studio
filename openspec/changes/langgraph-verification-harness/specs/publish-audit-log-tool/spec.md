## MODIFIED Requirements

### Requirement: Agent publishes draft AuditLog via custom tool
The agent SHALL have a `publish_audit_log` function that persists a validated AuditLog YAML as a **draft** via the internal Gateway endpoint. The function SHALL only be reachable via the `publish_draft` graph node (after the validation gate passes). It SHALL NOT be available as a direct LLM-callable tool.

#### Scenario: Graph reaches publish_draft after validation gate
- **WHEN** the `validate_draft` node sets `validation_result.valid` to `true`
- **AND** the human approves at the interrupt gate
- **THEN** the `publish_draft` node SHALL call the publish function with `draft_yaml` from state
- **THEN** the tool SHALL POST to `/internal/draft-audit-logs` with `policy_id`, `content`, `agent_reasoning`, `model`, and `prompt_version`

#### Scenario: LLM attempts to call publish_audit_log directly
- **WHEN** the LLM emits a `tool_calls` message targeting `publish_audit_log`
- **THEN** the tool SHALL NOT be in the LLM's bound tools list
- **THEN** the LLM SHALL receive an error indicating the tool is not available (standard LangGraph behavior for unbound tools)

#### Scenario: Agent calls publish_audit_log with invalid YAML
- **WHEN** the `publish_draft` node receives state where `draft_yaml` is unparseable YAML
- **THEN** the function SHALL return `{"error": "Invalid YAML: ..."}` without persisting
- **THEN** the graph SHALL route to `halt` (this indicates a graph logic error since validation should have caught it)

#### Scenario: Agent calls publish_audit_log with non-AuditLog type
- **WHEN** the `publish_draft` node receives state where `draft_yaml` has `metadata.type` not equal to `"AuditLog"`
- **THEN** the function SHALL return `{"error": "Expected metadata.type=AuditLog, got '...'"}`
