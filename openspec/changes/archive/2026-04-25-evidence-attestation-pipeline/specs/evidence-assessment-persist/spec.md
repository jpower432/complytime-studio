## ADDED Requirements

### Requirement: Agent emits structured EvidenceAssessment artifacts
The assistant agent SHALL emit a structured `EvidenceAssessment` artifact (JSON or YAML) containing per-evidence classifications when performing a posture check. The artifact SHALL include policy_id, plan_id, evidence_id, classification, reason, and assessed_at for each assessed evidence row.

#### Scenario: Posture check produces assessment artifact
- **WHEN** the agent completes a posture check for policy `p-access-review` covering 5 assessment plans
- **THEN** the agent SHALL emit an EvidenceAssessment artifact with one entry per evidence row assessed, each with a classification from the 7-state model

#### Scenario: Assessment includes provenance
- **WHEN** the agent emits an EvidenceAssessment artifact
- **THEN** the artifact SHALL include the model name and prompt version that produced the assessment

### Requirement: Gateway intercepts and persists EvidenceAssessment artifacts
The Gateway SHALL detect `EvidenceAssessment` artifacts in the A2A SSE stream and write the classifications to the `evidence_assessments` table in ClickHouse. This follows the same pattern as the existing AuditLog auto-persist interceptor.

#### Scenario: Valid assessment persisted
- **WHEN** the Gateway detects an EvidenceAssessment artifact with 3 classification entries
- **THEN** the Gateway SHALL validate the structure (required fields, valid classification enum) and write 3 rows to `evidence_assessments`

#### Scenario: Invalid assessment rejected
- **WHEN** the Gateway detects an EvidenceAssessment artifact with missing required fields or invalid classification values
- **THEN** the Gateway SHALL log a warning and NOT write to ClickHouse

#### Scenario: Agent never writes directly
- **WHEN** the agent performs a posture check
- **THEN** the agent SHALL NOT have write credentials or INSERT access to ClickHouse — all persistence is Gateway-mediated

### Requirement: evidence_assessments table stores classification history
The system SHALL maintain an `evidence_assessments` table in ClickHouse that records every classification produced by the agent, preserving history over time.

#### Scenario: Table schema
- **WHEN** the `evidence_assessments` table is created
- **THEN** it SHALL contain columns: `evidence_id`, `policy_id`, `plan_id`, `classification` (Enum8), `reason` (String), `assessed_at` (DateTime64), `assessed_by` (String — model + prompt version)

#### Scenario: Multiple assessments over time
- **WHEN** the agent assesses evidence `ev-123` on April 1 as "Healthy" and again on April 15 as "Failing"
- **THEN** both rows SHALL exist in `evidence_assessments` with their respective timestamps

#### Scenario: Queryable classifications
- **WHEN** a user or agent queries `SELECT * FROM evidence_assessments WHERE classification = 'Unfit Evidence'`
- **THEN** the query SHALL return all evidence ever classified as Unfit Evidence across all policies and time periods
