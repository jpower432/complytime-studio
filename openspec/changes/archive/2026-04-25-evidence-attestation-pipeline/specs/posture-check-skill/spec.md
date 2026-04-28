## MODIFIED Requirements

### Requirement: Skill classifies each plan into seven states
The skill SHALL define seven readiness states: Healthy, Failing, Wrong Source, Wrong Method, Unfit Evidence, Stale, Blind. Classification priority (worst wins): Blind > Wrong Source > Wrong Method > Unfit Evidence > Stale > Failing > Healthy.

#### Scenario: Wrong Source
- **WHEN** evidence exists but `engine_name` does not match the plan's `executor.id`, OR the attestation chain shows an unauthorized signer
- **THEN** the agent SHALL classify the plan as "Wrong Source"

#### Scenario: Wrong Method
- **WHEN** evidence exists and the actor matches, but the assessment method type or mode does not match the plan's `evaluation-methods[]` (e.g., plan says `mode: Automated` but evidence was manually uploaded)
- **THEN** the agent SHALL classify the plan as "Wrong Method"

#### Scenario: Unfit Evidence
- **WHEN** evidence exists, actor matches, method matches, but the evidence content does not satisfy the plan's `evidence-requirements` field (semantic mismatch — e.g., plan requires firewall rule export, evidence is a pod security report)
- **THEN** the agent SHALL classify the plan as "Unfit Evidence" with a reason explaining the mismatch

#### Scenario: Healthy
- **WHEN** evidence exists, actor matches, method matches, evidence fits requirements, cadence is current, and `eval_result` is Passed
- **THEN** the agent SHALL classify the plan as "Healthy"

#### Scenario: Priority ordering
- **WHEN** evidence has both a method mismatch AND a stale cadence
- **THEN** the agent SHALL classify as "Wrong Method" (higher priority than Stale)

### Requirement: Skill validates executor provenance
The skill SHALL instruct the agent to compare each evidence row's `engine_name` against the assessment plan's `evaluation-methods[].executor.id`. When `attestation_ref` is present, the agent SHALL perform cryptographic chain verification instead. A mismatch in either mode SHALL classify the plan as "Wrong Source."

#### Scenario: Executor matches (string comparison, no attestation)
- **WHEN** the assessment plan specifies `executor.id: nessus` AND evidence rows have `engine_name = 'nessus'` AND `attestation_ref` is NULL
- **THEN** the agent SHALL pass the provenance check using string comparison

#### Scenario: Executor mismatch (string comparison, no attestation)
- **WHEN** the assessment plan specifies `executor.id: nessus` AND evidence rows have `engine_name = 'qualys'` AND `attestation_ref` is NULL
- **THEN** the agent SHALL classify the plan as "Wrong Source" with message "Expected: nessus, Got: qualys"

#### Scenario: Attestation present and verified
- **WHEN** evidence has `attestation_ref` present AND the attestation chain verifies against the layout
- **THEN** the agent SHALL classify provenance as verified and note "Provenance: cryptographically verified" in the readiness table

#### Scenario: Attestation present but chain broken
- **WHEN** evidence has `attestation_ref` present AND the attestation chain fails verification
- **THEN** the agent SHALL classify the plan as "Wrong Source" with the specific chain failure reason

#### Scenario: Graceful fallback
- **WHEN** evidence has `attestation_ref` present but the OCI registry is unreachable
- **THEN** the agent SHALL fall back to `engine_name` string comparison and note "Attestation verification unavailable — fell back to engine_name check"

### Requirement: Skill checks method type and mode
The skill SHALL instruct the agent to compare each evidence row's collection method against the assessment plan's `evaluation-methods[].type` and `evaluation-methods[].mode`. A mismatch SHALL classify the plan as "Wrong Method."

#### Scenario: Automated plan, manual evidence
- **WHEN** the plan specifies `mode: Automated` AND evidence was submitted via manual REST upload (no OTel collector path)
- **THEN** the agent SHALL classify as "Wrong Method" with reason "Plan requires Automated, evidence was manually submitted"

#### Scenario: Behavioral plan, intent evidence
- **WHEN** the plan specifies `type: Behavioral` AND evidence metadata indicates an intent-based check
- **THEN** the agent SHALL classify as "Wrong Method" with reason "Plan requires Behavioral evaluation, evidence is Intent-based"

### Requirement: Skill evaluates evidence against plan requirements
The skill SHALL instruct the agent to compare evidence content against the assessment plan's `evidence-requirements` field. This is a semantic comparison — the agent uses reasoning to determine whether the evidence satisfies the described requirement.

#### Scenario: Evidence matches requirements
- **WHEN** the plan's `evidence-requirements` states "OPA evaluation output showing current access matrix" AND the evidence is an OPA access review result
- **THEN** the agent SHALL pass the evidence fitness check

#### Scenario: Evidence does not match requirements
- **WHEN** the plan's `evidence-requirements` states "Firewall rule export showing ingress/egress policies" AND the evidence is a Kyverno pod security report
- **THEN** the agent SHALL classify as "Unfit Evidence" with reason explaining the mismatch

### Requirement: Skill emits EvidenceAssessment artifact
The skill SHALL instruct the agent to emit a structured `EvidenceAssessment` artifact after completing the posture check. The artifact SHALL contain one entry per assessed evidence row with the 7-state classification, reason, and provenance metadata.

#### Scenario: Assessment artifact emitted
- **WHEN** the agent completes a posture check covering 4 assessment plans across 2 targets
- **THEN** the agent SHALL emit one EvidenceAssessment artifact containing all classification entries

#### Scenario: Assessment artifact includes all fields
- **WHEN** the agent emits an EvidenceAssessment entry for evidence `ev-123`
- **THEN** the entry SHALL include evidence_id, policy_id, plan_id, classification, reason, and assessed_at
