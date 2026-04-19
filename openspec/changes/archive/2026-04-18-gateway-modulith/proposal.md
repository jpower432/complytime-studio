## Why

`cmd/gateway/main.go` is 567 lines and growing. It mixes OCI registry proxying, A2A reverse proxying, agent directory management, platform config, workbench serving, and shared utilities in a single file. Adding a new endpoint means touching the same file everyone else touches — merge conflicts, hard-to-test handlers, and no enforced boundaries between concerns. A modulith restructuring keeps the single-binary deployment while giving each concern its own package with explicit interfaces.

## What Changes

- Extract registry proxy logic (~170 lines) into `internal/registry`
- Extract A2A agent proxy and agent directory into `internal/agents`
- Extract platform config endpoint into `internal/config`
- Extract workbench static serving into `internal/workbench`
- Extract shared utilities (`writeJSON`, `envOr`, `readBody`, `cookieSignKey`) into `internal/httputil`
- Reduce `cmd/gateway/main.go` to pure wiring: parse env, construct modules, register on mux, start server
- Define interfaces for cross-cutting concerns (e.g., `TokenProvider` for session-to-token extraction) to prevent sibling packages from importing each other

## Capabilities

### New Capabilities
- `gateway-module-boundaries`: Defines the internal package structure, module isolation rules, and the interfaces that connect modules to shared concerns (auth, HTTP utilities)

### Modified Capabilities
- `a2a-gateway-proxy`: Route registration moves from inline `main.go` closures to `internal/agents.Register(mux, ...)`

## Impact

- **Code**: `cmd/gateway/main.go` and new `internal/` packages. No new dependencies.
- **APIs**: Zero changes. All HTTP routes, request/response shapes, and behavior remain identical.
- **Tests**: Each extracted module becomes independently testable with `httptest`.
- **Deployment**: No changes. Single binary, single port, same container image.
