## ADDED Requirements

### Requirement: Delta values display tooltip on hover
Each delta value in the audit history table (`+N`, `-N`, `0`) SHALL display a tooltip on hover explaining the comparison.

#### Scenario: Positive delta tooltip
- **WHEN** the user hovers over a `(+2)` delta on the Strengths column
- **THEN** a tooltip SHALL display "2 more than prior audit"

#### Scenario: Negative delta tooltip
- **WHEN** the user hovers over a `(-1)` delta on the Findings column
- **THEN** a tooltip SHALL display "1 fewer than prior audit"

#### Scenario: Zero delta tooltip
- **WHEN** the user hovers over a `(0)` delta
- **THEN** a tooltip SHALL display "No change from prior audit"

#### Scenario: No prior audit (no delta shown)
- **WHEN** the row is the oldest audit with no prior comparison
- **THEN** no delta SHALL be displayed and no tooltip is needed
