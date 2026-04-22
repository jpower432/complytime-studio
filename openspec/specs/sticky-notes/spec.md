## ADDED Requirements

### Requirement: Sticky notes panel

The chat panel SHALL include a toggleable sticky notes panel accessible via a button in the chat header.

#### Scenario: Toggle panel open
- **WHEN** the user clicks the sticky notes button in the chat header
- **THEN** a notes panel SHALL appear above the input area
- **THEN** existing sticky notes SHALL be displayed as a list

#### Scenario: Toggle panel closed
- **WHEN** the user clicks the sticky notes button while the panel is open
- **THEN** the panel SHALL close

### Requirement: Add a sticky note

The sticky notes panel SHALL include a text input and "Add" action for creating new notes.

#### Scenario: Add a note
- **WHEN** the user types text into the sticky note input and clicks "Add"
- **THEN** a new note SHALL be created with a unique ID, the text, and current timestamp
- **THEN** the note SHALL appear in the notes list
- **THEN** the input SHALL be cleared

#### Scenario: Empty input
- **WHEN** the user clicks "Add" with an empty input
- **THEN** no note SHALL be created

### Requirement: Delete a sticky note

Each sticky note SHALL have a delete button that removes it.

#### Scenario: Delete a note
- **WHEN** the user clicks the delete button on a sticky note
- **THEN** the note SHALL be removed from the list and from localStorage

### Requirement: Sticky note character limit

Each sticky note SHALL be limited to 200 characters.

#### Scenario: Note at limit
- **WHEN** the user types 200 characters into the sticky note input
- **THEN** further input SHALL be prevented
- **THEN** a character count indicator SHALL show "200/200"

### Requirement: Sticky note count limit

The system SHALL enforce a maximum of 10 sticky notes.

#### Scenario: Limit reached
- **WHEN** 10 sticky notes exist
- **THEN** the "Add" input SHALL be disabled
- **THEN** a message SHALL indicate the limit has been reached

### Requirement: Sticky notes persist in localStorage

Sticky notes SHALL be stored under the `studio-sticky-notes` localStorage key as a JSON array of `{id: string, text: string, createdAt: string}`.

#### Scenario: Notes survive page reload
- **WHEN** the user adds a sticky note and reloads the page
- **THEN** the note SHALL still be present after reload

### Requirement: Sticky notes always injected on new tasks

Sticky notes SHALL be serialized as a `<sticky-notes>` block and included in every `streamMessage()` call (new A2A tasks).

#### Scenario: New task with sticky notes
- **WHEN** the user sends the first message of a new task and 3 sticky notes exist
- **THEN** the `streamMessage()` text SHALL include a `<sticky-notes>` block containing all 3 notes
- **THEN** the block SHALL appear before the `<conversation-history>` block

#### Scenario: New task without sticky notes
- **WHEN** the user sends the first message of a new task and no sticky notes exist
- **THEN** no `<sticky-notes>` block SHALL be included
