## ADDED Requirements

### Requirement: Prompt routes posture-check requests
The assistant prompt SHALL include a routing step that detects posture-check intent and dispatches to the posture workflow instead of the audit production workflow. Posture-check intent SHALL be recognized from queries containing terms like "posture", "readiness", "status", "how ready", "assessment plan status", or "are we compliant."

#### Scenario: User asks for posture
- **WHEN** the user says "What's my posture for policy ACP-01?"
- **THEN** the agent SHALL execute the posture-check workflow, not the audit production workflow

#### Scenario: User asks for audit
- **WHEN** the user says "Run an audit for policy ACP-01"
- **THEN** the agent SHALL execute the existing audit production workflow unchanged

#### Scenario: Ambiguous request
- **WHEN** the user's intent is unclear between posture check and audit production
- **THEN** the agent SHALL ask: "Do you want a posture check (readiness overview) or a full audit (AuditLog production)?"

### Requirement: Posture workflow collects policy and window inputs
The posture-check workflow SHALL require a policy identifier (name or `policy_id`) and an audit window. If either is missing, the agent SHALL ask once and stop.

#### Scenario: Both inputs provided
- **WHEN** the user provides "posture for ACP-01, Q2 2026"
- **THEN** the agent SHALL proceed with the posture check without additional prompts

#### Scenario: Missing audit window
- **WHEN** the user provides a policy but no window
- **THEN** the agent SHALL ask for the audit window before proceeding

### Requirement: Posture workflow returns per-plan readiness table
The posture-check workflow SHALL return a table with one row per assessment plan showing: Plan ID, Frequency, Last Evidence Date, Source Match status, Latest Result, and Classification (Healthy/Failing/Wrong Source/Stale/Blind).

#### Scenario: Policy with 4 assessment plans
- **WHEN** the agent completes a posture check against a policy with 4 plans
- **THEN** the agent SHALL return a table with 4 rows, one per plan, with all columns populated

#### Scenario: Summary line
- **WHEN** the posture check completes
- **THEN** the agent SHALL append a summary line stating how many plans are Healthy vs. total (e.g., "1/4 plans healthy. 1 failing, 1 wrong source, 1 blind.")

### Requirement: Posture workflow groups results by target
The posture-check workflow SHALL discover distinct targets from the evidence table (same logic as audit workflow step 3) and produce a separate readiness table per target.

#### Scenario: Multiple targets
- **WHEN** evidence exists for 3 targets under the given policy
- **THEN** the agent SHALL present 3 readiness tables, one per target, each with plan-level rows

#### Scenario: Single target
- **WHEN** evidence exists for only 1 target
- **THEN** the agent SHALL present 1 readiness table without a target selection prompt

### Requirement: Posture workflow does not produce AuditLog
The posture-check workflow SHALL NOT produce, validate, or publish an AuditLog artifact. It is a read-only diagnostic.

#### Scenario: Posture check completes
- **WHEN** the posture-check workflow finishes
- **THEN** the agent SHALL NOT call `validate_gemara_artifact` or `publish_audit_log`
- **THEN** the agent SHALL NOT emit a `TaskArtifactUpdateEvent`
