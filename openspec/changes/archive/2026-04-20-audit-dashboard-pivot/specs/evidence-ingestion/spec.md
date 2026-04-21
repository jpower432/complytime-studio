## MODIFIED Requirements

### Requirement: Multi-channel evidence ingestion
The system SHALL support three evidence ingestion channels: OpenTelemetry (OTLP receiver), REST API (`POST /api/evidence`), and file upload (`POST /api/evidence/upload`). All channels write to the same ClickHouse evidence table.

#### Scenario: Evidence from OTel
- **WHEN** an OTLP-compatible client sends evidence spans/logs to the gateway's OTel receiver
- **THEN** the system transforms them to the evidence schema and inserts into ClickHouse

#### Scenario: Evidence from REST API
- **WHEN** a client sends a JSON batch to `POST /api/evidence`
- **THEN** the system validates and inserts into ClickHouse

#### Scenario: Evidence from file upload
- **WHEN** a user uploads a CSV or JSON file via `POST /api/evidence/upload`
- **THEN** the system parses, validates, and inserts into ClickHouse

### Requirement: ClickHouse is required
The system SHALL require ClickHouse as a running dependency. The gateway SHALL fail startup health checks if ClickHouse is unreachable.

#### Scenario: ClickHouse unavailable
- **WHEN** ClickHouse is unreachable at gateway startup
- **THEN** the gateway reports unhealthy on its readiness probe and logs the connection error
