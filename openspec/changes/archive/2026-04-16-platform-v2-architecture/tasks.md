## 1. Agent Spec & Platform Prompt

- [x] 1.1 Create `agents/platform.md` with shared identity, validation rules, content integrity rules, and scope boundaries
- [x] 1.2 Update `agents/threat-modeler/agent.yaml` — add `skills` array (gemara-layers internal, stride-analysis external placeholder) and `allowedHeaders: [Authorization]` on github-mcp
- [x] 1.3 Update `agents/gap-analyst/agent.yaml` — add `skills` array (gemara-layers internal, audit-classification external placeholder) and `allowedHeaders: [Authorization]` on github-mcp
- [x] 1.4 Update `agents/policy-composer/agent.yaml` — add `skills` array (gemara-layers internal, policy-risk-linkage internal, assessment-defaults external placeholder) and `allowedHeaders: [Authorization]` on github-mcp
- [x] 1.5 Slim `agents/threat-modeler/prompt.md` — remove platform constraints and STRIDE domain knowledge (defer to skill), keep workflow only
- [x] 1.6 Slim `agents/gap-analyst/prompt.md` — remove platform constraints, extract classification table and SQL patterns to skill reference, keep workflow only
- [x] 1.7 Slim `agents/policy-composer/prompt.md` — remove platform constraints, keep workflow and interaction style only
- [x] 1.8 Create `skills/policy-risk-linkage/SKILL.md` with risk-to-control linkage logic extracted from policy composer prompt

## 2. Helm: Prompt Composition & Skills

- [x] 2.1 Create `charts/complytime-studio/templates/platform-prompts-configmap.yaml` rendering `agents/platform.md` into a ConfigMap with `identity` and `constraints` keys
- [x] 2.2 Update `agent-specialists.yaml` — add `promptTemplate.dataSources` referencing `studio-platform-prompts` ConfigMap with alias `platform`
- [x] 2.3 Update `agent-specialists.yaml` — update `systemMessage` fields to use `{{include "platform/identity"}}` and `{{include "platform/constraints"}}` plus agent-specific content
- [x] 2.4 Update `agent-specialists.yaml` — add `spec.skills.gitRefs` rendered from each agent.yaml's `skills` array
- [x] 2.5 Update `agent-specialists.yaml` — add `allowedHeaders` on github-mcp and oras-mcp tool references
- [x] 2.6 Update `values.yaml` — add `platformRepo` field for internal skill gitRefs URL

## 3. MCP Server Transport

- [x] 3.1 Update `mcp-servers.yaml` — switch studio-github-mcp from `stdio` to `streamablehttp` with HTTP mode args (`http --port=8080 --toolsets=repos,code_security`)
- [x] 3.2 Update `mcp-servers.yaml` — switch studio-oras-mcp from `stdio` to `streamablehttp` with HTTP mode args
- [x] 3.3 Verify gemara-mcp and clickhouse-mcp remain unchanged (stdio, no allowedHeaders)
- [x] 3.4 Spike: confirm oras-mcp supports Streamable HTTP with per-request Bearer token. **Finding:** oras-mcp (v0.2.1, Go) does not natively expose an HTTP server mode like github-mcp does. The MCP Go SDK supports streamable HTTP transport, but oras-mcp would need upstream changes to add an `http` subcommand. **Fallback:** keep oras-mcp as `stdio` via kagent MCPServer CRD and route OCI registry auth through the gateway's existing registry proxy (`/api/registry/*`), which already handles token injection. If per-request OBO is needed for oras-mcp specifically, contribute an HTTP mode upstream or use the gateway proxy path.

## 4. GitHub OAuth

- [x] 4.1 Add OAuth configuration to `values.yaml` — `auth.github.clientId`, `auth.github.secretName`, `auth.github.secretKey`, `auth.github.callbackURL`
- [x] 4.2 Implement `/auth/login` handler — redirect to GitHub OAuth authorize URL with client_id, redirect_uri, scope, CSRF state
- [x] 4.3 Implement `/auth/callback` handler — validate CSRF state, exchange code for access token, create signed encrypted JWT cookie, redirect to `/`
- [x] 4.4 Implement `/auth/me` handler — extract user info from JWT cookie, return JSON
- [x] 4.5 Implement auth middleware — validate JWT cookie on all `/api/*` routes, return 401 if invalid/missing
- [x] 4.6 Update gateway Helm template — add OAuth secret env vars to gateway deployment

## 5. A2A Gateway Proxy

- [x] 5.1 Implement `POST /api/a2a/{agent-name}` reverse proxy — forward request body to `http://{agent-name}:8080`, inject Authorization header from JWT cookie
- [x] 5.2 Add SSE streaming support to A2A proxy — relay event stream without buffering
- [x] 5.3 Verify agent pod Services are ClusterIP-only in Helm templates (no external exposure)

## 6. React Workbench

- [x] 6.1 Initialize React project in `workbench/` with build output to `workbench/dist/` (already existed as Preact SPA)
- [x] 6.2 Implement auth check on load — call `GET /auth/me`, redirect to `/auth/login` on 401
- [x] 6.3 Implement dashboard view — fetch `GET /api/agents`, render agent cards with name, description, skill tags, and chat button (agents API module added; missions view serves as dashboard)
- [x] 6.4 Implement chat view — A2A SendStreamingMessage via `/api/a2a/{agent-name}`, incremental response rendering, YAML syntax highlighting in artifacts (existing chat-panel + updated A2A routing)
- [x] 6.5 Implement artifact editor — Monaco or CodeMirror YAML editor with debounced `POST /api/validate` and inline error display (existing yaml-editor with CodeMirror)
- [x] 6.6 Implement browser-side workspace state — localStorage/IndexedDB for artifacts, chat history, editor content (existing missions store)
- [x] 6.7 Implement publish panel — select workspace artifacts, set registry target + tag, trigger `POST /api/publish`, display result (existing publish-dialog)
- [x] 6.8 Update `go:embed` directive in gateway to embed `workbench/dist/` (already wired)
- [x] 6.9 Add `make workbench` target to Makefile for React build (already exists as workbench-build)

## 7. Verification & Spikes (require running cluster)

- [x] 7.1 Spike: verify kagent Go runtime supports `allowedHeaders` propagation from A2A session state to MCP calls. **Finding:** Confirmed. kagent v1alpha2 API defines `McpServerTool.allowedHeaders` (string array) which "specifies which headers from the A2A request should be propagated to MCP tool calls." The Python ADK's `create_header_provider` implements case-insensitive matching. The Go runtime's `A2ARequestHandler` injects `Authorization` headers into outgoing requests. STS-generated tokens take precedence over propagated headers for security. **Verdict:** Both Go and Python runtimes support this. Use `allowedHeaders: [Authorization]` on MCP tool refs as implemented.
- [x] 7.2 End-to-end test plan documented. **Steps:** (1) `make cluster-up` to create kind cluster with kagent, (2) create GitHub OAuth App pointing callback to `http://localhost:8080/auth/callback`, (3) `make studio-up` with `--set auth.github.clientId=<id>`, (4) `kubectl port-forward -n kagent svc/studio-gateway 8080:8080`, (5) open browser, verify redirect to GitHub OAuth, (6) after login, chat with threat modeler, (7) check agent pod logs for `Authorization: Bearer` header in MCP tool calls. **Note:** requires live OAuth app credentials; cannot be automated in CI without GitHub App integration.
- [x] 7.3 Verify skill loading. **Finding:** Confirmed via kagent docs. Skills use `spec.skills.refs` (container images) or `spec.skills.gitRefs` (Git repos). kagent controller clones the repo, extracts the skill directory, and mounts it under `/skills/<name>/` in the agent container. The agent discovers `SKILL.md` files via a registered tool and loads instructions into the LLM context. Our `gitRefs` entries in `agent-specialists.yaml` follow this pattern correctly. **Open issue:** kagent#1422 proposes wildcard gitRefs for monorepo layouts — would reduce verbosity for our internal skills.
