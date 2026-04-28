## ADDED Requirements

### Requirement: Risk severity traversal query pattern
The `evidence-schema` skill SHALL document a query pattern that derives risk severity from evidence via the join path: `evidence` → `control_threats` → `risk_threats` → `risks`. The join key between `control_threats` and `risk_threats` SHALL be `threat_entry_id`.

#### Scenario: Query risk severity for failed controls
- **WHEN** the assistant executes `SELECT e.control_id, e.eval_result, r.risk_id, r.title AS risk_title, r.severity FROM evidence e JOIN control_threats ct ON e.control_id = ct.control_id JOIN risk_threats rt ON ct.threat_entry_id = rt.threat_entry_id JOIN risks r ON r.risk_id = rt.risk_id AND r.catalog_id = rt.catalog_id WHERE e.policy_id = ? AND e.eval_result = 'Failed'`
- **THEN** the result SHALL include the risk severity for each failed evidence row, derived through the threat graph

#### Scenario: Control with no linked risks
- **WHEN** a control has threats in `control_threats` but no matching `risk_threats` rows exist
- **THEN** the JOIN SHALL produce no risk rows for that control (LEFT JOIN would show NULL severity)

### Requirement: Risk exposure summary query pattern
The `evidence-schema` skill SHALL document a query pattern that summarizes risk exposure by severity across all evidence for a policy.

#### Scenario: Aggregate risk exposure
- **WHEN** the assistant executes `SELECT r.severity, count(DISTINCT r.risk_id) AS risk_count, count(DISTINCT e.control_id) AS affected_controls FROM evidence e JOIN control_threats ct ON e.control_id = ct.control_id JOIN risk_threats rt ON ct.threat_entry_id = rt.threat_entry_id JOIN risks r ON r.risk_id = rt.risk_id AND r.catalog_id = rt.catalog_id WHERE e.policy_id = ? AND e.eval_result = 'Failed' GROUP BY r.severity ORDER BY r.severity`
- **THEN** the result SHALL show the count of distinct risks and affected controls per severity level

### Requirement: Skill file updated with risk table schemas
The `skills/evidence-schema/SKILL.md` file SHALL document the `risks` and `risk_threats` table schemas alongside the existing tables.

#### Scenario: Skill file contains risk table DDL
- **WHEN** the assistant reads `skills/evidence-schema/SKILL.md`
- **THEN** it SHALL find DDL documentation for `risks` and `risk_threats` in addition to the existing tables
