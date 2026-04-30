# Design: Sub-Agent Registry

## Agent Card Schema

Expand `agents.Card` struct. Backward compatible — new fields are optional.

| Field | Type | Required | Notes |
|:--|:--|:--|:--|
| `id` | `string` | yes | Stable slug, unique across directory. Used in `/api/a2a/{id}` routing. Survives renames. |
| `name` | `string` | yes | Human-readable display name |
| `description` | `string` | yes | One-line description for UI and assistant context |
| `url` | `string` | no | Direct A2A URL (fallback when `kagent.a2aURL` unset) |
| `role` | `string` | no | Agent role category: `assistant`, `coordination`, `program`, `evidence`, `review`, `intelligence`, `framework`, `project-management` |
| `framework` | `string` | no | Runtime framework: `adk`, `langgraph`, `custom`. Informational — gateway does not branch on this. |
| `skills` | `[]Skill` | yes | A2A skills (existing) |
| `delegatable` | `bool` | no | If true, the assistant can auto-route to this agent. Default `false`. |
| `examples` | `[]string` | no | Example user queries for intent matching. Injected into assistant context. |
| `tools` | `[]string` | no | Tool names this agent has access to (informational for UI/assistant). |
| `status` | `string` | no | `active` (default), `deprecated`, `hidden`. Controls visibility in picker. |

### Skill struct (unchanged)

| Field | Type | Required |
|:--|:--|:--|
| `id` | `string` | yes |
| `name` | `string` | yes |
| `description` | `string` | yes |
| `tags` | `[]string` | no |

## Helm Values Shape

```yaml
agentDirectory:
  - id: studio-assistant
    name: Studio Assistant
    description: >-
      Audit preparation, evidence synthesis, cross-framework coverage
      analysis, and compliance guidance
    url: "http://studio-assistant:8080"
    role: assistant
    framework: adk
    delegatable: false
    skills:
      - id: compliance-assistant
        name: Studio Assistant
        description: >-
          Audit preparation, evidence synthesis, cross-framework coverage
          analysis, policy guidance, and AuditLog generation.
        tags: [assistant, audit, compliance]

  - id: studio-program-agent
    name: Program Agent
    description: >-
      Program lifecycle management — intake, monitoring, pipeline runs,
      and state transitions for a single compliance program
    url: "http://studio-program-agent:8080"
    role: program
    framework: langgraph
    delegatable: true
    examples:
      - "Run monitoring for fedramp-high"
      - "Full build for new-product-soc2"
    tools: [validate_gemara_artifact, search_knowledge_base]
    skills:
      - id: program-lifecycle
        name: Program Lifecycle
        description: >-
          Program intake, monitoring, pipeline execution, and state
          management for compliance programs.
        tags: [program, lifecycle, monitoring]

  - id: studio-evidence-agent
    name: Evidence Agent
    description: >-
      Evidence staleness monitoring, gap detection, and validation
      against control objectives across programs
    url: "http://studio-evidence-agent:8080"
    role: evidence
    framework: langgraph
    delegatable: true
    examples:
      - "Check evidence freshness for FedRAMP High"
      - "What evidence gaps do we have across all programs?"
    skills:
      - id: evidence-lifecycle
        name: Evidence Lifecycle
        description: >-
          Staleness monitoring, gap detection, evidence validation,
          and source integration tracking.
        tags: [evidence, staleness, gaps]

  # Extension agents (not shipped — operator-registered):
  #
  # Proprietary integrations (Jira, ServiceNow, Linear, etc.) are not
  # bundled in the open source project. Operators register them as
  # kagent BYO agent entries in this directory.
  #
  # Example:
  # - id: my-project-manager
  #   name: Project Manager
  #   description: >-
  #     Cross-program kanban tracking with Jira integration
  #   url: "http://my-project-manager:8080"
  #   role: project-management
  #   framework: langgraph
  #   delegatable: true
  #   tools: [create_issue, list_epics, list_tasks, search_tasks]
  #   skills:
  #     - id: project-management
  #       name: Project Management
  #       description: Kanban boards with Jira sync
  #       tags: [kanban, jira, tasks]
```

## Gateway Changes

### `internal/agents/agents.go`

**Card struct expansion:**

```go
type Card struct {
    ID          string     `json:"id"`
    Name        string     `json:"name"`
    Description string     `json:"description"`
    URL         string     `json:"url,omitempty"`
    Role        string     `json:"role,omitempty"`
    Framework   string     `json:"framework,omitempty"`
    Skills      []Skill    `json:"skills"`
    Delegatable bool       `json:"delegatable,omitempty"`
    Examples    []string   `json:"examples,omitempty"`
    Tools       []string   `json:"tools,omitempty"`
    Status      string     `json:"status,omitempty"`
    Model       *CardModel `json:"model,omitempty"`
}
```

Uniqueness constraint: `id` must be unique across the directory. Gateway validates at startup and rejects duplicates.

**No proxy logic changes.** The A2A proxy already routes by name and supports both direct URLs and kagent controller paths. New fields are metadata only.

### `GET /api/agents` response

Returns the full enriched card list. Agents with `status: hidden` are excluded. Workbench consumes it for the agent picker.

## Assistant Context Injection

The assistant's system prompt gets a dynamic section listing available sub-agents. Assembled at deploy time (Helm ConfigMap).

### Injected block format

```
## Available Sub-Agents

You can delegate work to these specialized agents using the a2a_delegate tool.
Only delegate to agents listed here. Do not delegate to yourself. Maximum
delegation depth: 2 (you → sub-agent → one more, no further).

### studio-program-agent (program)
Program lifecycle management — intake, monitoring, pipeline runs,
and state transitions for a single compliance program.
Examples: "Run monitoring for fedramp-high"
Tools: validate_gemara_artifact, search_knowledge_base

### studio-evidence-agent (evidence)
Evidence staleness monitoring, gap detection, and validation.
Examples: "Check evidence freshness for FedRAMP High"

[... one block per delegatable agent ...]
```

### Delegation guardrails

- Max depth: 2 hops (assistant → sub-agent → one more)
- Self-delegation denied (agent cannot call itself)
- Timeout: 120s per delegation call
- On sub-agent failure: assistant reports the error, does not retry silently

### `a2a_delegate` tool

Calls `http://localhost:8080/api/a2a/{agent-id}` through the gateway proxy. Streams response back. Implementation varies by framework:
- ADK: registered as a tool via `FunctionTool`
- LangGraph: `@tool` decorator

## Workbench Changes

### Agent picker

Sources from `GET /api/agents`. Displays:
- Agent name and description
- Role badge (color-coded)
- Framework badge (informational)
- Skill tags
- Examples as placeholder hints
- Hidden agents excluded

User selects an agent → workbench routes chat to `/api/a2a/{agent-id}`. Or user stays on the assistant, which auto-delegates.

## Docker-Compose / Non-Kubernetes

`AGENT_DIRECTORY` env var remains. Operators set it to a JSON array matching the schema. Only `studio-assistant` registered by default.
