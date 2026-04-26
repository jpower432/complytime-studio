## MODIFIED Requirements

### Requirement: Agent validates AuditLog via MCP tool
The agent prompt SHALL instruct the assistant to call the `validate_gemara_artifact` MCP tool with `definition: "#AuditLog"` on every generated AuditLog YAML block before publishing it. The agent SHALL fix validation errors and re-validate up to 3 times.

#### Scenario: Agent produces valid AuditLog
- **WHEN** the agent generates an AuditLog YAML artifact
- **THEN** the agent SHALL call `validate_gemara_artifact` before calling `publish_audit_log`

#### Scenario: Validation fails on first attempt
- **WHEN** `validate_gemara_artifact` returns errors
- **THEN** the agent SHALL fix the identified issues and re-validate, up to 3 total attempts

#### Scenario: Validation fails after 3 attempts
- **WHEN** the agent cannot produce valid YAML after 3 validation attempts
- **THEN** the agent SHALL report the validation errors to the user and halt

## REMOVED Requirements

### Requirement: Agent saves AuditLog artifact via callback
**Reason**: The `after_agent` callback lacked `ToolContext` and could never call `save_artifact`. Artifact emission now happens via the `publish_audit_log` function tool, which receives `ToolContext` from ADK.
**Migration**: `publish_audit_log` tool replaces the callback-based approach entirely.
