# Tasks: BYO RAG via Knowledge Base MCP Server

## MCP tool contract
- [ ] Define `search_knowledge_base` input/output JSON schema
- [ ] Document schema in `docs/design/knowledge-base-mcp-contract.md`

## Reference implementation
- [ ] Create `mcp-servers/knowledge-base/` directory
- [ ] Implement MCP server with `search_knowledge_base` tool
- [ ] HTTP proxy to `RAG_BACKEND_URL` backend
- [ ] Timeout handling, error responses
- [ ] Dockerfile for reference image

## Agent integration
- [ ] Add `KNOWLEDGE_BASE_MCP_URL` env var to LangGraph agent runtime
- [ ] Conditionally register `search_knowledge_base` tool when URL is set
- [ ] Graceful degradation: commands work without RAG if URL unset

## Helm
- [ ] Add `rag.*` values section
- [ ] Create `templates/knowledge-base-mcp.yaml` (gated on `rag.enabled` + `rag.image.repository`)
- [ ] Wire `KNOWLEDGE_BASE_MCP_URL` to LangGraph agent pods when RAG enabled
- [ ] Document external RAG server setup (operator manages, sets URL directly)

## Tests
- [ ] MCP tool: valid query, filters, empty results, timeout, backend down
- [ ] Agent: operates without RAG when URL unset
