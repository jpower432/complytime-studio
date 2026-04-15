## Why

The specialist agents (threat modeler, gap analyst, policy composer) contain zero custom Go logic. They are configuration — a prompt, MCP tool filters, and a model reference — wrapped in boilerplate constructors, switch statements, and an entire binary (`cmd/agents`). kagent's Declarative Agent CRD now supports Go runtime, MCP tool filtering via `toolNames`, and built-in A2A server configuration, making the BYO binary unnecessary. Eliminating it removes ~600 lines of wiring code, a Docker image, and the orchestrator routing layer (users pick specialists directly via the platform UI).

## What Changes

- **BREAKING**: Delete `cmd/agents/` binary and `internal/agents/` package entirely
- **BREAKING**: Delete orchestrator agent and routing skill (`agents/orchestrator.md`, `skills/orchestrator-routing/`)
- Replace BYO Agent CRDs in Helm templates with Declarative Agent CRDs (`runtime: go`)
- Move agent prompts from `//go:embed` Go files to a ConfigMap generated from markdown files
- Add `ModelConfig` CRD to Helm chart for model provider configuration
- Move publish bundle workflow from agent tool to gateway REST API
- Update gateway to expose specialist A2A agent cards as a directory for the frontend
- Create canonical agent definition files (`agents/<name>/agent.yaml` + `prompt.md`) as the source of truth, rendered into kagent CRDs by Helm

## Capabilities

### New Capabilities

- `declarative-agent-crds`: kagent Declarative Agent CRD rendering from canonical agent definitions, replacing BYO binary
- `agent-directory-api`: Gateway API endpoint exposing available specialist agent cards for frontend routing
- `gateway-publish-api`: Publish bundle workflow as a gateway REST endpoint instead of an agent function tool

### Modified Capabilities

_(none — no existing spec-level behavior changes)_

## Impact

- **Deleted code**: `cmd/agents/`, `internal/agents/` (~8 Go files), `Dockerfile` for agents image, `agents/orchestrator.md`, `skills/orchestrator-routing/`
- **Helm chart**: `agent-specialists.yaml` rewritten from BYO → Declarative; new `model-config.yaml` and `agent-prompts-configmap.yaml` templates
- **Dependencies**: `google.golang.org/adk` usage reduced to gateway/publish only; `github.com/Alcova-AI/adk-anthropic-go` removed from agents path; `github.com/a2aproject/a2a-go` removed from agents path (kagent handles A2A)
- **Gateway**: New `/api/agents` directory endpoint; `/api/publish` endpoint absorbs publish bundle logic from agent tool
- **Frontend**: Routes to specialists directly via A2A instead of through orchestrator
- **Local dev**: Requires kind/minikube with kagent operator for agent development; gateway and ingest remain standalone Go binaries
