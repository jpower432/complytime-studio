---
name: gemara-mcp
description: Gemara layer model and MCP server tools/resources for artifact validation and schema access
---

# Gemara MCP

## Layer Model

Gemara organizes security governance into seven layers. Each layer produces typed artifacts consumed by downstream layers.

| Layer | Name | Artifacts | Role |
|:--|:--|:--|:--|
| L1 | Guidance | GuidanceCatalog | Industry knowledge, vectors, guidelines |
| L2 | Controls | CapabilityCatalog, ThreatCatalog, ControlCatalog | Threat modeling, control objectives, assessment requirements |
| L3 | Policy | RiskCatalog, Policy | Risk appetite, organizational rules, adherence plans |
| L4 | Activity | ActivityLog | Sensitive activity tracking |
| L5 | Evaluation | EvaluationLog | Assessment findings — did the resource comply? |
| L6 | Enforcement | EnforcementLog | Preventive/remediative actions taken |
| L7 | Audit | AuditLog | Point-in-time compliance review with coverage classification |

**The Studio assistant produces L7 (AuditLog) artifacts.** It consumes L3 (Policy), L5 (EvaluationLog), and L6 (EnforcementLog) as inputs.

## MCP Tools

### validate_gemara_artifact

Validates YAML content against a Gemara CUE schema definition.

**Parameters:**
- `artifact_content` (required): YAML string of the artifact
- `definition` (required): CUE definition to validate against — e.g., `#AuditLog`, `#Policy`, `#EvaluationLog`, `#ControlCatalog`, `#ThreatCatalog`, `#GuidanceCatalog`
- `version` (optional): Schema version (defaults to latest)

**Validation workflow:**
1. Author the artifact YAML
2. Call `validate_gemara_artifact` with the content and definition
3. If validation fails, read the error, fix the YAML, re-validate
4. Maximum 3 attempts — if still failing, report the error to the user

### migrate_gemara_artifact

Migrates an older artifact to the current schema version using CUE transformations.

**Parameters:**
- `artifact_content` (required): YAML string of the artifact to migrate
- `artifact_type` (optional): `ThreatCatalog` or `ControlCatalog` — infer from structure if metadata.type is missing
- `gemara_version` (optional): Source version when metadata.gemara-version is missing

## MCP Resources

### gemara://lexicon

Term definitions for the Gemara security model. Use this to understand domain terminology before authoring artifacts. Key terms: Assessment, Control, Evaluation, Enforcement, Policy, Risk, Threat, Audit, Opinion.

### gemara://schema/definitions

Full CUE schema definitions for all artifact types. Use this to understand field names, types, required fields, and enum values before authoring YAML. Available as `gemara://schema/definitions` (latest) or `gemara://schema/definitions?version=X.Y.Z` for a specific version.

**Always read schema definitions before authoring a new artifact type for the first time.** The schema is the source of truth for field names and structure — do not rely on memory.

## Output Format

Return validated artifact YAML wrapped in a ```yaml fenced code block. The platform detects and imports artifacts from fenced blocks.
