## ADDED Requirements

### Requirement: CertificationHandler subscribes to evidence events
The `CertificationHandler` SHALL subscribe to `studio.evidence.>` via NATS, the same subject pattern as `PostureCheckHandler`.

#### Scenario: Evidence event received
- **WHEN** an `EvidenceEvent` is published on NATS
- **THEN** the `CertificationHandler` SHALL receive the event alongside (not instead of) the `PostureCheckHandler`

### Requirement: CertificationHandler queries new evidence rows
On receiving an `EvidenceEvent`, the handler SHALL query ClickHouse for evidence rows matching the event's `policy_id` within a recent ingestion window.

#### Scenario: New evidence found
- **WHEN** the handler queries for evidence matching the event's `policy_id`
- **THEN** it SHALL retrieve the newly ingested rows for certification

#### Scenario: No matching rows
- **WHEN** the query returns no rows (race condition or deletion)
- **THEN** the handler SHALL log a warning and return without error

### Requirement: CertificationHandler runs pipeline per row
The handler SHALL invoke the certifier pipeline for each evidence row returned by the query.

#### Scenario: Multiple rows ingested
- **WHEN** an event indicates 5 new evidence rows for a policy
- **THEN** the pipeline SHALL run against each of the 5 rows independently

### Requirement: CertificationHandler writes results
After running the pipeline, the handler SHALL batch INSERT all `CertResult` entries to the `certifications` table and UPDATE `evidence.certified` for each affected row.

#### Scenario: Results persisted
- **WHEN** the pipeline returns [pass, fail, skip] for an evidence row
- **THEN** 3 rows SHALL be inserted into `certifications` and `evidence.certified` SHALL be set to `false` (due to the fail)

### Requirement: CertificationHandler is non-blocking
The handler SHALL NOT block evidence ingestion. If the handler fails (ClickHouse error, pipeline panic), evidence remains in ClickHouse with `certified = false`.

#### Scenario: Handler failure
- **WHEN** the `CertificationHandler` encounters a ClickHouse write error
- **THEN** the error SHALL be logged, and previously ingested evidence SHALL remain unaffected with `certified = false`
