## ADDED Requirements

### Requirement: Deterministic intent router node
The graph SHALL include a `router` node that classifies user intent deterministically using keyword matching before falling back to an LLM classification call.

#### Scenario: User message contains posture keywords
- **WHEN** the user's first message contains any of: "posture", "readiness", "status", "how ready", "assessment plan", "evidence quality", "are we compliant"
- **THEN** the router SHALL set `intent` to `"posture_check"` without invoking the LLM
- **THEN** the graph SHALL route to the `posture_check` subgraph

#### Scenario: User message contains audit keywords
- **WHEN** the user's first message contains any of: "run an audit", "produce an auditlog", "audit", "generate audit", "audit results", "audit log"
- **THEN** the router SHALL set `intent` to `"audit_production"` without invoking the LLM
- **THEN** the graph SHALL route to the `audit_production` subgraph

#### Scenario: No keyword match — LLM fallback
- **WHEN** the user's first message does not match any keyword pattern
- **THEN** the router SHALL invoke the LLM with a constrained output schema: `{"intent": "posture_check" | "audit_production" | "ambiguous"}`
- **THEN** the LLM call SHALL use a minimal system prompt (classification only, no tools bound)

#### Scenario: LLM classifies as ambiguous
- **WHEN** the LLM fallback returns `"ambiguous"`
- **THEN** the graph SHALL route to a `clarify` node that emits: "Do you want a posture check (readiness overview) or a full audit (AuditLog production)?"
- **THEN** the graph SHALL wait for user reply and re-invoke the router

### Requirement: Router operates on first substantive message only
The router SHALL classify intent once per conversation. After `intent` is set to a non-empty value, subsequent messages SHALL bypass the router and proceed directly into the active subgraph.

#### Scenario: Follow-up message in active audit
- **WHEN** `intent` is `"audit_production"` and the user sends a follow-up message
- **THEN** the router SHALL be skipped and the message SHALL be delivered to the `audit_production` subgraph directly

### Requirement: Keyword list is centralized
The keyword patterns SHALL be defined in a single Python constant (not inline in the router function) to satisfy the Single Source of Truth principle.

#### Scenario: Adding a new keyword
- **WHEN** a developer needs to add a routing keyword
- **THEN** the change SHALL require modifying exactly one constant definition
- **THEN** no prompt.md changes SHALL be required for routing behavior
