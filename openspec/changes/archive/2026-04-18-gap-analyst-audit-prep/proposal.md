## Why

The Gap Analyst operates on a single policy + single target today. Real-world audits are combined audits: one internal criteria set mapped to multiple external compliance frameworks (SOC 2, ISO 27001, FedRAMP), with evidence spread across multiple targets. A compliance analyst preparing for audit needs to know which criteria are covered, how that coverage translates across frameworks, and where gaps exist — factually, using strength and confidence scores — so they can prioritize remediation before the auditor arrives.

The current agent treats MappingDocuments as optional enrichment. This change makes them central. The agent becomes an audit preparation assistant: derive inventory from evidence, assess coverage per criteria per target, translate coverage through MappingDocuments to external frameworks, and surface partial or missing coverage using strength and confidence so a human can address gaps.

## What Changes

- **Multi-target inventory**: Derive audit targets from distinct `target_id` values in L5/L6 evidence rather than requiring a single target upfront
- **User-provided audit timeline**: Replace policy-frequency-derived time window with explicit user-provided audit period
- **MappingDocument-driven cross-framework analysis**: Elevate MappingDocuments from optional enrichment to required input for combined audits; use `MappingTarget.strength` and `MappingTarget.confidence-level` to classify external framework coverage
- **Multi-document AuditLog output**: One AuditLog per target in a multi-YAML-document file, plus a cross-framework coverage summary
- **Prompt rewrite**: Gap Analyst prompt updated to reflect combined audit workflow with ~4-6 exchange guided conversation

## Capabilities

### Modified Capabilities

- `gap-analyst-agent`: Evolves from single-target evidence synthesizer to multi-target combined audit preparation assistant with cross-framework coverage analysis

## Impact

- **Agent prompt**: `agents/gap-analyst/prompt.md` rewritten for combined audit workflow
- **Agent spec**: `agents/gap-analyst/agent.yaml` description updated
- **No new infrastructure**: Uses existing ClickHouse evidence store and MCP tools
- **No schema changes**: Existing `evaluation_logs` and `enforcement_actions` tables support multi-target queries
- **Artifact output**: AuditLog per target (multi-YAML-doc), unchanged schema per document
