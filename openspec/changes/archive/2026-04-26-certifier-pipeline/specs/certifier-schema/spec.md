## ADDED Requirements

### Requirement: Schema certifier validates metadata presence
The schema certifier SHALL verify that required metadata fields on the evidence row are present and non-empty: `evidence_id`, `target_id`, `rule_id`, `eval_result`, `compliance_status`, `collected_at`.

#### Scenario: All required fields present
- **WHEN** an evidence row has all required metadata fields populated
- **THEN** the schema certifier SHALL return `pass`

#### Scenario: Missing required field
- **WHEN** an evidence row has an empty `target_id`
- **THEN** the schema certifier SHALL return `fail` with reason identifying the missing field

### Requirement: Schema certifier validates field types
The schema certifier SHALL verify that enum fields (`eval_result`, `compliance_status`) contain values within their defined ClickHouse enum sets.

#### Scenario: Invalid enum value
- **WHEN** an evidence row has `eval_result` set to a value not in the defined enum
- **THEN** the schema certifier SHALL return `fail` with reason identifying the invalid value

#### Scenario: Valid enum values
- **WHEN** all enum fields contain values within their defined sets
- **THEN** the schema certifier SHALL not fail on enum validation

### Requirement: Schema certifier validates timestamp
The schema certifier SHALL verify that `collected_at` is not zero-valued and not in the future (beyond a 5-minute clock skew tolerance).

#### Scenario: Future timestamp
- **WHEN** `collected_at` is more than 5 minutes in the future
- **THEN** the schema certifier SHALL return `fail` with reason "collected_at is in the future"

#### Scenario: Zero timestamp
- **WHEN** `collected_at` is the zero value
- **THEN** the schema certifier SHALL return `fail` with reason "collected_at is missing"
