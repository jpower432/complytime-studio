## Why

The assistant's prompt architecture has two reliability gaps exposed by comparison with the OllamaCrosswalker/Compliance Mapper project:

1. **No few-shot examples.** The assistant classifies evidence as Strength/Finding/Gap/Observation using prose rules in the `audit-methodology` skill. Mapper uses concrete few-shot examples with anti-patterns for every classification type, achieving measurably better consistency. Without examples, the assistant's classification accuracy depends entirely on the model's zero-shot capability.

2. **No audit provenance on AuditLog artifacts.** The `audit_logs` table stores `created_by` but not *how* the artifact was produced — which model or prompt version generated it. When a classification is disputed, there is no way to reproduce the conditions that produced it.

## What Changes

- **Add few-shot examples** for the three high-stakes classification decisions: result type (Strength/Finding/Gap/Observation), satisfaction determination (Satisfied/Partially/Not Satisfied/N/A), and coverage status (Covered/Partially/Weakly/Not Covered). Structured as YAML files under `agents/assistant/prompts/few-shot/`, loaded and formatted into the existing `load_prompt()` path.
- **Add provenance columns** to the `audit_logs` table: `model`, `prompt_version`. Populated by the gateway at insert time from assistant metadata. Links the AuditLog artifact to the conditions that produced it.

## Capabilities

### New Capabilities

- `few-shot-classification`: Structured examples for Strength/Finding/Gap/Observation and satisfaction determination — anchors model behavior on concrete cases

### Modified Capabilities

- `audit-log-gateway-enrichment`: `InsertAuditLog` extended with provenance columns. Gateway populates from assistant metadata.

## Impact

- **Agent runtime**: `agents/assistant/main.py` — `load_prompt()` extended to read and format few-shot YAML files into the instruction string
- **Prompt content**: New `agents/assistant/prompts/few-shot/` directory with 3 YAML files
- **Backend**: `internal/clickhouse/client.go` — two new columns in `audit_logs` DDL. `internal/store/store.go` — `AuditLog` struct and `InsertAuditLog` updated.
- **No frontend changes**
- **No API contract changes** — provenance columns have defaults; existing callers unaffected

## Future Work

If token cost or attention dilution becomes measurable (via Vertex AI billing or classification regression), evaluate:
- **Prompt layering**: YAML merge semantics separating workflow from identity from examples
- **Selective skill loading**: Keyword-triggered skill injection to reduce per-turn context size

These are deferred until the few-shot baseline is established and there is data to justify the added runtime complexity.
