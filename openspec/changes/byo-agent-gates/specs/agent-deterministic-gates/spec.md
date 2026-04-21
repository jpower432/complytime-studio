## ADDED Requirements

### Requirement: before_agent validates input

The agent SHALL run `before_agent_callback` to inspect the inbound user
message. Missing policy reference SHALL be logged as a warning without blocking
the run. Empty user messages SHALL be handled by returning `None` from the
callback so the agent stack applies its default behavior.

#### Scenario: User message lacks policy reference
- **WHEN** the user message does not contain an identifiable policy reference
- **THEN** the callback SHALL log a warning
- **THEN** the callback SHALL NOT block agent execution

#### Scenario: User message is empty
- **WHEN** the user message is empty
- **THEN** the callback SHALL return `None`
- **THEN** the agent runtime SHALL handle the empty input per ADK defaults

### Requirement: after_agent extracts AuditLog

The agent SHALL run `after_agent_callback` that scans agent output for YAML
bearing AuditLog markers. When such YAML is present, the callback SHALL call
`save_artifact` with MIME type `application/yaml`. When no qualifying YAML
exists, the callback SHALL return `None`.

#### Scenario: Output contains AuditLog YAML
- **WHEN** the agent output contains YAML with AuditLog markers
- **THEN** the callback SHALL call `save_artifact` with `application/yaml`

#### Scenario: Output has no YAML
- **WHEN** the agent output contains no YAML segment matching AuditLog markers
- **THEN** the callback SHALL return `None`

### Requirement: before_tool guards SQL

The agent SHALL run `before_tool_callback` that intercepts ClickHouse
`run_select_query` calls. Queries whose arguments match a deny-list of
DDL/DML keywords (`INSERT`, `UPDATE`, `DELETE`, `DROP`, `ALTER`, `CREATE`,
`TRUNCATE`, `GRANT`, `REVOKE`) SHALL be rejected with an error dict. All other
tools SHALL pass through with the callback returning `None`.

#### Scenario: run_select_query contains forbidden keywords
- **WHEN** `run_select_query` is invoked with arguments containing DDL/DML
  keywords (`INSERT`, `UPDATE`, `DELETE`, `DROP`, `ALTER`, `CREATE`,
  `TRUNCATE`, `GRANT`, `REVOKE`)
- **THEN** the callback SHALL return an error dict rejecting the query

#### Scenario: Non-ClickHouse tool invocation
- **WHEN** the tool name is not `run_select_query`
- **THEN** the callback SHALL return `None` (pass-through)
