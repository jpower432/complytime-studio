# Design: Direct Gemara-Based Evidence Ingestion

## Endpoint

```
POST /api/evidence/import
Content-Type: application/x-yaml  (or multipart/form-data with YAML file)
```

Accepts: Gemara EvaluationLog or EnforcementLog YAML body.

Returns:

```json
{
  "artifact_type": "EvaluationLog",
  "artifact_id": "eval-soc2-2026q1",
  "policy_id": "policy-soc2",
  "rows_ingested": 42,
  "certification": "pending"
}
```

## Handler Flow

```
POST /api/evidence/import
  │
  ├─ 1. Read body (YAML or multipart file)
  ├─ 2. Lightweight header parse → metadata.type
  ├─ 3. Branch on artifact type:
  │     ├─ EvaluationLog  → gemara.Load[gemara.EvaluationLog]
  │     └─ EnforcementLog → gemara.Load[gemara.EnforcementLog]
  ├─ 4. Derive policyID from metadata.mapping-references
  ├─ 5. Flatten → []ingest.EvidenceRow (existing logic)
  ├─ 6. ingest.Writer.InsertEvidenceRows(ctx, rows)
  ├─ 7. bus.PublishEvidence(policyID, len(rows))
  │     └─ NATS → CertificationHandler runs async
  └─ 8. Return 201 with summary
```

## Gateway Integration

### New handler: `importGemaraEvidenceHandler`

Location: `internal/store/handlers.go` (alongside existing `ingestEvidenceHandler`)

```go
func importGemaraEvidenceHandler(
    writer *ingest.Writer,
    pub EvidencePublisher,
) http.HandlerFunc
```

Dependencies:
- `internal/ingest` — `FlattenEvaluationLog`, `FlattenEnforcementLog`, `Writer`
- `go-gemara` — `gemara.Load`, type detection
- `internal/events` — `EvidencePublisher` for NATS

### Registration

```go
mux.HandleFunc("POST /api/evidence/import",
    importGemaraEvidenceHandler(ingestWriter, s.EventPublisher))
```

### Stores expansion

`Stores` struct gains `IngestWriter *ingest.Writer` field. The gateway's `main.go` initializes the writer alongside the existing ClickHouse store connection (reuses same connection pool or shares config).

## Reuse from `cmd/ingest`

| Component | Current location | Action |
|:--|:--|:--|
| `detectType` | `cmd/ingest/main.go` | Move to `internal/ingest/detect.go` |
| `derivePolicyID` | `cmd/ingest/main.go` | Move to `internal/ingest/policy.go` |
| `FlattenEvaluationLog` | `internal/ingest/flatten.go` | No change |
| `FlattenEnforcementLog` | `internal/ingest/flatten.go` | No change |
| `Writer` / `InsertEvidenceRows` | `internal/ingest/writer.go` | No change |
| `EvidenceRow` | `internal/ingest/types.go` | No change |
| `bytesFetcher` | `cmd/ingest/main.go` | Move to `internal/ingest/fetcher.go` |

After extraction, `cmd/ingest/main.go` becomes a thin wrapper calling `internal/ingest` — then deprecated.

## Content-Type Handling

| Content-Type | Behavior |
|:--|:--|
| `application/x-yaml`, `text/yaml` | Body is YAML artifact |
| `multipart/form-data` | File field `artifact` contains YAML |
| Other | 415 Unsupported Media Type |

Max body size: `consts.MaxRequestBody` (existing constant).

## Validation

The `go-gemara` SDK's `gemara.Load[T]` performs structural validation during deserialization. Invalid YAML or schema-violating artifacts fail with 400 and a descriptive error.

Optional: call `validate_gemara_artifact` MCP tool for CUE schema validation. Deferred to Phase 2 — SDK validation is sufficient for ingestion.

## Error Responses

| Status | Condition |
|:--|:--|
| 201 | Ingested successfully, certification pending |
| 400 | Invalid YAML, unknown artifact type, empty evaluations/actions |
| 413 | Body exceeds max size |
| 415 | Unsupported content type |
| 500 | ClickHouse write failure |

## Certification Flow (unchanged)

```
Gateway inserts rows
  → bus.PublishEvidence(policyID, count)
  → NATS subject: studio.evidence.{policyID}
  → CertificationHandler subscribes
  → Queries recent evidence
  → Runs certifier.Pipeline
  → Writes certifications table
  → Sets evidence.certified flag
```

No changes to the certification pipeline. The NATS event trigger is identical to the existing `POST /api/evidence` path.

## ADR Update

Supersede `docs/decisions/otel-native-ingestion.md`:

- **Old primary path**: OTel Collector → ClickHouse exporter
- **New primary path**: `POST /api/evidence/import` (Gemara YAML → flatten → ClickHouse → certify)
- **OTel Collector**: remains valid for operators who prefer it. Semconv alignment doc kept as reference contract.
- **`cmd/ingest`**: deprecated. Functionality subsumed by gateway endpoint.
- **`POST /api/evidence`** (JSON): unchanged. Still available for flat JSON records, seeding, non-Gemara producers.

## Helm / Env Changes

None. The gateway already connects to ClickHouse and NATS. The `ingest.Writer` reuses the same ClickHouse config (`CLICKHOUSE_HOST`, `CLICKHOUSE_PORT`, etc.) that the store uses.

## Tests

| Test | Validates |
|:--|:--|
| `TestImportEvaluationLog_Success` | Valid EvalLog YAML → 201 + correct row count |
| `TestImportEnforcementLog_Success` | Valid EnfLog YAML → 201 + correct row count |
| `TestImport_InvalidYAML` | Malformed body → 400 |
| `TestImport_UnknownArtifactType` | Valid YAML, wrong type → 400 |
| `TestImport_EmptyEvaluations` | EvalLog with no evaluations → 400 |
| `TestImport_MultipartFile` | multipart/form-data with file → 201 |
| `TestImport_UnsupportedContentType` | JSON body → 415 |
| `TestImport_NATSPublish` | Verify NATS event emitted with correct policyID and count |
| `TestImport_PolicyIDDerivation` | mapping-references parsed correctly |
