## ADDED Requirements

### Requirement: Publish bundle is gateway-only

The publish bundle workflow SHALL be exposed exclusively via the gateway `POST /api/publish` endpoint. No agent function tool for publishing SHALL exist.

#### Scenario: Agent publish tool deleted

- **WHEN** the codebase is inspected
- **THEN** `internal/publish/tool.go` does not exist

#### Scenario: Gateway publish endpoint operational

- **WHEN** a POST request is made to `/api/publish` with valid artifacts, target, and tag
- **THEN** the gateway assembles the OCI bundle and pushes it to the target registry, returning reference, digest, and tag

### Requirement: Publish bundle dependencies retained

The `internal/publish/` package SHALL retain `bundle.go`, `media_types.go`, `helpers.go`, and `sign.go` for use by the gateway handler. Only the agent function tool wrapper (`tool.go`) is removed.

#### Scenario: Bundle assembly available to gateway

- **WHEN** the gateway handles a publish request
- **THEN** it calls `publish.AssembleAndPush` from `internal/publish/bundle.go`
