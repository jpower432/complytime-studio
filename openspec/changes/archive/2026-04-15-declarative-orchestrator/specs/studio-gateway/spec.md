## ADDED Requirements

### Requirement: Gateway serves workbench SPA
The `studio-gateway` service SHALL serve the embedded workbench SPA at the root path (`/`).

#### Scenario: Browser loads workbench
- **WHEN** a browser requests `GET /`
- **THEN** the gateway SHALL return the workbench `index.html` with HTTP 200
- **THEN** all static assets (JS, CSS) SHALL be served from the embedded filesystem

### Requirement: Gateway proxies A2A to orchestrator
The gateway SHALL forward A2A JSON-RPC requests to the declarative orchestrator's A2A endpoint.

#### Scenario: Chat message forwarded to orchestrator
- **WHEN** the workbench sends a `POST /invoke` with a JSON-RPC `message/send` payload
- **THEN** the gateway SHALL forward the request to the orchestrator's A2A endpoint
- **THEN** the response SHALL be returned to the workbench

#### Scenario: EventSource stream forwarded
- **WHEN** the workbench opens an EventSource connection to `/invoke?taskId=<id>`
- **THEN** the gateway SHALL proxy the SSE stream from the orchestrator to the browser

### Requirement: Gateway proxies validate and migrate to gemara-mcp
The gateway SHALL proxy `/api/validate` and `/api/migrate` requests to the gemara-mcp service.

#### Scenario: Validate request proxied
- **WHEN** the workbench sends `POST /api/validate` with `{yaml, definition, version}`
- **THEN** the gateway SHALL call `validate_gemara_artifact` on gemara-mcp via MCP client
- **THEN** the result SHALL be returned as `{valid, errors}`

#### Scenario: Gemara-mcp unavailable
- **WHEN** gemara-mcp is not reachable
- **THEN** the gateway SHALL return HTTP 503 with `{"error": "gemara-mcp unavailable"}`

### Requirement: Gateway proxies registry operations to oras-mcp
The gateway SHALL proxy `/api/registry/*` requests to the oras-mcp service.

#### Scenario: List repositories proxied
- **WHEN** the workbench sends `GET /api/registry/repositories?registry=<url>`
- **THEN** the gateway SHALL call `list_repositories` on oras-mcp via MCP client
- **THEN** the result SHALL be returned as JSON

### Requirement: Gateway hosts publish endpoint
The gateway SHALL serve the `POST /api/publish` endpoint for OCI bundle assembly and push.

#### Scenario: Bundle published via gateway
- **WHEN** the workbench sends `POST /api/publish` with `{artifacts, target, tag, sign}`
- **THEN** the gateway SHALL assemble the artifacts into an OCI bundle and push to the target registry
- **THEN** the response SHALL include `{reference, digest, tag}`

### Requirement: Gateway deployed as Kubernetes Deployment and Service
The Helm chart SHALL deploy `studio-gateway` as a standard Kubernetes Deployment and ClusterIP Service.

#### Scenario: Chart renders gateway resources
- **WHEN** `helm template` is run with default values
- **THEN** the output SHALL contain a Deployment named `studio-gateway`
- **THEN** the output SHALL contain a Service named `studio-gateway` on port 8080
