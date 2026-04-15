## Why

ComplyTime needs a multi-agent platform for authoring, validating, and publishing Gemara GRC artifacts as OCI bundles. The prototype exists in `github.com/jpower432/gide` — a working Go ADK + A2A + kagent BYO system with threat modeling, mapping, and policy composition agents. This change rebrands it as ComplyTime Studio, replaces the Mapper with a Gap Analyst, adds a native OCI publishing pipeline, and promotes the workbench UI to a first-class product.

The compliance workflow this enables: specialist agents author criteria-phase artifacts (L1-L3), the orchestrator publishes signed bundles to an OCI registry, and downstream tools (complyctl, Lula) consume those bundles for runtime evaluation. The OCI registry is the bridge between authoring and enforcement.

## What Changes

- **Rebrand** from GIDE to ComplyTime Studio across Go module, agent names, prompts, Helm chart, Dockerfiles, and UI
- **Replace Mapper agent** with a Gap Analyst that consumes MappingDocuments as input and produces AuditLog artifacts (L7) classifying coverage as Gap/Finding/Observation/Strength
- **Add native `publish_bundle` tool** using oras-go for atomic assemble → push → sign (notation-go or cosign-go), replacing the current "print an oras command" approach
- **Multi-provider model support** carried forward: Gemini (Vertex), Anthropic (Vertex + direct), extensible via ADK `model.LLM` interface
- **Stateless session strategy**: in-memory for local dev, kagent controller DB for Kubernetes — no custom persistence layer
- **First-class workbench UI**: missions, chat, YAML editor + validation, OCI publishing, and artifact pulling from registries

## Capabilities

### New Capabilities
- `gap-analyst-agent`: Specialist agent that consumes MappingDocuments and reference frameworks, produces AuditLog artifacts with coverage classification per Gemara L7 schema
- `publish-bundle`: Native Go tool (oras-go) for OCI bundle assembly, push, and signing — registered as an ADK tool on the orchestrator
- `workbench-publishing`: UI workflow for selecting a registry target, confirming artifacts, and triggering publish + sign from the artifact panel
- `workbench-registry-browser`: UI view for browsing OCI registries, inspecting bundle layers, and pulling artifacts into the editor
- `studio-rebrand`: Rename GIDE → ComplyTime Studio across all code, config, prompts, Helm chart, and documentation

### Modified Capabilities

## Impact

- **Go module**: `github.com/jpower432/gide` → `github.com/complytime/complytime-studio`
- **Dependencies added**: `oras-go` (OCI push), `notation-go` or `cosign-go` (signing)
- **Dependencies removed**: Mapper agent code, mapper prompt, SCF/NIST embedded reference data (Gap Analyst doesn't need them — MappingDocument is input)
- **Agent topology**: 4 agents → 4 agents (Orchestrator, Threat Modeler, Gap Analyst, Policy Composer). Mapper dropped.
- **Helm chart**: `charts/gide/` → `charts/complytime-studio/`, Agent CRD names updated
- **Docker images**: renamed from `gide-agents` to `studio-agents`
- **Workbench**: vanilla JS prototype replaced with framework-based SPA (still embedded via `go:embed`)
- **API surface**: new `/api/publish` proxy endpoint; existing `/api/validate`, `/api/migrate` unchanged
