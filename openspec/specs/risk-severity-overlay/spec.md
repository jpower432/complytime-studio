## Requirements

### Requirement: Posture cards show risk severity badge
The system SHALL display a risk severity badge (Critical, High, Medium, Low) on each posture card derived from the highest-severity risk linked to failing controls for that policy.

#### Scenario: Policy with critical risk
- **WHEN** a policy has a control with eval_result "fail" linked to a risk with severity "Critical"
- **THEN** the posture card displays a "Critical" severity badge in red

#### Scenario: Policy with no risks
- **WHEN** a policy has no linked risks in the `risks`/`risk_threats` tables
- **THEN** the posture card displays no severity badge

### Requirement: Requirement matrix rows show risk indicator
The system SHALL display a risk severity column in the requirement matrix. Each row SHALL show the aggregate highest severity from risks linked to that requirement's controls.

#### Scenario: Requirement linked to high-severity risk
- **WHEN** a requirement row's controls are linked via `control_threats` → `risk_threats` → `risks` to a risk with severity "High"
- **THEN** the matrix row displays a "High" indicator in the risk column

#### Scenario: Requirement with no risk linkage
- **WHEN** a requirement row's controls have no linked risks
- **THEN** the risk column displays "—" (dash)

### Requirement: Risk severity API endpoint
The system SHALL provide a `GET /api/risks/severity?policy_id={id}` endpoint returning per-control risk severity derived from the graph join: `risks` → `risk_threats` → `threats` → `control_threats` → `controls`.

#### Scenario: API returns per-control severity
- **WHEN** a client calls `GET /api/risks/severity?policy_id=ampel-branch-protection`
- **THEN** the response includes `[{"control_id":"...","max_severity":"High","risk_count":3}, ...]`
