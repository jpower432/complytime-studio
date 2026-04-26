## ADDED Requirements

### Requirement: Staleness calculation uses policy assessment plan frequency
The system SHALL determine evidence staleness by comparing `age(collected_at)` against the cycle length derived from the matching assessment plan's `frequency` field in `Policy.adherence.assessment-plans[]`.

#### Scenario: Quarterly frequency with 100-day-old evidence
- **WHEN** an evidence row's `requirement_id` matches an assessment plan with `frequency: quarterly` (90 days) and the evidence is 100 days old
- **THEN** the evidence SHALL be classified as Aging (between 1 and 2 cycles)

#### Scenario: Daily frequency with 3-day-old evidence
- **WHEN** an evidence row matches an assessment plan with `frequency: daily` (1 day) and the evidence is 3 days old
- **THEN** the evidence SHALL be classified as Stale (between 2 and 3 cycles)

### Requirement: On-demand frequency means never stale
The system SHALL classify evidence as Current when the matching assessment plan has `frequency: on-demand`, regardless of evidence age.

#### Scenario: On-demand with old evidence
- **WHEN** an evidence row matches an assessment plan with `frequency: on-demand` and the evidence is 200 days old
- **THEN** the evidence SHALL be classified as Current

### Requirement: Fallback to 30-day threshold
The system SHALL fall back to a 30-day-based threshold when no assessment plan matches the evidence row's `requirement_id`. Fallback buckets: Current ≤7d, Aging ≤30d, Stale ≤90d, Very Stale >90d.

#### Scenario: No matching assessment plan
- **WHEN** an evidence row's `requirement_id` does not match any assessment plan in the policy
- **THEN** the system SHALL apply the 30-day fallback threshold for bucket classification

### Requirement: Frequency-to-days mapping
The system SHALL map frequency values to cycle days: daily=1, weekly=7, monthly=30, quarterly=90, annually=365.

#### Scenario: Frequency mapping
- **WHEN** an assessment plan has `frequency: monthly`
- **THEN** the cycle length SHALL be 30 days

### Requirement: Freshness buckets relative to cycle length
The system SHALL classify evidence into four buckets based on the ratio of age to cycle length: Current (≤1 cycle), Aging (≤2 cycles), Stale (≤3 cycles), Very Stale (>3 cycles).

#### Scenario: Evidence at 1.5 cycles
- **WHEN** evidence age is 1.5x the cycle length
- **THEN** the evidence SHALL be classified as Aging
