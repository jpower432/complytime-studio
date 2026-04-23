## MODIFIED Requirements

### Requirement: Agent validates AuditLog via MCP tool
The agent prompt SHALL instruct the assistant to call the `validate_gemara_artifact` MCP tool with `definition: "#AuditLog"` on every generated AuditLog YAML block before returning it to the user. The agent SHALL fix validation errors and re-validate up to 3 times.

#### Scenario: Agent produces valid AuditLog
- **WHEN** the agent generates an AuditLog YAML artifact
- **THEN** the agent SHALL call `validate_gemara_artifact` with the YAML content and `definition: "#AuditLog"` before returning it

#### Scenario: Validation fails on first attempt
- **WHEN** `validate_gemara_artifact` returns errors
- **THEN** the agent SHALL fix the identified issues and re-validate, up to 3 total attempts

#### Scenario: Validation fails after 3 attempts
- **WHEN** the agent cannot produce valid YAML after 3 validation attempts
- **THEN** the agent SHALL report the validation errors to the user and halt

## REMOVED Requirements

### Requirement: Agent saves AuditLog artifact via callback
**Reason**: The `after_agent` callback's `save_artifact` call never successfully fires in the current ADK version. AuditLog detection and persistence is moving to the frontend (inline-artifact-detection). The callback dead code is removed to avoid confusion.
**Migration**: AuditLog persistence is handled by the frontend "Save to Audit History" button. No agent-side artifact emission is needed.
