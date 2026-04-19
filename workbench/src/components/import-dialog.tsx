// SPDX-License-Identifier: Apache-2.0

import { useState } from "preact/hooks";
import { listRepositories, listTags, fetchManifest, fetchLayer, type Repository, type Tag, type Manifest } from "../api/registry";
import { isGemaraArtifact } from "../lib/artifact-detect";
import { extractMappingRefFromYaml } from "../lib/yaml-inject";
import { injectMappingReference } from "../store/editor";

type RegistryState =
  | { phase: "input" }
  | { phase: "repos"; registry: string; repos: Repository[] }
  | { phase: "tags"; registry: string; repo: string; tags: Tag[] }
  | { phase: "manifest"; registry: string; repo: string; tag: string; manifest: Manifest }
  | { phase: "layer"; registry: string; repo: string; tag: string; content: string; mediaType: string };

interface ImportDialogProps {
  onClose: () => void;
}

export function ImportDialog({ onClose }: ImportDialogProps) {
  const [state, setState] = useState<RegistryState>({ phase: "input" });
  const [registryUrl, setRegistryUrl] = useState("");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");
  const [feedback, setFeedback] = useState<{ ok: boolean; message: string } | null>(null);

  async function handleBrowse() {
    if (!registryUrl.trim()) return;
    setLoading(true); setError("");
    try {
      const repos = await listRepositories(registryUrl.trim());
      setState({ phase: "repos", registry: registryUrl.trim(), repos });
    } catch (e: unknown) { setError((e as Error).message); }
    finally { setLoading(false); }
  }

  async function handleSelectRepo(repo: string) {
    if (state.phase !== "repos") return;
    setLoading(true); setError("");
    try {
      const tags = await listTags(`${state.registry}/${repo}`);
      setState({ phase: "tags", registry: state.registry, repo, tags });
    } catch (e: unknown) { setError((e as Error).message); }
    finally { setLoading(false); }
  }

  async function handleSelectTag(tag: string) {
    if (state.phase !== "tags") return;
    setLoading(true); setError("");
    try {
      const manifest = await fetchManifest(`${state.registry}/${state.repo}:${tag}`);
      setState({ phase: "manifest", registry: state.registry, repo: state.repo, tag, manifest });
    } catch (e: unknown) { setError((e as Error).message); }
    finally { setLoading(false); }
  }

  async function handleInspectLayer(digest: string, mediaType: string) {
    if (state.phase !== "manifest") return;
    setLoading(true); setError("");
    try {
      const content = await fetchLayer(`${state.registry}/${state.repo}@${digest}`);
      setState({ phase: "layer", registry: state.registry, repo: state.repo, tag: state.tag, content, mediaType });
    } catch (e: unknown) { setError((e as Error).message); }
    finally { setLoading(false); }
  }

  function handleImportReference() {
    if (state.phase !== "layer") return;
    const ref = extractMappingRefFromYaml(state.content);
    if (!ref) { setFeedback({ ok: false, message: "Could not extract metadata from artifact" }); return; }
    ref.url = `${state.registry}/${state.repo}:${state.tag}`;
    injectMappingReference(ref);
    onClose();
  }

  function goBack() {
    if (state.phase === "layer" && "registry" in state) {
      setState({ phase: "input" });
    } else {
      setState({ phase: "input" });
    }
    setFeedback(null); setError("");
  }

  const layerIsGemara = state.phase === "layer" && isGemaraArtifact(state.content);

  return (
    <div class="dialog-overlay" onClick={onClose}>
      <div class="dialog dialog-wide import-dialog" onClick={(e) => e.stopPropagation()}>
        <div class="registry-header">
          <h3>Import from Registry</h3>
          {state.phase !== "input" && <button class="btn btn-secondary btn-sm" onClick={goBack}>&larr; Back</button>}
        </div>
        {error && <div class="registry-error">{error}</div>}

        {state.phase === "input" && (
          <div class="registry-input-section">
            <p class="registry-hint">Enter an OCI registry URL to browse for artifacts to import.</p>
            <div class="registry-input-row">
              <input type="text" class="dialog-input" placeholder="ghcr.io/jpower432" value={registryUrl}
                onInput={(e) => setRegistryUrl((e.target as HTMLInputElement).value)}
                onKeyDown={(e) => e.key === "Enter" && handleBrowse()} />
              <button class="btn btn-primary" onClick={handleBrowse} disabled={loading}>
                {loading ? "Loading..." : "Browse"}
              </button>
            </div>
          </div>
        )}

        {state.phase === "repos" && (
          <div class="registry-list">
            <h4>Repositories in {state.registry}</h4>
            {state.repos.length === 0
              ? <p class="text-muted">No repositories found.</p>
              : state.repos.map((r) => (
                <div key={r.name} class="registry-list-item" onClick={() => handleSelectRepo(r.name)}>
                  <span class="registry-item-icon">&#128230;</span>{r.name}
                </div>
              ))}
          </div>
        )}

        {state.phase === "tags" && (
          <div class="registry-list">
            <h4>{state.registry}/{state.repo}</h4>
            {state.tags.length === 0
              ? <p class="text-muted">No tags found.</p>
              : state.tags.map((t) => (
                <div key={t.name} class="registry-list-item" onClick={() => handleSelectTag(t.name)}>
                  <span class="registry-item-icon">&#127991;</span>{t.name}
                  {t.digest && <span class="text-muted registry-digest">{t.digest.slice(0, 19)}</span>}
                </div>
              ))}
          </div>
        )}

        {state.phase === "manifest" && (
          <div class="registry-manifest">
            <h4>{state.repo}:{state.tag}</h4>
            <div class="manifest-section">
              <h4>Layers</h4>
              {state.manifest.layers.map((layer) => (
                <div key={layer.digest} class="manifest-layer">
                  <div class="manifest-layer-header">
                    <span class="manifest-media-type">{layer.mediaType}</span>
                    <span class="text-muted">{(layer.size / 1024).toFixed(1)} KB</span>
                  </div>
                  <div class="manifest-layer-actions">
                    <button class="btn btn-secondary btn-sm" onClick={() => handleInspectLayer(layer.digest, layer.mediaType)} disabled={loading}>
                      Inspect
                    </button>
                  </div>
                </div>
              ))}
            </div>
          </div>
        )}

        {state.phase === "layer" && (
          <div class="registry-layer-view">
            <div class="registry-layer-toolbar">
              <span class="manifest-media-type">{state.mediaType}</span>
              {layerIsGemara && (
                <button class="btn btn-primary btn-sm" onClick={handleImportReference}>Import Reference</button>
              )}
            </div>
            {feedback && (
              <div class={`validation-result ${feedback.ok ? "valid" : "invalid"}`}>
                {feedback.ok ? "\u2713" : "\u2717"} {feedback.message}
              </div>
            )}
            <pre class="registry-layer-content">{state.content}</pre>
          </div>
        )}

        <div class="dialog-actions">
          <button class="btn btn-secondary" onClick={onClose}>Close</button>
        </div>
      </div>
    </div>
  );
}
