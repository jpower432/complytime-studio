## MODIFIED Requirements

### Requirement: Agent reads schema before authoring AuditLog
The agent prompt SHALL instruct the assistant to read the `gemara://schema/definitions` MCP resource to obtain the `#AuditLog` definition BEFORE authoring any AuditLog artifact. This ensures the agent uses the correct field structure.

#### Scenario: Agent prepares to author AuditLog
- **WHEN** the agent reaches the "Author AuditLog" workflow step
- **THEN** the agent SHALL first read `gemara://schema/definitions` to obtain the `#AuditLog` schema definition

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
