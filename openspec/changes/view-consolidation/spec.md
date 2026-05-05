# Spec: View Consolidation

## Capability

Program-centric compliance navigation, unified Gemara import, on-ingest posture, policy recommendations, inventory rollups, cross-view filter chips, and writer RBAC.

## Scenarios

### Program Creation

#### Successful create (writer)

| Clause | Statement |
|:--|:--|
| GIVEN | At least one imported GuidanceCatalog exists |
| AND | Authenticated user has `writer` or `admin` role |
| WHEN | User selects catalog, applicability values, confirms metadata, submits create |
| THEN | API persists a `programs` row with `guidance_catalog_id`, `applicability`, empty `policy_ids` |
| AND | Program detail shows zero baseline coverage until policies attach |
| AND | Recommendations become available when mapping-backed policies exist |

#### Denied create (reviewer)

| Clause | Statement |
|:--|:--|
| GIVEN | Authenticated user has `reviewer` role only |
| WHEN | User attempts Programs create via UI or `POST /api/programs` |
| THEN | Server rejects with forbidden |
| AND | UI hides or disables creation affordances |

### Unified Import

#### Auto-detect routing

| Clause | Statement |
|:--|:--|
| GIVEN | User uploads a supported Gemara YAML artifact |
| WHEN | Client calls `POST /api/import` |
| THEN | Server detects artifact type without manual selector |
| AND | Persistence matches type-specific tables (policy bundle unpacks all constituents) |

#### Policy bundle unpack

| Clause | Statement |
|:--|:--|
| GIVEN | Payload is a policy bundle with multiple nested artifacts |
| WHEN | Import completes |
| THEN | Policy leaf lands in `policies` |
| AND | Nested ControlCatalog / GuidanceCatalog / threat / risk artifacts populate structured tables |
| AND | New guidance catalogs appear in Program creation selector |

#### Mapping gap hint

| Clause | Statement |
|:--|:--|
| GIVEN | Fresh policy import finishes |
| AND | No MappingDocument references that policy's catalogs |
| WHEN | User views import result or policy detail |
| THEN | UI surfaces guidance to import mappings for coverage tracking |

### Policy Recommendation + Attach

#### On-demand suggestions

| Clause | Statement |
|:--|:--|
| GIVEN | Program references a guidance catalog with mapped guidelines |
| AND | Candidate policies share controls mapped to those guidelines |
| WHEN | User opens recommendations panel on Program detail |
| THEN | Server returns ranked suggestions using overlap primary, evidence quality and mapping strength as context |

#### Attach policy

| Clause | Statement |
|:--|:--|
| GIVEN | Writer or admin views a recommendation |
| WHEN | User activates **Attach** |
| THEN | Program `policy_ids` updates idempotently |
| AND | Posture reflects new policy scope on next compute cycle |

#### Dismiss recommendation

| Clause | Statement |
|:--|:--|
| GIVEN | Writer or admin dismisses a recommendation for the program |
| WHEN | User reloads panel or returns later |
| THEN | Dismissed item stays suppressed for that program |

#### Reviewer read-only

| Clause | Statement |
|:--|:--|
| GIVEN | User is reviewer-only |
| WHEN | User opens Program detail |
| THEN | Attach and dismiss controls are unavailable |
| AND | API rejects mutating recommendation endpoints |

### Posture Computation

#### Recompute on ingest

| Clause | Statement |
|:--|:--|
| GIVEN | Evidence ingest persists new EvaluationLog data for a policy |
| WHEN | NATS publish completes |
| THEN | Posture worker recomputes programs referencing that policy via assignments and mappings |

#### Regression notification

| Clause | Statement |
|:--|:--|
| GIVEN | Posture degrades relative to prior snapshot |
| WHEN | Worker persists results |
| THEN | System emits actionable notification consumed by Dashboard/Reviews indicators |

### Cross-view Navigation with Filter Chips

#### Deep-link sets chip

| Clause | Statement |
|:--|:--|
| GIVEN | User follows a Dashboard or Program detail link that encodes scope |
| WHEN | Destination view loads |
| THEN | URL/query hydrates filters |
| AND | Matching chips render above the table |
| AND | Data queries include chip predicates |

#### Clear chip resets scope

| Clause | Statement |
|:--|:--|
| GIVEN | One or more chips active |
| WHEN | User dismisses a chip |
| THEN | Corresponding query param and control state clear |
| AND | Result set expands accordingly |

### Writer Role Access Control

#### Writer mutates content

| Clause | Statement |
|:--|:--|
| GIVEN | User role is `writer` |
| WHEN | User creates or updates programs, imports artifacts, attaches policies |
| THEN | Server permits operations guarded as writer-capable |

#### Writer cannot administer users

| Clause | Statement |
|:--|:--|
| GIVEN | User role is `writer` |
| WHEN | User calls user-admin or role mutation endpoints |
| THEN | Server responds forbidden |

#### Settings restricted

| Clause | Statement |
|:--|:--|
| GIVEN | User role is `writer` or `reviewer` |
| WHEN | User navigates to Settings |
| THEN | UI blocks or server denies access unless `admin` |
