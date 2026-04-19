## ADDED Requirements

### Requirement: Gateway proxies A2A requests to agent pods
The gateway SHALL expose `POST /api/a2a/{agent-name}` which reverse-proxies the request body to the agent's A2A endpoint at `http://{agent-name}:8080`. Route registration SHALL be performed by `internal/agents.Register(mux, opts)` instead of inline closures in `main.go`.

#### Scenario: Proxied chat message
- **WHEN** the frontend sends `POST /api/a2a/studio-threat-modeler` with an A2A SendMessage payload
- **THEN** the gateway forwards the request to `http://studio-threat-modeler:8080`
- **THEN** the response (including streaming SSE) is relayed back to the frontend

#### Scenario: Unknown agent name
- **WHEN** the frontend sends `POST /api/a2a/nonexistent-agent`
- **THEN** the gateway returns HTTP 502 (Bad Gateway) if the upstream is unreachable

### Requirement: A2A proxy injects auth headers
The gateway SHALL inject the `Authorization: Bearer <token>` header on all proxied A2A requests before forwarding to the agent pod. The token SHALL be obtained via a `TokenProvider` interface, not by directly importing `internal/auth`.

#### Scenario: Header injection
- **WHEN** a proxied A2A request is forwarded
- **THEN** the `Authorization` header is set to the user's GitHub token obtained from `TokenProvider.TokenFromRequest(r)`
- **THEN** any pre-existing `Authorization` header from the frontend is overwritten

### Requirement: A2A proxy supports streaming
The gateway SHALL support Server-Sent Events (SSE) streaming for A2A `SendStreamingMessage` responses, relaying the event stream from the agent pod to the frontend without buffering.

#### Scenario: Streaming response relay
- **WHEN** the agent returns an SSE stream (TaskStatusUpdateEvent, TaskArtifactUpdateEvent)
- **THEN** the gateway relays each event to the frontend as it arrives
- **THEN** the gateway does not buffer the full response before sending

### Requirement: Frontend never contacts agent pods directly
All agent communication from the React SPA SHALL go through the gateway's `/api/a2a/{agent-name}` endpoint. Agent pod A2A ports are not exposed via Ingress.

#### Scenario: Network isolation
- **WHEN** the Helm chart is deployed
- **THEN** agent pod Services are ClusterIP only (no NodePort, no Ingress)
- **THEN** the gateway is the sole Ingress-exposed entry point
