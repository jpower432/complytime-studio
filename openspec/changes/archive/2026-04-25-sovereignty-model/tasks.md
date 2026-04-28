# Tasks — sovereignty model (`source_registry`)

## Schema migration

- [x] Add `source_registry Nullable(String)` to the `evidence` table via versioned `schemaMigrations()` in `internal/clickhouse/client.go` (additive `ALTER TABLE` following existing attestation pattern).
- [x] Update `CREATE TABLE IF NOT EXISTS evidence` baseline DDL for new deployments to include `source_registry` in column order consistent with the migration.
- [x] Regenerate or update any generated schema assets or Helm `ConfigMap` schema snippets if the repo duplicates DDL.

## Gateway handler changes

- [x] Extend evidence REST DTOs and handlers to accept optional `source_registry` on `POST` (and any batch/upload paths in scope for this change).
- [x] Plumb `source_registry` from JSON into `InsertEvidence` / batch insert for all REST evidence insert code paths in scope.
- [x] Add optional `eval_message` size (or length) check with structured warning logs; document threshold in code or config.
- [x] Add unit or handler tests for with/without `source_registry` and for oversized `eval_message` warning path (if implemented).

## OTel attribute mapping

- [x] Update `docs/design/evidence-semconv-alignment.md` with `compliance.source.registry` -> `source_registry` (`Nullable(String)`).
- [x] Align OTel Collector / exporter config or documentation so the attribute maps to the same column (environment-specific; keep Studio docs the source of truth for attribute name).

## Workbench display

- [x] Add `source_registry` to evidence types and API responses consumed by the Workbench.
- [x] Show `source_registry` on the evidence detail view when present; handle NULL/empty per UX.
- [x] Add copy or link affordance if the value is a URL (implementation detail).

## Attestation skill update

- [x] Update `skills/attestation-verification/SKILL.md` to require reading `source_registry` from the evidence row and passing registry context to oras-mcp when pulling by `attestation_ref`.
- [x] Document default-registry fallback when `source_registry` is NULL.
- [x] Adjust verdict text examples to mention cross-registry resolution failures distinctly from missing attestation.

## Boundary contract documentation

- [x] Add a "Trust boundary and evidence" subsection under `docs/design/architecture.md` stating: raw bundles in boundary OCI; Studio stores summaries, `attestation_ref`, and `source_registry` only; complyctl owns the push contract.
- [x] Cross-link from the sovereignty-model proposal or OpenSpec to that section.

## Testing

- [x] Integration or E2E: insert evidence with `source_registry` via REST, query back and assert column round-trip.
- [x] If OTel test harness exists: one fixture with `compliance.source.registry` -> assert `source_registry` in CH.
- [x] Manual or automated check: Workbench detail view displays `source_registry` for a seeded row.
- [x] Skill path: with a test registry URL in `source_registry`, verify the skill's instructions lead to oras-mcp using that registry (mock or local registry as appropriate).
