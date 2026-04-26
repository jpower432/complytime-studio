## ADDED Requirements

### Requirement: Skill extracts assessment plans from Policy YAML
The posture-check skill SHALL instruct the agent to parse the `policies.content` YAML column and extract the `adherence.assessment-plans[]` array. For each plan, the agent SHALL extract: `id`, `requirement-id`, `frequency`, `evaluation-methods[].executor` (id, type, version), `evaluation-methods[].mode`, and `evidence-requirements`.

#### Scenario: Policy with assessment plans
- **WHEN** the agent loads a Policy from ClickHouse whose `content` contains `adherence.assessment-plans` with 3 entries
- **THEN** the agent SHALL extract all 3 plans with their requirement-id, frequency, and evaluation-methods

#### Scenario: Policy without assessment plans
- **WHEN** the agent loads a Policy whose `content` has no `adherence.assessment-plans` array
- **THEN** the agent SHALL report "Policy has no assessment plans defined" and halt the posture check

### Requirement: Skill defines per-plan evidence query
The skill SHALL instruct the agent to query the `evidence` table for each assessment plan using `policy_id`, `requirement_id`, and a time window derived from the plan's `frequency`. The query SHALL select `engine_name`, `engine_version`, `eval_result`, `collected_at`, `confidence`, and `plan_id`.

#### Scenario: Query by requirement and frequency window
- **WHEN** an assessment plan has `requirement-id: AR-vuln-scan` and `frequency: quarterly`
- **THEN** the agent SHALL query evidence WHERE `requirement_id = 'AR-vuln-scan'` AND `collected_at` is within the most recent 90-day window

#### Scenario: Fallback when plan_id is NULL
- **WHEN** evidence rows matching a requirement have `plan_id` as NULL
- **THEN** the agent SHALL still match by `requirement_id` and note the missing plan linkage in the readiness table

### Requirement: Skill validates executor provenance
The skill SHALL instruct the agent to compare each evidence row's `engine_name` against the assessment plan's `evaluation-methods[].executor.id`. A mismatch SHALL classify the plan as "Wrong Source."

#### Scenario: Executor matches
- **WHEN** the assessment plan specifies `executor.id: nessus` and evidence rows have `engine_name = 'nessus'`
- **THEN** the agent SHALL pass the provenance check for that plan

#### Scenario: Executor mismatch
- **WHEN** the assessment plan specifies `executor.id: nessus` and evidence rows have `engine_name = 'qualys'`
- **THEN** the agent SHALL classify the plan as "Wrong Source" with message "Expected: nessus, Got: qualys"

#### Scenario: NULL engine_name
- **WHEN** evidence rows have `engine_name` as NULL
- **THEN** the agent SHALL classify as "Unknown Source" and note that provenance cannot be verified

### Requirement: Skill checks cadence against frequency
The skill SHALL instruct the agent to compute expected collection cycles using the same frequency mapping as the studio-audit skill (daily=1d, weekly=7d, monthly=30d, quarterly=90d, annually=365d). Missing cycles within the audit window SHALL classify the plan as "Stale."

#### Scenario: Evidence on cadence
- **WHEN** a monthly plan has evidence collected within the last 30 days
- **THEN** the agent SHALL classify cadence as current

#### Scenario: Evidence outside frequency window
- **WHEN** a quarterly plan's most recent evidence is 190 days old
- **THEN** the agent SHALL classify the plan as "Stale" with the age noted

### Requirement: Skill classifies each plan into five states
The skill SHALL define five readiness states: Healthy, Failing, Wrong Source, Stale, Blind. Classification priority (worst wins): Blind > Wrong Source > Stale > Failing > Healthy.

#### Scenario: Healthy plan
- **WHEN** evidence exists, executor matches, cadence is current, and latest `eval_result` is Passed
- **THEN** the agent SHALL classify the plan as "Healthy"

#### Scenario: Failing plan
- **WHEN** evidence exists, executor matches, cadence is current, and latest `eval_result` is Failed
- **THEN** the agent SHALL classify the plan as "Failing"

#### Scenario: Blind plan
- **WHEN** no evidence rows exist for the plan's `requirement_id` within the audit window
- **THEN** the agent SHALL classify the plan as "Blind"

#### Scenario: Multiple conditions apply
- **WHEN** evidence exists but both executor mismatches AND cadence is stale
- **THEN** the agent SHALL classify as "Wrong Source" (higher priority than Stale)
