## ADDED Requirements

### Requirement: Certifications table schema
The system SHALL create a `certifications` table in ClickHouse with columns: `evidence_id` (String), `certifier` (LowCardinality String), `certifier_version` (LowCardinality String), `result` (Enum: pass/fail/skip/error), `reason` (String), `certified_at` (DateTime64 DEFAULT now64).

#### Scenario: Table created on init
- **WHEN** ClickHouse schema initialization runs
- **THEN** the `certifications` table SHALL exist with the specified columns, partitioned by `toYYYYMM(certified_at)`, ordered by `(evidence_id, certifier, certified_at)`

### Requirement: Certifications table is append-only by convention
The system SHALL only INSERT to the `certifications` table, never UPDATE or DELETE. Re-running a certifier (e.g., after version bump) SHALL append a new row. Historical verdicts SHALL remain.

#### Scenario: Certifier re-run appends
- **WHEN** the schema certifier v1.1 re-certifies an evidence row previously certified by v1.0
- **THEN** both the v1.0 and v1.1 rows SHALL exist in the table

### Requirement: Evidence certified column
The system SHALL add a `certified` column (Bool DEFAULT false) to the `evidence` table.

#### Scenario: Column added
- **WHEN** schema migration runs
- **THEN** the `evidence` table SHALL have a `certified` column defaulting to `false`

### Requirement: Certified computed from certifications
The `evidence.certified` column SHALL be set to `true` when the latest run per certifier has at least one `pass` and zero `fail` verdicts. `skip` and `error` do not count as pass or fail.

#### Scenario: All pass
- **WHEN** certifiers return [pass, pass, skip]
- **THEN** `evidence.certified` SHALL be `true`

#### Scenario: Any fail
- **WHEN** certifiers return [pass, fail, pass]
- **THEN** `evidence.certified` SHALL be `false`

#### Scenario: Only skip and error
- **WHEN** certifiers return [skip, error, skip]
- **THEN** `evidence.certified` SHALL be `false` (no affirmative pass)

### Requirement: Pre-existing evidence defaults uncertified
All evidence rows existing before the `certified` column is added SHALL have `certified = false`. No grandfather clause.

#### Scenario: Existing data
- **WHEN** the migration adds the `certified` column
- **THEN** all existing rows SHALL have `certified = false`
