## ADDED Requirements

### Requirement: Standalone A2A proxy binary

The A2A proxy SHALL be a standalone Go binary (`cmd/a2a-proxy/`) that
registers and handles `POST /api/a2a/{agent}` by reverse-proxying to either
direct agent base URLs or kagent controller URLs. The binary SHALL have no
ClickHouse dependency and no session state.

#### Scenario: Route parity with gateway-embedded proxy
- **WHEN** the standalone A2A proxy is configured with the same agent directory and upstream URL settings as the current gateway
- **THEN** requests to `/api/a2a/{agent}` behave equivalently for success, streaming, and upstream error cases

#### Scenario: Stateless operation
- **WHEN** the A2A proxy binary starts
- **THEN** it requires no database connection, no session store, and no shared filesystem
- **THEN** multiple replicas can run concurrently without coordination

#### Scenario: kagent upstream
- **WHEN** `KAGENT_A2A_URL` points at a kagent-controlled endpoint
- **THEN** the standalone proxy forwards to that URL model without requiring clients to change path or payload shape

### Requirement: Authorization propagation for OBO

The A2A proxy SHALL accept `Authorization` headers from the trusted gateway
(or ingress) and propagate them to agent endpoints so the on-behalf-of GitHub
token flow continues to work.

#### Scenario: Bearer forwarded to agent
- **WHEN** an incoming request includes `Authorization: Bearer <token>`
- **THEN** the proxied upstream request includes that header so MCP and agent tooling receive the user token

#### Scenario: No silent token drop
- **WHEN** auth is enabled in the overall deployment
- **THEN** a missing token at the proxy results in a documented 401 response, not an ambiguous partial forward

### Requirement: Independent Kubernetes Deployment

The A2A proxy SHALL be deployed as its own Kubernetes Deployment and Service,
with independent replica count, resource limits, and health checks.

#### Scenario: Independent scaling
- **WHEN** chat load increases
- **THEN** the A2A proxy replica count can be increased without scaling the gateway

#### Scenario: Independent failure
- **WHEN** the A2A proxy is unavailable
- **THEN** the gateway continues serving dashboard CRUD, SPA, and other non-agent routes
