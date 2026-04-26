## ADDED Requirements

### Requirement: Provenance certifier checks source identity
The provenance certifier SHALL verify that the evidence row has at least one of `source_registry` or `attestation_ref` populated.

#### Scenario: Both source_registry and attestation_ref present
- **WHEN** an evidence row has both `source_registry` and `attestation_ref` non-null
- **THEN** the provenance certifier SHALL return `pass`

#### Scenario: Only source_registry present
- **WHEN** an evidence row has `source_registry` non-null and `attestation_ref` null
- **THEN** the provenance certifier SHALL return `pass`

#### Scenario: Only attestation_ref present
- **WHEN** an evidence row has `attestation_ref` non-null and `source_registry` null
- **THEN** the provenance certifier SHALL return `pass`

#### Scenario: No provenance fields
- **WHEN** an evidence row has both `source_registry` and `attestation_ref` null
- **THEN** the provenance certifier SHALL return `fail` with reason "no source_registry or attestation_ref"

### Requirement: Provenance certifier checks known registry
When `source_registry` is present, the provenance certifier SHALL check it against Studio's list of known/trusted registries.

#### Scenario: Known registry
- **WHEN** `source_registry` matches an entry in Studio's known registries
- **THEN** the provenance certifier SHALL include "known registry" in the pass reason

#### Scenario: Unknown registry
- **WHEN** `source_registry` does not match any known registry
- **THEN** the provenance certifier SHALL return `fail` with reason identifying the unrecognized registry
