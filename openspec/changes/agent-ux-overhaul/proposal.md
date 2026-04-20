## Why

Agent interactions are unreliable. Five symptoms compound into a broken experience:

1. **Agents forget context** -- follow-up replies (`streamReply`) send only the new message text; conversation history and workspace artifacts are lost after the first exchange.
2. **Agents block on domain jargon** -- prompts demand user decisions on RACI roles, scope dimensions, and enforcement approaches that non-expert users cannot answer.
3. **Chat splits thoughts** -- each streaming finalization boundary creates a new message bubble, fragmenting a single agent response into 3-5 disconnected blocks.
4. **Agents ignore tools** -- prompts reference gemara-mcp "prompts" (e.g., `threat_assessment`) that are not exposed as callable tools; agents hardcode schema shapes instead of calling `validate_gemara_artifact` early to learn the required structure.
5. **Artifacts fail validation** -- without schema guidance in context, agent-produced YAML rarely conforms to Gemara schemas.

Root cause: prompts assume a multi-turn stateful runtime, but kagent's declarative runtime is stateless. Each A2A request starts a fresh LLM context.

## What Changes

- **Context replay on follow-up messages** -- `streamReply` re-sends full conversation history and workspace context artifacts so agents retain state across turns.
- **Conversation history size management** -- enforce a token budget on replayed history; truncate oldest messages first, always preserve system context and the most recent exchange.
- **Single-shot prompt rewrites** -- threat-modeler and gap-analyst prompts rewritten to run the full pipeline in one turn without mid-workflow questions.
- **Reduced-turn prompt rewrite** -- policy-composer prompt restructured into 2 phases (derive-all, confirm-once) with sensible defaults for fields the user doesn't provide.
- **Gemara authoring skill** -- extract schema guidance (currently trapped in gemara-mcp prompts) into a `skills/gemara-authoring/SKILL.md` so agents always have structural knowledge in context.
- **Message grouping in chat** -- consecutive agent messages and tool calls rendered as a single cohesive block instead of separate bubbles.
- **Input sanitization on context replay** -- conversation history assembled client-side must not allow prompt injection via user message content.

## Capabilities

### New Capabilities

- `context-replay`: Client-side conversation history and artifact replay on follow-up A2A messages, with token budget and truncation strategy.
- `gemara-authoring-skill`: Reusable skill containing Gemara schema shapes and authoring guidance, replacing reliance on MCP prompts the agent cannot call.
- `chat-message-grouping`: Consecutive agent messages and interleaved tool calls rendered as a single visual block in the chat panel.

### Modified Capabilities

- `streaming-chat`: Message rendering changes to support grouped blocks; input handling changes for context replay assembly.
- `agent-spec-skills`: Threat-modeler, policy-composer, and gap-analyst prompts rewritten for stateless runtime compatibility.
- `platform-prompt-composition`: Platform prompt updated with context replay expectations and output format enforcement.

## Impact

| Area | Change |
|---|---|
| `workbench/src/api/a2a.ts` | `streamReply` assembles and sends conversation history + context |
| `workbench/src/components/chat-panel.tsx` | Message grouping renderer |
| `workbench/src/components/chat-drawer.tsx` | History assembly, size budget enforcement |
| `agents/threat-modeler/prompt.md` | Single-shot rewrite |
| `agents/gap-analyst/prompt.md` | Single-shot rewrite |
| `agents/policy-composer/prompt.md` | 2-phase reduced-turn rewrite |
| `agents/platform.md` | Context replay expectations, output format |
| `skills/gemara-authoring/SKILL.md` | New skill (schema guidance extracted from MCP prompts) |
| `charts/complytime-studio/` | Helm sync for prompt changes |
| Token usage | Increases proportionally with conversation length due to replay; bounded by budget |
