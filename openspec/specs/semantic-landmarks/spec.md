## ADDED Requirements

### Requirement: Views use semantic container elements
Every workbench view component SHALL wrap its root content in a `<section>` element with an accessible heading (`<h2>` or equivalent).

#### Scenario: Posture view renders as section
- **WHEN** the posture view loads with policy data
- **THEN** the root element is `<section>` containing an `<h2>` heading

#### Scenario: Empty state renders as section
- **WHEN** any view loads with no data
- **THEN** the empty state is wrapped in `<section>` with an `<h2>` heading

### Requirement: Posture cards use article elements
Each posture card SHALL render as an `<article>` element so screen readers announce card boundaries.

#### Scenario: Screen reader announces card
- **WHEN** a screen reader user navigates to a posture card
- **THEN** the card is announced as an article with the policy title as its heading

### Requirement: Data tables use native table elements
All tabular data in evidence view, audit history, and draft review SHALL use `<table>`, `<thead>`, `<th>`, `<tbody>`, and `<td>` elements.

#### Scenario: Evidence list renders as table
- **WHEN** evidence records load in the evidence view
- **THEN** results render in a `<table>` with `<th>` column headers

#### Scenario: Audit history list renders as table
- **WHEN** audit logs load in audit history
- **THEN** logs render in a `<table>` with `<th>` column headers

### Requirement: App shell uses landmark elements
The app shell SHALL use `<header>` for the top bar, `<aside>` for the sidebar (already present), and `<main>` for the content area.

#### Scenario: Landmark navigation
- **WHEN** a screen reader user lists landmarks
- **THEN** banner (header), navigation (sidebar nav), and main content are listed as distinct regions
