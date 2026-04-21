## ADDED Requirements

### Requirement: Evidence ingestion endpoint
The system SHALL expose `POST /api/evidence` accepting a JSON array of evidence records and inserting them into ClickHouse.

#### Scenario: Successful batch insert
- **WHEN** a client sends a valid JSON array of evidence records to `POST /api/evidence`
- **THEN** the system inserts all records into ClickHouse and returns 201 with the count of inserted rows

#### Scenario: Schema validation failure
- **WHEN** a record is missing required fields (policy_id, target_id, control_id, collected_at)
- **THEN** the system rejects the entire batch with 400 and a list of validation errors

### Requirement: Evidence query endpoint
The system SHALL expose `GET /api/evidence` with query parameters for policy_id, target_id, control_id, time range (start, end), and pagination (limit, offset).

#### Scenario: Filtered query
- **WHEN** a client sends `GET /api/evidence?policy_id=X&start=2026-01-01&end=2026-03-31`
- **THEN** the system returns matching evidence rows from ClickHouse ordered by collected_at descending
