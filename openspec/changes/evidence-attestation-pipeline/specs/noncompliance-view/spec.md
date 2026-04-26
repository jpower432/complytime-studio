## ADDED Requirements

### Requirement: Materialized view surfaces non-compliant evidence
The system SHALL maintain a ClickHouse materialized view that incrementally captures evidence rows where `eval_result` is `Failed` or `Needs Review`, or `compliance_status` is `Non-Compliant`.

#### Scenario: Failed evidence captured
- **WHEN** an evidence row is inserted with `eval_result = 'Failed'`
- **THEN** the row SHALL appear in the materialized view

#### Scenario: Passing evidence excluded
- **WHEN** an evidence row is inserted with `eval_result = 'Passed'` and `compliance_status = 'Compliant'`
- **THEN** the row SHALL NOT appear in the materialized view

#### Scenario: Needs Review evidence captured
- **WHEN** an evidence row is inserted with `eval_result = 'Needs Review'`
- **THEN** the row SHALL appear in the materialized view

### Requirement: Agent queries non-compliance view for posture questions
The assistant agent SHALL query the non-compliance materialized view when answering posture, readiness, or compliance status questions to surface failing evidence efficiently.

#### Scenario: Posture question routes to view
- **WHEN** the auditor asks "what's failing?" or "are we compliant with SOX?"
- **THEN** the agent SHALL query the non-compliance view rather than scanning the full evidence table

#### Scenario: View returns current data
- **WHEN** new failing evidence is ingested 1 minute before the auditor asks a posture question
- **THEN** the agent's response SHALL include the newly ingested failing evidence
