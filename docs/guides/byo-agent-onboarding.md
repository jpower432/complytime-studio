# BYO Agent Onboarding

Register a custom agent with ComplyTime Studio in three steps.

## Prerequisites

- Container image serving A2A at `/.well-known/agent-card.json`
- Helm access to the Studio chart

## Step 1: Add Agent Directory Entry

Add an entry to `agentDirectory` in `charts/complytime-studio/values.yaml`:

```yaml
agentDirectory:
  - id: my-agent                # K8s Service name (must match CRD metadata.name)
    name: My Agent              # Human-friendly display name
    description: >-
      What this agent does (shown in workbench picker)
    role: reviewer              # Logical role (auditor, reviewer, composer, etc.)
    framework: langgraph        # Agent framework (adk, langgraph, crewai, custom)
    status: active              # active = shown in picker, hidden = internal-only
    url: "http://my-agent:8080" # Internal service URL
    tools:                      # MCP tools this agent may invoke (CEL-enforced)
      - validate_gemara_artifact
    examples:                   # Prompt suggestions for workbench
      - "Analyze the latest evidence batch"
    skills:
      - id: my-skill
        name: My Skill
        description: What this A2A skill does
        tags: [review]
```

## Step 2: Create BYO Agent CRD Template

Create `charts/complytime-studio/templates/byo-my-agent.yaml`:

```yaml
apiVersion: kagent.dev/v1alpha2
kind: Agent
metadata:
  name: my-agent
  namespace: {{ .Release.Namespace }}
spec:
  description: What this agent does
  type: BYO
  byo:
    deployment:
      image: "{{ .Values.myAgent.image.repository }}:{{ .Values.myAgent.image.tag }}"
      env:
        - name: GEMARA_MCP_URL
          value: "http://agentgateway-proxy/mcp/gemara-mcp"
        - name: AGENT_ID
          value: "my-agent"
        - name: KAGENT_URL
          value: "http://kagent-controller:8083"
        - name: APP_NAME
          value: "my-agent"
      resources:
        requests:
          memory: "256Mi"
          cpu: "100m"
```

The `AGENT_ID` env var configures the agent to send `X-Agent-ID: my-agent` on
MCP requests, which AgentGateway uses for CEL tool-access enforcement.

## Step 3: Deploy

```bash
make deploy
```

Verify the agent appears in the workbench picker via `GET /api/agents`.

## Tool Access

AgentGateway enforces per-agent tool allowlists via CEL policies generated from
`agentDirectory[].tools`. Only declared tools are permitted — all other MCP
calls are denied. To grant access to additional tools, add them to the `tools`
list and redeploy.

## Architecture

```
Browser → /a2a/{agent-id} → AgentGateway → Agent Pod
                                  ↕
                            CEL Policies
                                  ↕
Agent Pod → /mcp/{server} → AgentGateway → MCP Server Pod
```

AgentGateway provides:
- Protocol-aware A2A/MCP routing with SSE streaming
- OpenTelemetry traces for every tool call
- CEL-based authorization (deny-all default)
- Session management for multi-turn conversations

---

## LangGraph Framework Guide

LangGraph is the recommended framework for BYO agents. It provides native A2A
support and integrates with kagent's checkpointer for durable conversations.

### Minimal graph.py

```python
from typing import Annotated, Sequence, TypedDict

from kagent_langgraph import KAgentApp
from langchain_core.messages import BaseMessage
from langgraph.graph import StateGraph
from langgraph.graph.message import add_messages


class State(TypedDict):
    messages: Annotated[Sequence[BaseMessage], add_messages]


async def agent_node(state: State, config):
    # Your LLM call with tools bound
    ...

builder = StateGraph(State)
builder.add_node("agent", agent_node)
builder.add_edge("__start__", "agent")

app = KAgentApp(
    graph_builder=builder,
    agent_card={
        "name": "my-agent",
        "description": "What this agent does",
        "version": "0.1.0",
        "capabilities": {"streaming": True},
        "defaultInputModes": ["text"],
        "defaultOutputModes": ["text"],
        "skills": [],
    },
    kagent_url=os.environ["KAGENT_URL"],
    app_name=os.environ["APP_NAME"],
)

graph = app.build()
```

### langgraph.json

```json
{
  "$schema": "https://langgra.ph/schema.json",
  "graphs": {
    "my_agent": "./graph.py:graph"
  },
  "python_version": "3.12",
  "dependencies": ["./requirements.txt"]
}
```

### requirements.txt

```
kagent-langgraph>=0.9.2
langchain-mcp-adapters>=0.2.2
langchain-anthropic>=0.3.0
pyyaml
httpx
```

### Dockerfile

```dockerfile
FROM python:3.12-slim
WORKDIR /app
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt
COPY . .
EXPOSE 8080
CMD ["uvicorn", "graph:graph", "--host", "0.0.0.0", "--port", "8080"]
```

### Checkpointing

`kagent-langgraph` uses `KAgentCheckpointer` automatically. Conversation state
is stored in kagent's platform storage — no PostgreSQL configuration needed.
State survives pod restarts and is visible in the kagent dashboard.

### MCP Tools

Use `langchain-mcp-adapters` to connect to MCP servers:

```python
from langchain_mcp_adapters.client import MultiServerMCPClient

servers = {
    "gemara-mcp": {
        "transport": "streamable_http",
        "url": os.environ["GEMARA_MCP_URL"],
    },
}

async with MultiServerMCPClient(servers) as client:
    tools = client.get_tools()
```

AgentGateway enforces tool access via CEL policies — only tools listed in
`agentDirectory[].tools` are permitted.
