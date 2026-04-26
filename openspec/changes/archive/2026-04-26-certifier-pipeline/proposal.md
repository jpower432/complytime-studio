## Why

Evidence enters ClickHouse without any trust signal. The `PostureCheckHandler` computes pass-rate deltas but never evaluates whether evidence is well-formed, has known provenance, or was produced by a registered engine. Manual CSV upload and form entry create an unverifiable backdoor. Studio currently reaches into the client's trust boundary (OCI registry) for on-demand attestation verification — conflating client-controlled artifacts with Studio-controlled trust decisions.

## What Changes

- **Remove manual evidence upload paths** — drop CSV import and manual form entry from the UI and REST API. All non-OTel evidence flows through structured Gemara artifact uploads via `cmd/ingest`. **BREAKING**
- **Introduce a certifier pipeline** — async, post-ingest certifiers that watch for new evidence via NATS and independently assess trust from within Studio's boundary. Certifiers never reach into the client's trust boundary.
- **Add `certifications` table** — append-only operational metadata in ClickHouse. One row per certifier per evidence row. Not a ledger (see `docs/decisions/transparency-ledger.md`).
- **Add `evidence.certified` column** — denormalized bool on the evidence table for fast reads. Computed from certifications: true = at least one pass, no fails.
- **Day-one certifiers** — schema (metadata valid), provenance (source known to Studio), executor (engine registered), attestation (bundle verifies from Studio's OCI copy).
- **Surface certification status in the UI** — row-level indicator, certification bar, expand for per-certifier breakdown, filterable via existing filter chip system.

## Capabilities

### New Capabilities
- `certifier-interface`: Extensible certifier contract (Name, Certify → pass/fail/skip/error) and pipeline runner that executes registered certifiers against new evidence rows.
- `certifier-schema`: Day-one schema certifier — validates evidence row metadata presence and types.
- `certifier-provenance`: Day-one provenance certifier — checks `source_registry` against Studio's known registries.
- `certifier-executor`: Day-one executor certifier — checks `engine_name` against a registered engine list.
- `certifier-attestation`: Day-one attestation certifier — verifies in-toto bundle from Studio's OCI registry copy (structure, signatures, chain integrity). No layout comparison (requires policy context, remains agent-driven).
- `certifications-table`: ClickHouse schema for the `certifications` table and `evidence.certified` column. Includes `certifier_version` for tracking rule changes.
- `certification-handler`: NATS handler that replaces manual upload event handling. Subscribes to `EvidenceEvent`, runs the certifier pipeline, writes results to the certifications table, updates `evidence.certified`.
- `certification-ui`: Evidence table row indicator, certification summary bar, per-certifier detail expand, certification as a filterable field.
- `remove-manual-upload`: Remove CSV import endpoint, manual form entry UI, and upload button from evidence views.

### Modified Capabilities
- `evidence-event-bus`: `EvidenceEvent` unchanged, but the `CertificationHandler` becomes a new subscriber alongside the existing `PostureCheckHandler`.

## Impact

- **Backend** — `internal/events/`: new `CertificationHandler`, certifier interface and implementations. `cmd/ingest/`: no changes (already Gemara-only). `internal/store/handlers.go`: remove `POST /api/evidence/upload` and CSV import handler.
- **Schema** — `clickhouse-schema-configmap.yaml`: add `certifications` table, add `certified` column to `evidence` table.
- **Frontend** — `evidence-view.tsx`: remove upload button and form entirely (not just hide when embedded), add certification column/indicator, add certification bar, add certification to filter menu. `add-filter-menu.tsx`: add certification field option.
- **API** — `POST /api/evidence/upload` removed. **BREAKING** for any client using direct upload.
- **Dependencies** — No new external dependencies. Certifiers use existing OCI/oras infrastructure for attestation verification.
- **Decisions** — `docs/decisions/transparency-ledger.md` already documents the future path to Trillian for tamper-evident logging.
