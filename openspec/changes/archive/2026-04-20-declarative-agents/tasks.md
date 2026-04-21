## 1. Canonical Agent Definitions

- [x] 1.1 Create `agents/threat-modeler/agent.yaml` with name, description, MCP tools (gemara + github with tool filters), A2A skills, and model reference
- [x] 1.2 Move `internal/agents/threatmodeler_prompt.md` to `agents/threat-modeler/prompt.md`
- [x] 1.3 Create `agents/gap-analyst/agent.yaml` with MCP tools (gemara + github + `ClickHouse/mcp-clickhouse` with tool filters) and A2A skills
- [x] 1.4 Move `internal/agents/gap_analyst_prompt.md` to `agents/gap-analyst/prompt.md`
- [x] 1.5 Create `agents/policy-composer/agent.yaml` with MCP tools (gemara + github with tool filters) and A2A skills
- [x] 1.6 Move `internal/agents/policy_composer_prompt.md` to `agents/policy-composer/prompt.md`

## 2. Helm Chart — Declarative Agent CRDs

- [x] 2.1 Add `model-config.yaml` template rendering a kagent `ModelConfig` CRD from `values.yaml` model settings
- [x] 2.2 Add `agent-prompts-configmap.yaml` template rendering a ConfigMap from `agents/*/prompt.md` files
- [x] 2.3 Rewrite `agent-specialists.yaml` — replace BYO Agent CRDs with Declarative Agent CRDs (`runtime: go`, `systemMessageFrom`, `tools[].toolNames`, `a2aConfig`)
- [x] 2.4 Update `values.yaml` — replace `agents.image`/`agents.modelProvider`/`agents.modelName` with `model.provider`/`model.name`/`model.apiKeySecret` structure; remove orchestrator section

## 3. Gateway — Agent Directory API

- [x] 3.1 Add `/api/agents` GET endpoint returning JSON array of specialist agent cards (name, description, url, skills)
- [x] 3.2 Configure agent directory entries from environment variables or values-driven config
- [x] 3.3 Update gateway `values.yaml` section to include agent endpoint URLs for directory

## 4. Delete BYO Agent Code

- [x] 4.1 Delete `cmd/agents/` directory
- [x] 4.2 Delete `internal/agents/` directory (all Go files and remaining prompt files)
- [x] 4.3 Delete `internal/publish/tool.go` (agent function tool wrapper)
- [x] 4.4 Delete `agents/orchestrator.md`
- [x] 4.5 Delete `skills/orchestrator-routing/` directory
- [x] 4.6 Remove agents Dockerfile if separate from gateway (`Dockerfile.agents` or equivalent)

## 5. Dependency Cleanup

- [x] 5.1 Run `go mod tidy` — verify `google.golang.org/adk` agent imports are removed (keep if gateway still uses ADK types)
- [x] 5.2 Verify `github.com/a2aproject/a2a-go` is still needed (gateway may use it for card types) or remove
- [x] 5.3 Verify `github.com/Alcova-AI/adk-anthropic-go` is fully unused and remove from `go.mod`

## 6. Cancel Stale Change

- [x] 6.1 Cancel or archive `externalize-agent-skills` change — superseded by this change

## 7. Verification

- [x] 7.1 `go vet ./...` and `go build ./...` — zero errors (gateway, ingest, workbench still build)
- [x] 7.2 `go test ./...` — all remaining tests pass
- [x] 7.3 `helm template` renders valid Declarative Agent CRDs, ModelConfig, and prompts ConfigMap
- [x] 7.4 Verify no references to `cmd/agents` or `internal/agents` remain in Makefile, CI, or Dockerfiles
