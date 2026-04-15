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
  # Internal skills (from this repo)
  - path: skills/gemara-layers
  # External skills (from other repos)
  - repo: https://github.com/org/skill-repo.git
    ref: main
    path: skills/skill-name

model:
  provider: AnthropicVertexAI
  name: claude-sonnet-4-20250514

mcp:
  - server: studio-gemara-mcp
    tools:
      - validate_gemara_artifact
      - migrate_gemara_artifact
  - server: studio-github-mcp
    allowedHeaders: [Authorization]
    tools:
      - get_file_contents
      - search_code
      - search_repositories

a2a:
  skills:
    - id: <skill-id>
      name: <Human-Readable Skill Name>
      description: >-
        What this A2A skill does. Shown in the platform dashboard.
      tags: [tag1, tag2]
```

### 3. Write `prompt.md`

Keep the prompt focused on **workflow only**. Platform identity and constraints are injected automatically from `agents/platform.md`.

```markdown
You specialize in <Layer N (Name)>: <what you do>.

## Workflow

1. **Gather context**: Use github-mcp tools to fetch relevant files.
2. **Analyze**: Apply your skills to the input data.
3. **Author**: Produce the Gemara artifact YAML.
4. **Validate**: Call `validate_gemara_artifact`. Fix and re-validate.
5. **Return**: Return validated artifact YAML.
```

**Rules:**
- Do NOT repeat platform constraints (they come from `agents/platform.md`)
- Do NOT embed domain knowledge (put it in a SKILL.md instead)
- DO define the step-by-step workflow
- DO specify required/optional inputs
- DO define interaction style (propose defaults vs. interrogate)

### 4. Register in Helm

Add the agent to `charts/complytime-studio/templates/agent-specialists.yaml`. The Helm template reads `agent.yaml` and renders a kagent Declarative Agent CRD.

### 5. Sync prompts

```bash
make sync-prompts
```

---

## Creating Skills

Skills are reusable knowledge packs. Any agent can reference them.

### Internal skills (this repo)

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

kagent clones the repo and mounts the skill under `/skills/<skill-name>/` in the agent container.

---

## Architecture

```
┌─────────────────────────────────────────────┐
│ agents/platform.md  (shared identity)       │
│  ↓ injected via promptTemplate ConfigMap    │
├─────────────────────────────────────────────┤
│ agents/<name>/prompt.md  (workflow)         │
│  ↓ embedded in Helm systemMessage           │
├─────────────────────────────────────────────┤
│ skills/  (knowledge packs)                  │
│  ↓ mounted via kagent gitRefs               │
├─────────────────────────────────────────────┤
│ MCP servers  (tools)                        │
│  ↓ declared in agent.yaml mcp block         │
└─────────────────────────────────────────────┘
```

### Prompt composition

The final system prompt is assembled by kagent at runtime:

1. **Platform layer** — `agents/platform.md` rendered into a ConfigMap, included via `{{include "platform/platform"}}`
2. **Agent layer** — `agents/<name>/prompt.md` embedded directly in the Helm template

### MCP server transport

| Server | Transport | Auth Model |
|:--|:--|:--|
| studio-gemara-mcp | stdio | Static (no user auth) |
| studio-clickhouse-mcp | stdio | Static credentials via Secret |
| studio-github-mcp | streamablehttp | Per-request Bearer token (OBO) |
| studio-oras-mcp | stdio | Gateway proxy handles auth |

Servers using `streamablehttp` accept per-request `Authorization` headers propagated from A2A requests via kagent's `allowedHeaders` mechanism.

### On-Behalf-Of (OBO) flow

```
Browser → Gateway → A2A Agent Pod → MCP Server
  │         │            │              │
  │ cookie  │ inject     │ allowedHeaders
  │         │ Bearer     │ propagates   │
  │         │ header     │ Authorization│
```

The gateway extracts the user's GitHub token from the session cookie and injects it as an `Authorization: Bearer` header on A2A requests. kagent propagates this header to MCP tool calls for servers with `allowedHeaders: [Authorization]`.

---

## Existing Agents

| Agent | Layer | A2A Skills |
|:--|:--|:--|
| studio-threat-modeler | L2 (Controls) | threat-assessment, control-authoring |
| studio-gap-analyst | L7 (Audit) | gap-analysis |
| studio-policy-composer | L3 (Policy) | policy-authoring |

---

## Checklist

- [ ] `agents/<name>/agent.yaml` with name, description, skills, mcp, a2a
- [ ] `agents/<name>/prompt.md` with workflow steps only
- [ ] Skills extracted to `skills/<name>/SKILL.md` if reusable
- [ ] Agent added to `agent-specialists.yaml` Helm template
- [ ] `make sync-prompts` copies prompt to chart
- [ ] `allowedHeaders: [Authorization]` on github-mcp tool ref (for OBO)
