## Context

All three agents (`studio-threat-modeler`, `studio-gap-analyst`, `studio-policy-composer`) reference a single shared `ModelConfig` CRD named `studio-model`. The Helm chart renders one `ModelConfig` from `model.*` values. Agent CRDs hardcode `modelConfig: studio-model`.

This creates three limitations:
1. Model changes are all-or-nothing — switching to Gemini Flash for gap analysis forces the same model on threat modeling.
2. No cost attribution — a single model config makes per-agent usage tracking impossible.
3. No A/B evaluation — comparing model quality across agents requires full redeployment cycles.

## Goals / Non-Goals

**Goals:**
- Each agent can run on a different provider/model combination
- Global defaults still work — zero config for users who want one model everywhere
- Per-agent model info is visible in the workbench agent picker
- `agent.yaml` canonical definitions declare a model preference that Helm can override

**Non-Goals:**
- Runtime model switching (hot-swap without redeploy) — out of scope
- Per-user model selection — belongs in a future multi-tenancy proposal
- Cost metering or billing — this enables attribution but does not implement it
- Model routing or load balancing between providers

## Decisions

### D1: One ModelConfig CRD per agent, global fallback

**Decision**: Render a dedicated `ModelConfig` CRD per agent (e.g., `studio-model-threat-modeler`). When no per-agent override is set in `values.yaml`, the CRD inherits the global `model.*` values.

**Alternatives considered**:
- *Single shared ModelConfig (current state)*: Cannot differentiate models per agent.
- *Override at agent CRD level only*: kagent's `modelConfig` is a name reference, not an inline spec. The CRD must exist.

**Rationale**: One CRD per agent is the minimum unit kagent supports. The global fallback keeps the zero-config path simple.

### D2: values.yaml override structure

**Decision**: Add an `agents` map in `values.yaml` keyed by agent short name. Each entry supports `model.provider` and `model.name` overrides. Missing keys inherit from the global `model.*` block.

```yaml
agents:
  threat-modeler:
    model:
      provider: AnthropicVertexAI
      name: claude-sonnet-4
  gap-analyst:
    model:
      provider: GeminiVertexAI
      name: gemini-2.5-flash
  policy-composer: {}  # inherits global
```

**Alternatives considered**:
- *Flat env vars per agent (THREAT_MODEL_PROVIDER)*: Doesn't scale and pollutes the Makefile.
- *JSON override file*: Adds complexity for a three-agent system.

**Rationale**: Nested YAML matches existing `values.yaml` conventions and works with `--set agents.gap-analyst.model.name=gemini-2.5-flash`.

### D3: Helm template renders per-agent ModelConfigs

**Decision**: Replace the single `model-config.yaml` template with a loop that renders one `ModelConfig` per agent entry in `agentDirectory`. Each resolves its provider/model by checking `agents.<name>.model.*` first, falling back to `model.*`.

**Rationale**: `agentDirectory` is the existing registry of deployed agents — iterating it avoids duplicating the agent list.

### D4: Agent directory includes model metadata

**Decision**: Extend `agentDirectory` entries with a `model` field containing `provider` and `name`. The gateway serializes this in the `/api/agents` response. The workbench renders it as a badge on agent picker cards.

**Rationale**: Users need visibility into which model powers each agent before starting a job. The data is already available at deploy time.

### D5: Makefile support

**Decision**: Add optional per-agent env vars following the pattern `<AGENT>_MODEL_PROVIDER` and `<AGENT>_MODEL_NAME` (e.g., `GAP_ANALYST_MODEL_PROVIDER`). These feed into `--set agents.<name>.model.*` flags. When unset, agents inherit the global `MODEL_PROVIDER` / `MODEL_NAME`.

**Rationale**: Keeps the simple `make studio-up` path working while enabling per-agent overrides for development.

## Risks / Trade-offs

| Risk | Mitigation |
|:--|:--|
| CRD proliferation — 3 ModelConfigs instead of 1 | Manageable at current scale. Review if agent count exceeds ~10. |
| Helm template complexity increases | Loop is <30 lines. Template tests via `helm template` catch regressions. |
| `agent.yaml` model field diverges from deployed config | `agent.yaml` declares the *default preference*. Helm override is authoritative. Document this clearly. |
| Vertex AI project/location may differ per provider | The template already handles `anthropicVertexAI` and `geminiVertexAI` blocks conditionally. Per-agent configs inherit the same logic. |
