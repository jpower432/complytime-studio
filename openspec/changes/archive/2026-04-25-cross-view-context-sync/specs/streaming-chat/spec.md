## ADDED Requirements

### Requirement: Agent context reflects populated signals
The dashboard context injected into agent messages SHALL include non-null values for all shared signals: `policy_id`, `time_range_start`, `time_range_end`, `control_id`, `requirement_id`, `eval_result`.

#### Scenario: Full context after cross-view navigation
- **WHEN** the user has navigated posture -> requirements (setting policy and time range) and opens chat
- **THEN** the injected context JSON includes `policy_id`, `time_range_start`, and `time_range_end` with the values from the shared signals
