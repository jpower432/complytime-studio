## ADDED Requirements

### Requirement: OTel Collector receives OTLP evidence signals
The Helm chart SHALL deploy an OTel Collector that receives evidence signals via OTLP (gRPC and HTTP) and exports to ClickHouse.

#### Scenario: Collector receives OTLP/gRPC signal
- **WHEN** a producer sends an OTLP/gRPC log record with `policy.*` attributes to the collector endpoint
- **THEN** the collector accepts the record and exports it to the `evidence` table in ClickHouse

#### Scenario: Collector receives OTLP/HTTP signal
- **WHEN** a producer sends an OTLP/HTTP log record with `policy.*` attributes to the collector endpoint
- **THEN** the collector accepts the record and exports it to the `evidence` table in ClickHouse

#### Scenario: Collector maps semconv attributes to ClickHouse columns
- **WHEN** a log record with `beacon.evidence` attributes is received
- **THEN** each semconv attribute SHALL map to its corresponding ClickHouse column as defined in the semconv-alignment spec

### Requirement: Collector deployment is conditional
The OTel Collector deployment SHALL be conditional on a Helm values flag.

#### Scenario: OTel enabled
- **WHEN** `otel.enabled=true` in Helm values
- **THEN** the collector Deployment, Service, and ConfigMap are rendered

#### Scenario: OTel disabled
- **WHEN** `otel.enabled=false` in Helm values
- **THEN** no collector resources are rendered
- **THEN** `cmd/ingest` remains the only ingestion path

### Requirement: Collector exposes an in-cluster OTLP endpoint
The collector Service SHALL expose OTLP receivers accessible within the cluster.

#### Scenario: In-cluster producer sends evidence
- **WHEN** a pod in the same namespace sends OTLP to `studio-otel-collector:4317` (gRPC) or `studio-otel-collector:4318` (HTTP)
- **THEN** the collector receives and processes the signal

### Requirement: Collector pipeline supports passthrough and enrichment modes
The collector SHALL support two pipeline modes based on the intake path.

#### Scenario: Path A — Gemara-native signal (passthrough)
- **WHEN** a log record arrives with both `policy.*` and `compliance.*` attributes populated
- **THEN** the collector exports the record to ClickHouse without modification

#### Scenario: Path B — raw policy signal (enrichment required)
- **WHEN** a log record arrives with `policy.*` attributes only and `compliance.*` attributes absent
- **THEN** the truthbeam processor enriches the record with `compliance.*` attributes from Gemara artifacts
- **THEN** `compliance.enrichment.status` is set to reflect the enrichment outcome

### Requirement: Collector configuration is documented for multiple topologies
The Helm chart and documentation SHALL describe gateway, agent, and direct deployment patterns.

#### Scenario: Gateway topology (default)
- **WHEN** the chart is installed with default values
- **THEN** a single collector Deployment is created with OTLP receivers and ClickHouse exporter

#### Scenario: Documentation covers agent topology
- **WHEN** a user reads the deployment documentation
- **THEN** instructions describe how to deploy the collector as a sidecar alongside evidence producers

#### Scenario: Documentation covers direct topology
- **WHEN** a user reads the deployment documentation
- **THEN** instructions describe how to run the collector locally with a ClickHouse exporter for development
