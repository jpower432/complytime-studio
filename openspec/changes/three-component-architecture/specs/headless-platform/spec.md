## ADDED Requirements

### Requirement: Platform gateway serves no UI assets
The platform gateway binary SHALL NOT embed or serve SPA assets. All non-API request paths SHALL return HTTP 404 with a JSON error body.

#### Scenario: Browser hits root path
- **WHEN** a request is made to `GET /`
- **THEN** the platform returns `404 {"error": "not found"}`

#### Scenario: Browser hits arbitrary path
- **WHEN** a request is made to `GET /policies/ac-1`
- **THEN** the platform returns `404 {"error": "not found"}`

#### Scenario: API routes still work
- **WHEN** a request is made to `GET /api/policies`
- **THEN** the platform returns `200` with the policies JSON array

### Requirement: CORS configuration is mandatory
The platform SHALL require `CORS_ORIGINS` to be configured when Studio is deployed separately. The gateway SHALL include CORS headers on all `/api/*` and `/a2a/*` responses for configured origins.

#### Scenario: Studio origin is allowed
- **WHEN** `CORS_ORIGINS` includes `http://studio:3000` and the SPA sends a preflight request from that origin
- **THEN** the gateway responds with `Access-Control-Allow-Origin: http://studio:3000`

#### Scenario: Unknown origin is rejected
- **WHEN** a request arrives from an origin not in `CORS_ORIGINS`
- **THEN** the gateway does not include `Access-Control-Allow-Origin` in the response

#### Scenario: Credentials are allowed
- **WHEN** a cross-origin request includes credentials (cookies/auth header)
- **THEN** the gateway includes `Access-Control-Allow-Credentials: true`

### Requirement: Helm chart supports headless deployment via studio.enabled toggle
The Helm chart SHALL support `studio.enabled: false` to deploy only the Platform and Agents without the Studio SPA container.

#### Scenario: Headless deployment
- **WHEN** `helm install` is run with `--set studio.enabled=false`
- **THEN** no Studio Deployment or Service is created
- **THEN** Platform and Agent Deployments are created normally

#### Scenario: Full deployment (default)
- **WHEN** `helm install` is run with default values (`studio.enabled: true`)
- **THEN** Studio, Platform, and Agent Deployments are all created

### Requirement: Health check is independent of Studio
The platform `/healthz` endpoint SHALL report health based on PostgreSQL connectivity only, with no dependency on Studio availability.

#### Scenario: Studio is down but platform is healthy
- **WHEN** the Studio container is unavailable but PostgreSQL is reachable
- **THEN** `GET /healthz` returns `200 ok`
