// SPDX-License-Identifier: Apache-2.0

import { useState, useRef, useEffect } from "preact/hooks";
import { editorContent, editorFilename, editorDefinition, pendingProposal, applyProposal, dismissProposal, setEditorContent, setEditorDefinition } from "../store/editor";
import { allArtifacts, capacityWarning } from "../store/workspace";
import { getJob, type Artifact } from "../store/jobs";
import { currentJobId } from "../app";
import { validate } from "../api/a2a";
import { detectDefinition, ALL_DEFINITIONS } from "../lib/artifact-detect";
import { YamlEditor } from "./yaml-editor";
import { ArtifactTabs } from "./artifact-tabs";
import { PublishDialog } from "./publish-dialog";
import { ImportDialog } from "./import-dialog";
import { ChatDrawer } from "./chat-drawer";

function downloadYaml(filename: string, content: string) {
  const blob = new Blob([content], { type: "application/x-yaml" });
  const url = URL.createObjectURL(blob);
  const a = document.createElement("a");
  a.href = url;
  a.download = filename;
  a.click();
  URL.revokeObjectURL(url);
}

export function WorkspaceView() {
  const content = editorContent.value;
  const filename = editorFilename.value;
  const [validationResult, setValidationResult] = useState<{ valid: boolean; message: string } | null>(null);
  const [showPublish, setShowPublish] = useState(false);
  const [showImport, setShowImport] = useState(false);
  const [chatOpen, setChatOpen] = useState(true);
  const [copyFeedback, setCopyFeedback] = useState(false);
  const [menuOpen, setMenuOpen] = useState(false);
  const menuRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!menuOpen) return;
    function onClickOutside(e: MouseEvent) {
      if (menuRef.current && !menuRef.current.contains(e.target as Node)) setMenuOpen(false);
    }
    document.addEventListener("mousedown", onClickOutside);
    return () => document.removeEventListener("mousedown", onClickOutside);
  }, [menuOpen]);

  const jobId = currentJobId.value;
  const currentJob = jobId ? getJob(jobId) : null;
  const wsArtifacts = allArtifacts.value;

  function handleEditorChange(val: string) {
    setEditorContent(val);
    const detected = detectDefinition(val);
    if (detected) setEditorDefinition(detected);
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

  async function handleCopy() {
    try { await navigator.clipboard.writeText(content); }
    catch { const ta = document.createElement("textarea"); ta.value = content; document.body.appendChild(ta); ta.select(); document.execCommand("copy"); ta.remove(); }
    setCopyFeedback(true);
    setTimeout(() => setCopyFeedback(false), 1500);
  }

  const publishArtifacts: Artifact[] = wsArtifacts.length > 0
    ? wsArtifacts.map((a) => ({ name: a.name, yaml: a.yaml, definition: a.definition }))
    : content.trim()
      ? [{ name: filename, yaml: content, definition }]
      : [];

  return (
    <div class="workspace-layout">
      <div class={`workspace-editor-area ${currentJob && chatOpen ? "with-drawer" : ""}`}>
        <ArtifactTabs />
        <div class="artifact-toolbar">
          <select
            class="definition-select"
            value={definition}
            onChange={(e) => { setEditorDefinition((e.target as HTMLSelectElement).value); }}
          >
            {ALL_DEFINITIONS.map((d) => <option key={d} value={d}>{d.replace("#", "")}</option>)}
          </select>
          <button class="btn btn-primary btn-sm" onClick={handleValidate} disabled={!content.trim()}>Validate</button>
          <button class="btn btn-accent btn-sm" onClick={() => setShowPublish(true)} disabled={publishArtifacts.length === 0}>Publish</button>
          {currentJob && !chatOpen && (
            <button class="btn btn-secondary btn-sm workspace-chat-toggle" onClick={() => setChatOpen(true)}>Chat</button>
          )}
          {copyFeedback && <span class="copy-toast">Copied!</span>}
          <div class="toolbar-overflow" ref={menuRef}>
            <button class="btn btn-secondary btn-sm toolbar-overflow-trigger" onClick={() => setMenuOpen(!menuOpen)} title="More actions">
              <svg width="14" height="14" viewBox="0 0 16 16" fill="currentColor"><circle cx="3" cy="8" r="1.5"/><circle cx="8" cy="8" r="1.5"/><circle cx="13" cy="8" r="1.5"/></svg>
            </button>
            {menuOpen && (
              <div class="toolbar-overflow-menu">
                <button class="toolbar-overflow-item" disabled={!content.trim()} onClick={() => { handleCopy(); setMenuOpen(false); }}>Copy YAML</button>
                <button class="toolbar-overflow-item" disabled={!content.trim()} onClick={() => { downloadYaml(filename, content); setMenuOpen(false); }}>Download YAML</button>
                {wsArtifacts.length > 1 && (
                  <button class="toolbar-overflow-item" onClick={() => { wsArtifacts.forEach((a) => downloadYaml(a.name, a.yaml)); setMenuOpen(false); }}>Download All</button>
                )}
                <button class="toolbar-overflow-item" onClick={() => { setShowImport(true); setMenuOpen(false); }}>Import</button>
              </div>
            )}
          </div>
          <span class="workspace-filename">{filename}</span>
        </div>
        {capacityWarning.value && (
          <div class="validation-result invalid">{capacityWarning.value}</div>
        )}
        {pendingProposal.value && (
          <div class="proposal-banner">
            <span class="proposal-banner-text">Agent suggests: <strong>{pendingProposal.value.name}</strong></span>
            <div class="proposal-banner-actions">
              <button class="btn btn-primary btn-sm" onClick={applyProposal}>Apply</button>
              <button class="btn btn-secondary btn-sm" onClick={dismissProposal}>Dismiss</button>
            </div>
          </div>
        )}
        <YamlEditor content={content} onChange={handleEditorChange} />
        {validationResult && (
          <div class={`validation-result ${validationResult.valid ? "valid" : "invalid"}`}>
            {validationResult.valid ? "\u2713" : "\u2717"} {validationResult.message}
          </div>
        )}
      </div>
      {currentJob && chatOpen && (
        <ChatDrawer job={currentJob} onClose={() => setChatOpen(false)} />
      )}
      {showPublish && publishArtifacts.length > 0 && (
        <PublishDialog artifacts={publishArtifacts} onClose={() => setShowPublish(false)} />
      )}
      {showImport && <ImportDialog onClose={() => setShowImport(false)} />}
    </div>
  );
}
