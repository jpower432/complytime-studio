## ADDED Requirements

### Requirement: Extended State schema with working memory
The LangGraph State TypedDict SHALL include structured fields beyond `messages` to persist working memory across the message window boundary.

#### Scenario: State fields available after graph resume
- **WHEN** the graph resumes from an interrupt (e.g., human approval gate)
- **THEN** all structured state fields (`intent`, `draft_yaml`, `evidence_refs`, `validation_result`, `validation_attempts`, `target_inventory`) SHALL be preserved with their pre-interrupt values

#### Scenario: Evidence summary survives message truncation
- **WHEN** the message list exceeds the conversation window and older messages are truncated by the checkpointer
- **THEN** `evidence_refs` and `target_inventory` SHALL remain accessible because they are top-level State fields, not embedded in messages

### Requirement: State field definitions
The State SHALL define the following typed fields:

| Field | Type | Purpose |
|:--|:--|:--|
| `messages` | `Annotated[Sequence[BaseMessage], add_messages]` | Chat history (existing) |
| `intent` | `str` | Classified intent: `"posture_check"`, `"audit_production"`, or `""` |
| `draft_yaml` | `str` | Current draft artifact YAML content |
| `evidence_refs` | `list[str]` | Evidence IDs referenced in the current draft |
| `validation_result` | `dict` | Latest validation outcome: `{"valid": bool, "errors": list}` |
| `validation_attempts` | `int` | Counter for retry budget enforcement |
| `target_inventory` | `list[dict]` | Discovered targets: `[{"target_id": str, "target_name": str}]` |

#### Scenario: Agent node populates draft_yaml
- **WHEN** the LLM generates a complete AuditLog YAML in its response
- **THEN** the graph SHALL extract the YAML content and set `draft_yaml` in state before routing to `validate_draft`

#### Scenario: Agent node populates evidence_refs
- **WHEN** the LLM drafts an AuditLog referencing evidence
- **THEN** the graph SHALL parse `evidence_refs` from the draft's `results[].evidence[].location.reference-id` fields and set them in state

### Requirement: State initialization
All extended State fields SHALL initialize to their zero values (`""` for strings, `[]` for lists, `{}` for dicts, `0` for ints) at graph start. The graph SHALL NOT require these fields in the initial input.

#### Scenario: New conversation starts
- **WHEN** a new A2A task is created with only a user message
- **THEN** the graph SHALL start with `intent=""`, `draft_yaml=""`, `evidence_refs=[]`, `validation_result={}`, `validation_attempts=0`, `target_inventory=[]`
