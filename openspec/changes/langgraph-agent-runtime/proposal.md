# Proposal: LangGraph Agent Runtime

## User Story

As a platform operator, I need to deploy specialized compliance agents that run on LangGraph so that my team can execute structured compliance commands through Studio's workbench.

As a developer, I need to build new agent personas by writing markdown spec files and registering them in a single container image so that adding agents does not require new infrastructure.

## Problem

Studio has one agent: `studio-assistant` (Python ADK, speaks A2A). There is no mechanism to deploy LangGraph-based agents that speak A2A, load structured command specifications, filter tools per command, or checkpoint multi-turn conversations.

## Solution

Create a single parameterized container image that wraps LangGraph `CompiledStateGraph` instances in A2A endpoints via `kagent-langgraph`. An `AGENT_TYPE` env var selects the persona (markdown spec file). kagent manages deployment, scaling, and A2A exposure. Studio's gateway routes to them via the existing A2A proxy.

## Scope

| In Scope | Out of Scope |
|:--|:--|
| LangGraph agent container image with A2A adapter | Rewriting existing assistant in LangGraph |
| Spec loading (constitution, commands, agent personas) | Runtime agent hot-swap |
| MCP tool integration via `kagent-langgraph` / LangChain adapters | Proprietary integrations (extension agents) |
| `AGENT_TYPE` env var to select persona | |
| kagent BYO Agent CRD templates | |
| PostgreSQL checkpointer for multi-turn chat | |
| Quality gate validation via MCP tool (per governance spec) | |
| 3 personas for v1: `program`, `evidence`, `coordinator` | Remaining personas deferred until validated |
