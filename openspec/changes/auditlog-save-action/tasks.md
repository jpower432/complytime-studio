## 1. YAML Extraction Utility

- [ ] 1.1 Create `workbench/src/lib/yaml-detect.ts` with `extractAuditLogs(text: string): { name: string; content: string }[]` — extracts fenced YAML blocks, parses each, returns those with `metadata.type === "AuditLog"`
- [ ] 1.2 Use `js-yaml` or a lightweight YAML parser already in the project (check dependencies before adding)

## 2. Chat Message Finalization

- [ ] 2.1 In `chat-assistant.tsx`, modify the `finalize` function to call `extractAuditLogs()` on the stream buffer
- [ ] 2.2 For each detected AuditLog, append an artifact card message (reuse existing `ChatMessage.artifact` shape) with `name` from `metadata.id`, `content` as the raw YAML, `mimeType: "application/yaml"`
- [ ] 2.3 Strip the extracted YAML blocks from the text message so they don't render as both text and artifact cards

## 3. Agent Callback Cleanup

- [ ] 3.1 In `agents/assistant/callbacks.py`, remove the AuditLog detection and `save_artifact` logic from `after_agent` — keep only the empty return (or remove the callback if no other logic remains)
- [ ] 3.2 Remove `_extract_yaml_blocks()` helper function from `callbacks.py`
- [ ] 3.3 Remove unused `yaml` import from `callbacks.py`

## 4. Verification

- [ ] 4.1 Build workbench (`cd workbench && npm run build`) and verify no TypeScript errors
- [ ] 4.2 Deploy and verify the save button appears on AuditLog YAML in chat
- [ ] 4.3 Click "Save to Audit History" and verify the AuditLog appears in the Audit History view
