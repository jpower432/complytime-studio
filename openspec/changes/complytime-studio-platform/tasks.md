## 1. Rebrand: GIDE → ComplyTime Studio

- [x] 1.1 Copy GIDE source into complytime-studio repo (Go code, prompts, templates, workbench, charts, deploy, scripts, Dockerfiles)
- [x] 1.2 Update `go.mod` module path to `github.com/complytime/complytime-studio` and fix all import paths
- [x] 1.3 Rename agent names in Go code: `gide-orchestrator` → `studio-orchestrator`, `gide-threat-modeler` → `studio-threat-modeler`, `gide-policy` → `studio-policy-composer`
- [x] 1.4 Replace "GIDE" with "ComplyTime Studio" in all prompt `.md` files
- [x] 1.5 Rename Helm chart directory `charts/gide/` → `charts/complytime-studio/` and update `Chart.yaml` name, description, home, sources
- [x] 1.6 Update Helm templates: Agent CRD `metadata.name`, `NOTES.txt`, `_helpers.tpl`
- [x] 1.7 Rename Docker image references from `gide-agents` to `studio-agents` in Dockerfiles, docker-compose.yaml, Makefile, and Helm values
- [x] 1.8 Update `Makefile` targets: rename `gide-up/gide-down/gide-template` → `studio-up/studio-down/studio-template`, update Kind cluster name
- [x] 1.9 Update `README.md` with ComplyTime Studio branding and usage
- [x] 1.10 Update workbench HTML title and header from "GIDE" to "ComplyTime Studio"

## 2. Remove Mapper Agent

- [x] 2.1 Delete `internal/agents/mapper.go` and `internal/agents/mapper_prompt.md`
- [x] 2.2 Remove mapper agent creation and A2A server startup from `cmd/agents/main.go`
- [x] 2.3 Remove mapper-related config fields (`MapperPort`) and A2A URL method from `config.go`
- [x] 2.4 Remove mapper references from orchestrator prompt (routing rules, specialist listing)
- [x] 2.5 Remove `internal/agents/data/nist_sp800_53_rev5.json` and `internal/agents/data/embed.go` (embedded reference data was mapper-only)

## 3. Gap Analyst Agent

- [x] 3.1 Create `internal/agents/gap_analyst.go` with `NewGapAnalyst(cfg, model)` constructor following threatmodeler.go pattern
- [x] 3.2 Create `internal/agents/gap_analyst_prompt.md` with instructions for MappingDocument consumption and AuditLog production
- [x] 3.3 Wire gemara-mcp and github-mcp toolsets in the Gap Analyst constructor
- [x] 3.4 Add Gap Analyst A2A server startup in `cmd/agents/main.go` with `gap-analysis` skill in agent card
- [x] 3.5 Register Gap Analyst as orchestrator sub-agent
- [x] 3.6 Update orchestrator prompt with Gap Analyst routing rules (MappingDocument input → AuditLog output, L7 scope)
- [x] 3.7 Add Gap Analyst port config (`GAP_ANALYST_PORT`, default 8002) and A2A URL method to `config.go`
- [x] 3.8 Update Helm chart Agent CRD template with Gap Analyst port environment variable

## 4. Publish Bundle Tool

- [x] 4.1 Add `oras-go` dependency to `go.mod`
- [x] 4.2 Create `internal/publish/media_types.go` with centralized Gemara artifact type → OCI media type mapping table
- [x] 4.3 Create `internal/publish/bundle.go` implementing OCI manifest assembly from artifact YAML strings using oras-go
- [x] 4.4 Create `internal/publish/sign.go` with signing wrapper (notation-go or cosign-go) for manifest digest signing
- [x] 4.5 Create `internal/publish/tool.go` registering `publish_bundle` as an ADK `tool.Func` with inputs: artifacts[], target, tag, sign
- [x] 4.6 Register the publish tool on the orchestrator in `orchestrator.go`
- [x] 4.7 Update orchestrator prompt with bundle assembly and publishing instructions (replace "print oras command" with tool invocation)
- [x] 4.8 Write tests for media type mapping, bundle assembly, and publish tool registration

## 5. Workbench UI Rebuild

- [x] 5.1 Choose UI framework (Preact + Vite) and set up build tooling in `workbench/` with output to `workbench/dist/`
- [x] 5.2 Update `workbench/embed.go` to embed from `dist/` instead of raw source files
- [x] 5.3 Add `make workbench-build` target to compile frontend assets
- [x] 5.4 Implement missions view: list, create, resume, status badges
- [x] 5.5 Implement chat panel: A2A JSON-RPC streaming, message rendering, HITL reply input
- [x] 5.6 Implement artifact panel: tabbed CodeMirror YAML editor, validate button, copy button
- [x] 5.7 Implement publish workflow: publish button in artifact toolbar, publish dialog with registry reference + signing toggle, progress/result display
- [x] 5.8 Implement registry browser: sidebar navigation, registry URL input, repository/tag listing, layer inspection
- [x] 5.9 Add `/api/registry/*` proxy endpoints on the orchestrator backend for registry browser requests (delegates to oras-mcp)
- [x] 5.10 Apply ComplyTime Studio branding: header, colors, page title

## 6. Integration and Deployment

- [x] 6.1 Update `docker-compose.yaml` with renamed services, ports, and image names
- [x] 6.2 Update `.env.example` with any new environment variables (GAP_ANALYST_PORT, signing config)
- [x] 6.3 Update Kind setup script (`deploy/kind/setup.sh`) with renamed cluster and chart references
- [x] 6.4 Update Helm `values.yaml` with Gap Analyst port and any new MCP server config
- [ ] 6.5 Verify `make compose-up` starts all agents with correct routing
- [ ] 6.6 Verify `make studio-up` deploys to Kind with kagent BYO Agent CRD
- [x] 6.7 Run `make test` and `make lint` clean
