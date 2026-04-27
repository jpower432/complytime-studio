## ADDED Requirements

### Requirement: Attestation certifier skips when no ref
The attestation certifier SHALL skip evidence rows where `attestation_ref` is null.

#### Scenario: No attestation_ref
- **WHEN** an evidence row has `attestation_ref` null
- **THEN** the attestation certifier SHALL return `skip` with reason "no attestation_ref"

### Requirement: Attestation certifier resolves bundle from Studio registry (Phase 1)
The attestation certifier SHALL attempt to resolve the `attestation_ref` digest from Studio's own OCI registry via the `BundleResolver` interface. When no resolver is configured, the certifier SHALL skip. It SHALL NOT use the `source_registry` field on the evidence row.

#### Scenario: No resolver configured
- **WHEN** the certifier has no `BundleResolver` configured
- **THEN** the attestation certifier SHALL return `skip` with reason "no bundle resolver configured"

#### Scenario: Bundle found in Studio registry
- **WHEN** `attestation_ref` resolves to a non-empty bundle in Studio's OCI registry
- **THEN** the attestation certifier SHALL return `pass` with reason "bundle resolved from Studio registry"

#### Scenario: Bundle not in Studio registry
- **WHEN** `attestation_ref` does not resolve in Studio's OCI registry
- **THEN** the attestation certifier SHALL return `fail` with reason "bundle not found in Studio registry"

#### Scenario: Empty bundle
- **WHEN** `attestation_ref` resolves but the bundle content is empty
- **THEN** the attestation certifier SHALL return `fail` with reason "empty bundle"

### Requirement: In-toto verification deferred to Phase 2
Structural parsing of in-toto link files, Ed25519 signature verification, and material/product chain integrity checks are deferred to a future phase. Phase 1 treats successful bundle resolution from Studio's registry as the primary trust signal.

### Requirement: No layout comparison at ingest time
The attestation certifier SHALL NOT perform layout comparison (authorized steps, authorized signers per policy). Layout validation requires policy context and remains the agent's lazy verification job.
