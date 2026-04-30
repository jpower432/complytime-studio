# Proposal: BYO RAG via Knowledge Base MCP Server

## User Story

As a platform operator, I need to plug in my organization's existing RAG implementation (Haystack, LlamaIndex, Vectara, custom PGVector service) so that Studio's agents can search our compliance knowledge base without Studio prescribing a specific RAG stack.

As an agent developer, I need a standard tool interface for knowledge base search so that agents work with any backend without per-backend tool wiring.

## Problem

Studio agents need access to a compliance knowledge base for context retrieval (regulatory guidance, prior assessments, organizational policies). There is no standard interface for this. Without one, every RAG backend requires custom tool integration per agent framework, and switching backends requires rewiring agents.

## Solution

Define a `knowledge-base-mcp` server interface with a single tool: `search_knowledge_base`. Any RAG backend that implements this MCP server contract works with Studio agents. Studio ships a reference implementation (thin MCP proxy to an HTTP search endpoint) and operators swap the backend.

## Scope

| In Scope | Out of Scope |
|:--|:--|
| `search_knowledge_base` MCP tool contract (input/output schema) | Embedding model selection |
| Reference MCP server implementation (HTTP proxy) | Document ingestion pipeline |
| Helm template for optional RAG MCP server deployment | PGVector schema management |
| Agent wiring via `KNOWLEDGE_BASE_MCP_URL` env var | Specific RAG framework integration (Haystack, LlamaIndex, etc.) |
| | Chunking, indexing, or embedding strategies |
