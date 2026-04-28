## ADDED Requirements

### Requirement: Agent verifies attestation chain on demand
The assistant agent SHALL verify an evidence row's attestation chain when an auditor requests provenance verification. Verification SHALL be deterministic tool output, not LLM reasoning.

#### Scenario: Verified chain
- **WHEN** the auditor asks to verify evidence `ev-123` AND `attestation_ref` is present AND the layout expects steps `[fetch-policy, evaluate]` AND both signed links exist with authorized signers AND material/product hashes chain correctly
- **THEN** the agent SHALL return "CHAIN VERIFIED" with a summary of each step (signer, timestamp, material/product hashes)

#### Scenario: Broken chain — unauthorized signer
- **WHEN** the attestation bundle contains a link signed by `key-id-3` but the layout authorizes only `key-id-1` and `key-id-2`
- **THEN** the agent SHALL return "BROKEN CHAIN" with the specific step and key mismatch

#### Scenario: Broken chain — material/product hash mismatch
- **WHEN** step 2's material hash does not match step 1's product hash
- **THEN** the agent SHALL return "BROKEN CHAIN" identifying the gap between the two steps

#### Scenario: Missing attestation
- **WHEN** the auditor asks to verify evidence `ev-456` AND `attestation_ref` is NULL
- **THEN** the agent SHALL report "No attestation available for this evidence. Provenance cannot be cryptographically verified. Source identity from engine_name: [value]."

### Requirement: Agent retrieves attestation bundle from OCI
The agent SHALL use oras-mcp to pull attestation bundles from the OCI registry using the `attestation_ref` digest stored in the evidence row.

#### Scenario: Successful retrieval
- **WHEN** the agent requests attestation bundle at digest `sha256:abc123`
- **THEN** oras-mcp SHALL return the bundle contents (signed link JSON files)

#### Scenario: Registry unavailable
- **WHEN** the OCI registry is unreachable during a verification request
- **THEN** the agent SHALL report "Cannot verify attestation — registry unavailable" without failing the conversation

### Requirement: Agent retrieves layout from Policy OCI reference
The agent SHALL use oras-mcp to pull the in-toto layout from the Policy's OCI manifest using the `layout` reference in the assessment plan.

#### Scenario: Layout present
- **WHEN** the Policy assessment plan has a `layout` field with digest `sha256:def456`
- **THEN** the agent SHALL pull the layout and use it for chain verification

#### Scenario: Layout absent
- **WHEN** the Policy assessment plan has no `layout` field
- **THEN** the agent SHALL report "No layout defined for this assessment plan. Attestation chain cannot be verified against expected steps."

### Requirement: Verification workflow is routed by agent prompt
The assistant prompt SHALL recognize verification intent and route to the attestation verification workflow.

#### Scenario: Verification keywords
- **WHEN** the auditor uses keywords like "verify", "provenance", "attestation", "chain", "sample evidence"
- **THEN** the agent SHALL execute the attestation verification workflow

#### Scenario: Disambiguation
- **WHEN** the auditor says "check evidence ev-123" without clear intent
- **THEN** the agent SHALL ask whether they want a posture check (readiness) or provenance verification (attestation chain)
