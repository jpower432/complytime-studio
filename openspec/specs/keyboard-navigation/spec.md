## ADDED Requirements

### Requirement: Skip-to-main link
The app shell SHALL render a visually hidden skip link as the first focusable element that jumps focus to `<main>` on activation.

#### Scenario: Keyboard user skips navigation
- **WHEN** a keyboard user presses Tab on page load
- **THEN** the first focused element is a "Skip to main content" link
- **WHEN** the user activates the link
- **THEN** focus moves to the `<main>` element

### Requirement: Visible focus indicators
All interactive elements (buttons, links, inputs, table rows, select elements) SHALL display a visible focus outline when focused via keyboard (`:focus-visible`).

#### Scenario: Button receives keyboard focus
- **WHEN** a user tabs to a button
- **THEN** a visible outline (minimum 2px, contrast ratio 3:1 against adjacent colors) appears

#### Scenario: Mouse click does not show outline
- **WHEN** a user clicks a button with a mouse
- **THEN** no focus outline is displayed

### Requirement: Logical tab order
Tab order SHALL follow the visual layout: skip link -> header -> sidebar nav items -> main content interactive elements -> chat FAB.

#### Scenario: Tab through app shell
- **WHEN** a keyboard user tabs through the page
- **THEN** focus moves through header, sidebar items (top to bottom), main content, then chat button

### Requirement: Chat input is keyboard accessible
The chat overlay input SHALL be reachable via keyboard and support Enter to send without requiring mouse interaction.

#### Scenario: Send message via keyboard
- **WHEN** the chat overlay is open and the user tabs to the input
- **THEN** the input receives focus
- **WHEN** the user types a message and presses Enter
- **THEN** the message is sent

### Requirement: WCAG AA contrast compliance
All text/background combinations SHALL meet WCAG AA contrast ratios: 4.5:1 for normal text, 3:1 for large text (18px+ or 14px+ bold).

#### Scenario: Light theme contrast
- **WHEN** the light theme is active
- **THEN** all text/background pairs meet WCAG AA ratios

#### Scenario: Dark theme contrast
- **WHEN** the dark theme is active
- **THEN** all text/background pairs meet WCAG AA ratios
