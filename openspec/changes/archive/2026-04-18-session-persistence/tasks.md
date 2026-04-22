## 1. Backend: Chat Store

- [x] 1.1 Add `ChatSession` struct to `internal/auth/session_store.go` with `Messages json.RawMessage`, `TaskID string`, `UpdatedAt int64`
- [x] 1.2 Add `ChatStore` interface: `PutChat(ctx, email, ChatSession)`, `GetChat(ctx, email) (*ChatSession, error)`, `DeleteChat(ctx, email)`
- [x] 1.3 Implement `ChatStore` on `MemorySessionStore` with TTL matching `sessionMaxAge` (8h)
- [x] 1.4 Write tests for `ChatStore`: put/get, TTL expiration, delete, concurrent access

## 2. Backend: REST Endpoints

- [x] 2.1 Create `handleGetChatHistory` handler — extracts email from session context, calls `ChatStore.GetChat`, returns `{"messages":[],"taskId":null}` on miss
- [x] 2.2 Create `handlePutChatHistory` handler — extracts email from session context, calls `ChatStore.PutChat`, returns 204
- [x] 2.3 Register `GET /api/chat/history` and `PUT /api/chat/history` in `cmd/gateway/main.go`
- [x] 2.4 Ensure `PUT /api/chat/history` is NOT admin-gated (viewers can save their own chat)
- [x] 2.5 Write tests for both handlers: authenticated, unauthenticated, empty state

## 3. Frontend: Remove Checkpoints

- [x] 3.1 Remove `CHECKPOINT_KEY`, `saveCheckpointSummary`, `loadAndClearCheckpointSummary` from `chat-assistant.tsx`
- [x] 3.2 Remove `handleCheckpoint` function and checkpoint-related state
- [x] 3.3 Remove checkpoint button from chat header controls
- [x] 3.4 Remove checkpoint divider rendering (`msg.isCheckpoint` branch) from message list
- [x] 3.5 Remove `isCheckpoint` field from `ChatMessage` interface
- [x] 3.6 Remove `checkpointSummary` parameter from `buildInjectedContext`, update callers
- [x] 3.7 Remove `.chat-checkpoint-divider` and related styles from `global.css`

## 4. Frontend: Server-Side Persistence

- [x] 4.1 Add `fetchChatHistory(): Promise<{messages, taskId}>` to `workbench/src/api/chat.ts`
- [x] 4.2 Add `saveChatHistory(messages, taskId): Promise<void>` to `workbench/src/api/chat.ts`
- [x] 4.3 On `ChatAssistant` mount, call `fetchChatHistory` and hydrate `messages` state and `taskIdRef`
- [x] 4.4 After each agent response completes (`onDone` with state "completed"), call `saveChatHistory`
- [x] 4.5 On "New Session", call `saveChatHistory([], null)` before clearing client state
- [x] 4.6 Handle fetch errors gracefully — fall back to empty state, log warning

## 5. Verification

- [x] 5.1 `go build ./...` passes
- [x] 5.2 `go test ./...` passes
- [x] 5.3 TypeScript `tsc --noEmit` passes
- [ ] 5.4 Send message, refresh page, verify messages and taskId restored
- [ ] 5.5 Click New Session, refresh page, verify clean state
- [ ] 5.6 Wait 8h (or mock TTL), verify state expires
- [ ] 5.7 Verify checkpoint button is gone from chat header
