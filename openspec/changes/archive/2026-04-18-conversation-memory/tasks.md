## 1. ChatMessage Model & Storage

- [x] 1.1 Add optional `pinned: boolean` field to `ChatMessage` interface in `chat-assistant.tsx`
- [x] 1.2 Create `studio-sticky-notes` localStorage helpers: `loadStickyNotes()`, `saveStickyNotes()` with `StickyNote` type `{id: string, text: string, createdAt: string}`
- [x] 1.3 Create `studio-pinned-cache` localStorage helpers: `savePinnedCache()`, `loadAndClearPinnedCache()`

## 2. Sticky Notes Panel

- [x] 2.1 Create `StickyNotesPanel` component with note list, add input (200 char limit), delete buttons
- [x] 2.2 Enforce 10-note maximum with disabled input and limit message
- [x] 2.3 Add sticky notes toggle button to chat header (replaces or sits alongside session controls)
- [x] 2.4 Add CSS styles for sticky notes panel, note items, character counter

## 3. Pin UI

- [x] 3.1 Add pin toggle button to agent messages (hover/focus reveal)
- [x] 3.2 Implement `togglePin(index)` handler that flips `pinned` on the message at the given index
- [x] 3.3 Enforce 5-pin limit — auto-unpin oldest and show notification when exceeded
- [x] 3.4 Add CSS for pinned indicator and pin button states

## 4. Session Controls

- [x] 4.1 Replace "Clear" button with "New Session" button
- [x] 4.2 "New Session" handler: write pinned messages to `studio-pinned-cache`, clear messages, null `taskIdRef`
- [x] 4.3 Add "Checkpoint" button to chat header
- [x] 4.4 "Checkpoint" handler: serialize turns since last checkpoint into summary string, insert checkpoint divider message into `messages[]`, null `taskIdRef`, store summary for next injection
- [x] 4.5 Add CSS for checkpoint divider visual separator

## 5. Context Assembly

- [x] 5.1 Create `buildInjectedContext()` function that composes sticky notes + pinned cache + checkpoint summary into tagged text blocks
- [x] 5.2 Serialize sticky notes as `<sticky-notes>` block (bullet list of note texts)
- [x] 5.3 Serialize pinned cache + checkpoint summary as `<conversation-history>` block (each pin truncated to 500 chars)
- [x] 5.4 Enforce 4500-char total budget — truncate pins further if over budget
- [x] 5.5 Update `send()` function: when `taskIdRef` is null, call `buildInjectedContext()` and prepend to `streamMessage()` text
- [x] 5.6 After injection, clear pinned cache and checkpoint summary from localStorage

## 6. Context Injection Indicator

- [x] 6.1 Add collapsed "Memory context sent to agent" block as the first item in messages when context was injected
- [x] 6.2 Expanding the block shows the exact injected text
- [x] 6.3 Add CSS for the context indicator block (collapsed/expanded states)

## 7. Agent Prompt Update

- [x] 7.1 Add `<sticky-notes>` tag convention to `agents/assistant/prompt.md` constraints section
- [x] 7.2 Add instruction for agent to suggest sticky notes when user establishes persistent scope/dates/priorities
- [x] 7.3 Copy updated prompt to `charts/complytime-studio/agents/assistant/prompt.md`

## 8. Verification

- [x] 8.1 Test pin flow: pin 2 messages, click "New Session", send message — verify injected context includes pins
- [x] 8.2 Test checkpoint flow: have 8+ turn conversation, click "Checkpoint", send message — verify condensed summary injected
- [x] 8.3 Test sticky notes: add 3 notes, start new session, send message — verify `<sticky-notes>` block in injected context
- [x] 8.4 Test pin limit: pin 6 messages — verify oldest auto-unpinned with notification
- [x] 8.5 Test sticky note limit: add 11 notes — verify input disabled at 10
- [x] 8.6 Test context indicator: verify collapsed block shows after injection, expands to show full text
