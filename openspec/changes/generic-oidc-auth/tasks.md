# Tasks: Generic OIDC Authentication via OAuth2 Proxy

- [x] Rewrite openspec (proposal, design, tasks) for OAuth2 Proxy architecture
- [x] Delete `internal/auth/oidc.go`, `internal/auth/jwks.go`
- [x] Delete `internal/auth/oidc_test.go`, `internal/auth/auth_test.go`
- [x] Extract `ChatStore` + `MemoryChatStore` from `session_store.go` into `chat_store.go`, delete `session_store.go`
- [x] Rewrite `internal/auth/auth.go`: thin `Handler` with header-trust middleware, `TokenFromRequest` via `X-Forwarded-Access-Token`
- [x] Simplify `internal/auth/config.go`: remove OIDC config, keep `STUDIO_API_TOKEN` only
- [x] Update `internal/auth/metrics.go`: remove OIDC/JWKS/discovery counters, keep request-level metrics
- [x] Update `cmd/gateway/main.go`: remove OIDC wiring, discovery retry, cookie secret, session store; simplify `NewHandler`
- [x] Add OAuth2 Proxy sidecar to `charts/complytime-studio/templates/gateway.yaml`
- [x] Rewrite `charts/complytime-studio/values.yaml` auth section for OAuth2 Proxy
- [x] Remove OIDC env vars from gateway deployment template
- [x] Update `docs/decisions/echo-framework-migration.md` for OAuth2 Proxy auth model
- [x] Update `docs/design/architecture.md` for OAuth2 Proxy auth model
- [x] Write auth tests: header-trust middleware, user upsert from headers, role seeding, anonymous fallback
- [x] Verify: `go build`, `go vet`, `go test`, `helm template`
