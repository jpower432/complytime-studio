## ADDED Requirements

### Requirement: OpenAPI 3.1 spec documents all public REST endpoints
A hand-authored OpenAPI specification SHALL document every endpoint served by the platform gateway on the public port. The spec SHALL live at `docs/api/openapi.yaml`.

#### Scenario: Spec covers all API groups
- **WHEN** a consumer reads `docs/api/openapi.yaml`
- **THEN** it contains path definitions for: policies, evidence, audit-logs, draft-audit-logs, posture, mappings, requirements, catalogs, programs, agents, config, system-info, notifications, certifications, threats, risks, validate, migrate, auth (me, bootstrap)

#### Scenario: Request and response schemas defined
- **WHEN** a consumer reads an endpoint definition
- **THEN** it includes request body schema (for POST/PUT/PATCH) and response schema with field types

### Requirement: OpenAPI spec is the contract between Platform and Studio
The Studio SPA SHALL only call endpoints documented in the OpenAPI spec. Any new endpoint required by Studio MUST be added to the spec before the SPA consumes it.

#### Scenario: Studio uses undocumented endpoint
- **WHEN** a PR adds a fetch call to an endpoint not in `openapi.yaml`
- **THEN** the PR SHALL be rejected until the endpoint is added to the spec

### Requirement: Spec includes authentication requirements
Each endpoint in the spec SHALL declare its security requirements: bearer token, OAuth2 session cookie, or no auth.

#### Scenario: Public endpoints
- **WHEN** a consumer reads `/healthz` definition
- **THEN** it shows no security requirement (public)

#### Scenario: Protected endpoints
- **WHEN** a consumer reads `/api/policies` definition
- **THEN** it shows `bearerAuth` or `oauth2` security scheme

### Requirement: Spec includes error response schemas
Each endpoint SHALL document error responses (400, 401, 403, 404, 500) with consistent error body schema `{"error": "string"}`.

#### Scenario: Validation error
- **WHEN** a POST request has invalid body
- **THEN** the spec documents a 400 response with `{"error": "<description>"}`
