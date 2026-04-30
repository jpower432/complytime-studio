# Design: Agent Governance

## Layer 1: Constitution as Skill

### File: `skills/constitution/SKILL.md`

```markdown
---
name: constitution
description: >-
  Behavioral mandates for all compliance agents — evidence integrity,
  source citation, scope boundaries, output standards.
---

# Agent Constitution

## Evidence Integrity
- Never fabricate, infer, or extrapolate evidence. If data is missing, classify as Gap.
- Always cite the ClickHouse query or artifact reference that produced a claim.

## Scope Boundaries
- Refuse requests outside compliance domain (code generation, general Q&A).
- When uncertain about scope, ask the user rather than guessing.

## Output Standards
- Use ## markdown headings for sections (not numbered lists as headings).
- Fenced JSON blocks must be valid JSON.
- No emoji in body text.

## Decision Ordering
- Present data before recommendations.
- Flag assumptions explicitly.
- Distinguish Observations (evidence exists, needs review) from Findings (evidence insufficient).
```

Target: under 500 tokens. Invariant rules only — nothing that changes per persona or per deployment.

### Integration

- Added to every agent's `agent.yaml` skills block
- Prompt assembly order: `platform.md` → constitution → persona → domain skills
- Studio's `platform.md` provides identity; constitution provides behavior

## Layer 2: Quality Gates as MCP Tool

### Tool: `validate_command_output`

Added to `gemara-mcp` server (reuses existing MCP infrastructure).

**Input:**

```json
{
  "command_name": "daily-brief",
  "output_text": "## Program Status\n...",
  "expected_sections": ["Program Status", "Action Items", "Blockers"]
}
```

**Output:**

```json
{
  "structural": {
    "passed": true,
    "missing_sections": []
  },
  "format": {
    "passed": false,
    "issues": ["Invalid JSON in fenced block at line 42"]
  },
  "overall": "fail"
}
```

### Checks

**Structural (Gate 2):**
- Each entry in `expected_sections` appears as a `##` heading in the output
- `expected_sections` comes from command spec YAML frontmatter (`outputs` field)
- Agent passes them explicitly — no ConfigMap coupling in the gateway

**Format (Gate 3):**
- No numbered headings as section headers
- No emoji in body text
- Fenced ` ```json ` blocks parse as valid JSON

### Agent Integration

Agents call `validate_command_output` after producing a command response. On failure:
1. Agent attempts self-correction (re-generate the failing section)
2. If still failing after one retry, include quality gate result in response metadata
3. Workbench renders pass/fail badge on command output

### A2A Response Metadata

Agents include quality gate results in A2A task status metadata:

```json
{
  "quality_gate": {
    "structural": { "passed": true },
    "format": { "passed": false, "issues": ["..."] },
    "overall": "fail"
  }
}
```

Gateway passes this through unchanged. No gateway logic, no buffering.

### Workbench Rendering

- Quality gate badge: green check / red X next to command output
- Expandable issues list when a gate fails
- Badge only appears for command outputs (not free-form chat)

### Why MCP Tool (Not Gateway Interceptor)

| Concern | MCP Tool | Gateway Interceptor |
|:--|:--|:--|
| Stream buffering | None — agent validates before responding | Must buffer full stream |
| Self-correction | Agent can fix and retry | Gateway can only annotate |
| Coupling | Agent owns command context | Gateway needs ConfigMap of all command specs |
| Latency | Part of agent processing time | Adds post-stream delay |
| Complexity | One tool on existing MCP server | New SSE event type + interceptor logic |

## Tests

| Test | Validates |
|:--|:--|
| `TestValidateOutput_AllSectionsPresent` | Structural pass when all headings found |
| `TestValidateOutput_MissingSections` | Structural fail with correct missing list |
| `TestValidateOutput_ValidJSON` | Format pass for valid fenced JSON |
| `TestValidateOutput_InvalidJSON` | Format fail with line reference |
| `TestValidateOutput_NumberedHeadings` | Format fail for `1.` style headings |
| `TestValidateOutput_Emoji` | Format fail for emoji in body |
| `TestConstitutionTokenCount` | Constitution skill is under 500 tokens |
