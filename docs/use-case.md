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

**Audit liaison** — prepares for audits, presents evidence to auditors, coordinates with control owners. Needs a push-based model where the system surfaces what needs attention. Uses the Inbox, posture drill-down, and chat assistant daily.

**Compliance engineer** — maintains GRC artifacts in Git. Uses `complyctl` and local tooling (Cursor, Claude Code) + gemara-mcp to author Policies, MappingDocuments, and control catalogs. Uses Studio to see how those artifacts perform against real evidence.

**Control owner** — responsible for a specific inventory item (repo, cluster, service). Receives evidence from scans and needs to know when posture changes. Appears as the RACI "accountable" contact on posture cards.

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

## End-to-End Audit Workflow

The workflow below traces a single audit cycle from evidence arrival to auditor deliverable.

```
Evidence arrives → Posture check → Notification → Review → Audit → Export
```

### 1. Setup (one-time)

A compliance engineer authors two artifacts outside Studio and imports them:

- **Policy** — defines your controls, assessment plans, and cadence requirements. Imported via `POST /api/policies/import` or OCI registry pull.
- **MappingDocument** — crosswalks your controls to an external framework (e.g., BP-1 supports SOC 2 CC8.1 with strength 9/10). Imported via `POST /api/mappings/import`.

The MappingDocument is the bridge that lets Studio speak the auditor's language.

### 2. Evidence Ingestion (continuous)

Evidence flows in through two paths:

| Path | Source | Trigger |
|:--|:--|:--|
| REST API | `POST /api/evidence` — JSON, CSV, or multipart with file attachment | CI pipeline, manual upload, `make seed` |
| CLI job | `cmd/ingest` — reads Gemara EvaluationLog/EnforcementLog YAML | CronJob, `complyctl` pipeline |

Both paths insert into ClickHouse and publish a NATS event per policy.

### 3. Posture Check (automatic)

NATS event → Debouncer (30s window, coalesces per policy) → PostureCheckHandler:

1. Query current pass rate for the policy
2. Compare to last-known rate
3. If delta detected → insert a `posture_change` notification
4. Notification appears in the audit liaison's Inbox with the delta (e.g., "95% → 82%")

### 4. Triage (audit liaison)

The liaison opens Studio and sees:

- **Posture** — cards per policy showing target count, control count, evidence freshness, RACI owner, risk severity
- **Inbox badge** — red dot when unread notifications exist
- **Inbox** — combined feed of posture change notifications and draft audit logs

Clicking a posture card drills into the policy with tabs for Requirements, Evidence, and History.

### 5. Audit Production (agent-assisted)

The liaison opens the chat assistant and asks for an audit. The agent:

1. Loads the Policy and its MappingDocuments
2. Queries evidence per control within the audit window
3. Classifies each result: Strength, Finding, Gap, or Observation
4. Joins through the MappingDocument to map findings to framework criteria (e.g., "BP-1 is a Gap → SOC 2 CC8.1 and CC6.1 affected")
5. Produces a draft AuditLog artifact, validates it against the Gemara schema
6. Publishes the draft to the Inbox

### 6. Review (audit liaison)

The draft appears in the Inbox with an unread indicator. The liaison clicks it and sees:

- Each result card with the agent's classification and reasoning
- A type override dropdown to reclassify (e.g., agent said Gap, liaison says Finding)
- A reviewer note field per result
- Auto-save on edit (debounced 1s)

When satisfied, the liaison clicks **Save to History** to promote the draft to the official audit record.

### 7. Export and Delivery

From the History tab inside the policy drill-down:

- **Download YAML** — raw Gemara AuditLog for machine consumption
- **Export Excel** — spreadsheet with findings, evidence references, and framework mappings
- **Export PDF** — formatted report for the auditor interview

The auditor receives a deliverable that maps your controls to their framework, backed by timestamped evidence, with reviewer notes explaining any overrides.

## What It Is Not

- **Not an authoring tool.** Policies and MappingDocuments are created by engineers in local tooling. Studio consumes them.
- **Not a policy engine.** Studio reads evidence that policy engines (OPA, Kyverno) produce.
- **Not a replacement for complyctl.** `complyctl` is the CLI for validation, transformation, and CI automation. Studio is the dashboard.
