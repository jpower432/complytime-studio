## ADDED Requirements

### Requirement: Admin email allowlist configuration
The gateway SHALL read admin emails from an `ADMIN_EMAILS` environment variable (comma-separated). The Helm chart SHALL populate this from `auth.admins` in `values.yaml`. Any authenticated user whose email matches the list SHALL be assigned the `admin` role. All other authenticated users SHALL be assigned the `viewer` role.

#### Scenario: User in admin list
- **WHEN** an authenticated user with email `eddie@acme.com` makes a request and `ADMIN_EMAILS` contains `eddie@acme.com`
- **THEN** the user's role SHALL be resolved as `admin`

#### Scenario: User not in admin list
- **WHEN** an authenticated user with email `auditor@acme.com` makes a request and `ADMIN_EMAILS` does not contain `auditor@acme.com`
- **THEN** the user's role SHALL be resolved as `viewer`

#### Scenario: Empty admin list
- **WHEN** `ADMIN_EMAILS` is empty or unset
- **THEN** all authenticated users SHALL be assigned the `admin` role (fail-open for dev clusters)

### Requirement: RequireAdmin middleware for write endpoints
The gateway SHALL apply a `RequireAdmin` middleware to all mutating API endpoints. Requests from `viewer` users to protected endpoints SHALL receive HTTP 403 with `{"error": "admin role required"}`.

#### Scenario: Admin writes evidence
- **WHEN** an `admin` user sends `POST /api/evidence`
- **THEN** the request SHALL proceed normally

#### Scenario: Viewer writes evidence
- **WHEN** a `viewer` user sends `POST /api/evidence`
- **THEN** the gateway SHALL return HTTP 403

#### Scenario: Viewer reads evidence
- **WHEN** a `viewer` user sends `GET /api/evidence?policy_id=...`
- **THEN** the request SHALL proceed normally (reads are not protected)

#### Scenario: Viewer chats with agent
- **WHEN** a `viewer` user sends `POST /api/a2a/studio-assistant`
- **THEN** the request SHALL proceed normally (agent chat is not write-protected)

### Requirement: Protected endpoints list
The `RequireAdmin` middleware SHALL be applied to exactly these endpoints: `POST /api/policies/import`, `POST /api/evidence`, `POST /api/evidence/upload`, `POST /api/audit-logs`, `POST /api/catalogs/import`, `POST /api/mappings/import`, `POST /api/publish`. All `GET` endpoints and `POST /api/a2a/*` SHALL NOT be protected.

#### Scenario: All write endpoints protected
- **WHEN** a `viewer` user sends `POST` to any of the protected endpoints
- **THEN** each SHALL return HTTP 403

#### Scenario: Read endpoints unprotected
- **WHEN** a `viewer` user sends `GET` to `/api/policies`, `/api/evidence`, `/api/audit-logs`, or `/api/agents`
- **THEN** each SHALL return normally

### Requirement: Frontend hides write controls for viewers
The workbench SHALL read the `role` field from `/auth/me` and hide import buttons, upload buttons, and audit trigger prompts when `role` is `viewer`.

#### Scenario: Viewer sees read-only UI
- **WHEN** a `viewer` user loads the workbench
- **THEN** the policy import button, evidence upload button, and catalog import controls SHALL NOT be rendered

#### Scenario: Admin sees full UI
- **WHEN** an `admin` user loads the workbench
- **THEN** all write controls SHALL be rendered normally
