## Context

`internal/ingest/gemara.go` defines 13 Gemara document structs (EvaluationLog, EnforcementLog, and their sub-types) by hand. `cmd/ingest/main.go` parses YAML with raw `goccy/go-yaml` unmarshal. `internal/publish/tool.go` uses another partial struct for type detection. These duplicate the CUE-generated types in `github.com/gemaraproj/go-gemara`, which provides strict decoding, typed enums, and a generic `Load[T]` API.

The SDK also plans to ship OCI bundle packing logic, which will overlap with the current `internal/publish/bundle.go` implementation.

## Goals / Non-Goals

**Goals:**

- Replace hand-rolled Gemara structs with SDK-generated types
- Use the SDK's `Load[T]` + `Fetcher` for parsing instead of raw YAML unmarshal
- Leverage SDK typed enums (`Result`, `Disposition`, `ConfidenceLevel`, `ArtifactType`) instead of raw strings
- Prepare `internal/publish/bundle.go` for future SDK packing API adoption with comments
- Keep the ClickHouse row schemas (`EvalRow`, `EnforcementRow`) and flatten logic as project-specific code

**Non-Goals:**

- Adopting SDK OSCAL/SARIF conversion (agents use MCP for this today)
- Replacing OCI bundle packing now (SDK packing API not yet available)
- Changing the ClickHouse schema or ingest pipeline behavior

## Decisions

### 1. Delete `internal/ingest/gemara.go`, import SDK types directly

**Rationale:** The 13 local structs are a subset of the SDK's CUE-generated types. Maintaining two copies creates drift risk. The SDK types carry additional fields (Author, Owner, etc.) that are ignored by the flatten logic but do no harm.

**Alternative considered:** Thin adapter types wrapping SDK types. Rejected — adds indirection with no benefit since flatten.go can access SDK fields directly.

### 2. Field name migration in `flatten.go`

The SDK uses CUE-conventional casing. Key renames:

| Local field | SDK field | Type change |
|:---|:---|:---|
| `.ID` | `.Id` | — |
| `.ReferenceID` | `.ReferenceId` | — |
| `.EntryID` | `.EntryId` | — |
| `.URL` | `.Url` | — |
| `.Result` (string) | `.Result` | `gemara.Result` (has `.String()`) |
| `.Disposition` (string) | `.Disposition` | `gemara.Disposition` (has `.String()`) |
| `.ConfidenceLevel` (string) | `.ConfidenceLevel` | `gemara.ConfidenceLevel` (has `.String()`) |
| `.Start` (`time.Time`) | `.Start` | `gemara.Datetime` (string alias) |
| `.End` (`*time.Time`) | `.End` | `gemara.Datetime` (string, zero-value = `""`) |
| `.StepsExecuted` (`*int`) | `.StepsExecuted` | `int64` (zero-value = `0`) |
| `.Message` (string, ActionResult) | `.Message` | `*string` |
| `.Evaluations` (slice of value) | `.Evaluations` | `[]*ControlEvaluation` (pointer slice) |
| `.AssessmentLogs` (slice of value) | `.AssessmentLogs` | `[]*AssessmentLog` (pointer slice) |
| `.Actions` (slice of value) | `.Actions` | `[]*ActionResult` (pointer slice) |

Flatten functions will call `.String()` on enum types and parse `Datetime` strings for `time.Time` columns in ClickHouse rows.

### 3. Replace `detectType()` and `artifactMeta` with SDK metadata

**Rationale:** Both `cmd/ingest/main.go` and `internal/publish/tool.go` define partial structs to extract `metadata.type`. The SDK's `Metadata.Type` field is `ArtifactType` — a validated enum. Use `gemara.Load[gemara.Metadata]` or a lightweight header load to get the typed value.

**Alternative considered:** Keep partial struct for speed (avoids full parse). Rejected — the full parse happens immediately after in both call sites, so parsing once with the SDK is simpler.

### 4. Use `fetcher.URI` for ingest input

**Rationale:** The ingest command reads from a file path or stdin. For file paths, `fetcher.URI{}` dispatches `file://` to the SDK's `fetcher.File`. For stdin, retain the existing `io.ReadAll(os.Stdin)` path since the SDK fetcher interface expects a source string.

### 5. Keep `internal/publish/media_types.go` constants, derive map from `ArtifactType`

**Rationale:** The SDK exports `ArtifactType` enum values but does not yet export OCI media type strings. Keep the media type constants local. Replace the `artifactTypeToMediaType` map keys with `ArtifactType` enum values instead of raw strings.

### 6. Comment `bundle.go` for future SDK packing API

**Rationale:** The SDK plans OCI bundle packing. Mark `AssembleAndPush` and its helpers with comments indicating the planned migration path so future contributors know not to invest heavily in the local implementation.

## Risks / Trade-offs

| Risk | Mitigation |
|:---|:---|
| SDK Datetime (string) vs ClickHouse time.Time conversion overhead | Write a small `parseDatetime` helper; SDK guarantees ISO 8601 format |
| SDK pointer slices (`[]*ControlEvaluation`) require nil checks in loops | Add nil guards in flatten logic; SDK guarantees non-nil entries from valid documents |
| SDK enum `.String()` output may diverge from current raw strings stored in ClickHouse | Pin SDK version; add tests comparing enum string output to expected ClickHouse values |
| SDK transitive deps (`go-oscal`, `go-cmp`) increase module size | Acceptable — these are well-maintained, small modules. `go-oscal` is already aligned with our domain |
| Future SDK packing API may not match current `AssembleAndPush` interface | Comments mark the boundary; migration is a separate change when the API ships |
