## MODIFIED Requirements

### Requirement: Gateway handles OAuth callback
The gateway SHALL expose `/auth/callback` which exchanges the authorization code for an access token, resolves Google Groups membership, and sets a session cookie.

#### Scenario: Successful callback
- **WHEN** Google redirects to `/auth/callback?code=<code>&state=<state>`
- **THEN** the gateway validates the CSRF state parameter
- **THEN** the gateway exchanges the code for a Google access token
- **THEN** the gateway extracts user info (name, email, avatar) from the ID token
- **THEN** the gateway extracts Google Groups from the `groups` claim if present
- **THEN** the gateway creates a `ServerSession` with `Groups` populated
- **THEN** the gateway sets the session cookie and redirects to `/`

#### Scenario: ID token without groups claim
- **WHEN** the Google ID token does not contain a `groups` claim
- **THEN** the gateway sets `ServerSession.Groups` to an empty slice
- **THEN** login still succeeds — email-only RACI matching is the fallback

#### Scenario: Invalid state parameter
- **WHEN** the `state` parameter does not match the expected CSRF token
- **THEN** the gateway returns HTTP 403 and does not exchange the code

### Requirement: Gateway serves user info endpoint
The gateway SHALL expose `GET /auth/me` which returns the authenticated user's profile information and policy access set.

#### Scenario: Authenticated request
- **WHEN** a request to `GET /auth/me` includes a valid session cookie
- **THEN** the gateway returns JSON with `login`, `name`, `avatar_url`, `email`, and `policies` fields
- **THEN** `policies` is a map of `policy_id → raci_role` from the user's resolved access set

#### Scenario: Unauthenticated request
- **WHEN** a request to `GET /auth/me` has no session cookie or an expired/invalid cookie
- **THEN** the gateway returns HTTP 401
