## ADDED Requirements

### Requirement: Threat impact traversal query pattern
The `evidence-schema` skill SHALL document a query pattern that traverses from a threat ID to all evidence rows via the `control_threats` junction table. The pattern SHALL use: `threats` → `control_threats` → `evidence`.

#### Scenario: Query evidence affected by a specific threat
- **WHEN** the assistant executes `SELECT e.target_id, e.eval_result, e.control_id FROM control_threats ct JOIN evidence e ON e.control_id = ct.control_id WHERE ct.threat_entry_id = ?`
- **THEN** the result SHALL include all evidence for controls that mitigate the specified threat

### Requirement: Coverage completeness query pattern
The `evidence-schema` skill SHALL document a query pattern that identifies controls with no evidence. The pattern SHALL LEFT JOIN `controls` with `evidence` and filter for NULL evidence rows.

#### Scenario: Find controls with no evidence
- **WHEN** the assistant executes `SELECT c.control_id, c.title FROM controls c LEFT JOIN evidence e ON e.control_id = c.control_id AND e.policy_id = ? WHERE e.control_id IS NULL AND c.catalog_id = ?`
- **THEN** the result SHALL include all controls from the catalog that have no corresponding evidence rows

### Requirement: Requirement text enrichment query pattern
The `evidence-schema` skill SHALL document a query pattern that enriches evidence with the assessment requirement text. The pattern SHALL JOIN `evidence` with `assessment_requirements`.

#### Scenario: Evidence enriched with requirement text
- **WHEN** the assistant executes `SELECT e.control_id, e.eval_result, ar.text FROM evidence e JOIN assessment_requirements ar ON ar.control_id = e.control_id AND ar.requirement_id = e.requirement_id`
- **THEN** the result SHALL include the human-readable requirement text alongside each evidence row

### Requirement: Framework-to-threat traversal query pattern
The `evidence-schema` skill SHALL document a multi-hop query pattern: `mapping_entries` → `controls` → `control_threats` → `threats`. This enables "which threats does framework X address?" queries.

#### Scenario: Threats covered by a framework
- **WHEN** the assistant executes `SELECT DISTINCT t.threat_id, t.title FROM mapping_entries me JOIN controls c ON c.control_id = me.control_id JOIN control_threats ct ON ct.control_id = c.control_id JOIN threats t ON t.threat_id = ct.threat_entry_id WHERE me.framework = ?`
- **THEN** the result SHALL include all threats mitigated by controls mapped to the specified framework

### Requirement: Skill file updated with new table schemas
The `skills/evidence-schema/SKILL.md` file SHALL document the `controls`, `assessment_requirements`, `threats`, and `control_threats` table schemas alongside the existing tables.

#### Scenario: Skill file contains all table DDL
- **WHEN** the assistant reads `skills/evidence-schema/SKILL.md`
- **THEN** it SHALL find DDL documentation for `controls`, `assessment_requirements`, `threats`, and `control_threats` in addition to the existing tables
