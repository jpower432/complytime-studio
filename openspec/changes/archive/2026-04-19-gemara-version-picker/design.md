## Context

The `validate()` function in `a2a.ts` accepts a `version` parameter (defaults to `"latest"`) and passes it to the gateway, which forwards it to gemara-mcp's `validate_gemara_artifact` tool. The CUE registry resolves the version to a specific module tag. The UI currently hardcodes `"latest"` with no way to override.

## Goals / Non-Goals

**Goals:**

- Let users specify a Gemara version for validation.
- Store the version per-artifact in the workspace.
- Keep it simple: text input with `latest` default.

**Non-Goals:**

- Fetching available versions from the CUE registry (future enhancement via `GET /api/gemara/versions`).
- Version selection for publishing (separate concern).
- Validating that the user-entered version string is real before submission.

## Decisions

| # | Decision | Rationale |
|:--|:--|:--|
| 1 | Text input, not dropdown | No version list API exists yet. Text input works immediately with zero backend changes. |
| 2 | Default to `latest` | Matches current behavior. Most users want the latest schema. |
| 3 | Store per-artifact | Different artifacts in a workspace may target different Gemara releases. |
| 4 | Place next to definition dropdown | Version and definition are tightly coupled -- you validate a `#ThreatCatalog` at a specific Gemara version. Colocating them is natural. |

## Risks / Trade-offs

| Risk | Mitigation |
|:--|:--|
| User typos an invalid version | gemara-mcp returns a clear error ("version not found"). Display it in the validation result bar. |
| Toolbar width increases | Input is narrow (~80px). Fits within the existing toolbar alongside the overflow menu. |
