// SPDX-License-Identifier: Apache-2.0
import { useState } from "preact/hooks";
import type { Artifact } from "../store/missions";
import { publishBundle } from "../api/a2a";

interface PublishDialogProps { artifacts: Artifact[]; onClose: () => void }

export function PublishDialog({ artifacts, onClose }: PublishDialogProps) {
  const [target, setTarget] = useState("");
  const [tag, setTag] = useState("");
  const [sign, setSign] = useState(false);
  const [selected, setSelected] = useState<boolean[]>(artifacts.map(() => true));
  const [status, setStatus] = useState<"idle" | "publishing" | "success" | "error">("idle");
  const [result, setResult] = useState<string>("");
  function toggleArtifact(index: number) { setSelected((prev) => prev.map((v, i) => (i === index ? !v : v))); }
  async function handlePublish() {
    if (!target.trim()) { setResult("Target registry reference is required."); setStatus("error"); return; }
    const selectedArtifacts = artifacts.filter((_, i) => selected[i]).map((a) => a.yaml);
    if (selectedArtifacts.length === 0) { setResult("Select at least one artifact."); setStatus("error"); return; }
    setStatus("publishing"); setResult("");
    try {
      const res = await publishBundle({ artifacts: selectedArtifacts, target: target.trim(), tag: tag.trim() || undefined, sign });
      setStatus("success"); setResult(`Published to ${res.reference}\nDigest: ${res.digest}\nTag: ${res.tag}`);
    } catch (e: unknown) { setStatus("error"); setResult((e as Error).message); }
  }
  return (
    <div class="dialog-overlay" onClick={onClose}>
      <div class="dialog dialog-wide" onClick={(e) => e.stopPropagation()}>
        <h3>Publish OCI Bundle</h3>
        <label class="dialog-label">Target Registry Reference<input type="text" class="dialog-input" placeholder="ghcr.io/org/repo" value={target} onInput={(e) => setTarget((e.target as HTMLInputElement).value)} /></label>
        <label class="dialog-label">Tag (optional)<input type="text" class="dialog-input" placeholder="v1.0.0 (default: latest)" value={tag} onInput={(e) => setTag((e.target as HTMLInputElement).value)} /></label>
        <div class="dialog-label">Artifacts to include</div>
        <div class="publish-artifacts">{artifacts.map((a, i) => (<label key={a.name} class="publish-artifact-row"><input type="checkbox" checked={selected[i]} onChange={() => toggleArtifact(i)} /><span class="artifact-name-mono">{a.name}</span></label>))}</div>
        <label class="publish-sign-row"><input type="checkbox" checked={sign} onChange={() => setSign(!sign)} />Sign bundle after push</label>
        {result && (<div class={`publish-result ${status === "success" ? "valid" : "invalid"}`}><pre>{result}</pre></div>)}
        <div class="dialog-actions">
          <button class="btn btn-secondary" onClick={onClose}>{status === "success" ? "Done" : "Cancel"}</button>
          {status !== "success" && (<button class="btn btn-primary" disabled={status === "publishing"} onClick={handlePublish}>{status === "publishing" ? "Publishing..." : "Publish"}</button>)}
        </div>
      </div>
    </div>
  );
}
