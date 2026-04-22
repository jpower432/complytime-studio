## 1. Rewrite callbacks for current ADK API

- [x] 1.1 Rewrite `callbacks.py` with correct ADK callback signatures (`before_agent_callback(callback_context)`, `after_agent_callback(callback_context)`, `before_tool_callback(tool, args, tool_context)`)
- [x] 1.2 Implement `before_agent` — parse user message for policy reference and audit timeline, log warnings if missing
- [x] 1.3 Implement `after_agent` — extract YAML blocks from output, validate AuditLog markers, call `save_artifact` with `application/yaml` MIME
- [x] 1.4 Implement `before_tool` — regex SQL guard rejecting DDL/DML in `run_select_query` args

## 2. Custom event converter

- [x] 2.1 Rewrite `event_converter.py` as an `AdkEventToA2AEventsConverter` callable matching `A2aAgentExecutorConfig.adk_event_converter` signature
- [x] 2.2 Detect `artifact_delta` on ADK events and emit `TaskArtifactUpdateEvent` with `application/yaml` mimeType metadata
- [x] 2.3 Pass through text-only events as standard `TaskStatusUpdateEvent`

## 3. Replace `to_a2a()` with manual executor

- [x] 3.1 Construct `Runner` with `InMemorySessionService`, `InMemoryArtifactService`, `InMemoryMemoryService`
- [x] 3.2 Create `A2aAgentExecutorConfig` with the custom `adk_event_converter`
- [x] 3.3 Create `A2aAgentExecutor` with runner and config
- [x] 3.4 Build `AgentCardBuilder` for auto-generated agent card
- [x] 3.5 Wire `DefaultRequestHandler`, `InMemoryTaskStore`, Starlette app with lifespan, and uvicorn

## 4. Enable MCP resources

- [x] 4.1 Add `use_mcp_resources=True` to gemara-mcp `McpToolset` constructor
- [x] 4.2 Verify clickhouse-mcp `McpToolset` does NOT set `use_mcp_resources`

## 5. Wire callbacks into LlmAgent

- [x] 5.1 Import callbacks and pass `before_agent_callback`, `after_agent_callback`, `before_tool_callback` to `LlmAgent` constructor
- [x] 5.2 Add `callbacks.py` and `event_converter.py` back to Dockerfile COPY

## 6. Chat assistant artifact handling

- [x] 6.1 Update `chat-assistant.tsx` to detect `TaskArtifactUpdateEvent` in `onArtifact` callback
- [x] 6.2 Render artifact card with YAML content and `application/yaml` badge
- [x] 6.3 Add "Save to Audit History" button that POSTs YAML to `/api/audit-logs`
- [x] 6.4 Show success/error toast after save

## 7. Build and verify

- [x] 7.1 Rebuild gap analyst image and deploy to kind
- [x] 7.2 Verify MCP resource access in startup logs (`gemara://schema/definitions`)
- [ ] 7.3 Test SQL guard by sending a message that triggers a ClickHouse query with DDL
- [ ] 7.4 Test artifact emission by requesting an AuditLog and confirming `TaskArtifactUpdateEvent` in SSE stream
