## 1. Shared Utilities — `internal/httputil`

- [x] 1.1 Create `internal/httputil/httputil.go` with exported `WriteJSON`, `EnvOr`, `ReadBody`, `UnavailableHandler` functions
- [x] 1.2 Define `TokenProvider` interface in `internal/httputil/httputil.go`: `TokenFromRequest(r *http.Request) (string, bool)`
- [x] 1.3 Add `TokenFromRequest` method to `internal/auth.Handler` that satisfies the `TokenProvider` interface (extract token from session cookie)
- [x] 1.4 Update `internal/proxy` to import `httputil.WriteJSON` and delete its local `writeJSON` copy

## 2. Registry Module — `internal/registry`

- [x] 2.1 Create `internal/registry/registry.go` with `Options` struct (MCP URL, insecure registries list) and `Register(mux, opts)` function
- [x] 2.2 Move `registryProxy` struct, all handler methods (`handleListRepositories`, `handleListTags`, `handleFetchManifest`, `handleFetchLayer`, `toolCall`), and MCP client setup from `main.go`
- [x] 2.3 Move `splitReference`, `splitRepoTag`, `splitRepoDigest`, `parseInsecureRegistries` as unexported helpers in `internal/registry`
- [x] 2.4 Replace local `writeJSON` calls with `httputil.WriteJSON`

## 3. Agents Module — `internal/agents`

- [x] 3.1 Create `internal/agents/agents.go` with `AgentCard`, `AgentSkill` types, `ParseDirectory(raw string) []AgentCard`, and `Options` struct (cards, token provider)
- [x] 3.2 Move `registerAgentDirectory` into `internal/agents` as part of `Register(mux, opts)`
- [x] 3.3 Move A2A reverse proxy logic into `internal/agents` — use `TokenProvider` interface for header injection instead of importing `internal/auth`
- [x] 3.4 Replace `auth2.SessionFrom(req.Context())` with `opts.TokenProvider.TokenFromRequest(req)` in the A2A proxy director

## 4. Config and Workbench Modules

- [x] 4.1 Create `internal/config/config.go` with `Options` struct (key-value map) and `Register(mux, opts)` function for `/api/config`
- [x] 4.2 Create `internal/web/serve.go` with `Register(mux, assets fs.FS)` function that mounts the SPA file server with history-mode fallback
- [x] 4.3 Pass `workbench.Assets` from `main.go` into `web.Register`

## 5. Slim `cmd/gateway/main.go`

- [x] 5.1 Remove all extracted handler logic, types, and utility functions from `main.go`
- [x] 5.2 Wire `main()` to call `Register` functions from each module with appropriate `Options` structs
- [x] 5.3 Verify `main.go` contains only: env parsing, module construction, mux registration, auth middleware, server lifecycle (~80 lines)

## 6. Publish Module Update

- [x] 6.1 Update `internal/publish` to accept a `TokenProvider` in a new `Register(mux, opts)` function that mounts `/api/publish`
- [x] 6.2 Move the publish handler closure from `main.go` into `internal/publish`
- [x] 6.3 Replace direct `auth2.SessionFrom` call with `TokenProvider.TokenFromRequest` in the publish handler

## 7. Verification

- [x] 7.1 Run `go build ./cmd/gateway` — confirm clean compilation with no circular imports
- [x] 7.2 Run `go vet ./...` — confirm no import violations between sibling packages
- [x] 7.3 Verify no `internal/` package (except `httputil`) appears in another sibling's import list
- [ ] 7.4 Deploy to Kind cluster and smoke test: login, agent chat, validate, publish, registry browse — all routes behave identically
