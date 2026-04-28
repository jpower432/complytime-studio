# Spec: Audit History UX

## Capability

Audit history browsing and workspace context panel.

## Requirements

### Expandable Inline Cards

| ID | Requirement | Acceptance |
|:--|:--|:--|
| AH-1 | Audit history list renders cards with collapse/expand toggle | Click card header toggles expanded state |
| AH-2 | Expanded card shows full YAML content fetched on demand | Content loads from `/api/audit-logs/{id}` on first expand |
| AH-3 | Expanded card includes "Open Workspace" action | Button navigates to workspace view for deeper review |
| AH-4 | Only one card expanded at a time | Expanding a card collapses the previously expanded card |

### Simplified Workspace Panel

| ID | Requirement | Acceptance |
|:--|:--|:--|
| WP-1 | Right panel displays audit summary stats | Strengths, findings, gaps counts visible |
| WP-2 | Right panel displays audit metadata | Period, framework, creator, model, prompt version shown |
| WP-3 | Right panel provides deep links to policy detail tabs | Buttons navigate to Requirements, Evidence, History views at the policy level |
| WP-4 | Right panel does not embed full sub-views | No RequirementMatrixView, EvidenceView, or AuditHistoryView components |
