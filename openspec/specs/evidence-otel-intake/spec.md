## SUPERSEDED

This spec is superseded by two architecture decisions:

- [OTel Collector Is Environment Infrastructure](../../../docs/decisions/otel-collector-out-of-chart.md) — collector is operator-managed, not deployed by the Studio Helm chart.
- [OTel-Native Ingestion via Collector](../../../docs/decisions/otel-native-ingestion.md) — evidence flows through the collector's ClickHouse exporter directly into the `evidence` table. Studio has no OTLP receiver or ingest binary.

The attribute→column mapping contract is defined in [evidence-semconv-alignment.md](../../../docs/design/evidence-semconv-alignment.md).
