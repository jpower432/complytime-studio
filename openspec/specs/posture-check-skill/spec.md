## Requirements

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

### Requirement: Skill uses five-state classification
The skill SHALL instruct the agent to classify each assessment plan into one of five states based on evidence presence, recency, source match, and result. The states in priority order are: Healthy, Failing, Wrong Source, Stale, No Evidence. The classification "No Evidence" replaces the previous "Blind" label and indicates zero evidence rows exist for the plan within the frequency window.

#### Scenario: No evidence exists
- **WHEN** the evidence query returns zero rows for an assessment plan within its frequency window
- **THEN** the agent SHALL classify the plan as "No Evidence"

#### Scenario: Classification table uses updated labels
- **WHEN** the agent produces a readiness table
- **THEN** the classification column uses "No Evidence" (not "Blind") for plans with no evidence

### Requirement: Posture check supports event-triggered mode
The posture-check skill SHALL support an event-triggered mode where it receives a `policy_id` and recent evidence summary from the gateway (via A2A) instead of requiring the user to specify a policy in chat. The result SHALL be a lightweight posture delta — not a full audit log.

#### Scenario: Event-triggered posture check
- **WHEN** the gateway triggers a posture check for policy "ampel-branch-protection" after evidence arrival
- **THEN** the agent runs the posture-check skill with the policy ID, compares current pass rate to last known, and returns a structured delta: `{"policy_id":"...","previous_pass_rate":89,"current_pass_rate":72,"new_findings":3}`

#### Scenario: No change detected
- **WHEN** the gateway triggers a posture check and the pass rate has not changed
- **THEN** the agent returns a no-change result and no inbox notification is created

### Requirement: Posture check result creates inbox notification
The system SHALL create an inbox notification when the event-triggered posture check detects a meaningful change (pass rate delta > 2% or new findings > 0).

#### Scenario: Pass rate drop creates notification
- **WHEN** the posture check detects a pass rate drop from 89% to 72%
- **THEN** a notification is created in the inbox with type "posture_change", the policy ID, and the delta details
