## ADDED Requirements

### Requirement: Gemara document types from SDK

The system SHALL use `github.com/gemaraproj/go-gemara` generated types for all Gemara document structs instead of locally defined types. No local Gemara document structs SHALL exist in the codebase.

#### Scenario: EvaluationLog parsed via SDK types

- **WHEN** an EvaluationLog YAML document is ingested
- **THEN** the system decodes it into `gemara.EvaluationLog` from the SDK, not a local struct

#### Scenario: EnforcementLog parsed via SDK types

- **WHEN** an EnforcementLog YAML document is ingested
- **THEN** the system decodes it into `gemara.EnforcementLog` from the SDK, not a local struct

#### Scenario: No local Gemara struct definitions

- **WHEN** the codebase is inspected
- **THEN** `internal/ingest/gemara.go` SHALL NOT exist (or SHALL contain only import aliases, no struct definitions)

### Requirement: SDK loader for document parsing

The system SHALL use the SDK's `gemara.Load[T]` generic loader with a `Fetcher` implementation for parsing Gemara documents from files. Raw `yaml.Unmarshal` into Gemara document types SHALL NOT be used.

#### Scenario: File-based ingest uses SDK loader

- **WHEN** a file path is provided to the ingest command
- **THEN** the system uses `gemara.Load[gemara.EvaluationLog]` or `gemara.Load[gemara.EnforcementLog]` with a file-capable fetcher

#### Scenario: Stdin ingest retains byte-based path

- **WHEN** input is piped via stdin (no file path)
- **THEN** the system MAY read bytes from stdin and use `gemara.Load[T]` with an appropriate adapter, or parse via the SDK's YAML codec

### Requirement: Typed enums replace raw strings

The system SHALL use SDK enum types (`gemara.Result`, `gemara.Disposition`, `gemara.ConfidenceLevel`, `gemara.ArtifactType`) instead of raw strings for Gemara-defined values. Enum `.String()` methods SHALL be used when converting to ClickHouse string columns.

#### Scenario: Result enum in flatten output

- **WHEN** an EvaluationLog is flattened to ClickHouse rows
- **THEN** `ControlResult` and `AssessmentResult` columns contain the `.String()` output of `gemara.Result` values

#### Scenario: ArtifactType enum for type detection

- **WHEN** artifact type detection is performed in ingest or publish
- **THEN** the system reads `gemara.Metadata.Type` as `gemara.ArtifactType`, not a raw string

### Requirement: Datetime conversion for ClickHouse columns

The system SHALL convert `gemara.Datetime` (ISO 8601 string alias) values to `time.Time` when populating ClickHouse row structs. A shared conversion helper SHALL handle parsing.

#### Scenario: Start time conversion

- **WHEN** an AssessmentLog's `Start` field (type `gemara.Datetime`) is flattened
- **THEN** the `CollectedAt` ClickHouse column receives a `time.Time` parsed from the ISO 8601 string

#### Scenario: Optional End time conversion

- **WHEN** an AssessmentLog's `End` field is empty (`""`)
- **THEN** the `CompletedAt` ClickHouse column receives a nil `*time.Time`

### Requirement: Media type map uses ArtifactType enum keys

The system SHALL key the `artifactTypeToMediaType` map in `internal/publish/media_types.go` using `gemara.ArtifactType` enum values instead of raw strings. Media type string constants remain locally defined.

#### Scenario: Lookup by ArtifactType

- **WHEN** `MediaTypeForArtifact` is called
- **THEN** it accepts a `gemara.ArtifactType` parameter (not a raw string)

### Requirement: Future SDK packing API comments

The system SHALL include comments in `internal/publish/bundle.go` indicating that `AssembleAndPush` and related OCI packing logic are candidates for replacement by the `go-gemara` SDK's upcoming packing API.

#### Scenario: Comment presence in bundle.go

- **WHEN** `internal/publish/bundle.go` is inspected
- **THEN** `AssembleAndPush` has a comment referencing the planned SDK packing API migration

### Requirement: ClickHouse row schemas unchanged

The system SHALL NOT modify the `EvalRow` or `EnforcementRow` struct definitions or their ClickHouse column mappings. These are project-specific and independent of the SDK.

#### Scenario: EvalRow struct unchanged

- **WHEN** the refactor is complete
- **THEN** `EvalRow` field names, types, and `ch:` tags are identical to pre-refactor
