## 1. Enable kagent Streaming

- [x] 1.1 Add `stream: true` to `studio-threat-modeler` declarative block in `agent-specialists.yaml`
- [x] 1.2 Add `stream: true` to `studio-gap-analyst` declarative block in `agent-specialists.yaml`
- [x] 1.3 Add `stream: true` to `studio-policy-composer` declarative block in `agent-specialists.yaml`
- [x] 1.4 Verify `helm template` renders `stream: true` on all three Agent CRDs

## 2. Job Store Lifecycle

- [x] 2.1 Add `acceptedAt`, `acceptNote` fields to `Job` interface in `store/jobs.ts`
- [x] 2.2 Add `cancelJob(taskId)` function — closes SSE if active, sets status `cancelled`, sets `updatedAt`
- [x] 2.3 Add `acceptJob(taskId, note)` function — sets status `accepted`, `acceptedAt`, `acceptNote`
- [x] 2.4 Add `deleteJob(taskId)` function — removes job from localStorage
- [x] 2.5 Add `purgeHistory()` function — removes history jobs older than 7 days (based on `acceptedAt` or `updatedAt`)
- [x] 2.6 Map kagent `completed` state to workbench `ready` in the `onStatus` callback of `ChatDrawer`
- [x] 2.7 Extend `Message` interface with `partial?: boolean` and `toolCall?: { name, args, result, status }` fields

## 3. Streaming Chat Panel

- [x] 3.1 Add streaming text accumulator — partial events append to a live message buffer, finalize on `partial: false`
- [x] 3.2 Render live message bubble with typing cursor (CSS `@keyframes blink`) while `partial: true`
- [x] 3.3 Render tool call blocks for `function_call` DataParts — collapsible, show tool name and spinner while executing
- [x] 3.4 Update tool call blocks on `function_response` — show result summary, auto-collapse
- [x] 3.5 Render approve/reject buttons on tool call blocks with `is_long_running: true` metadata
- [x] 3.6 Wire approve button to send A2A approval message via `sendReply` with HITL decision payload
- [x] 3.7 Wire reject button to send A2A rejection message via `sendReply` with HITL decision payload

## 4. SSE Event Handler Updates

- [x] 4.1 Update `onMessage` in `ChatDrawer` to detect `kagent.adk_partial` metadata on TextParts
- [x] 4.2 Route partial text to the streaming accumulator instead of creating new messages
- [x] 4.3 Detect `DataPart` with `kagent.type = function_call` and render as tool call message
- [x] 4.4 Detect `DataPart` with `kagent.type = function_response` and update matching tool call
- [x] 4.5 Coalesce finalized partial chunks into a single stored message to prevent localStorage bloat

## 5. Lifecycle Controls

- [x] 5.1 Add "Cancel Job" button to chat panel footer, visible for `submitted`, `working`, `input-required`, `ready`
- [x] 5.2 Wire cancel button to `cancelJob()` which closes SSE and transitions state
- [x] 5.3 Add "Accept" button to chat panel footer, visible only for `ready` status
- [x] 5.4 Create accept dialog component with optional note textarea
- [x] 5.5 Wire accept dialog to `acceptJob()` store function

## 6. Jobs View Split

- [x] 6.1 Split `JobsView` into Active and Recent sections
- [x] 6.2 Active section: list jobs with status `submitted`, `working`, `input-required`, `ready`
- [x] 6.3 Recent section: list jobs with status `accepted`, `cancelled` — show acceptance notes and timestamps
- [x] 6.4 Add delete button to history job cards
- [x] 6.5 Hide Recent section when no history jobs exist
- [x] 6.6 Show "No active jobs" empty state with New Job button when no active jobs exist

## 7. History Auto-Purge

- [x] 7.1 Call `purgeHistory()` on app mount in `app.tsx`
- [x] 7.2 Set up 60-minute interval to call `purgeHistory()` in `app.tsx`
- [x] 7.3 Clean up interval on app unmount

## 8. CSS and Styling

- [x] 8.1 Add styles for streaming live message bubble with blinking cursor
- [x] 8.2 Add styles for tool call blocks (collapsible, executing spinner, completed state)
- [x] 8.3 Add styles for HITL approval blocks (distinct background, "Waiting for approval" label)
- [x] 8.4 Add styles for lifecycle controls footer in chat panel
- [x] 8.5 Add styles for accept dialog
- [x] 8.6 Add styles for Active/Recent sections in jobs view with history card variant
