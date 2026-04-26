## ADDED Requirements

### Requirement: Executor certifier checks engine registration
The executor certifier SHALL verify that the evidence row's `engine_name` is present and matches a registered engine in Studio's configuration.

#### Scenario: Known engine
- **WHEN** `engine_name` matches a registered engine
- **THEN** the executor certifier SHALL return `pass`

#### Scenario: Unknown engine
- **WHEN** `engine_name` is present but does not match any registered engine
- **THEN** the executor certifier SHALL return `fail` with reason identifying the unknown engine name

#### Scenario: Missing engine_name
- **WHEN** `engine_name` is null
- **THEN** the executor certifier SHALL return `fail` with reason "engine_name is missing"

### Requirement: Executor certifier skips when not applicable
The executor certifier SHALL skip rows that have no meaningful engine context (e.g., manually enriched rows that predate this change).

#### Scenario: Enrichment-only row
- **WHEN** the evidence row has `enrichment_status = 'Skipped'` and `engine_name` is null
- **THEN** the executor certifier SHALL return `skip` with reason "no engine context"
