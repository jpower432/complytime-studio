## 1. Dependency Setup

- [x] 1.1 Add `github.com/gemaraproj/go-gemara` to `go.mod` via `go get`
- [x] 1.2 Run `go mod tidy` and verify transitive deps resolve cleanly

## 2. Ingest Package — Type Migration

- [x] 2.1 Delete `internal/ingest/gemara.go` (all 13 local Gemara structs)
- [x] 2.2 Add a `parseDatetime` helper in `internal/ingest/flatten.go` to convert `gemara.Datetime` → `time.Time`
- [x] 2.3 Update `FlattenEvaluationLog` to accept `*gemara.EvaluationLog` — remap field names (`.Id`, `.ReferenceId`, `.EntryId`), call `.String()` on `Result`/`ConfidenceLevel` enums, parse `Datetime` fields, handle pointer slices
- [x] 2.4 Update `FlattenEnforcementLog` to accept `*gemara.EnforcementLog` — remap field names, call `.String()` on `Disposition`/`Result` enums, dereference `*string` Message, handle pointer slices
- [x] 2.5 Verify `EvalRow` and `EnforcementRow` structs are unchanged (types and `ch:` tags identical)

## 3. Ingest Command — Loader Migration

- [x] 3.1 Replace `detectType()` and `artifactHeader` in `cmd/ingest/main.go` with SDK metadata load (parse `gemara.Metadata.Type` as `ArtifactType`)
- [x] 3.2 Replace `yaml.Unmarshal` in `ingestEvaluationLog` with `gemara.Load[gemara.EvaluationLog]` using a file fetcher for file-path input
- [x] 3.3 Replace `yaml.Unmarshal` in `ingestEnforcementLog` with `gemara.Load[gemara.EnforcementLog]`
- [x] 3.4 Update `derivePolicyID` to use `gemara.MappingReference` fields (`.Id`, `.Title`)
- [x] 3.5 Retain stdin path with appropriate adapter (byte-based load or temp file)

## 4. Publish Package — Type Alignment

- [x] 4.1 Replace `artifactMeta` partial struct in `internal/publish/tool.go` with SDK metadata type detection
- [x] 4.2 Change `artifactTypeToMediaType` map keys in `media_types.go` from raw strings to `gemara.ArtifactType` enum values
- [x] 4.3 Update `MediaTypeForArtifact` signature to accept `gemara.ArtifactType` instead of `string`
- [x] 4.4 Update callers of `MediaTypeForArtifact` in `bundle.go` to pass `ArtifactType`

## 5. Future SDK Packing Markers

- [x] 5.1 Add comment to `AssembleAndPush` in `bundle.go` noting planned replacement by `go-gemara` SDK packing API
- [x] 5.2 Add comment to `ArtifactInput` struct noting it will be superseded by SDK bundle types
- [x] 5.3 Add comment to `copyToRemote` and OCI helpers noting they are candidates for SDK consolidation

## 6. Tests and Verification

- [x] 6.1 Update `internal/publish/*_test.go` for renamed types and `ArtifactType` enum parameters
- [x] 6.2 Add test for `parseDatetime` helper (valid ISO 8601, empty string, edge cases)
- [x] 6.3 Run `go vet ./...` and `go build ./...` — zero errors
- [x] 6.4 Run existing test suite — all pass with no behavior changes
