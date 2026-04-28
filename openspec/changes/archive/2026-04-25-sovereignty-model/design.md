# Sovereignty model — design

## Context

Studio ingests compliance evidence from OTel and REST, persists it in ClickHouse, and uses Workbench plus the agent for analysis. Regulated and PII-heavy raw artifacts must stay in trust boundaries; a central Studio instance should still list posture and trace to full artifacts when auditors need them.

## Decision 1: Only `source_registry` as schema addition

**Choice:** Add `source_registry Nullable(String)` to `evidence`. Do **not** add `region_id`, `tenant_id`, or other sovereignty dimension columns in this change.

**Rationale:** Provenance for raw bundles is content-addressed: `attestation_ref` is the digest; `source_registry` is the OCI host where the bundle lives. Extra dimensions duplicate information that the OCI reference and registry access model already express, and they invite inconsistent labeling across clients.

**Consequences:** Optional nullable column, additive migration, backward compatible. Evidence without a registry still works; sovereignty tracking is best-effort per row.

## Decision 2: Sovereignty is architectural, not a CHECK constraint

**Choice:** Enforce the “no raw PII in Studio” rule through product documentation, complyctl behavior, and optional operational warnings (e.g. large `eval_message` logs). Do not add ClickHouse `CHECK` constraints or reject rows for “summary shape.”

**Rationale:** Studio cannot cryptographically prove that a string field is PII-free. Enforcement belongs at the client boundary (complyctl packages raw data into OCI, sends summaries) and in organizational process.

**Consequences:** Misconfigured clients could still load large text into `eval_message`; the Gateway may warn to surface mistakes without blocking all ingestion.

## Decision 3: complyctl owns the boundary contract

**Choice:** Formally document: **complyctl** (or equivalent scan client) pushes raw attestation bundles to the **boundary** OCI registry, then sends **summary** evidence to Studio (OTel or REST) with `attestation_ref` and `source_registry`. **Studio** stores summaries and digests; it does not ingest raw artifacts as a requirement for normal operation.

**Rationale:** Single writer for “what left the boundary” — the tool that already has credentials to the local registry. Studio remains a read/query surface for OCI when verifying attestations, not a second hop for raw bundle upload.

**Consequences:** Documentation and complyctl release notes must stay aligned; Studio APIs stay minimal (no new raw-upload contract in this design).

Normative architecture write-up: [Trust Boundary Contract](../../../docs/design/architecture.md#trust-boundary-contract).

## Decision 4: Attestation-verification cross-registry resolution

**Choice:** When `source_registry` is set, the attestation-verification skill uses that value to direct oras-mcp (or equivalent) to pull `attestation_ref` from the correct registry, rather than defaulting to a single Studio-adjacent registry. When `source_registry` is `NULL`, behavior matches pre-change defaults (e.g. default registry from environment or policy context).

**Rationale:** Bundles for different trust boundaries live on different OCI hosts; the digest alone does not say which host to query. Default-only resolution would fail for multi-boundary deployments.

**Consequences:** oras-mcp (or the skill’s invocation) must accept per-request registry base URL or an equivalent parameter. The skill’s SKILL.md and verification steps are updated to read `source_registry` from the evidence row.

**Failure modes:** If the registry is wrong or credentials are missing, the skill reports a registry/availability outcome (per existing `REGISTRY UNAVAILABLE` style) without fabricating a verified chain.

## Decision 5: Optional raw-data heuristics at ingest

**Choice:** The Gateway may compare `eval_message` length (and optionally other cheap heuristics such as high base64-like density) against thresholds and log warnings. This is **not** a hard block unless a future decision adds it.

**Rationale:** Aligns with the risk in cloud-native posture correction: if clients send dumps into summary fields, operators need a signal. Schema cannot fix discipline; logging can.

**Consequences:** Tunable threshold or static default; document that warnings are heuristics, not legal classification of data sensitivity.

## Related documents

- `docs/decisions/cloud-native-posture-correction.md` — sovereignty and summary-only ingestion
- `docs/design/architecture.md` — evidence pipeline, Gateway, oras-mcp, Workbench
- `openspec/changes/evidence-attestation-pipeline/design.md` — attestation_ref, OCI, agent verification
- `docs/design/evidence-semconv-alignment.md` — extend with `compliance.source.registry` → `source_registry`
