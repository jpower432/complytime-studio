# Proposal: Workbench Program Views

## User Story

As a compliance program manager, I need to manage program lifecycles — creation, status tracking, evidence oversight, structured command execution, and agent chat — through Studio's web interface so that I can run compliance programs without CLI tools.

## Problem

Studio's workbench is organized around policies, evidence, and audit logs. There is no concept of programs — no way to group policies into a managed lifecycle, track program health, execute structured commands against a program, or see cross-program posture.

## Solution

Add four views to the Preact workbench, backed by new gateway API endpoints (from the dual-store data layer spec). Views are implemented in Preact with signals — following Studio's existing patterns.

## Scope

| In Scope | Out of Scope |
|:--|:--|
| Programs list view | Kanban board view (extension agent territory) |
| Program detail view (tabbed: overview, evidence, commands, chat) | Calendar view (Phase 2) |
| Command bar + structured command output | PatternFly component adoption (stay with current CSS) |
| Dashboard enhancement with program health cards | Mobile-responsive layout changes |
| Create/edit program flow (modal or wizard) | Portfolio metrics (deferred until coordinator agent validated) |
| Sidebar navigation expansion | |
| API client functions for programs and commands | |
