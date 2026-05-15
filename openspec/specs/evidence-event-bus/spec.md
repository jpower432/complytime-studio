## Requirements

### Requirement: Ingest publishes evidence events to NATS
The system SHALL publish an event to NATS subject `studio.evidence.{policy_id}` after successfully inserting evidence records. The event payload SHALL include `policy_id`, `record_count`, and `timestamp`.

#### Scenario: Evidence insert triggers NATS publish
- **WHEN** `cmd/ingest` inserts 5 evidence records for policy "ampel-branch-protection"
- **THEN** a message is published to `studio.evidence.ampel-branch-protection` with `{"policy_id":"ampel-branch-protection","record_count":5,"timestamp":"..."}`

#### Scenario: NATS unavailable does not block ingest
- **WHEN** NATS is unreachable and evidence records are inserted
- **THEN** the ingest succeeds, the NATS publish fails silently with a warning log, and no data is lost

### Requirement: Gateway subscribes to evidence events for certification
The system SHALL subscribe to `studio.evidence.*` on startup. Received events SHALL be debounced per `policy_id` over a configurable window before triggering the certification pipeline.

#### Scenario: Debounced event triggers certification
- **WHEN** 3 evidence events arrive for policy "ampel-branch-protection" within 10 seconds
- **THEN** the gateway waits until the debounce window closes, then runs exactly one certification pass for that policy

#### Scenario: Concurrent policies handled independently
- **WHEN** events arrive for policy A and policy B within the same debounce window
- **THEN** the gateway triggers separate certification passes for A and B

### Requirement: NATS deployed in-chart
The system SHALL include a single-node NATS server in the Helm chart when `nats.enabled` is true. The NATS server SHALL use pure pub/sub (no JetStream).

#### Scenario: NATS disabled by default
- **WHEN** `values.yaml` has `nats.enabled: false`
- **THEN** no NATS Deployment or Service is rendered

#### Scenario: NATS enabled renders resources
- **WHEN** `values.yaml` has `nats.enabled: true`
- **THEN** Helm renders a NATS Deployment, Service, and NetworkPolicy restricting ingress to the gateway pod

### Requirement: Gateway limits concurrent certification checks
The system SHALL track in-flight certification checks per policy and skip triggering a new check if one is already running for the same policy.

#### Scenario: Duplicate check suppressed
- **WHEN** a certification check is in-flight for policy A and another evidence event arrives
- **THEN** the gateway skips triggering a second check and logs a debug message
