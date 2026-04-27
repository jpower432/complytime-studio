## ADDED Requirements

### Requirement: Evidence table certification indicator
The evidence table SHALL display a certification status icon per row. Certified rows SHALL show a checkmark (✓). Uncertified rows SHALL show a warning indicator (⚠).

#### Scenario: Certified row
- **WHEN** an evidence row has `certified = true`
- **THEN** the row SHALL display a ✓ icon in the certification column

#### Scenario: Uncertified row
- **WHEN** an evidence row has `certified = false`
- **THEN** the row SHALL display a ⚠ icon in the certification column

### Requirement: Per-certifier detail expand
Clicking the certification indicator on an uncertified row SHALL expand a detail panel showing the per-certifier breakdown from the `certifications` table.

#### Scenario: Expand uncertified detail
- **WHEN** a user clicks the ⚠ icon on an uncertified evidence row
- **THEN** a detail panel SHALL display each certifier's name, result (pass/fail/skip/error), and reason

#### Scenario: Expand certified detail
- **WHEN** a user clicks the ✓ icon on a certified evidence row
- **THEN** a detail panel SHALL display each certifier's name and pass result

### Requirement: Certification summary bar
The evidence view SHALL display a certification bar above the table showing the proportion of certified vs uncertified evidence. The bar SHALL use two segments.

#### Scenario: Bar renders counts
- **WHEN** the evidence view loads with 80 certified and 20 uncertified rows
- **THEN** the certification bar SHALL show segments proportional to 80/20 with count labels

#### Scenario: Bar click filters
- **WHEN** a user clicks the "uncertified" segment of the certification bar
- **THEN** a filter chip for `Certification: Uncertified` SHALL be added and the table SHALL filter to uncertified rows only

### Requirement: Certification as filterable field
The "+ Filter" menu SHALL include `Certification` as a filterable field with options: `Certified`, `Uncertified`.

#### Scenario: Filter by uncertified
- **WHEN** a user adds a filter for `Certification: Uncertified`
- **THEN** the evidence table SHALL show only rows where `certified = false`

#### Scenario: Filter by certified
- **WHEN** a user adds a filter for `Certification: Certified`
- **THEN** the evidence table SHALL show only rows where `certified = true`

#### Scenario: Filter chip dismissal
- **WHEN** a user dismisses the `Certification` filter chip
- **THEN** the table SHALL return to showing all evidence regardless of certification status
