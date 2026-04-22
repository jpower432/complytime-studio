## MODIFIED Requirements

### Requirement: A2A proxy removed from gateway binary

The gateway binary SHALL NOT register `/api/a2a/` routes. The A2A proxy
is a separate binary and Deployment. The gateway MAY retain the agent
directory endpoint (`GET /api/agents`) for the workbench to discover
available agents.

#### Scenario: Gateway does not proxy A2A
- **WHEN** a request arrives at the gateway for `/api/a2a/{agent}`
- **THEN** the gateway returns 404 (or ingress routes it to the proxy before it reaches the gateway)

#### Scenario: Agent directory stays in gateway
- **WHEN** the workbench requests `GET /api/agents`
- **THEN** the gateway returns the agent card directory as before
