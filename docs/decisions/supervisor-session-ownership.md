# Supervisor Session Ownership

**Date**: 2026-05-11
**Status**: Accepted

## Decision

The Studio Assistant is the permanent session owner. BYO agents are invoked by the assistant via A2A delegation, not selected by the user in a picker. The agent picker dropdown is deprecated as a routing mechanism.

## Problem

A consumer has a BYO agent that returns domain-specific data. The current architecture forces either/or:

1. User talks to the assistant (has compliance context, tools, evidence access)
2. User switches to the BYO agent via picker (loses all session context)

Switching agents calls `handleNewSession()` which wipes messages and taskId. The user must manually re-provide context to the BYO agent, then manually re-provide the BYO agent's output back to the assistant. This is the same "copy-paste between tools" workflow Studio exists to eliminate.

The assistant also has no A2A client -- it can call MCP tools but cannot invoke other agents. Even if the picker is removed, the assistant has no delegation path.

## Solution

The assistant owns the session for its entire lifetime. When the user's request requires capabilities from a BYO agent, the assistant delegates via A2A through AgentGateway and merges the response into its own state. The user never leaves the conversation.

```
User → Assistant (session owner)
              │
              ├── MCP tools (gemara, postgres, oras)
              │
              └── A2A delegation → BYO Agent (stateless)
                        │
                        └── domain-specific data returned
                              │
                              └── merged into assistant State
```

## What Changes

| Before | After |
|:--|:--|
| User picks agent in dropdown | Assistant is always the session agent |
| Switching agents wipes context | No switching -- assistant delegates |
| Assistant has MCP tools only | Assistant has MCP tools + A2A delegation |
| BYO agent is a session peer | BYO agent is a stateless worker |

## Scope

- One concrete BYO consumer agent returning domain-specific data
- The BYO agent does NOT produce Gemara artifacts -- the assistant incorporates its data into its own work
- Generalize to ANS / capability routing only when agent count exceeds 2

## What This Does NOT Include

| Deferred | Trigger |
|:--|:--|
| Agent Naming Service (dynamic discovery) | >2 BYO agents registered |
| Capability Router (domain → agent mapping) | >2 BYO agents with overlapping skills |
| Verification gate on worker responses | BYO agent starts producing Gemara artifacts |
| Context compaction before dispatch | Worker payloads exceed token budget |
| Multi-model orchestration (planner/worker split) | Operational data shows token cost justifies it |

## Relationship to Agent Interaction Model ADR

The agent-interaction-model ADR remains valid: Studio is a HITL chatbot. This ADR extends it -- the assistant is still HITL, but it can now delegate sub-tasks to BYO workers as part of drafting. The human still reviews and approves all outputs.

## Relationship to Verification Harness

The `langgraph-verification-harness` change builds the graph infrastructure (structured state, subgraphs, deterministic nodes) that makes delegation possible. Delegation is a node in the graph that writes worker responses into State. The verification gate applies to the assistant's own AuditLog output, not to raw worker data.
