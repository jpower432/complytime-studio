## ADDED Requirements

### Requirement: CertificationHandler subscribes alongside PostureCheckHandler
The NATS evidence event bus SHALL support multiple subscribers on `studio.evidence.>`. The `CertificationHandler` SHALL receive the same `EvidenceEvent` as the `PostureCheckHandler` without interfering with it.

#### Scenario: Both handlers receive event
- **WHEN** an `EvidenceEvent` is published for policy "ampel-branch-protection"
- **THEN** both the `CertificationHandler` and the `PostureCheckHandler` SHALL receive the event and execute independently

#### Scenario: CertificationHandler failure does not affect PostureCheckHandler
- **WHEN** the `CertificationHandler` encounters an error processing an event
- **THEN** the `PostureCheckHandler` SHALL still process the same event and produce its posture delta notification

### Requirement: Debounce applies to certification
The existing 30-second debounce window per `policy_id` SHALL apply to the `CertificationHandler` the same way it applies to the `PostureCheckHandler`. Multiple evidence events within the window SHALL trigger one certification pass, not N.

#### Scenario: Debounced certification
- **WHEN** 3 evidence events arrive for the same policy within 10 seconds
- **THEN** the `CertificationHandler` SHALL run once after the 30-second window closes, certifying all rows from the batch
