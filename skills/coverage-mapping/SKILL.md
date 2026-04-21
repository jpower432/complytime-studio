---
name: coverage-mapping
description: Cross-framework coverage analysis using MappingDocuments with strength and confidence scoring
---

# Coverage Mapping

Cross-framework coverage analysis translates internal AuditResults into external framework coverage using MappingDocuments. This analysis is **optional** — only performed when MappingDocuments are provided. If none are provided, state that cross-framework translation is skipped and proceed with internal-only analysis.

## MappingDocument Structure

A MappingDocument links internal Policy/Catalog entries to entries in an external compliance framework (e.g., SOC 2, ISO 27001, FedRAMP).

Each mapping entry contains:
- `source`: Internal criteria reference (control + requirement from the Policy)
- `targets[]`: External framework entries, each with:
  - `reference`: The external framework entry ID
  - `strength`: Integer 1-10 indicating how strongly the internal control addresses the external requirement
  - `confidence-level`: How confident the mapping author is in the mapping (High, Medium, Low)

## Join Logic

For each MappingDocument:

1. Match `AuditResult.criteria-reference` entries to `Mapping.source` entries
2. Follow `Mapping.targets[]` to identify which external framework entries are addressed
3. Read `strength` and `confidence-level` from each target

An AuditResult may map to multiple external entries. An external entry may be mapped from multiple AuditResults.

## Coverage Status Derivation

For each external framework entry, combine the AuditResult type with the mapping quality:

| AuditResult type | Mapping strength | Confidence | Framework coverage |
|:--|:--|:--|:--|
| Strength | 8-10 | High | Covered |
| Strength | 5-7 | Medium or High | Partially Covered |
| Strength | 1-4 | any | Weakly Covered |
| Finding | any | any | Not Covered (finding) |
| Gap | any | any | Not Covered (no evidence) |
| Observation | any | any | Needs Review |
| (no mapping) | — | — | Unmapped |

## Multi-Mapping Resolution

When multiple internal controls map to the same external entry, use the **strongest coverage**. Note weaker mappings in recommendations for completeness.

Example: If Control A maps to SOC2 CC6.1 with strength 9 (Covered) and Control B also maps with strength 3 (Weakly Covered), the framework coverage for CC6.1 is **Covered** with a note that Control B provides additional weak coverage.

## Coverage Matrix Format

Present a summary per framework:

```
### Framework: [name]
Total entries: N | Covered: N | Partially: N | Weakly: N | Not Covered: N | Unmapped: N

| External Entry | Coverage | Internal Control | Strength | Confidence | Notes |
|:--|:--|:--|:--|:--|:--|
| CC6.1 | Covered | AC-2 | 9 | High | — |
| CC6.2 | Not Covered | AC-3 | 7 | Medium | Finding: failed eval |
```

Follow with an **attention items** table sorted by risk:
1. Not Covered (finding) — highest priority
2. Not Covered (no evidence)
3. Weakly Covered
4. Partially Covered

Include the internal control, mapping strength, and a brief gap description for each attention item.

## Embedding in AuditResults

After coverage analysis, embed framework-specific context into `AuditResult.recommendations[]` so the AuditLog artifact itself carries the cross-framework detail. The coverage matrix in the response is a summary view — the artifact is the authoritative record.

## Validation

Before analysis, validate MappingDocuments:
- Warn if `strength` or `confidence-level` fields are missing on mapping targets
- Warn if source references don't match any AuditResult criteria-reference
- Missing fields reduce coverage confidence but do not block analysis
