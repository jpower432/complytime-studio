# Design: LangGraph Agent Runtime

## Architecture

```
kagent controller
  └── BYO Agent CRD (per persona)
       └── Pod: langgraph-agent container
            ├── kagent-langgraph A2A adapter (FastAPI, /.well-known/agent-card.json, /task)
            ├── LangGraph create_react_agent (CompiledStateGraph)
            ├── Spec loader (constitution + persona + command specs)
            ├── MCP tool connections (gemara-mcp, clickhouse-mcp, knowledge-base-mcp)
            └── PostgreSQL checkpointer (AsyncPostgresSaver)
```

Studio gateway routes A2A traffic via `KAGENT_A2A_URL`:
```
Workbench → Gateway /api/a2a/{agent-id} → kagent controller → Agent pod A2A endpoint
```

## Container Image

Single parameterized image. `AGENT_TYPE` env var selects persona.

### Dockerfile

```dockerfile
FROM python:3.13-slim
WORKDIR /app

COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

COPY config/ /app/config/
COPY agents/ /app/agents/
COPY commands/ /app/commands/
COPY functions/ /app/functions/
COPY engine/ /app/engine/
COPY skills/ /app/skills/
COPY runtime/ /app/runtime/
COPY main.py .

EXPOSE 8080
CMD ["python", "main.py"]
```

### `main.py` — A2A entrypoint

```python
import os
from kagent.langgraph import KAgentApp
from runtime.agent.builder import build_agent
from runtime.spec_loader import SpecLoader
from runtime.tools.registry import get_all_tools

AGENT_TYPE = os.environ.get("AGENT_TYPE", "program")

spec_loader = SpecLoader("/app")
constitution = spec_loader.load_constitution()
persona = spec_loader.load_agent(AGENT_TYPE)

system_prompt = f"{constitution}\n\n---\n\n{persona}"

tools = get_all_tools()
graph = build_agent(system_prompt=system_prompt, tools=tools)

app = KAgentApp(
    graph=graph,
    agent_card={
        "name": f"studio-{AGENT_TYPE}",
        "description": persona.description,
        "capabilities": {"streaming": True},
        "skills": [...]  # from persona YAML frontmatter
    },
)
```

### Requirements

```
kagent-langgraph
langchain-google-genai
langchain-core
langgraph
mcp
langchain-mcp-adapters
psycopg[binary]
pyyaml
httpx
```

## Persona Selection (v1)

| `AGENT_TYPE` | Persona file | Role |
|:--|:--|:--|
| `program` | `agents/program-agent.md` | Program lifecycle |
| `evidence` | `agents/evidence-agent.md` | Evidence staleness, gaps |
| `coordinator` | `agents/coordinator.md` | Portfolio aggregation, routing |

Additional personas (framework specialists, review, intelligence) are deferred until the v1 personas are validated in production. The image supports them — they just aren't deployed.

## Spec Loading

| Component | Source | Loaded at |
|:--|:--|:--|
| Constitution | `config/constitution.md` | Startup |
| Agent persona | `agents/{type}.md` (YAML frontmatter + body) | Startup |
| Command specs | `commands/{name}.md` | On command execution |
| Function specs | `functions/{name}.md` | When referenced by command body |

Prompt assembly for commands:
```
constitution + agent persona body + command body + referenced function bodies
```

Prompt assembly for free-form chat:
```
constitution + agent persona body
```

Constitution must stay under 500 tokens (per governance spec).

## Tool Integration

### MCP tools via environment

| Env var | MCP server | Tools |
|:--|:--|:--|
| `GEMARA_MCP_URL` | gemara-mcp | `validate_gemara_artifact`, `migrate_gemara_artifact`, `validate_command_output` |
| `CLICKHOUSE_MCP_URL` | clickhouse-mcp | `run_select_query`, `list_databases`, `list_tables` |
| `KNOWLEDGE_BASE_MCP_URL` | BYO knowledge base MCP | `search_knowledge_base` (see BYO RAG spec) |

### Per-command tool filtering

A `COMMAND_TOOL_MAP` controls which tools each command can access. Commands not in the map get no tools (LLM-only). This prevents tool misuse and reduces prompt noise.

## Chat Checkpointing

`AsyncPostgresSaver` from `langgraph.checkpoint.postgres.aio` connects to the shared PostgreSQL instance (same one from dual-store spec). Thread ID format: `{program_id}_{user_id}`.

Env var: `POSTGRES_URL` — same connection string as gateway.

Checkpointer is used for **free-form chat** (multi-turn). Command execution is **stateless** (no checkpointer).

Checkpoint data may contain PII/prompt text. Retention policy and encryption-at-rest follow the PostgreSQL instance's configuration.

## LLM Provider

Single provider per deployment. Configured via env vars:

| Env var | Default | Notes |
|:--|:--|:--|
| `LLM_PROVIDER` | `google` | `google` or `anthropic` |
| `LLM_MODEL` | `gemini-2.5-pro` | Model name for selected provider |
| `GOOGLE_API_KEY` | — | Required when provider is `google` |
| `ANTHROPIC_API_KEY` | — | Required when provider is `anthropic` |

`build_llm()` returns the appropriate LangChain chat model. One provider per deployment — no runtime switching. Second provider is escape hatch, not a first-class multi-provider story.

## kagent BYO Agent CRD

One CRD per deployed persona. Example for program agent:

```yaml
apiVersion: kagent.dev/v1alpha2
kind: Agent
metadata:
  name: studio-program-agent
  namespace: kagent
spec:
  description: >-
    Program lifecycle management — intake, monitoring,
    pipeline runs, and state transitions
  type: BYO
  byo:
    deployment:
      image: studio-langgraph-agent:latest
      env:
        - name: AGENT_TYPE
          value: "program"
        - name: LLM_PROVIDER
          value: "google"
        - name: GOOGLE_API_KEY
          valueFrom:
            secretKeyRef:
              name: kagent-google
              key: GOOGLE_API_KEY
        - name: GEMARA_MCP_URL
          value: "http://studio-gemara-mcp:8080"
        - name: CLICKHOUSE_MCP_URL
          value: "http://studio-clickhouse-mcp:8080"
        - name: POSTGRES_URL
          valueFrom:
            secretKeyRef:
              name: studio-postgres-credentials
              key: url
```

## Deployment Tiers

| Tier | Personas | Use case |
|:--|:--|:--|
| **Minimal** | `program` | Program lifecycle only |
| **Standard** | `program`, `evidence`, `coordinator` | Multi-program management |
| **Full** | All personas (when validated) | Complete compliance fleet |

Each persona is a separate Helm toggle:
```yaml
agents:
  programAgent:
    enabled: true
  evidenceAgent:
    enabled: false
  coordinator:
    enabled: false
```
