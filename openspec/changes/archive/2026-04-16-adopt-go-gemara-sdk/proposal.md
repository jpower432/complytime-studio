## Why

The `internal/ingest` and `internal/publish` packages hand-roll Gemara document types and YAML parsing instead of using the `go-gemara` SDK (`github.com/gemaraproj/go-gemara`). The SDK provides CUE-generated types for every Gemara artifact, a generic `Load[T]` parser with strict decoding, typed enums with validation, and an `ArtifactType` registry. Duplicating these in-tree creates drift risk, skips the SDK's decode-time validation, and blocks adoption of upcoming SDK features (OCI bundle packing).

## What Changes

- Replace 13 hand-rolled structs in `internal/ingest/gemara.go` with imports from `go-gemara` generated types (`gemara.EvaluationLog`, `gemara.EnforcementLog`, etc.)
- Replace raw `yaml.Unmarshal` calls in `cmd/ingest/main.go` with `gemara.Load[T]` and the `fetcher` abstraction
- Replace the `artifactMeta` partial-unmarshal pattern in `internal/publish/tool.go` with SDK type detection via `gemara.Metadata` / `ArtifactType` enum
- Replace the local `artifactTypeToMediaType` map in `internal/publish/media_types.go` with the SDK's `ArtifactType` enum (media type constants remain local until the SDK ships them)
- Update `internal/ingest/flatten.go` field access to match SDK-generated struct field names
- Add `github.com/gemaraproj/go-gemara` to `go.mod`
- Leave forward-looking comments in `internal/publish/bundle.go` for the SDK's upcoming OCI packing API

## Capabilities

### New Capabilities

- `sdk-type-adoption`: Replace local Gemara structs and parsing with `go-gemara` SDK types and loaders across ingest and publish packages

### Modified Capabilities

_(none — no spec-level behavior changes, only implementation)_

## Impact

- **Dependencies**: Adds `github.com/gemaraproj/go-gemara` (and its transitive deps: `go-oscal`, `go-yaml`, `go-cmp`) to `go.mod`
- **Code**: `internal/ingest/gemara.go` shrinks to an import alias or is deleted; `flatten.go` field references update; `cmd/ingest/main.go` parsing path changes; `internal/publish/tool.go` and `media_types.go` refactored
- **Tests**: `internal/publish/*_test.go` may need minor updates for renamed types
- **API surface**: No external-facing changes; ClickHouse row schemas (`EvalRow`, `EnforcementRow`) and OCI bundle output are unchanged
