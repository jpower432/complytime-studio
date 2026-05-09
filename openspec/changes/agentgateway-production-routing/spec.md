# Spec: AgentGateway Production Routing

## Capability

Production A2A and MCP routing through AgentGateway with protocol-aware streaming, CEL-enforced tool access, enriched agent directory, and declarative route generation from Helm values.

## Scenarios

### A2A Routing

#### Browser reaches agent through AgentGateway

| Clause | Statement |
|:--|:--|
| GIVEN | AgentGateway is deployed with an HTTPRoute for `studio-assistant` |
| AND | OAuth2 Proxy injects `Authorization: Bearer` header on `/a2a/*` |
| WHEN | Browser sends `POST /a2a/studio-assistant` with A2A `message/stream` payload |
| THEN | AgentGateway routes request to `studio-assistant` Service on port 8080 |
| AND | Response streams back as SSE events to the browser |

#### Unknown agent rejected

| Clause | Statement |
|:--|:--|
| GIVEN | No HTTPRoute exists for agent `nonexistent-agent` |
| WHEN | Browser sends `POST /a2a/nonexistent-agent` |
| THEN | AgentGateway returns 404 |

#### Studio gateway is NOT in the A2A path

| Clause | Statement |
|:--|:--|
| GIVEN | AgentGateway is the backend for `/a2a/*` at the ingress |
| WHEN | Browser sends any request to `/a2a/*` |
| THEN | Request does NOT pass through Studio gateway |
| AND | Studio gateway logs show zero A2A traffic |

### MCP Tool Access Enforcement

#### Allowed tool call succeeds

| Clause | Statement |
|:--|:--|
| GIVEN | Agent `studio-assistant` has `tools: [validate_gemara_artifact]` in agentDirectory |
| AND | CEL policy allows `studio-assistant` to call `validate_gemara_artifact` |
| WHEN | Agent pod sends MCP `tools/call` for `validate_gemara_artifact` through AgentGateway |
| THEN | Request reaches `studio-gemara-mcp` and returns a valid response |

#### Disallowed tool call rejected

| Clause | Statement |
|:--|:--|
| GIVEN | Agent `studio-assistant` does NOT have `dangerous_tool` in its tools list |
| AND | CEL deny-all policy is active |
| WHEN | Agent pod sends MCP `tools/call` for `dangerous_tool` through AgentGateway |
| THEN | AgentGateway rejects the request before it reaches the MCP server |
| AND | Response indicates policy denial |

#### New agent inherits tool restrictions

| Clause | Statement |
|:--|:--|
| GIVEN | Operator adds agent `evidence-analyst` with `tools: [query_database]` to agentDirectory |
| WHEN | Helm renders and applies |
| THEN | CEL policy allows `evidence-analyst` to call only `query_database` |
| AND | Calls to `validate_gemara_artifact` from `evidence-analyst` are rejected |

### Agent Directory

#### GET /api/agents returns enriched cards

| Clause | Statement |
|:--|:--|
| GIVEN | agentDirectory has entries with `id`, `name`, `role`, `framework`, `status`, `tools`, `examples` |
| WHEN | Workbench calls `GET /api/agents` |
| THEN | Response includes all fields for each active agent |
| AND | Response does NOT include `url` (internal routing detail) |

#### Hidden agents excluded from directory

| Clause | Statement |
|:--|:--|
| GIVEN | Agent `internal-tool` has `status: hidden` in agentDirectory |
| WHEN | Workbench calls `GET /api/agents` |
| THEN | Response does NOT include `internal-tool` |
| AND | AgentGateway HTTPRoute for `internal-tool` still exists (reachable by ID) |

### Route Generation

#### Helm renders routes from agentDirectory

| Clause | Statement |
|:--|:--|
| GIVEN | agentDirectory has entry with `id: studio-assistant` and `status: active` |
| WHEN | `helm template` renders the chart |
| THEN | Output includes `AgentgatewayBackend` named `studio-assistant-backend` |
| AND | Output includes `HTTPRoute` named `studio-assistant-route` with path prefix `/studio-assistant` |

#### Adding an agent generates all routing resources

| Clause | Statement |
|:--|:--|
| GIVEN | Operator adds new entry to agentDirectory with `id: threat-modeler` |
| WHEN | `helm upgrade` applies |
| THEN | New `AgentgatewayBackend`, `HTTPRoute`, and CEL policy rules are created |
| AND | Agent is reachable at `/a2a/threat-modeler` through AgentGateway |
| AND | Agent appears in `GET /api/agents` response |

### Workbench Integration

#### Workbench sends A2A to correct path

| Clause | Statement |
|:--|:--|
| GIVEN | Workbench `a2aEndpoint()` returns `/a2a/${agentName}` |
| WHEN | User selects an agent and sends a message |
| THEN | Request goes to `/a2a/{agent-id}` (NOT `/api/a2a/{agent-id}`) |
| AND | Session cookie is included (`credentials: "same-origin"`) |

### BYO Agent Onboarding

#### Operator registers LangGraph agent

| Clause | Statement |
|:--|:--|
| GIVEN | Operator has a container image serving A2A at `/` |
| AND | Operator adds kagent BYO Agent CRD template to Helm |
| AND | Operator adds `agentDirectory` entry with `id`, `name`, `role`, `framework`, `status`, `tools` |
| WHEN | `helm upgrade` applies |
| THEN | kagent creates Deployment + Service for the agent |
| AND | AgentGateway routes traffic to the agent via generated HTTPRoute |
| AND | CEL policy restricts agent to declared tools |
| AND | Agent appears in workbench picker |
