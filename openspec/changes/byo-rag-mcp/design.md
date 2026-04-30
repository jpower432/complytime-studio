# Design: BYO RAG via Knowledge Base MCP Server

## MCP Tool Contract

### `search_knowledge_base`

**Description:** Search the compliance knowledge base for relevant documents.

**Input:**

```json
{
  "query": "What are the FedRAMP High baseline requirements for access control?",
  "filters": {
    "framework": "fedramp-high",
    "document_type": "guidance",
    "tags": ["access-control"]
  },
  "top_k": 5
}
```

| Parameter | Type | Required | Notes |
|:--|:--|:--|:--|
| `query` | `string` | yes | Natural language search query |
| `filters` | `object` | no | Key-value pairs for metadata filtering. Keys are backend-defined. |
| `top_k` | `integer` | no | Max results to return. Default: 5. |

**Output:**

```json
{
  "results": [
    {
      "content": "AC-2: Account Management. The organization manages...",
      "source": "nist-800-53-rev5/ac-2",
      "score": 0.92,
      "metadata": {
        "framework": "fedramp-high",
        "document_type": "guidance",
        "title": "AC-2 Account Management"
      }
    }
  ],
  "total": 42,
  "query_ms": 120
}
```

| Field | Type | Required | Notes |
|:--|:--|:--|:--|
| `results[].content` | `string` | yes | Retrieved text chunk |
| `results[].source` | `string` | yes | Document identifier / path |
| `results[].score` | `number` | no | Relevance score (0-1). Backend-specific. |
| `results[].metadata` | `object` | no | Arbitrary metadata from the backend |
| `total` | `integer` | no | Total matching documents (not just top_k) |
| `query_ms` | `integer` | no | Query execution time in milliseconds |

## Reference Implementation

Thin MCP server that proxies `search_knowledge_base` calls to an HTTP backend.

### Architecture

```
Agent → MCP protocol → knowledge-base-mcp → HTTP POST → RAG backend
                         (reference impl)                (operator-provided)
```

### Configuration

| Env var | Default | Notes |
|:--|:--|:--|
| `RAG_BACKEND_URL` | `""` | HTTP endpoint for the RAG backend. Required. |
| `RAG_TIMEOUT` | `30s` | Request timeout |
| `MCP_PORT` | `8080` | MCP server listen port |

### HTTP Backend Contract

The reference MCP server forwards to:

```
POST {RAG_BACKEND_URL}/search
Content-Type: application/json

{
  "query": "...",
  "filters": {...},
  "top_k": 5
}
```

Response must match the `results` schema above. This is the contract between the reference MCP server and the RAG backend.

### Operators can replace entirely

The reference implementation is convenience. Operators can deploy any MCP server that implements `search_knowledge_base` with the schema above. The agent doesn't care what's behind the MCP interface.

## Agent Integration

Agents connect via `KNOWLEDGE_BASE_MCP_URL` env var:

```python
# In tool registry
if os.environ.get("KNOWLEDGE_BASE_MCP_URL"):
    tools.append(mcp_tool("search_knowledge_base", url=os.environ["KNOWLEDGE_BASE_MCP_URL"]))
```

Tool is **optional**. If `KNOWLEDGE_BASE_MCP_URL` is not set, agents operate without RAG. Commands that reference `search_knowledge_base` in `COMMAND_TOOL_MAP` gracefully degrade (LLM-only, no retrieval).

## Helm Integration

```yaml
# values.yaml
rag:
  enabled: false
  image:
    repository: ""    # operator provides image
    tag: latest
  backendUrl: ""      # HTTP endpoint for RAG backend (used by reference impl)
  resources:
    requests:
      memory: "512Mi"
      cpu: "500m"
    limits:
      memory: "2Gi"
      cpu: "2"
```

When `rag.enabled` and `rag.image.repository` set, Helm deploys the MCP server pod and wires `KNOWLEDGE_BASE_MCP_URL` to LangGraph agent env vars.

When `rag.enabled` but `rag.image.repository` empty, Helm skips the deployment — operator manages the MCP server externally and sets the URL directly.

## Examples: Plugging In Common Backends

### PGVector (custom service)

Operator deploys a FastAPI service that queries PGVector embeddings. Points `RAG_BACKEND_URL` at it. Reference MCP server proxies.

### Haystack

Operator deploys Haystack pipeline with a REST API. Points `RAG_BACKEND_URL` at the Haystack `/query` endpoint. May need a thin adapter if Haystack's response format differs.

### Vectara / Pinecone / Weaviate

Same pattern: operator deploys their preferred vector DB with an HTTP search API, wraps it to match the `/search` contract, points `RAG_BACKEND_URL` at it.

### No RAG

Leave `rag.enabled: false`. Agents work without retrieval. Knowledge comes from system prompts and skills only.

## Tests

| Test | Validates |
|:--|:--|
| `TestSearchKnowledgeBase_ValidQuery` | MCP tool returns results matching schema |
| `TestSearchKnowledgeBase_WithFilters` | Filters forwarded to backend |
| `TestSearchKnowledgeBase_EmptyResults` | Graceful empty response |
| `TestSearchKnowledgeBase_BackendTimeout` | Timeout returns error, not hang |
| `TestSearchKnowledgeBase_BackendDown` | Returns MCP error, agent continues without RAG |
| `TestAgent_NoRAGConfigured` | Agent operates normally when KNOWLEDGE_BASE_MCP_URL unset |
