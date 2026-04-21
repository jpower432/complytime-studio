# ComplyTime Studio: Use Case

## What It Does

Studio is the aggregation point in the ComplyTime ecosystem. Policies and evidence converge here for audit preparation.

| Function | How |
|:--|:--|
| Import policies | Pull Gemara Policy artifacts from OCI registries into ClickHouse |
| Ingest evidence | Accept assessment results via API, file upload, or OTel collector |
| Map frameworks | Load MappingDocuments that crosswalk internal criteria to external frameworks |
| Prepare audits | Agentic assistant queries evidence, validates cadence, classifies results, produces AuditLog artifacts |
| Publish artifacts | Push validated artifacts back to OCI registries |

## Who It Is For

**Compliance engineer** maintaining GRC artifacts in Git. Uses `complyctl` and local tooling (Cursor, Claude Code) + gemara-mcp to author artifacts. Uses Studio to see how those artifacts perform against real evidence.

**Security team lead** preparing for audits. Needs a view across policies, evidence, and frameworks without writing SQL. Uses the chat assistant to synthesize AuditLogs from ClickHouse data.

## Ecosystem Position

Studio does not own the compliance lifecycle end-to-end. It occupies the **audit preparation** slot in a decoupled pipeline.

```
Author           Distribute       Assess           Analyze
──────           ──────────       ──────           ───────
Cursor +         OCI registries   OPA / Kyverno    ComplyTime
gemara-mcp       (GHCR, Zot)     complyctl/Lula   Studio
complyctl                         OTel collector
```

The [Gemara schema](https://gemara.openssf.org/) is the shared contract across all stages. Policies, evidence, and audit logs are portable — any tool that reads Gemara can participate.

### What Each Tool Owns

| Concern | Tool |
|:--|:--|
| Artifact authoring | Engineer's local tooling + gemara-mcp |
| Validation and transformation | `complyctl` (CLI, CI/CD) |
| Distribution | OCI registries |
| Policy evaluation | OPA, Kyverno, policy engines |
| Evidence collection | OTel collector, `complyctl` ProofWatch |
| Evidence storage | ClickHouse (deployed by Studio or external) |
| Audit preparation | **Studio** — assistant queries evidence, produces AuditLogs |
| Audit review | Studio workbench — editor, validation, publish |

## What It Is Not

- **Not an authoring tool.** Artifact creation happens in local developer tooling. Studio consumes those artifacts.
- **Not a policy engine.** Studio reads evidence that policy engines produce.
- **Not a replacement for complyctl.** `complyctl` is the CLI for validation, transformation, and automation. Studio is the dashboard that shows how artifacts perform.
