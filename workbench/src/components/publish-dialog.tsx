// SPDX-License-Identifier: Apache-2.0
import { useState, useEffect } from "preact/hooks";
import type { Artifact } from "../store/jobs";
import { publishBundle } from "../api/a2a";
import { apiFetch } from "../api/fetch";

type RegistryOption = { label: string; prefix: string; needsRepo: boolean };

interface PublishDialogProps { artifacts: Artifact[]; onClose: () => void }

export function PublishDialog({ artifacts, onClose }: PublishDialogProps) {
  const [registries, setRegistries] = useState<RegistryOption[]>([]);
  const [registryIdx, setRegistryIdx] = useState(0);
  const [repoPath, setRepoPath] = useState("");
  const [tag, setTag] = useState("");
  const [selected, setSelected] = useState<boolean[]>(artifacts.map(() => true));
  const [status, setStatus] = useState<"idle" | "publishing" | "success" | "error">("idle");
  const [result, setResult] = useState<string>("");

  useEffect(() => {
    apiFetch("/api/config").then((r) => r.json()).then((cfg: Record<string, string>) => {
      const opts: RegistryOption[] = [];
      if (cfg.registry_insecure) {
        opts.push({ label: `In-cluster (${cfg.registry_insecure})`, prefix: cfg.registry_insecure, needsRepo: true });
      }
      opts.push({ label: "GitHub Container Registry (ghcr.io)", prefix: "ghcr.io", needsRepo: true });
      setRegistries(opts);
    }).catch(() => {
      setRegistries([{ label: "GitHub Container Registry (ghcr.io)", prefix: "ghcr.io", needsRepo: true }]);
    });
  }, []);

  function toggleArtifact(index: number) { setSelected((prev) => prev.map((v, i) => (i === index ? !v : v))); }

  const registry = registries[registryIdx];
  const target = registry && repoPath.trim() ? `${registry.prefix}/${repoPath.trim()}` : "";

  async function handlePublish() {
    if (!target) { setResult("Repository path is required."); setStatus("error"); return; }
    const selectedArtifacts = artifacts.filter((_, i) => selected[i]).map((a) => a.yaml);
    if (selectedArtifacts.length === 0) { setResult("Select at least one artifact."); setStatus("error"); return; }
    setStatus("publishing"); setResult("");
    try {
      const res = await publishBundle({ artifacts: selectedArtifacts, target, tag: tag.trim() || undefined });
      setStatus("success"); setResult(`Published to ${res.reference}\nDigest: ${res.digest}\nTag: ${res.tag}`);
    } catch (e: unknown) { setStatus("error"); setResult((e as Error).message); }
  }

  return (
    <div class="dialog-overlay" onClick={onClose}>
      <div class="dialog dialog-wide" onClick={(e) => e.stopPropagation()}>
        <h3>Publish OCI Bundle</h3>

        <label class="dialog-label">Registry
          <select class="dialog-input" value={registryIdx} onChange={(e) => setRegistryIdx(Number((e.target as HTMLSelectElement).value))}>
            {registries.map((r, i) => <option key={i} value={i}>{r.label}</option>)}
          </select>
        </label>

        <label class="dialog-label">
          Repository path
          <div class="publish-target-preview">
            <span class="publish-prefix">{registry?.prefix}/</span>
            <input type="text" class="dialog-input publish-repo-input" placeholder="org/bundle-name" value={repoPath} onInput={(e) => setRepoPath((e.target as HTMLInputElement).value)} />
          </div>
        </label>

        <label class="dialog-label">Tag (optional)<input type="text" class="dialog-input" placeholder="v1.0.0 (default: latest)" value={tag} onInput={(e) => setTag((e.target as HTMLInputElement).value)} /></label>

        <div class="dialog-label">Artifacts to include</div>
        <div class="publish-artifacts">{artifacts.map((a, i) => (<label key={a.name} class="publish-artifact-row"><input type="checkbox" checked={selected[i]} onChange={() => toggleArtifact(i)} /><span class="artifact-name-mono">{a.name}</span></label>))}</div>

        {result && (<div class={`publish-result ${status === "success" ? "valid" : "invalid"}`}><pre>{result}</pre></div>)}
        <div class="dialog-actions">
          <button class="btn btn-secondary" onClick={onClose}>{status === "success" ? "Done" : "Cancel"}</button>
          {status !== "success" && (<button class="btn btn-primary" disabled={status === "publishing" || !target} onClick={handlePublish}>{status === "publishing" ? "Publishing..." : "Publish"}</button>)}
        </div>
      </div>
    </div>
  );
}
