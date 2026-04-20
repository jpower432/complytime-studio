## ADDED Requirements

### Requirement: Each agent has a dedicated ModelConfig CRD
The Helm chart SHALL render one `ModelConfig` CRD per agent in the `agentDirectory`. Each CRD SHALL be named `studio-model-<agent-short-name>` (e.g., `studio-model-threat-modeler`).

#### Scenario: Three agents deployed
- **WHEN** the Helm chart is rendered with the default `agentDirectory` containing three agents
- **THEN** three `ModelConfig` CRDs are created: `studio-model-threat-modeler`, `studio-model-gap-analyst`, `studio-model-policy-composer`
- **THEN** each agent's `declarative.modelConfig` references its own `ModelConfig` name

### Requirement: Per-agent model override via values.yaml
The Helm chart SHALL support an `agents.<name>.model` block in `values.yaml` that overrides `provider` and `name` for a specific agent. When the block is absent or empty, the agent SHALL inherit the global `model.provider` and `model.name`.

#### Scenario: Agent with explicit override
- **WHEN** `values.yaml` sets `agents.gap-analyst.model.provider: GeminiVertexAI` and `agents.gap-analyst.model.name: gemini-2.5-flash`
- **THEN** `studio-model-gap-analyst` renders with `provider: GeminiVertexAI` and `model: gemini-2.5-flash`
- **THEN** `studio-model-threat-modeler` and `studio-model-policy-composer` render with the global `model.provider` and `model.name`

#### Scenario: No per-agent override
- **WHEN** `values.yaml` does not set `agents.policy-composer.model`
- **THEN** `studio-model-policy-composer` renders with the global `model.provider` and `model.name`

### Requirement: Provider-specific config inherits correctly
Each per-agent `ModelConfig` SHALL include the provider-specific configuration block (`anthropicVertexAI` or `geminiVertexAI`) matching the resolved provider for that agent.

#### Scenario: Mixed providers across agents
- **WHEN** `studio-threat-modeler` uses `AnthropicVertexAI` and `studio-gap-analyst` uses `GeminiVertexAI`
- **THEN** `studio-model-threat-modeler` includes the `anthropicVertexAI` block with `projectID` and `location`
- **THEN** `studio-model-gap-analyst` includes the `geminiVertexAI` block with `projectID` and `location`

### Requirement: Credentials secret is shared across all ModelConfigs
All per-agent `ModelConfig` CRDs SHALL reference the same `apiKeySecret` as the global config. Per-agent credential overrides are not supported.

#### Scenario: Global credentials apply to all agents
- **WHEN** `model.anthropicVertexAI.credentialsSecret` is set to `studio-gcp-credentials`
- **THEN** all `ModelConfig` CRDs include `apiKeySecret: studio-gcp-credentials`

### Requirement: Agent directory includes model metadata
Each entry in the `agentDirectory` SHALL include a `model` object with `provider` and `name` fields reflecting the resolved model for that agent.

#### Scenario: Gateway returns model info
- **WHEN** a client calls `GET /api/agents`
- **THEN** each agent entry includes `model.provider` and `model.name` matching the deployed `ModelConfig`

### Requirement: Makefile supports per-agent model overrides
The Makefile SHALL accept optional per-agent environment variables following the pattern `<AGENT_PREFIX>_MODEL_PROVIDER` and `<AGENT_PREFIX>_MODEL_NAME`. When unset, agents inherit `MODEL_PROVIDER` / `MODEL_NAME`.

#### Scenario: Override gap-analyst model
- **WHEN** user runs `GAP_ANALYST_MODEL_PROVIDER=GeminiVertexAI GAP_ANALYST_MODEL_NAME=gemini-2.5-flash make studio-up`
- **THEN** the Helm install includes `--set agents.gap-analyst.model.provider=GeminiVertexAI --set agents.gap-analyst.model.name=gemini-2.5-flash`

#### Scenario: No per-agent vars set
- **WHEN** user runs `make studio-up` without agent-specific env vars
- **THEN** no per-agent `--set` flags are passed and all agents inherit the global model
