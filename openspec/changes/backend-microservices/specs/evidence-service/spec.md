## DEFERRED Requirements

> Evidence service extraction is deferred. The requirements below document the
> future extraction contract, enabled by the modulith interface boundaries
> established in this change. Implement when measured load justifies the split.

### Requirement: Evidence service owns write path when extracted

IF the evidence service is deployed as a separate binary, THEN the evidence
service SHALL own `POST /api/evidence` (JSON body ingestion),
`POST /api/evidence/upload` (file upload), and the OpenTelemetry evidence
intake endpoint. The gateway SHALL NOT insert into or mutate the evidence
ClickHouse tables for those paths.

### Requirement: Evidence read API compatibility when extracted

IF the evidence service is extracted, THEN the evidence service SHALL expose
`GET /api/evidence` with the same query parameters and response semantics as
the pre-split gateway behavior.

### Requirement: ClickHouse evidence schema ownership when extracted

IF the evidence service is extracted, THEN DDL and migrations for ClickHouse
evidence tables SHALL be owned and applied by the evidence service. Other
services MAY read evidence tables as read-only consumers.
