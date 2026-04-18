// SPDX-License-Identifier: Apache-2.0

import { useState } from "preact/hooks";
import { editorContent, editorFilename, editorDefinition, setEditorArtifact } from "../store/editor";
import { getActiveMission, getMissionAgent, type Artifact } from "../store/missions";
import { validate } from "../api/a2a";
import { saveToWorkspace } from "../api/workspace";
import { detectDefinition, ALL_DEFINITIONS } from "../lib/artifact-detect";
import { YamlEditor } from "./yaml-editor";
import { PublishDialog } from "./publish-dialog";
import { ImportDialog } from "./import-dialog";
import { ChatDrawer } from "./chat-drawer";

export function WorkspaceView() {
  const content = editorContent.value;
  const filename = editorFilename.value;
  const [validationResult, setValidationResult] = useState<{ valid: boolean; message: string } | null>(null);
  const [saveStatus, setSaveStatus] = useState<{ ok: boolean; message: string } | null>(null);
  const [showPublish, setShowPublish] = useState(false);
  const [showImport, setShowImport] = useState(false);
  const [chatOpen, setChatOpen] = useState(true);
  const [copyFeedback, setCopyFeedback] = useState(false);

  const activeMission = getActiveMission();

  function handleEditorChange(val: string) {
    editorContent.value = val;
    editorDefinition.value = detectDefinition(val);
  }

  const definition = editorDefinition.value;

  async function handleValidate() {
    if (!content.trim()) return;
    setValidationResult({ valid: true, message: "Validating..." });
    try {
      const result = await validate(content, definition);
      if (result.valid) setValidationResult({ valid: true, message: `Valid ${definition}` });
      else setValidationResult({ valid: false, message: `Invalid: ${result.errors?.join(", ") || "unknown error"}` });
    } catch (e: unknown) { setValidationResult({ valid: false, message: `Error: ${(e as Error).message}` }); }
  }

  async function handleSave() {
    if (!content.trim()) return;
    setSaveStatus(null);
    try {
      const result = await saveToWorkspace(filename, content);
      setSaveStatus({ ok: true, message: `Saved to ${result.path}` });
    } catch (e: unknown) { setSaveStatus({ ok: false, message: (e as Error).message }); }
  }

  async function handleCopy() {
    try { await navigator.clipboard.writeText(content); }
    catch { const ta = document.createElement("textarea"); ta.value = content; document.body.appendChild(ta); ta.select(); document.execCommand("copy"); ta.remove(); }
    setCopyFeedback(true);
    setTimeout(() => setCopyFeedback(false), 1500);
  }

  const publishArtifacts: Artifact[] = content.trim()
    ? [{ name: filename, yaml: content, definition: editorDefinition.value }]
    : [];

  return (
    <div class="workspace-layout">
      <div class={`workspace-editor-area ${activeMission && chatOpen ? "with-drawer" : ""}`}>
        <div class="artifact-toolbar">
          <select
            class="definition-select"
            value={definition}
            onChange={(e) => { editorDefinition.value = (e.target as HTMLSelectElement).value; }}
          >
            {ALL_DEFINITIONS.map((d) => <option key={d} value={d}>{d.replace("#", "")}</option>)}
          </select>
          <button class="btn btn-primary btn-sm" onClick={handleValidate} disabled={!content.trim()}>Validate</button>
          <button class="btn btn-secondary btn-sm" onClick={handleCopy} disabled={!content.trim()}>Copy</button>
          {copyFeedback && <span class="copy-toast">Copied!</span>}
          <button class="btn btn-secondary btn-sm" onClick={handleSave} disabled={!content.trim()}>Save</button>
          <button class="btn btn-secondary btn-sm" onClick={() => setShowImport(true)}>Import</button>
          <button class="btn btn-accent btn-sm" onClick={() => setShowPublish(true)} disabled={!content.trim()}>Publish</button>
          {activeMission && !chatOpen && (
            <button class="btn btn-secondary btn-sm workspace-chat-toggle" onClick={() => setChatOpen(true)}>Chat</button>
          )}
          <span class="workspace-filename">{filename}</span>
        </div>
        <YamlEditor content={content} onChange={handleEditorChange} />
        {validationResult && (
          <div class={`validation-result ${validationResult.valid ? "valid" : "invalid"}`}>
            {validationResult.valid ? "\u2713" : "\u2717"} {validationResult.message}
          </div>
        )}
        {saveStatus && (
          <div class={`validation-result ${saveStatus.ok ? "valid" : "invalid"}`}>
            {saveStatus.ok ? "\u2713" : "\u2717"} {saveStatus.message}
          </div>
        )}
      </div>
      {activeMission && chatOpen && (
        <ChatDrawer mission={activeMission} onClose={() => setChatOpen(false)} />
      )}
      {showPublish && publishArtifacts.length > 0 && (
        <PublishDialog artifacts={publishArtifacts} onClose={() => setShowPublish(false)} />
      )}
      {showImport && <ImportDialog onClose={() => setShowImport(false)} />}
    </div>
  );
}
