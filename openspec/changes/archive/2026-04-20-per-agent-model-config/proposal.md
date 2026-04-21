## Why

All three agents share a single `ModelConfig` CRD (`studio-model`). Switching providers or models requires redeploying every agent, and there is no way to assign a model suited to a specific task. Threat modeling benefits from strong reasoning (Claude Sonnet), while gap analysis may perform well on a faster model (Gemini Flash). A shared config also blocks A/B evaluation and cost attribution per agent.

## What Changes

- Introduce per-agent `ModelConfig` CRDs (e.g., `studio-model-threat-modeler`, `studio-model-gap-analyst`, `studio-model-policy-composer`) each referencing a provider/model pair.
- Keep a shared default `ModelConfig` (`studio-model`) as the fallback when no per-agent override is set.
- Extend `values.yaml` with an `agents.<name>.model` override block. When absent, the agent inherits the global `model.*` values.
- Update `agent-specialists.yaml` to reference the per-agent `ModelConfig` name instead of hardcoded `studio-model`.
- Update `agent.yaml` canonical definitions to declare the model as an overridable default.
- Surface per-agent model info in the agent directory (`/api/agents`) response so the workbench can display which model backs each agent.

## Capabilities

### New Capabilities
- `per-agent-model`: Per-agent model configuration — independent ModelConfig CRDs per agent with global fallback.

### Modified Capabilities
- `agent-picker`: Agent picker cards display the model name/provider backing each agent.

## Impact

- **Helm chart**: New `model-configs.yaml` template (or loop in existing `model-config.yaml`). `agent-specialists.yaml` references per-agent model names.
- **values.yaml**: New `agents.<name>.model` override structure.
- **agent.yaml**: `model` block becomes the declared default; Helm override takes precedence.
- **Gateway**: `agentDirectory` entries gain a `model` field; `/api/agents` response includes it.
- **Workbench**: Agent picker cards show model badge. No functional change to job creation.
- **Makefile**: Per-agent model env vars (e.g., `THREAT_MODEL_PROVIDER`) or a single JSON override.
