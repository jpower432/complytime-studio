## ADDED Requirements

### Requirement: Attestation bundles are stored as OCI artifacts by the client
The client (complyctl) SHALL push in-toto attestation bundles to the OCI registry after producing them. Studio does not participate in attestation storage — it receives the OCI digest as a semconv attribute on the evidence row.

#### Scenario: Client pushes attestation bundle to OCI
- **WHEN** complyctl completes a scan and produces an attestation bundle
- **THEN** complyctl SHALL push the bundle to the OCI registry and include the digest as `compliance.attestation.ref` in the evidence OTel attributes

#### Scenario: Pull attestation bundle by digest
- **WHEN** the agent needs to verify an attestation for evidence with `attestation_ref = 'sha256:abc123'`
- **THEN** the agent SHALL use oras-mcp to pull the bundle directly from the OCI registry

### Requirement: Evidence rows reference attestation bundles by OCI digest
The `evidence` table SHALL include an `attestation_ref` column of type `Nullable(String)` containing the OCI digest of the attestation bundle associated with that evidence row.

#### Scenario: Evidence ingested with attestation
- **WHEN** evidence is ingested with `compliance.attestation.ref` set in OTel attributes
- **THEN** the `attestation_ref` column SHALL contain the OCI digest (e.g., `sha256:abc123`)

#### Scenario: Evidence ingested without attestation
- **WHEN** evidence is ingested via REST API upload or legacy OTel path without attestation metadata
- **THEN** the `attestation_ref` column SHALL be NULL

### Requirement: Studio has no attestation upload endpoint
Studio SHALL NOT expose REST endpoints for uploading or retrieving attestation bundles. The client pushes to OCI directly. The agent reads from OCI via oras-mcp. No Gateway intermediary needed.
