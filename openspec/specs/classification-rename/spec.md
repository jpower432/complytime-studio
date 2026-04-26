## ADDED Requirements

### Requirement: Classification uses No Evidence instead of Blind
All references to the "Blind" classification SHALL be renamed to "No Evidence" across frontend, backend, skills, and agent prompts.

#### Scenario: Frontend filter shows No Evidence
- **WHEN** a user opens the classification filter in the requirement matrix
- **THEN** the option reads "No Evidence" (not "Blind")

#### Scenario: Badge renders No Evidence
- **WHEN** a requirement has no associated evidence
- **THEN** the classification badge displays "No Evidence"

#### Scenario: Backend query returns No Evidence
- **WHEN** ClickHouse classification logic evaluates a requirement with zero evidence rows
- **THEN** the classification value is "No Evidence"

#### Scenario: Skill references No Evidence
- **WHEN** the posture-check skill classifies a requirement with no evidence
- **THEN** the classification table uses "No Evidence" as the label

#### Scenario: Agent prompt uses No Evidence
- **WHEN** the assistant prompt references classification states
- **THEN** "No Evidence" is used consistently (not "Blind")
