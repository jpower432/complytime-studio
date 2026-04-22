## ADDED Requirements

### Requirement: Evidence write requires responsible or accountable role
`POST /api/evidence` and `POST /api/evidence/upload` SHALL verify the user has `responsible` or `accountable` RACI role for the target `policy_id` before accepting the payload.

#### Scenario: Responsible user ingests evidence
- **WHEN** a user with `responsible` role for `ampel-bp` sends `POST /api/evidence` with `policy_id=ampel-bp`
- **THEN** the evidence is accepted and stored

#### Scenario: Consulted user attempts evidence ingestion
- **WHEN** a user with `consulted` role for `ampel-bp` sends `POST /api/evidence` with `policy_id=ampel-bp`
- **THEN** the gateway returns HTTP 403 with `{"error": "insufficient permissions"}`

#### Scenario: API token bypass
- **WHEN** a request uses `Authorization: Bearer <STUDIO_API_TOKEN>`
- **THEN** evidence ingestion is allowed regardless of RACI role (existing behavior preserved for seeding)
