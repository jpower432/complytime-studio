# Proposal: Sub-Agent Registry

## User Story

As a platform operator, I need to register specialized AI agents alongside Studio's default assistant so that users can interact with the right agent for their task — whether it's program lifecycle, evidence monitoring, or framework-specific guidance.

## Problem

Studio's agent directory is a single-entry static JSON blob (`AGENT_DIRECTORY`) describing only `studio-assistant`. There is no mechanism to register additional agents built with different frameworks (ADK, LangGraph, raw A2A), no metadata to support intent-based routing, and no workbench UI for agent selection.

## Solution

Expand the agent directory to support multiple heterogeneous agents registered at deploy time via Helm values. Each agent entry carries enough metadata for the gateway to proxy A2A traffic, the assistant to route by intent, and the workbench to display an agent picker. Proprietary integrations are not bundled — operators register them as extension agents.

## Scope

| In Scope | Out of Scope |
|:--|:--|
| Expanded `agentDirectory` schema in `values.yaml` | Runtime agent registration (add/remove without redeploy) |
| Agent card enrichment (`id`, `role`, `framework`, `examples`, `tools`, `status`) | Agent lifecycle management (kagent owns this) |
| Assistant prompt injection of sub-agent directory | Agent health probing / liveness checks |
| `a2a_delegate` tool for assistant-to-agent routing | CRD watch / dynamic discovery |
| Workbench agent picker sourced from `GET /api/agents` | |
| `AGENT_DIRECTORY` backward compat for docker-compose | |
| Delegation depth limit (max 2 hops, no self-delegation) | |
