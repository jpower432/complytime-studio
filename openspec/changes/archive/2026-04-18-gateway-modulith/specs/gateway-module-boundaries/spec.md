## ADDED Requirements

### Requirement: Gateway internal packages follow one-package-per-domain
Each HTTP API domain (registry, agents, config, workbench) SHALL reside in its own `internal/` package. `cmd/gateway/main.go` SHALL contain only environment parsing, module construction, mux registration, and server lifecycle.

#### Scenario: main.go is wiring-only
- **WHEN** `cmd/gateway/main.go` is inspected
- **THEN** it contains no HTTP handler logic (no `w.Write`, no `json.NewDecoder` on request bodies)
- **THEN** each API domain is registered via a single `Register(mux, ...)` call

#### Scenario: Package inventory matches domains
- **WHEN** the `internal/` directory is listed
- **THEN** it contains at minimum: `auth`, `proxy`, `publish`, `ingest`, `registry`, `agents`, `config`, `workbench`, `httputil`

### Requirement: No cross-imports between sibling internal packages
No `internal/` package SHALL import another sibling `internal/` package directly, except for `internal/httputil` which is a shared leaf dependency. Cross-cutting concerns SHALL flow through interfaces.

#### Scenario: Registry does not import auth
- **WHEN** `internal/registry` is compiled
- **THEN** its import list does not contain `internal/auth`, `internal/agents`, `internal/config`, or `internal/workbench`

#### Scenario: Agents does not import registry
- **WHEN** `internal/agents` is compiled
- **THEN** its import list does not contain `internal/registry`, `internal/config`, or `internal/workbench`

### Requirement: Shared HTTP utilities live in internal/httputil
Common HTTP functions (`WriteJSON`, `EnvOr`, `ReadBody`, `UnavailableHandler`) SHALL be exported from `internal/httputil`. Modules SHALL import these instead of defining local copies.

#### Scenario: WriteJSON is centralized
- **WHEN** any `internal/` package needs to write a JSON response
- **THEN** it calls `httputil.WriteJSON(w, status, v)`
- **THEN** no package defines its own `writeJSON` function

### Requirement: Cross-cutting auth uses TokenProvider interface
Modules that need the user's session token SHALL accept a `TokenProvider` interface, not a concrete `*auth.Handler`. The `TokenProvider` interface SHALL be defined in `internal/httputil`.

#### Scenario: A2A proxy receives token via interface
- **WHEN** `internal/agents.Register` is called
- **THEN** it accepts a `TokenProvider` in its options
- **THEN** the A2A reverse proxy calls `TokenProvider.TokenFromRequest(r)` to get the Bearer token

#### Scenario: Publish module receives token via interface
- **WHEN** `internal/publish` needs the user's OAuth token
- **THEN** it obtains it through a `TokenProvider` passed at registration, not by importing `internal/auth`

### Requirement: Each module exposes a Register function
Every extracted `internal/` package SHALL export a `Register(mux *http.ServeMux, opts <ModuleOptions>) error` function that mounts its routes on the provided mux.

#### Scenario: Registry module registration
- **WHEN** `registry.Register(mux, opts)` is called with a valid MCP URL
- **THEN** routes `/api/registry/repositories`, `/api/registry/tags`, `/api/registry/manifest`, `/api/registry/layer` are registered on the mux

#### Scenario: Agents module registration
- **WHEN** `agents.Register(mux, opts)` is called with agent cards
- **THEN** routes `/api/agents` and `/api/a2a/` are registered on the mux

### Requirement: Zero HTTP API changes
All existing HTTP routes, request/response schemas, and status codes SHALL remain identical after the refactor. No route paths, query parameters, or JSON field names SHALL change.

#### Scenario: Registry routes unchanged
- **WHEN** `GET /api/registry/repositories?registry=localhost:5050` is called
- **THEN** the response shape and status codes are identical to pre-refactor behavior

#### Scenario: A2A proxy routes unchanged
- **WHEN** `POST /api/a2a/studio-threat-modeler` is called with an A2A payload
- **THEN** the proxy behavior (header injection, streaming relay, error responses) is identical to pre-refactor behavior

#### Scenario: Config route unchanged
- **WHEN** `GET /api/config` is called
- **THEN** the response contains `github_org`, `github_repo`, and `registry_insecure` fields with identical values
