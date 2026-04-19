## ADDED Requirements

### Requirement: Job state machine

The system SHALL manage jobs through a defined state machine: `submitted → working → ready → accepted`. The `cancelled` state SHALL be reachable from `submitted`, `working`, `input-required`, and `ready`. The `failed` state SHALL be set when the agent reports failure.

#### Scenario: Agent completes work
- **WHEN** the SSE stream emits a `TaskStatusUpdateEvent` with `state: "completed"`
- **THEN** the job status SHALL transition to `ready` (not removed from the active list)

#### Scenario: Agent fails
- **WHEN** the SSE stream emits a `TaskStatusUpdateEvent` with `state: "failed"`
- **THEN** the job status SHALL transition to `failed` and display the error message from the event

#### Scenario: Agent requests input
- **WHEN** the SSE stream emits a `TaskStatusUpdateEvent` with `state: "input-required"`
- **THEN** the job status SHALL transition to `input-required` and the reply input SHALL be enabled

### Requirement: Cancel job

The system SHALL allow the user to cancel any active job (status `submitted`, `working`, `input-required`, or `ready`). Cancel is client-side only.

#### Scenario: User cancels a working job
- **WHEN** the user clicks "Cancel" on a job with status `working`
- **THEN** the system SHALL close the SSE EventSource connection
- **THEN** the job status SHALL transition to `cancelled`
- **THEN** the job SHALL move to the Recent history section

#### Scenario: User cancels a ready job
- **WHEN** the user clicks "Cancel" on a job with status `ready`
- **THEN** the job status SHALL transition to `cancelled` with no SSE cleanup needed
- **THEN** the job SHALL move to the Recent history section

### Requirement: Accept job with note

The system SHALL allow the user to accept a job in `ready` status. Acceptance SHALL open a dialog with an optional text note field.

#### Scenario: Accept with note
- **WHEN** the user clicks "Accept" on a `ready` job and enters a note
- **THEN** the system SHALL store `acceptedAt` timestamp and `acceptNote` on the job
- **THEN** the job SHALL move to the Recent history section with status `accepted`

#### Scenario: Accept without note
- **WHEN** the user clicks "Accept" on a `ready` job and leaves the note empty
- **THEN** the system SHALL store `acceptedAt` timestamp with an empty note
- **THEN** the job SHALL move to the Recent history section

### Requirement: Delete history job

The system SHALL allow the user to delete jobs in the Recent history section (`accepted` or `cancelled`).

#### Scenario: Delete a history job
- **WHEN** the user clicks "Delete" on a history job
- **THEN** the job SHALL be removed from localStorage permanently

### Requirement: Auto-purge history after 7 days

The system SHALL automatically remove history jobs older than 7 days. Age is measured from `acceptedAt` for accepted jobs and `updatedAt` for cancelled jobs.

#### Scenario: Purge on app load
- **WHEN** the workbench app loads
- **THEN** the system SHALL remove all history jobs older than 7 days from localStorage

#### Scenario: Periodic purge
- **WHEN** 60 minutes have elapsed since the last purge check
- **THEN** the system SHALL remove all history jobs older than 7 days from localStorage

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
