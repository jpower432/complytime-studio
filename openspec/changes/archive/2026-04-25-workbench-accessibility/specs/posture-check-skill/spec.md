## MODIFIED Requirements

### Requirement: Skill uses five-state classification
The skill SHALL instruct the agent to classify each assessment plan into one of five states based on evidence presence, recency, source match, and result. The states in priority order are: Healthy, Failing, Wrong Source, Stale, No Evidence. The classification "No Evidence" replaces the previous "Blind" label and indicates zero evidence rows exist for the plan within the frequency window.

#### Scenario: No evidence exists
- **WHEN** the evidence query returns zero rows for an assessment plan within its frequency window
- **THEN** the agent SHALL classify the plan as "No Evidence"

#### Scenario: Classification table uses updated labels
- **WHEN** the agent produces a readiness table
- **THEN** the classification column uses "No Evidence" (not "Blind") for plans with no evidence
