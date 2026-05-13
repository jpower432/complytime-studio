## ADDED Requirements

### Requirement: Impact query pattern in evidence-schema skill
The assistant's evidence-schema skill SHALL include a query pattern that joins `evidence` and `mapping_entries` to resolve which framework objectives are affected by control failures.

#### Scenario: Assistant answers "what certifications are affected"
- **WHEN** a user asks which framework objectives are affected by failures for a given policy and date range
- **THEN** the assistant SHALL use `run_select_query` with a JOIN between `evidence` and `mapping_entries` on `(policy_id, control_id)`
- **THEN** the result SHALL include `control_id`, `target_name`, `eval_result`, `framework`, `reference`, `strength`, and `confidence`

#### Scenario: No mapping entries exist for the policy
- **WHEN** the user asks about framework impact but no `mapping_entries` rows exist for the policy
- **THEN** the assistant SHALL inform the user that no framework mappings are available for impact analysis

### Requirement: Impact aggregation by framework objective
The skill SHALL include a query pattern that aggregates failed controls by framework objective.

#### Scenario: Multiple controls affect the same objective
- **WHEN** BP-2, BP-4, and BP-5 all fail and all map to CC8.1
- **THEN** the aggregated result SHALL show CC8.1 with 3 failed controls, the count of affected targets, and the maximum strength value

### Requirement: Impact query filters by date range
The impact query SHALL accept a date range to scope evidence to an audit period.

#### Scenario: Scoped to audit window
- **WHEN** the user specifies an audit period of April 1-18 2026
- **THEN** only evidence rows with `collected_at` within that range SHALL be included in the impact results
