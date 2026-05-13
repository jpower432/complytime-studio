## MODIFIED Requirements

### Requirement: Platform binary has no SPA dependency
The platform gateway binary SHALL build and run without any reference to SPA assets, the `workbench` Go package, or the `internal/web` package.

#### Scenario: Build without workbench
- **WHEN** `go build ./cmd/gateway/` is executed
- **THEN** compilation succeeds without `workbench/embed.go` or `internal/web/serve.go` in the dependency graph

#### Scenario: No embed directive in platform
- **WHEN** the platform source tree is inspected
- **THEN** no `//go:embed` directive references SPA assets

### Requirement: Studio has no Go import dependency
The Studio SPA build SHALL not import, reference, or depend on any Go source file. It is a pure TypeScript/JavaScript project.

#### Scenario: Studio builds independently
- **WHEN** `cd studio && npm run build` is executed
- **THEN** the build succeeds without Go toolchain installed
- **THEN** the output is a static `dist/` directory

### Requirement: Agents have no direct Go store imports at runtime
Agent containers SHALL not import `internal/store` or `internal/postgres` at runtime. Data access is exclusively via MCP protocol.

#### Scenario: Agent container has no Go dependencies
- **WHEN** the assistant agent container image is inspected
- **THEN** it contains Python runtime (LangGraph) and no Go binaries other than MCP sidecars

### Requirement: Boundary rules documented in AGENTS.md
The `AGENTS.md` file SHALL contain an explicit boundary rules section defining what each component may and may not depend on.

#### Scenario: Boundary rules are present
- **WHEN** a contributor reads `AGENTS.md`
- **THEN** they find a section listing: Studio must not import Go; Agents must not import Studio; Agents read platform data through studio-mcp only; Platform must not import from studio/
