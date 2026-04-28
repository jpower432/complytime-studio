## ADDED Requirements

### Requirement: Risk severity traversal via threat graph
The `evidence-schema` skill SHALL document a multi-hop query pattern: `evidence` → `control_threats` → `risk_threats` → `risks`. This enables "what is the risk severity for this failed control?" queries.

#### Scenario: Risk severity for a specific control
- **WHEN** the assistant executes `SELECT DISTINCT r.risk_id, r.title, r.severity FROM control_threats ct JOIN risk_threats rt ON ct.threat_entry_id = rt.threat_entry_id JOIN risks r ON r.risk_id = rt.risk_id AND r.catalog_id = rt.catalog_id WHERE ct.control_id = ?`
- **THEN** the result SHALL include all risks linked to the control through their common threats, with severity

### Requirement: Unmitigated risk identification query pattern
The `evidence-schema` skill SHALL document a query pattern that identifies risks with no corresponding controls (risks linked to threats that no control addresses).

#### Scenario: Find risks with no mitigating controls
- **WHEN** the assistant executes `SELECT r.risk_id, r.title, r.severity FROM risk_threats rt JOIN risks r ON r.risk_id = rt.risk_id AND r.catalog_id = rt.catalog_id LEFT JOIN control_threats ct ON rt.threat_entry_id = ct.threat_entry_id WHERE ct.control_id IS NULL`
- **THEN** the result SHALL include risks whose linked threats have no corresponding control
