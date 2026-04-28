## ADDED Requirements

### Requirement: Assessment plans support an optional layout reference
The Gemara `#AssessmentPlan` schema SHALL support an optional `layout` field that references an in-toto layout artifact stored in OCI by digest.

#### Scenario: Assessment plan with layout
- **WHEN** a Policy YAML contains an assessment plan with `layout: { reference-id: "oci-layouts", digest: "sha256:abc123" }`
- **THEN** the system SHALL treat this as a reference to an in-toto layout artifact for provenance verification

#### Scenario: Assessment plan without layout
- **WHEN** a Policy YAML contains an assessment plan without a `layout` field
- **THEN** the system SHALL treat this as valid and skip layout-based verification for that plan

### Requirement: Layout defines expected pipeline steps
An in-toto layout artifact SHALL define the expected steps for evidence collection, including step name, expected command pattern, authorized key identities, and material/product chaining rules.

#### Scenario: Layout with two steps
- **WHEN** a layout defines steps `[fetch-policy, evaluate]` with `fetch-policy` producing materials consumed by `evaluate`
- **THEN** verification SHALL check that both steps have signed attestations AND that `evaluate`'s materials match `fetch-policy`'s products by hash

#### Scenario: Layout with authorized keys
- **WHEN** a layout step specifies `authorized-keys: ["key-id-1", "key-id-2"]` with threshold 1
- **THEN** verification SHALL pass if at least 1 of the listed keys signed the step's attestation

### Requirement: Layout is stored as an OCI artifact alongside the policy
The in-toto layout SHALL be stored in the OCI registry, either as an additional layer in the Policy's OCI manifest or as a separately tagged artifact in the same OCI repository.

#### Scenario: Layout pushed alongside policy
- **WHEN** a policy is published to `oci://registry/policies/access-review:v1`
- **THEN** the layout SHALL be retrievable from the same repository by its digest

#### Scenario: Layout versioned independently
- **WHEN** the layout is updated (new authorized key added) without changing the Policy content
- **THEN** a new layout digest SHALL be produced without requiring a new Policy version
