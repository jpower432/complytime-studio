## ADDED Requirements

### Requirement: Draft Review auto-saves reviewer edits
The Draft Review UI SHALL auto-save reviewer edits (type overrides and notes) to the server via `PATCH /api/draft-audit-logs/{id}` with a 1-second debounce after each change. A "Saving..." / "Saved" indicator SHALL be displayed.

#### Scenario: Type override triggers auto-save
- **WHEN** the reviewer changes a result type from "Finding" to "Strength"
- **THEN** the UI debounces for 1 second and sends a PATCH with the updated `reviewer_edits`
- **THEN** a "Saved" indicator appears after successful save

#### Scenario: Note input triggers auto-save
- **WHEN** the reviewer types a note on a result
- **THEN** the UI debounces for 1 second and sends a PATCH with the updated `reviewer_edits`

### Requirement: Draft Review loads persisted edits on open
When the reviewer opens a draft detail, the UI SHALL read `reviewer_edits` from the GET response and pre-fill type overrides and notes for each result card.

#### Scenario: Reopen draft with saved edits
- **WHEN** the reviewer navigates away and returns to a draft with saved edits
- **THEN** the type override dropdowns and notes reflect the previously saved values

#### Scenario: Open draft with no edits
- **WHEN** the reviewer opens a draft that has no reviewer edits
- **THEN** all result cards show the original agent classification with empty notes

### Requirement: Save indicator shows auto-save state
The Draft Review detail panel SHALL display a save indicator with three states: idle (hidden), saving ("Saving..."), saved ("Saved").

#### Scenario: Save lifecycle
- **WHEN** an edit triggers auto-save
- **THEN** the indicator shows "Saving..." during the PATCH request
- **THEN** the indicator shows "Saved" after a successful response
- **THEN** the indicator fades after 2 seconds
