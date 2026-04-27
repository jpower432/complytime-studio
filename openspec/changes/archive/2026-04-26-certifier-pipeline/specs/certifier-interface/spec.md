## ADDED Requirements

### Requirement: Certifier contract
The system SHALL define a `Certifier` interface with `Name() string`, `Version() string`, and `Certify(ctx, row) CertResult` methods. `CertResult` SHALL contain `Certifier`, `Version`, `Verdict`, and `Reason` fields. `Verdict` SHALL be one of `pass`, `fail`, `skip`, or `error`.

#### Scenario: Certifier returns pass
- **WHEN** a certifier determines the evidence row satisfies its check
- **THEN** it SHALL return a `CertResult` with `Verdict = pass` and a human-readable `Reason`

#### Scenario: Certifier returns fail
- **WHEN** a certifier determines the evidence row does not satisfy its check
- **THEN** it SHALL return a `CertResult` with `Verdict = fail` and a `Reason` describing the failure

#### Scenario: Certifier returns skip
- **WHEN** a certifier determines its check does not apply to the evidence row
- **THEN** it SHALL return a `CertResult` with `Verdict = skip` and a `Reason` explaining why the check was inapplicable

#### Scenario: Certifier returns error
- **WHEN** a certifier encounters an external failure (timeout, unreachable service)
- **THEN** it SHALL return a `CertResult` with `Verdict = error` and a `Reason` describing the failure

### Requirement: Pipeline runner executes all certifiers
The system SHALL provide a `Pipeline` that accepts an ordered list of `Certifier` implementations and runs each one sequentially against a given `EvidenceRow`. The pipeline SHALL NOT short-circuit — all certifiers run regardless of prior verdicts.

#### Scenario: All certifiers run
- **WHEN** the pipeline is invoked with 4 registered certifiers and the first returns `fail`
- **THEN** all 4 certifiers SHALL execute and return their individual `CertResult`

#### Scenario: Pipeline returns all results
- **WHEN** the pipeline completes
- **THEN** it SHALL return a slice of `CertResult` with one entry per registered certifier

### Requirement: Certifier registration
The system SHALL allow certifiers to be registered with the pipeline at initialization. Adding a new certifier SHALL require only implementing the `Certifier` interface and adding it to the registration list.

#### Scenario: New certifier added
- **WHEN** a developer implements the `Certifier` interface and registers it with the pipeline
- **THEN** the pipeline SHALL include the new certifier in subsequent runs without changes to the pipeline code
