## 1. Helm Chart — Per-Agent ModelConfig CRDs

- [x] 1.1 Add `agents` override block to `values.yaml` with empty defaults for `threat-modeler`, `gap-analyst`, `policy-composer`
- [x] 1.2 Refactor `model-config.yaml` template to loop over `agentDirectory` entries and render one `ModelConfig` per agent, resolving provider/model from `agents.<name>.model.*` with global fallback
- [x] 1.3 Include conditional `anthropicVertexAI` / `geminiVertexAI` blocks per resolved provider in each `ModelConfig`
- [x] 1.4 Remove the old shared `studio-model` CRD (replaced by per-agent CRDs)
- [x] 1.5 Update `agent-specialists.yaml` to set `modelConfig: studio-model-<agent-name>` instead of hardcoded `studio-model`

## 2. Agent Directory — Model Metadata

- [x] 2.1 Add `model.provider` and `model.name` fields to each `agentDirectory` entry in `values.yaml`
- [x] 2.2 Verify `GET /api/agents` gateway response includes the new `model` field per agent

## 3. Makefile — Per-Agent Env Vars

- [x] 3.1 Add `THREAT_MODELER_MODEL_PROVIDER` / `THREAT_MODELER_MODEL_NAME` optional env vars with `--set` flags
- [x] 3.2 Add `GAP_ANALYST_MODEL_PROVIDER` / `GAP_ANALYST_MODEL_NAME` optional env vars with `--set` flags
- [x] 3.3 Add `POLICY_COMPOSER_MODEL_PROVIDER` / `POLICY_COMPOSER_MODEL_NAME` optional env vars with `--set` flags

## 4. Canonical Agent Definitions

- [x] 4.1 Update `agents/threat-modeler/agent.yaml` model block comment to note Helm override takes precedence
- [x] 4.2 Update `agents/gap-analyst/agent.yaml` model block comment
- [x] 4.3 Update `agents/policy-composer/agent.yaml` model block comment

## 5. Workbench — Agent Picker Model Badge

- [x] 5.1 Update agent picker card component to render a model badge from the `model` field in the agent directory response
- [x] 5.2 Handle missing `model` field gracefully (omit badge, render all other content)

## 6. Validation

- [x] 6.1 Run `helm template` and verify three `ModelConfig` CRDs render correctly with global defaults
- [x] 6.2 Run `helm template` with per-agent overrides and verify mixed provider/model combinations
- [x] 6.3 Deploy to Kind and confirm all three agent pods start with their respective ModelConfigs
