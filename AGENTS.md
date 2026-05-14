# AGENTS.md

Agent creation guide for ComplyTime Studio. Every agent follows the JTBD (Jobs-to-be-Done) framework.

---

## JTBD Framework

Each agent answers four questions:

| Dimension | Question | Artifact |
|:--|:--|:--|
| **Identity** | Who am I? | `agent.yaml` — `name`, `description` |
| **Instructions** | What do I need to do? | `prompt.md` — workflow steps |
| **Knowledge** | What do I need to know? | `skills/` — reusable SKILL.md packs |
| **Tools** | What tools do I need? | `agent.yaml` — `mcp` block |

---

## Quick Start: Create a New Agent

### 1. Create the agent directory

Agent source lives in the [complytime-agents](https://github.com/complytime/complytime-agents) repo:

```
agents/<agent-name>/
├── agent.yaml    # Canonical spec (framework-agnostic)
└── prompt.md     # Workflow instructions
```

### 2. Write `agent.yaml`

```yaml
# SPDX-License-Identifier: Apache-2.0
name: studio-<agent-name>
description: >-
  One-line description of what the agent does

prompt: prompt.md

skills:
 # Internal skills (from complytime-agents repo)
 - path: skills/studio-audit
 - path: skills/posture-check
 # External skills (from other repos)
 - repo: https://github.com/rhaml-23/prompt.git
 ref: main
 path: skills/research.md

model:
  provider: AnthropicVertexAI
  name: claude-sonnet-4

mcp:
 - server: studio-gemara-mcp
 tools:
 - validate_gemara_artifact
 - migrate_gemara_artifact

a2a:
  skills:
    - id: <skill-id>
      name: <Human-Readable Skill Name>
      description: >-
        What this A2A skill does. Shown in the platform dashboard.
      tags: [tag1, tag2]
```

### 3. Write `prompt.md`

Keep the prompt focused on **workflow only**. Platform identity and constraints are injected automatically from the platform prompt ConfigMap.

```markdown
You specialize in <Layer N (Name)>: <what you do>.

## Workflow

1. **Gather context**: Ask the user for relevant context or use available MCP tools.
2. **Analyze**: Apply your skills to the input data.
3. **Author**: Produce the Gemara artifact YAML.
4. **Validate**: Call `validate_gemara_artifact`. Fix and re-validate.
5. **Return**: Return validated artifact YAML.
```

**Rules:**
- Do NOT repeat platform constraints (they come from the platform prompt ConfigMap)
- Do NOT embed domain knowledge (put it in a SKILL.md instead)
- DO define the step-by-step workflow
- DO specify required/optional inputs
- DO define interaction style (propose defaults vs. interrogate)

### 4. Register in the Workbench

Add the agent to the `AGENT_DIRECTORY` JSON in `studio-deploy/charts/complytime/values.yaml` → `workbench.agentDirectory`. The workbench reads this env var to populate `/workbench/agents` and route A2A traffic.

```json
[
  {
    "name": "studio-<agent-name>",
    "description": "One-line description",
    "url": "http://localhost:8080/",
    "skills": [{"id": "<skill-id>", "name": "<Skill Name>"}]
  }
]
```

The agent must serve A2A on the URL specified. For co-located agents (running in the same container as the workbench), use `http://localhost:<port>/`.

**Framework-agnostic:** The agent can use any framework (LangGraph, Google ADK, CrewAI, custom) as long as it speaks A2A.

### 5. Set agent prompt

Add the agent's system prompt to `studio-deploy/charts/complytime/values.yaml` under `agentPrompts.<name>`, or override via `--set` at deploy time. The prompt ConfigMap is rendered by Helm from these values.

---

## Platform Constraints — Auth & Data

Authentication and data persistence changed significantly in 2026-05.

| Concern | Previous | Current |
|:--|:--|:--|
| Auth | In-process OIDC (gateway managed sessions) | OAuth2 Proxy sidecar — gateway trusts `X-Forwarded-*` headers |
| Persistence | ClickHouse primary | PostgreSQL primary (ClickHouse optional via FDW) |
| Token bypass | Static `STUDIO_API_TOKEN` in values | Auto-generated secret (`studio-cookie-secret`) |

**Key references:**
- Auth design: `openspec/changes/generic-oidc-auth/design.md`
- Helm auth values: `studio-deploy/charts/complytime/values.yaml` → `auth.oauth2Proxy.*`
- Architecture: `docs/design/architecture.md`
- ADR: `docs/decisions/postgres-with-extensions.md`

**Agent implications:** Agents communicate with the gateway via the internal port (8081). They do not pass through OAuth2 Proxy. Agent-to-gateway auth is network-enforced via NetworkPolicy, not token-based.

---

## Creating Skills

Skills are reusable knowledge packs. Any agent can reference them.

### Internal skills (complytime-agents repo)

```
skills/<skill-name>/
└── SKILL.md
```

### SKILL.md format

```markdown
---
name: <skill-name>
description: <One-line description of the skill>
---

# <Skill Title>

<Domain knowledge, decision tables, classification logic, etc.>
```

**Rules:**
- Frontmatter (`---`) with `name` and `description` is required
- Content is injected into the agent's context when the skill is loaded
- Keep skills focused — one concern per skill
- Skills should contain knowledge, not workflow (workflow goes in `prompt.md`)

### External skills (separate repo)

Reference in `agent.yaml`:

```yaml
skills:
  - repo: https://github.com/org/skill-repo.git
    ref: main
    path: skills/skill-name
```

The workbench container clones the repo at build time and mounts the skill under `/skills/<skill-name>/`.

---

## Architecture

```
┌──────────────────────────────────────────────────────┐
│  Browser → Nginx (studio-ui)                         │
│              ├── /api/*        → Data Platform (Go)  │
│              ├── /auth/*       → Data Platform       │
│              ├── /workbench/*  → Studio Workbench    │
│              └── /*            → static SPA          │
├──────────────────────────────────────────────────────┤
│  Studio Workbench (complytime-agents)                │
│    ├── /workbench/agents       → agent directory     │
│    ├── /workbench/a2a/{name}   → reverse-proxy A2A   │
│    ├── /workbench/validate     → gemara-mcp          │
│    ├── /workbench/publish      → oras-mcp            │
│    └── /workbench/chat/history → chat state           │
│    ↕                                                 │
│    LangGraph agents (in-process, port 8080)          │
├──────────────────────────────────────────────────────┤
│  Data Platform (complytime-studio)                   │
│    ├── Evidence, Posture, Certs, Policies, AuditLogs │
│    ├── NATS certifier pipeline                       │
│    └── OAuth2 Proxy + session management             │
├──────────────────────────────────────────────────────┤
│  MCP servers (tools)                                 │
│    ├── studio-mcp    → platform data access          │
│    ├── gemara-mcp    → artifact validation           │
│    └── oras-mcp      → OCI publish/browse            │
└──────────────────────────────────────────────────────┘
```

### Deployment topology

The workbench container runs both the Starlette HTTP server (port 8090) and the LangGraph agent (port 8080) in a single pod. The workbench reverse-proxies A2A traffic to the co-located agent. MCP servers are sidecar containers or standalone services accessible over HTTP.

### MCP server transport

| Server | Transport | Auth Model |
|:--|:--|:--|
| studio-gemara-mcp | http | Static (no user auth) |
| studio-mcp | http | Typed `studio://` resources + tools (platform data access) |
| studio-oras-mcp | http | Gateway proxy handles auth |

### On-Behalf-Of (OBO) flow

```
Browser → Nginx → Studio Workbench → Agent → MCP Server
  │                   │                         │
  │ cookie/bearer     │ propagates              │
  │                   │ Authorization header    │
```

The workbench propagates `Authorization` headers from the browser through to agent → MCP calls for user-scoped operations.

---

## Existing Agents

| Agent | Framework | Container | A2A skill `id` |
|:--|:--|:--|:--|
| studio-assistant | LangGraph (Python) | `studio-workbench` (co-located) | `compliance-assistant` |

Canonical spec: [`agents/assistant/agent.yaml`](https://github.com/complytime/complytime-agents/blob/main/agents/assistant/agent.yaml) in the complytime-agents repo.

All agents run inside the Studio Workbench container. The workbench's `/workbench/a2a/{name}` endpoint reverse-proxies A2A traffic to the agent running on `localhost:8080`. The `AGENT_DIRECTORY` environment variable (JSON) declares available agents and their URLs.

**studio-assistant internal skills** (in complytime-agents `skills/*/SKILL.md`):

| Skill | Purpose |
|:--|:--|
| `studio-audit` | Classification criteria, coverage mapping, PostgreSQL schema reference |
| `posture-check` | Pre-audit readiness — cadence, provenance, method, evidence fitness |

External git-mounted skills (see `agent.yaml`): `research.md`, `gemara.md` from `rhaml-23/prompt`.

> Threat modeler and policy composer have been removed. Artifact authoring is handled by engineers using local tooling (Cursor, Claude Code) + gemara-mcp. See `docs/decisions/audit-dashboard-pivot.md`.

---

## Git Commit Conventions

All commits created by agents MUST:

1. Use `-S -s` to GPG-sign and add a `Signed-off-by` trailer.
2. Include an `Assisted-by: Cursor (<model used>)` trailer.

```bash
git commit -S -s -m "$(cat <<'EOF'
feat: description of the change

Assisted-by: Cursor (claude-sonnet-4-20250514)
EOF
)"
```

---

## Checklist

- [ ] `agents/<name>/agent.yaml` in complytime-agents with name, description, skills, mcp, a2a
- [ ] `agents/<name>/prompt.md` in complytime-agents with workflow steps only
- [ ] Skills extracted to `skills/<name>/SKILL.md` in complytime-agents if reusable
- [ ] Agent entry added to `workbench.agentDirectory` JSON in `studio-deploy/charts/complytime/values.yaml`
- [ ] Agent prompt added to `agentPrompts` in `studio-deploy/charts/complytime/values.yaml`
- [ ] Agent process started in `Dockerfile.workbench` entrypoint
- [ ] Agent serves A2A on declared URL (co-located: `localhost:<port>`)
- [ ] MCP server URLs available via workbench container env vars

---

## QE Instructions

When archiving a change that modifies an agent, generate test instructions covering the happy path and edge cases. This ensures changes are verifiable before merge.

### Happy Path

Test the agent's primary workflow end-to-end through the workbench.

| Step | Action | Expected Result |
|:-----|:-------|:----------------|
| 1 | Open workbench, click "+ New Job" | Agent picker displays the agent with updated description |
| 2 | Select the agent, provide valid inputs per `prompt.md` Required Inputs | Job created, SSE stream connects, agent responds |
| 3 | Follow the guided conversation through each phase | Agent proposes defaults, presents tables, waits for confirmation at each phase boundary |
| 4 | Confirm/adjust at each decision point | Agent proceeds to next phase without re-asking resolved questions |
| 5 | Verify artifact appears in editor | `detectDefinition()` identifies correct type, YAML renders in editor pane |
| 6 | Click Validate | `validate_gemara_artifact` returns valid for the expected definition |
| 7 | Click Download YAML | YAML file downloaded with correct filename |
| 8 | Click Publish (if applicable) | OCI bundle pushed, reference and digest returned |

### Edge Cases

Test boundary conditions and error handling.

| Case | Action | Expected Result |
|:-----|:-------|:----------------|
| Missing required input | Start job without one or more required inputs | Agent responds with the specific guidance message defined in `prompt.md` (not a generic error) |
| Invalid input | Provide malformed YAML or wrong artifact type | Agent identifies the issue, requests correction |
| MCP server unavailable | Start job when a required MCP server is down | Agent reports the specific unavailability (not a hang or generic failure) |
| Multi-turn interruption | Close browser mid-conversation, reopen | Job resumes from last status; SSE reconnects or reports disconnected |
| Validation failure | Manually edit artifact YAML to be invalid, click Validate | Validation returns specific errors referencing the CUE definition |
| Empty evidence (assistant) | Query a policy_id/timeline with no evidence data | Agent classifies all criteria as Gap, does not fabricate evidence |
| Cadence gap (assistant) | Evidence exists but with missing assessment cycles | Agent produces Findings (not Observations) for each missing cycle with specific dates |
| No MappingDocuments (assistant) | Start audit without MappingDocuments | Agent offers internal-only analysis, skips cross-framework phase |
| Partial mapping strength (assistant) | MappingDocument has targets with low strength scores | Coverage matrix shows Partially/Weakly Covered with correct strength values |
| Concurrent job | Attempt to start a second job while one is active | "+ New Job" button disabled with tooltip explaining why |

### Helm Verification

The Helm chart lives in [studio-deploy](https://github.com/complytime/studio-deploy). After any agent change, verify the Kubernetes deployment renders correctly from that repo.

| Check | Command (from studio-deploy/) | Expected Result |
|:------|:------------------------------|:----------------|
| Chart renders | `make helm-template` | Workbench deployment contains updated `AGENT_DIRECTORY` env |
| Values match | Compare `charts/complytime/values.yaml` workbench.agentDirectory | Description and skills match `agent.yaml` |
| No stale refs | Search chart templates for old descriptions | Zero matches |

## Convention Packs

This repository uses convention packs scaffolded by
unbound-force. Agents MUST read the applicable pack(s)
before writing or reviewing code.

- `.opencode/uf/packs/default.md`
- `.opencode/uf/packs/default-custom.md`
- `.opencode/uf/packs/severity.md`
- `.opencode/uf/packs/content.md`
- `.opencode/uf/packs/content-custom.md`
- `.opencode/uf/packs/go.md`
- `.opencode/uf/packs/go-custom.md`

## Boundary Rules

- `complytime-studio` (Go) owns data CRUD, auth, and certifier pipeline only
- `complytime-agents` (Python) owns A2A routing, agent lifecycle, and Gemara/OCI tooling
- `studio-ui` owns the SPA, Nginx routing, and all client-side code
- `studio-deploy` owns deployment orchestration (Docker Compose, Helm umbrella)
- Agents MUST NOT import `internal/store` or `internal/postgres` at runtime
- Agents access platform data exclusively through `studio-mcp` MCP resources
- All cross-component communication uses REST API or MCP protocol
- Workbench endpoints live under `/workbench/*` path prefix (never `/api/*`)
