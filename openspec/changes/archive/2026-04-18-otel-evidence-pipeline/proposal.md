## Why

The evidence pipeline uses a batch Go CLI (`cmd/ingest`) that parses Gemara YAML, flattens nested structures, and inserts rows into ClickHouse via native protocol. This works for initial development but does not scale to the ComplyTime ecosystem where evidence originates from multiple sources: `complyctl` assessments (Gemara-native, instrumented via ProofWatch), admission controllers, CI scanners, and other policy engines (OPA, Gatekeeper) that have no knowledge of Gemara. The current pipeline also splits evaluation and enforcement into separate ClickHouse tables, forcing the gap-analyst to correlate across tables for what is logically a single evidence record.

The broader ComplyTime ecosystem (`complytime-collector-components`) is already defining OTel semantic conventions for policy and compliance signals and building collector components (truthbeam enrichment processor). Studio's evidence store should align with this standard rather than maintaining a parallel ingestion path.

## What Changes

- Replace the two-table ClickHouse schema (`evaluation_logs`, `enforcement_actions`) with a single `evidence` table aligned to the `beacon.evidence` OTel entity. Evaluation and remediation data co-located as a single evidence record.
- Define the mapping between `beacon.evidence` semconv attributes and ClickHouse columns. Document gaps in the current semconv that require new attributes (`compliance.policy.id`, `compliance.assessment.requirement.id`, `compliance.assessment.plan.id`, `compliance.assessment.confidence`, `compliance.assessment.steps`).
- Add an OTel Collector deployment to the Helm chart with the ClickHouse exporter, supporting two intake paths:
  - **Path A (complyctl/ProofWatch):** Gemara-native signals with full `policy.*` + `compliance.*` attributes. Collector passes through to ClickHouse.
  - **Path B (raw policy engines):** Signals with `policy.*` attributes only. The truthbeam collector processor enriches with `compliance.*` context from Gemara artifacts (Policy, ControlCatalog, MappingDocuments).
- Retain `cmd/ingest` as a local testing tool for loading Gemara YAML directly into ClickHouse without requiring the full OTel stack. Update it to write to the merged `evidence` table.
- Update the gap-analyst prompt and example queries for single-table access.
- Document collector deployment patterns: gateway (centralized), agent (co-located sidecar), and direct (local collector with ClickHouse exporter).

## Capabilities

### New Capabilities

- `evidence-otel-intake`: OTel Collector deployment with OTLP receiver and ClickHouse exporter. Receives evidence signals from complyctl (Path A) and raw policy engines (Path B). Supports gateway and agent deployment topologies.
- `semconv-alignment`: Mapping between the `beacon.evidence` OTel semantic convention and the ClickHouse `evidence` table schema. Documents required semconv additions for Gemara-specific attributes.
- `evidence-schema-merge`: Migration from two-table schema (`evaluation_logs`, `enforcement_actions`) to a single `evidence` table with nullable remediation columns. Includes DDL, sort key, partitioning, and TTL.

### Modified Capabilities

- `evidence-ingestion`: Update `cmd/ingest` to write to the merged `evidence` table instead of two separate tables. Retain as a local testing tool — not the production ingestion path.

## Impact

- **ClickHouse schema**: Breaking change — existing `evaluation_logs` and `enforcement_actions` tables replaced by `evidence`. Requires data migration or re-ingestion.
- **Helm chart**: New OTel Collector Deployment/Service templates (conditional on `otel.enabled` or similar). New ConfigMap for collector pipeline configuration.
- **`cmd/ingest`**: Updated to target merged table. No longer the primary ingestion path but retained for local dev.
- **Gap-analyst prompt**: Queries change from two-table to single-table. Simpler queries, no cross-table correlation.
- **External dependency**: `complytime-collector-components` repository — semconv definitions and truthbeam processor. Studio consumes the convention; the collector components repo owns it.
- **Semconv upstream**: New attributes proposed to `complytime-collector-components/model/attributes.yaml` in the `registry.compliance` group.
