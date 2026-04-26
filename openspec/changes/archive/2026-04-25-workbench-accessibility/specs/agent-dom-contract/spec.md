## ADDED Requirements

### Requirement: Main element exposes current view
The `<main>` element SHALL include a `data-view` attribute set to the active view name (e.g., `posture`, `requirements`, `evidence`).

#### Scenario: Agent reads current view
- **WHEN** the user navigates to the requirements view
- **THEN** `<main data-view="requirements">` is present in the DOM

### Requirement: Policy-scoped elements expose policy ID
Elements displaying policy-scoped data (posture cards, filter selects, table rows) SHALL include `data-policy-id` with the policy identifier.

#### Scenario: Posture card exposes policy
- **WHEN** a posture card renders for policy "nist-800-53"
- **THEN** the card element has `data-policy-id="nist-800-53"`

### Requirement: Classification badges expose machine-readable value
Classification badge elements SHALL include `data-classification` with the classification string value.

#### Scenario: Badge value readable by agent
- **WHEN** a requirement row shows classification "No Evidence"
- **THEN** the badge element has `data-classification="No Evidence"`

### Requirement: Matrix rows expose requirement ID
Requirement matrix row elements SHALL include `data-requirement-id` with the requirement identifier.

#### Scenario: Agent reads requirement context
- **WHEN** a requirement row renders for requirement "AC-2.1"
- **THEN** the row element has `data-requirement-id="AC-2.1"`

### Requirement: Evidence rows expose evidence ID
Evidence table rows SHALL include `data-evidence-id` with the evidence identifier.

#### Scenario: Agent reads evidence row
- **WHEN** an evidence row renders
- **THEN** the row element has `data-evidence-id` matching the record's `evidence_id`

### Requirement: Expandable rows expose expanded state
Expandable elements (requirement matrix rows, filter panels) SHALL include `data-expanded="true"` or `data-expanded="false"`.

#### Scenario: Collapsed row
- **WHEN** a requirement row is collapsed
- **THEN** the row has `data-expanded="false"`

#### Scenario: Expanded row
- **WHEN** a requirement row is expanded to show evidence
- **THEN** the row has `data-expanded="true"`

### Requirement: Data attribute contract is documented
The complete set of `data-*` attributes SHALL be documented in `docs/design/architecture.md` under a new "Agent DOM Contract" section.

#### Scenario: Documentation exists
- **WHEN** a developer reads architecture docs
- **THEN** a table lists every `data-*` attribute, its element scope, and its purpose
