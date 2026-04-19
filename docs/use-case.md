# ComplyTime Studio: Use Case and Market Fit

ComplyTime Studio is an agentic platform for authoring, reviewing, and shipping machine-readable GRC (Governance, Risk, and Compliance) artifacts. It occupies a gap in the open source compliance ecosystem where schemas exist, CLIs exist, but no opinionated authoring workflow connects them.

## The Gap

The open source compliance ecosystem has the primitives but not the workflow.

| Layer | What Exists | What's Missing |
|:--|:--|:--|
| **Schema** | [Gemara](https://gemara.openssf.org/) defines machine-readable types (ThreatCatalog, ControlCatalog, Policy, AuditLog, etc.) | No guided authoring experience — users hand-write YAML |
| **CLI** | [complyctl](https://github.com/complytime/complyctl) validates and transforms artifacts | Validation catches errors after the fact, not during authoring |
| **Registry** | OCI registries (GHCR, Zot) store bundles | No integrated publish flow from authoring to registry |
| **Assessment** | Policy engines (OPA, Kyverno) produce evidence | Evidence sits in logs — not connected to the artifacts that define what to assess |

Each tool solves one piece. No tool owns the end-to-end workflow: *author → validate → iterate → bundle → publish*.

## Who This Is For

**Primary persona: The compliance engineer who maintains GRC artifacts in Git.**

They know their frameworks (NIST, SOC2, FedRAMP). They know YAML. They don't want a SaaS platform that locks their data behind a login. They want artifacts in Git, versioned, reviewable, publishable as OCI bundles — the same workflow they use for code.

**Secondary persona: The security team lead who needs artifacts but lacks deep Gemara expertise.**

They can describe what they need ("analyze threats for Kyverno using STRIDE") but shouldn't have to know the exact YAML structure. Agents bridge the expertise gap.

## What It Should Do

Studio is an **agentic platform with artifact review** — not a general-purpose code editor, not a GRC SaaS dashboard.

```
┌─────────────────────────────────────────────────────┐
│                    User Workflow                     │
│                                                     │
│   1. Direct    "Analyze threats for Kyverno"        │
│        │                                            │
│        ▼                                            │
│   2. Agent     Threat modeler runs STRIDE,          │
│      works     calls GitHub MCP for repo context,   │
│                produces ThreatCatalog YAML           │
│        │                                            │
│        ▼                                            │
│   3. Review    Artifact appears as proposal.        │
│                User applies, edits, validates.       │
│        │                                            │
│        ▼                                            │
│   4. Chain     "Now write a policy from this."      │
│                User selects ThreatCatalog as         │
│                context → Policy agent runs.          │
│        │                                            │
│        ▼                                            │
│   5. Ship      Publish workspace artifacts as        │
│                OCI bundle to GHCR or Zot.            │
└─────────────────────────────────────────────────────┘
```

### Core Capabilities

| Capability | Description |
|:--|:--|
| **Agent-driven authoring** | Specialist agents (threat modeler, policy composer, gap analyst) produce Gemara-compliant artifacts from natural language prompts |
| **Artifact workspace** | Multi-artifact workspace accumulates output across jobs. Tabs for switching, localStorage persistence. |
| **Agent chaining** | Feed one agent's output as context to another. ThreatCatalog → ControlCatalog → Policy is a multi-job workflow, not a copy-paste exercise. |
| **Proposal gating** | Agents propose artifacts. Users accept or dismiss. No silent overwrites. Undo/redo via CodeMirror history. |
| **Inline validation** | Validate against Gemara schemas during authoring, not after commit. |
| **OCI bundle publish** | Publish one or many workspace artifacts as a Gemara OCI bundle to GHCR or an in-cluster registry. |
| **Registry import** | Browse OCI registries, inspect layers, import artifacts into the workspace as starting points. |
| **Evidence-backed audit** | Gap analyst queries ClickHouse for assessment evidence, producing AuditLogs grounded in real data. |

## Why Not Just Use Cursor or Claude Code?

Local AI-assisted editors are powerful for general coding. Studio's value is in what they can't provide:

| Concern | Local AI Editor | ComplyTime Studio |
|:--|:--|:--|
| **Schema awareness** | Generic YAML — no Gemara knowledge unless prompted | Built-in Gemara validation and definition detection |
| **Agent specialization** | General-purpose model, no domain prompts | STRIDE threat modeler, policy composer, gap analyst with domain skills |
| **Artifact context** | Manual copy-paste between sessions | Workspace artifacts are selectable as input context for new jobs |
| **Evidence grounding** | No access to assessment data | Gap analyst queries ClickHouse evidence directly |
| **Publishing** | Manual `oras push` or script | Integrated bundle publish to GHCR/Zot from the UI |
| **Approval flow** | AI writes directly to files | Agent proposes, user reviews and accepts |

Studio is not competing with general-purpose editors. It provides the **opinionated compliance workflow** that general tools cannot.

## What It Is Not

- **Not a GRC SaaS dashboard.** No database of controls. No compliance scorecards. Artifacts live in Git and OCI registries.
- **Not a general-purpose code editor.** It edits Gemara YAML, not arbitrary code.
- **Not a policy engine.** It authors the artifacts that policy engines consume, and ingests the evidence they produce.
- **Not a replacement for complyctl.** complyctl is the CLI for validation, transformation, and automation. Studio is the authoring experience that produces what complyctl processes.

## Ecosystem Position

```
┌──────────────────────────────────────────────────────────┐
│                   Compliance Lifecycle                    │
│                                                          │
│  Author          Validate       Assess         Audit     │
│  ──────          ────────       ──────         ─────     │
│                                                          │
│  ┌────────────┐  ┌──────────┐  ┌────────────┐  ┌──────┐│
│  │ ComplyTime │  │complyctl │  │  OPA /     │  │Audit ││
│  │  Studio    │──│  + CI    │──│  Kyverno   │──│ Log  ││
│  │            │  │          │  │            │  │      ││
│  │ Agents     │  │ Validate │  │ Evaluate   │  │Report││
│  │ Author     │  │ Transform│  │ Remediate  │  │      ││
│  │ Publish    │  │ Bundle   │  │ Evidence→  │  │      ││
│  └────────────┘  └──────────┘  │ ClickHouse │  └──────┘│
│       │                        └────────────┘     ▲     │
│       │              OCI Registry                 │     │
│       └──────────── bundles ──────────────────────┘     │
│                                                          │
│  Gemara schema governs all artifact types across         │
│  the entire lifecycle.                                   │
└──────────────────────────────────────────────────────────┘
```

Studio owns the **left side** of this lifecycle: authoring, iteration, and initial publish. Everything downstream — CI validation, policy engine assessment, audit reporting — consumes the artifacts Studio produces.
