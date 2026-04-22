## ADDED Requirements

### Requirement: Policy list filtered by access set
`GET /api/policies` SHALL return only policies present in the user's access set.

#### Scenario: User has access to two of three policies
- **WHEN** the system has policies `ampel-bp`, `soc2-corp`, `nist-internal` and the user's access set contains only `ampel-bp` and `soc2-corp`
- **THEN** `GET /api/policies` returns only `ampel-bp` and `soc2-corp`

#### Scenario: User has no policy access
- **WHEN** the user's access set is empty and `policy_contacts` has rows (enforcement is active)
- **THEN** `GET /api/policies` returns an empty list

### Requirement: Policy detail filtered by access set
`GET /api/policies/{id}` SHALL return 404 if the policy_id is not in the user's access set.

#### Scenario: User requests a policy they have access to
- **WHEN** the user requests `GET /api/policies/ampel-bp` and `ampel-bp` is in their access set
- **THEN** the policy is returned normally

#### Scenario: User requests a policy they lack access to
- **WHEN** the user requests `GET /api/policies/nist-internal` and `nist-internal` is not in their access set
- **THEN** the gateway returns HTTP 404

### Requirement: Evidence queries filtered by access set
`GET /api/evidence` SHALL inject `policy_id IN (...)` from the user's access set into the query filter.

#### Scenario: Evidence query scoped to allowed policies
- **WHEN** a user queries `GET /api/evidence` with access set `{ampel-bp, soc2-corp}`
- **THEN** results include only evidence rows where `policy_id` is `ampel-bp` or `soc2-corp`

### Requirement: Audit logs filtered by access set
`GET /api/audit-logs` and `GET /api/audit-logs/{id}` SHALL be scoped to the user's access set.

#### Scenario: Audit log list scoped
- **WHEN** a user queries audit logs and their access set does not include the requested `policy_id`
- **THEN** the endpoint returns an empty list (list) or HTTP 404 (detail)

### Requirement: /auth/me returns access set
`GET /auth/me` SHALL include a `policies` field containing the user's resolved access map (`policy_id → raci_role`).

#### Scenario: Authenticated user with access
- **WHEN** an authenticated user requests `GET /auth/me`
- **THEN** the response includes `"policies": {"ampel-bp": "responsible", "soc2-corp": "consulted"}`

#### Scenario: API token bypass
- **WHEN** a request uses `Authorization: Bearer <STUDIO_API_TOKEN>`
- **THEN** `/auth/me` is not available (token bypass skips session resolution)
- **THEN** all other API endpoints return unfiltered results (existing behavior preserved)
