## MODIFIED Requirements

### Requirement: Platform prompt for single agent
The platform prompt SHALL be simplified to address a single gap analyst agent. References to multi-agent selection, agent naming via `{{.AgentName}}`, and cross-agent coordination SHALL be removed.

#### Scenario: Agent receives platform prompt
- **WHEN** the BYO gap analyst starts a conversation
- **THEN** the system prompt includes platform identity, constraints (domain knowledge from MCP, no fabrication), and output format (fenced YAML for AuditLogs)
