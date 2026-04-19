## Context

The Go gateway (`cmd/gateway/main.go`) is a single 567-line file that contains route registration, request handling, type definitions, and utilities for six unrelated concerns: Gemara MCP proxying, OCI registry proxying, OCI bundle publishing, A2A agent proxying, platform config, and workbench serving. The existing `internal/` packages (`auth`, `proxy`, `publish`, `ingest`) already demonstrate the target pattern — self-contained packages with a `Register` or constructor function. The remaining logic in `main.go` needs the same treatment.

## Goals / Non-Goals

**Goals:**
- Reduce `cmd/gateway/main.go` to pure wiring (~80 lines: env parsing, module construction, mux registration, server start)
- Each domain module lives in its own `internal/` package with an exported `Register(mux, deps)` function
- No sibling `internal/` package imports another sibling directly — cross-cutting concerns flow through interfaces defined in a shared package
- Each module is independently unit-testable with `httptest`
- Zero external behavior change — all HTTP routes, payloads, and status codes remain identical

**Non-Goals:**
- Splitting into multiple binaries or containers
- Introducing dependency injection frameworks
- Changing any HTTP API contract
- Refactoring `internal/auth`, `internal/proxy`, or `internal/publish` (already well-structured)

## Decisions

### D1: Package layout follows one-package-per-domain

| Package | Responsibility | Extracted from |
|:--|:--|:--|
| `internal/httputil` | `WriteJSON`, `EnvOr`, `ReadBody`, `UnavailableHandler` | `main.go` utilities |
| `internal/registry` | `registryProxy` struct + 4 handlers + OCI string helpers | `main.go` lines 117–347 |
| `internal/agents` | `AgentCard`/`AgentSkill` types, directory parsing, A2A reverse proxy | `main.go` lines 405–505 |
| `internal/config` | Platform config endpoint (`/api/config`) | `main.go` lines 444–453 |
| `internal/workbench` | SPA file server with history-mode fallback | `main.go` lines 507–529 |

**Why one-package-per-domain over a single `internal/gateway` package:** Separate packages enforce isolation at the compiler level. A `registry` handler cannot accidentally import an `agents` type. A single package would allow internal coupling to re-emerge.

**Why not extract `cookieSignKey` into `internal/auth`:** It's env-parsing and randomness, not auth logic. It stays in `main.go` as server bootstrap.

### D2: Cross-cutting concerns use interfaces, not direct imports

Two concerns cross module boundaries:

1. **Session/token extraction** — the `agents` and `publish` modules need the user's OAuth token from the session.
2. **JSON response writing** — every module needs `writeJSON`.

For (1), define a `TokenProvider` interface in `internal/httputil`:

```go
type TokenProvider interface {
    TokenFromRequest(r *http.Request) (string, bool)
}
```

`internal/auth.Handler` satisfies this interface. `main.go` passes the auth handler (as `TokenProvider`) to `agents.Register()` and `publish.Register()`.

For (2), `internal/httputil.WriteJSON` is a plain function — no interface needed, direct import is fine since `httputil` is a leaf dependency with no sibling imports.

**Why interfaces over passing `*auth.Handler` directly:** The `agents` package should not know about GitHub OAuth, session cookies, or HMAC signing. It only needs "give me a token for this request." This also enables testing with a stub `TokenProvider`.

### D3: Each module exposes a `Register` function

Every extracted package exports a single entry point:

```go
func Register(mux *http.ServeMux, opts Options) error
```

Where `Options` is a module-specific struct containing dependencies (e.g., MCP URL, token provider, insecure registries). This mirrors the pattern already used by `internal/auth`:

```go
authHandler.Register(mux)
```

**Why `Register(mux)` over returning `http.Handler`:** The mux-based approach lets each module own its route paths. Returning a handler would push path knowledge back to `main.go`.

### D4: String helpers stay with their consumer

`splitReference`, `splitRepoTag`, `splitRepoDigest` are OCI-specific string parsers. They move to `internal/registry` as unexported functions, not to a shared utility package.

**Why not `internal/httputil`:** These functions have no reuse outside registry operations. Putting them in a shared package would invite inappropriate coupling.

## Risks / Trade-offs

| Risk | Mitigation |
|:--|:--|
| Large diff makes review harder | Split into sequential PRs: (1) extract `httputil`, (2) extract `registry`, (3) extract `agents` + `config` + `workbench`, (4) slim `main.go` |
| Duplicate `writeJSON` during transition | The proxy package already has its own `writeJSON`. After extraction, all modules import `httputil.WriteJSON` and delete local copies |
| Module boundary enforcement is convention-only | Go compiler prevents circular imports. A CI lint rule (`go vet` + `depguard`) can block sibling imports explicitly |
| Behavior regression from refactor | No logic changes — only moving code. Existing integration tests and manual smoke tests cover the same routes |
