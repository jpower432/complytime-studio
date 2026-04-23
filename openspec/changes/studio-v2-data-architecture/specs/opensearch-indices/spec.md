## ADDED Requirements

### Requirement: compliance-evidence index for evidence search
The system SHALL maintain an OpenSearch index `compliance-evidence` with fields: `evidence_id` (keyword), `target_name` (text + keyword), `engine_name` (keyword), `rule_name` (text), `rule_id` (keyword), `eval_result` (keyword), `policy_id` (keyword), `control_id` (keyword), `requirement_id` (keyword), `collected_at` (date), `labels` (object), `eval_message` (text), `gate_status` (keyword). The index SHALL support full-text search, keyword faceting, date range filtering, and nested label queries.

#### Scenario: Search evidence by label
- **WHEN** a user searches for evidence with label "quarterly-firewall-review"
- **THEN** the system SHALL return matching evidence documents ranked by relevance

#### Scenario: Faceted evidence filtering
- **WHEN** the UI requests evidence filtered by `engine_name = "nessus"` and `eval_result = "Failed"`
- **THEN** the system SHALL return matching documents with facet counts for other field values

### Requirement: compliance-knowledge index for semantic search
The system SHALL maintain an OpenSearch index `compliance-knowledge` with fields for control objectives, assessment requirement text, threat descriptions, and guideline objectives. Each text field SHALL have a corresponding dense vector field for semantic similarity search.

#### Scenario: Semantic control search
- **WHEN** the agent or user searches "which controls relate to access reviews?"
- **THEN** the system SHALL return controls ranked by vector similarity to the query, not just keyword match

### Requirement: Index templates managed declaratively
OpenSearch index templates SHALL be defined as JSON files in the Helm chart and applied at Gateway startup. Template changes SHALL be backward-compatible (additive fields only) or versioned with reindex support.

#### Scenario: Gateway applies index template on startup
- **WHEN** the Gateway starts and the index template has changed
- **THEN** the system SHALL apply the updated template; new documents SHALL use the updated mapping
