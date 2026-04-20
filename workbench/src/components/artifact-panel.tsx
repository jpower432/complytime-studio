// SPDX-License-Identifier: Apache-2.0
import { useState, useRef } from "preact/hooks";
import type { Artifact } from "../store/jobs";
import { validate } from "../api/a2a";
import { detectDefinition } from "../lib/artifact-detect";
import { downloadYaml } from "../lib/download";
import { YamlEditor } from "./yaml-editor";
import { PublishDialog } from "./publish-dialog";

interface ArtifactPanelProps { artifacts: Artifact[]; jobId: string }

export function ArtifactPanel({ artifacts, jobId }: ArtifactPanelProps) {
  const [activeTab, setActiveTab] = useState(0);
  const [validationResult, setValidationResult] = useState<{ valid: boolean; message: string } | null>(null);
  const [showPublish, setShowPublish] = useState(false);
  const editorContentRef = useRef(artifacts[activeTab]?.yaml ?? "");
  if (artifacts.length === 0) return (<div class="artifact-panel"><div class="artifact-empty">Artifacts will appear here as the agent produces them.</div></div>);
  function getCurrentContent(): string { return editorContentRef.current || artifacts[activeTab]?.yaml || ""; }
  async function handleValidate() {
    const content = getCurrentContent(); const definition = detectDefinition(content) || "#ThreatCatalog";
    setValidationResult({ valid: true, message: "Validating..." });
    try {
      const result = await validate(content, definition);
      if (result.valid) setValidationResult({ valid: true, message: `Valid ${definition}` });
      else setValidationResult({ valid: false, message: `Invalid ${definition}: ${result.errors?.join(", ") || "unknown error"}` });
    } catch (e: unknown) { setValidationResult({ valid: false, message: `Error: ${(e as Error).message}` }); }
  }
  async function handleCopy() { const content = getCurrentContent(); try { await navigator.clipboard.writeText(content); } catch { const ta = document.createElement("textarea"); ta.value = content; document.body.appendChild(ta); ta.select(); document.execCommand("copy"); ta.remove(); } }
  function handleDownload() {
    const content = getCurrentContent();
    const name = artifacts[activeTab]?.name || "artifact.yaml";
    downloadYaml(name, content);
  }
  function switchTab(index: number) { setActiveTab(index); editorContentRef.current = artifacts[index]?.yaml ?? ""; setValidationResult(null); }
  return (
    <div class="artifact-panel">
      <div class="artifact-tabs">{artifacts.map((a, i) => (<button key={a.name} class={`artifact-tab ${i === activeTab ? "active" : ""}`} onClick={() => switchTab(i)}>{a.name}</button>))}</div>
      <div class="artifact-toolbar">
        <button class="btn btn-primary btn-sm" onClick={handleValidate}>Validate</button>
        <button class="btn btn-secondary btn-sm" onClick={handleCopy}>Copy YAML</button>
        <button class="btn btn-secondary btn-sm" onClick={handleDownload}>Download YAML</button>
        <button class="btn btn-accent btn-sm" onClick={() => setShowPublish(true)}>Publish</button>
      </div>
      <YamlEditor content={artifacts[activeTab]?.yaml ?? ""} onChange={(val) => { editorContentRef.current = val; }} />
      {validationResult && (<div class={`validation-result ${validationResult.valid ? "valid" : "invalid"}`}>{validationResult.valid ? "\u2713" : "\u2717"} {validationResult.message}</div>)}
      {showPublish && <PublishDialog artifacts={artifacts} onClose={() => setShowPublish(false)} />}
    </div>
  );
}
