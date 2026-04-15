## Why

ComplyTime Studio's agent platform has outgrown its initial architecture. Agent identity is split across wiring YAML and monolithic prompts with duplicated constraints. Skill knowledge is embedded in prompt prose rather than composable, reusable packs. MCP servers use shared platform credentials instead of acting on behalf of authenticated users. The frontend is a vanilla JS prototype with no component framework, and there is no user authentication. These gaps block the path to a shared hosted platform where teams deploy agents via Helm and interact through a production-quality workbench.

## What Changes

- **Agent spec evolution**: Add `skills` (git-based knowledge packs) and `allowedHeaders` (OBO token forwarding) to the canonical `agent.yaml` format. Drop embedded domain knowledge from prompts.
- **Platform prompt composition**: Create `agents/platform.md` as shared identity and constraints. Use kagent `promptTemplate` with ConfigMap data sources to compose platform + agent-specific prompts. Eliminate copy-pasted constraints from individual `prompt.md` files.
- **Skill pack architecture**: Define internal skills (this repo) and external skill references (other repos) in `agent.yaml`. Render to kagent `gitRefs`. Extract domain knowledge from prompts into SKILL.md packs.
- **GitHub OAuth authentication**: Add GitHub OAuth App flow to the gateway. Store user's GitHub token in signed JWT cookie. Gateway injects `Authorization` header when proxying A2A requests to agents.
- **OBO token forwarding**: Switch github-mcp and oras-mcp from stdio to Streamable HTTP transport. Use kagent's native `allowedHeaders` on tool references to propagate user's token from A2A requests to MCP tool calls. Agents act on behalf of the authenticated user.
- **React frontend**: Replace vanilla JS workbench with React SPA. Dashboard with agent cards, A2A chat, YAML editor with live validation, publish panel, registry browser. Browser-side state (localStorage/IndexedDB, Excalidraw model).
- **A2A reverse proxy**: Gateway proxies A2A requests to agent pods, injecting user auth headers. Frontend talks to gateway, not directly to agent A2A endpoints.

## Capabilities

### New Capabilities
- `agent-spec-skills`: Git-based skill pack references in canonical agent.yaml, rendered to kagent gitRefs
- `platform-prompt-composition`: Shared platform.md identity composed with agent-specific prompts via kagent promptTemplate
- `github-oauth`: GitHub OAuth App login flow in gateway with JWT session cookie
- `obo-token-forwarding`: Per-request user token propagation from A2A to MCP via kagent allowedHeaders
- `a2a-gateway-proxy`: Gateway reverse proxy for A2A with auth header injection
- `react-workbench`: React SPA with dashboard, agent chat, artifact editor, publish panel, and registry browser

### Modified Capabilities
- `mcpserver-crd-transport`: github-mcp and oras-mcp switch from stdio to streamablehttp transport to accept per-request Authorization headers

## Impact

- **Gateway** (`cmd/gateway/`): New OAuth endpoints, A2A reverse proxy, session cookie management. Moderate code addition.
- **Agent definitions** (`agents/`): New `platform.md` file. All `agent.yaml` files gain `skills` and `allowedHeaders` fields. All `prompt.md` files slimmed (platform constraints removed, domain knowledge extracted to skills).
- **Skills** (`skills/`): New SKILL.md packs extracted from prompts. External skill repo references added.
- **Helm chart**: `mcp-servers.yaml` rewritten for HTTP transport on github-mcp and oras-mcp. `agent-specialists.yaml` gains allowedHeaders and skills.gitRefs. New `studio-platform-prompts` ConfigMap template. New OAuth secret reference in values.yaml.
- **Frontend** (`workbench/`): Full rewrite from vanilla JS to React. Build output still embedded via `go:embed`.
- **Dependencies**: GitHub OAuth library for Go. React + build toolchain for frontend.
- **External**: GitHub OAuth App registration required per deployment.
