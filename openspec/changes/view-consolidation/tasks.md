# Tasks: View Consolidation

## Phase 1 — Schema + Backend Foundation

| ID | Task | Status |
|:---|:--|:--:|
| P1-1 | Bump `go-gemara` to v0.4.x for bundle unpack; drop `replace` when upstream ready | [x] |
| P1-2 | Migration `005_programs.sql`: `programs`, `jobs`; `guidance_catalog_id`, `applicability`; guidance entry applicability | [x] |
| P1-3 | Migration `006_writer_role.sql`: allow `writer` on `users.role` | [x] |
| P1-4 | Migration `007_recommendation_dismissals.sql` (or equivalent dismissals storage) | [x] |
| P1-5 | `ProgramStore`, job store, `InventoryStore` interfaces; postgres implementation | [x] |
| P1-6 | `POST /api/import` unified entry; GuidanceCatalog parser → `guidance_entries` | [x] |
| P1-7 | `GET /api/inventory` with policy/program filters | [x] |
| P1-8 | Writer-aware `writeProtect`; admin-only settings | [x] |

## Phase 2 — Posture + Recommendations

| ID | Task | Status |
|:---|:--|:--:|
| P2-1 | `internal/posture/` engine; NATS subscribe on evidence ingest | [x] |
| P2-2 | Posture regression notifications | [x] |
| P2-3 | Recommendations: overlap + evidence quality + mapping strength; dismiss + attach | [x] |

## Phase 3 — Navigation + Dashboard

| ID | Task | Status |
|:---|:--|:--:|
| P3-1 | Sidebar: Dashboard, Programs, Policies, Inventory, Evidence, Reviews + Settings gear | [x] |
| P3-2 | `app.tsx` routing for new views | [x] |
| P3-3 | Dashboard: metrics, actionable cards, filtered deep-links | [x] |

## Phase 4 — Source-of-Truth Views

| ID | Task | Status |
|:---|:--|:--:|
| P4-1 | Programs list + create (guidance catalog + applicability) + detail | [x] |
| P4-2 | Reviews queue + workspace (scope/period/methodology, evidence projection, promote) | [x] |
| P4-3 | Inventory standalone view + chips | [x] |
| P4-4 | Evidence/Policies: program chip + URL initial filters | [x] |
| P4-5 | Unified Import UI: header button; contextual Programs/Policies actions | [x] |

## Phase 5 — Stripe Design Tokens

| ID | Task | Status |
|:---|:--|:--:|
| P5-1 | Map tokens to CSS variables in `global.css` | [x] |
| P5-2 | Align component styles (header, sidebar, views) to tokens | [x] |

## Phase 6 — Seed Data + Validation

| ID | Task | Status |
|:---|:--|:--:|
| P6-1 | Demo seed: guidance catalogs, programs, mappings, policies, evidence | [ ] |
| P6-2 | E2E validation: import guidance → program → assign policies → ingest evidence → posture → review | [ ] |

## Phase 7 — OpenSpec Documentation

| ID | Task | Status |
|:---|:--|:--:|
| P7-1 | `openspec/changes/view-consolidation/` proposal, design, tasks, spec | [x] |
