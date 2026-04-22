## MODIFIED Requirements

### Requirement: Gateway serves user info endpoint
The gateway SHALL expose `GET /auth/me` which returns the authenticated user's profile information including their resolved `role` (`"admin"` or `"viewer"`).

#### Scenario: Authenticated admin request
- **WHEN** a request to `GET /auth/me` includes a valid session cookie for an admin user
- **THEN** the gateway SHALL return JSON with `login`, `name`, `avatar_url`, `email`, and `role: "admin"` fields

#### Scenario: Authenticated viewer request
- **WHEN** a request to `GET /auth/me` includes a valid session cookie for a non-admin user
- **THEN** the gateway SHALL return JSON with `login`, `name`, `avatar_url`, `email`, and `role: "viewer"` fields

#### Scenario: Unauthenticated request
- **WHEN** a request to `GET /auth/me` has no session cookie or an expired/invalid cookie
- **THEN** the gateway SHALL return HTTP 401

## REMOVED Requirements

### Requirement: AuthzActive flag in /auth/me response
**Reason**: Replaced by the `role` field. The `authz_active` flag was part of the raci-authz fail-visible design which is being reverted.
**Migration**: Frontend checks `user.role` instead of `user.authz_active`.

### Requirement: Policies access map in /auth/me response
**Reason**: Per-policy RACI access scoping is being reverted. All authenticated users see all policies; write access is controlled by role, not per-policy.
**Migration**: Frontend removes `policies` map usage. Write control visibility is driven by `user.role`.
