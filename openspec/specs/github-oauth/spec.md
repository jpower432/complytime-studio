## ADDED Requirements

### Requirement: Gateway serves OAuth login flow
The gateway SHALL expose `/auth/login` which redirects the user to GitHub's OAuth authorization URL with the configured client ID and requested scopes.

#### Scenario: Unauthenticated user visits login
- **WHEN** a user requests `GET /auth/login`
- **THEN** the gateway redirects (HTTP 302) to `https://github.com/login/oauth/authorize` with `client_id`, `redirect_uri`, `scope`, and a CSRF `state` parameter

### Requirement: Gateway handles OAuth callback
The gateway SHALL expose `/auth/callback` which exchanges the authorization code for an access token and sets a session cookie.

#### Scenario: Successful callback
- **WHEN** GitHub redirects to `/auth/callback?code=<code>&state=<state>`
- **THEN** the gateway validates the CSRF state parameter
- **THEN** the gateway exchanges the code for a GitHub access token via `POST https://github.com/login/oauth/access_token`
- **THEN** the gateway creates a signed, encrypted JWT containing the GitHub access token and user info
- **THEN** the gateway sets the JWT as an HttpOnly, Secure, SameSite=Strict cookie
- **THEN** the gateway redirects the user to the workbench root (`/`)

#### Scenario: Invalid state parameter
- **WHEN** the `state` parameter does not match the expected CSRF token
- **THEN** the gateway returns HTTP 403 and does not exchange the code

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

### Requirement: OAuth configuration via Helm values
The GitHub OAuth client ID and secret SHALL be configurable via `values.yaml` and stored in a Kubernetes Secret.

#### Scenario: Values configuration
- **WHEN** `values.yaml` includes `auth.github.clientId` and `auth.github.secretName`
- **THEN** Helm renders the gateway deployment with environment variables referencing the Secret
- **THEN** the gateway reads OAuth credentials from environment at startup

### Requirement: Protected API endpoints require authentication
All `/api/*` endpoints (except `/auth/*`) SHALL require a valid session cookie. Unauthenticated requests SHALL receive HTTP 401.

#### Scenario: Unauthenticated API access
- **WHEN** a request to `/api/agents` has no valid session cookie
- **THEN** the gateway returns HTTP 401 with `{"error": "authentication required"}`
