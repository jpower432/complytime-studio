## MODIFIED Requirements

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
