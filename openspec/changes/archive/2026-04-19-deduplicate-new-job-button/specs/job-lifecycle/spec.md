## MODIFIED Requirements

### Requirement: Jobs view split

The jobs view SHALL display two sections: Active and Recent.

#### Scenario: Active section shows in-progress jobs
- **WHEN** jobs exist with status `submitted`, `working`, `input-required`, or `ready`
- **THEN** the Active section SHALL list those jobs with status badges and available actions

#### Scenario: Recent section shows history
- **WHEN** jobs exist with status `accepted` or `cancelled`
- **THEN** the Recent section SHALL list those jobs with acceptance notes (if any) and delete action

#### Scenario: Empty active state
- **WHEN** no active jobs exist
- **THEN** the Active section SHALL display "No active jobs" with descriptive copy
- **THEN** the Active section SHALL NOT render a duplicate New Job button (the header button is the single entry point)

#### Scenario: Empty history state
- **WHEN** no history jobs exist
- **THEN** the Recent section SHALL be hidden entirely
