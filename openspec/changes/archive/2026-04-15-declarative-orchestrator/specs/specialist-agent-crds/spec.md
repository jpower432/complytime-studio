## ADDED Requirements

### Requirement: Specialists deployed as separate BYO Agent CRDs
The Helm chart SHALL deploy each specialist agent (threat modeler, gap analyst, policy composer) as an independent `Agent` resource with `spec.type: BYO`.

#### Scenario: Chart renders three BYO Agent CRDs
- **WHEN** `helm template` is run with default values
- **THEN** the output SHALL contain Agent resources named `studio-threat-modeler`, `studio-gap-analyst`, and `studio-policy-composer`
- **THEN** each resource SHALL have `spec.type: BYO`

#### Scenario: Specialist pods run independently
- **WHEN** the specialist Agent CRDs are applied
- **THEN** each specialist SHALL run in its own pod
- **THEN** each pod SHALL reach `Ready` status independently of the other specialists

### Requirement: Specialist binary supports mode selection
The `studio-agents` binary SHALL accept an `AGENT_MODE` environment variable to control which specialist agent starts.

#### Scenario: Single specialist mode
- **WHEN** `AGENT_MODE=threat-modeler` is set
- **THEN** the binary SHALL start only the threat modeler agent and its A2A endpoint
- **THEN** the binary SHALL NOT start the gap analyst or policy composer

#### Scenario: All specialists mode for local dev
- **WHEN** `AGENT_MODE` is unset or set to `all`
- **THEN** the binary SHALL start all three specialist agents (threat modeler, gap analyst, policy composer) with separate A2A endpoints

### Requirement: Specialists expose A2A endpoints
Each specialist BYO agent SHALL expose an A2A endpoint with an agent card at `/.well-known/agent.json`.

#### Scenario: Agent card discoverable
- **WHEN** an HTTP GET request is sent to `/.well-known/agent.json` on a specialist's port
- **THEN** the response SHALL contain a valid A2A agent card with the specialist's name, description, and skills

#### Scenario: Orchestrator discovers specialist via A2A
- **WHEN** the declarative orchestrator resolves a `type: Agent` tool reference
- **THEN** kagent SHALL fetch the specialist's agent card from the BYO agent's A2A endpoint
- **THEN** the specialist's skills SHALL be available as tool descriptions for the orchestrator's LLM

### Requirement: Specialists preserve runtime portability
Specialist agent binaries SHALL function without Kubernetes or kagent when run locally.

#### Scenario: Local execution with stdio MCP transport
- **WHEN** the specialist binary is run locally with `MCP_TRANSPORT=stdio` (default)
- **THEN** the binary SHALL spawn MCP server processes (gemara-mcp, github-mcp) as child processes
- **THEN** the specialist SHALL be fully functional without any Kubernetes cluster

#### Scenario: Kubernetes execution with SSE MCP transport
- **WHEN** the specialist binary runs in Kubernetes with `MCP_TRANSPORT=sse`
- **THEN** the binary SHALL connect to MCP servers via HTTP URLs provided in environment variables
