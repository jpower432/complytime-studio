### Requirement: URL hash encodes active filters
The URL hash SHALL encode the active view and non-null filter signals in the format `#/<view>?param=value&param=value`.

#### Scenario: Navigate with policy selected
- **WHEN** the user navigates to requirements with policy "nist-800-53" selected
- **THEN** the URL hash becomes `#/requirements?policy=nist-800-53`

#### Scenario: Navigate with policy and date range
- **WHEN** the user searches audit history with policy "nist-800-53", start "2026-01-01", end "2026-03-31"
- **THEN** the URL hash becomes `#/audit-history?policy=nist-800-53&start=2026-01-01&end=2026-03-31`

### Requirement: Page load restores filters from hash
On page load or hash change, the app SHALL parse filter parameters from the URL hash and write them to the corresponding shared signals.

#### Scenario: Open deep link
- **WHEN** a user opens `#/requirements?policy=nist-800-53&start=2026-01-01`
- **THEN** `selectedPolicyId` is set to "nist-800-53"
- **THEN** `selectedTimeRange.start` is set to "2026-01-01"
- **THEN** the requirements view pre-fills the policy dropdown and start date

#### Scenario: Hash with view only
- **WHEN** a user opens `#/posture` with no parameters
- **THEN** shared signals remain null and views show default empty filter state
