## ADDED Requirements

### Requirement: Posture cards display inventory context
The system SHALL display enriched inventory information on each posture card: target count, control count, evidence freshness (relative time), and RACI owner (Accountable contact).

#### Scenario: Card shows target and control counts
- **WHEN** the posture view loads and a policy has 12 distinct targets and 47 controls
- **THEN** the posture card displays "12 targets" and "47 controls"

#### Scenario: Card shows evidence freshness
- **WHEN** the most recent evidence for a policy was collected 2 hours ago
- **THEN** the posture card displays "Last evidence: 2h ago"

#### Scenario: Card shows no evidence state
- **WHEN** a policy has zero evidence records
- **THEN** the posture card displays "No evidence yet" instead of freshness

### Requirement: Posture cards display RACI owner
The system SHALL display the Accountable contact from the Policy artifact on the posture card. If no Accountable contact exists, the card SHALL display "No owner".

#### Scenario: Owner from Policy contacts
- **WHEN** the policy has a RACI contact with role "accountable" and name "platform-team"
- **THEN** the posture card displays "Owner: platform-team"

#### Scenario: No owner defined
- **WHEN** the policy has no contacts or no accountable contact
- **THEN** the posture card displays "No owner" with muted styling

### Requirement: Enriched posture API endpoint
The system SHALL provide a `GET /api/posture/summary` endpoint (or extend `GET /api/posture`) returning enriched posture rows with `target_count`, `control_count`, `latest_evidence_at`, and `owner` fields alongside existing fields.

#### Scenario: API returns enriched data
- **WHEN** a client calls `GET /api/posture`
- **THEN** each row includes `target_count`, `control_count`, `latest_evidence_at`, and `owner` fields
