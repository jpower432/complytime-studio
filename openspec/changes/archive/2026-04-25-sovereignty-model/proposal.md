## Why

Compliance evidence (screenshots, logs, scan output) often contains PII or regulated data subject to GDPR, EUCS, or other residency requirements. This data cannot leave its trust boundary. Studio has no mechanism to prevent raw evidence from entering ClickHouse, no concept of where evidence originates, and no way for auditors to trace back to the source when they need the full artifact.

The [cloud-native posture correction](../../docs/decisions/cloud-native-posture-correction.md) establishes the sovereignty model: Studio is deployed centrally and receives summaries only. Raw evidence stays in per-boundary OCI registries as attestation bundles.

## What Changes

- **`source_registry` column** on the `evidence` table: nullable string populated by complyctl in OTel attributes or REST payload. Contains the hostname/URL of the OCI registry where the raw attestation bundle resides.
- **Boundary contract documentation**: complyctl pushes raw evidence to the boundary's OCI registry, then sends a summary row to Studio containing the OCI digest (`attestation_ref`) and registry location (`source_registry`). Studio stores the summary. Raw evidence never crosses the boundary. Formal split of responsibilities: [Trust Boundary Contract](../../../docs/design/architecture.md#trust-boundary-contract).
- **Evidence source display**: Workbench evidence views show `source_registry` so auditors can identify where to retrieve raw artifacts. The attestation-verification skill uses `source_registry` to resolve OCI references when verifying provenance.
- **Ingestion validation**: Gateway optionally warns when evidence rows contain fields that suggest raw data (large `eval_message` payloads, embedded base64) rather than summaries.

## Capabilities

### New Capabilities
- `source-registry-tracking`: Evidence rows carry `source_registry` identifying the OCI registry in the originating trust boundary
- `boundary-contract`: Documented contract for what crosses a trust boundary (summary metadata and OCI digests only) and what stays (raw artifacts)

### Modified Capabilities
- `evidence-semconv-alignment`: Add `source_registry Nullable(String)` column to the evidence table and semconv mapping
- `attestation-verification`: Use `source_registry` to resolve OCI references when the registry differs from Studio's default

## Impact

- **ClickHouse**: New `source_registry` column on `evidence` table (nullable, additive migration)
- **Gateway**: Accept `source_registry` in REST payload and OTel attribute mapping
- **Workbench**: Evidence detail view displays source registry with link/copy affordance
- **Agent**: Attestation-verification skill passes `source_registry` to oras-mcp when pulling bundles
- **complyctl (external)**: Populates `compliance.source.registry` OTel attribute after pushing to boundary registry
- **Documentation**: Boundary contract added to `docs/design/`

## Constitution Alignment

### I. Autonomous Collaboration

**Assessment**: PASS

The boundary contract is artifact-based. complyctl produces attestation bundles and summary rows independently. Studio consumes summaries without knowledge of boundary internals.

### II. Composability First

**Assessment**: PASS

`source_registry` is optional. Evidence without it works identically — sovereignty tracking degrades gracefully to "unknown source." No mandatory dependency on regional infrastructure.

### III. Observable Quality

**Assessment**: PASS

Every evidence row with `source_registry` and `attestation_ref` is fully traceable to its origin. Auditors can verify provenance end-to-end.

### IV. Testability

**Assessment**: PASS

Testable with a local OCI registry standing in for a boundary registry. Integration tests verify that summary rows with `source_registry` resolve correctly via oras-mcp.
