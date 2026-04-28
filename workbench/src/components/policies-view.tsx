// SPDX-License-Identifier: Apache-2.0

import { useState, useEffect } from "preact/hooks";
import { apiFetch } from "../api/fetch";
import { currentUser } from "../app";
import { fmtDate } from "../lib/format";

interface Policy {
  policy_id: string;
  title: string;
  version: string;
  oci_reference: string;
  imported_at: string;
}

interface PolicyDetail {
  policy: Policy & { content: string };
  mappings: { mapping_id: string; framework: string; content: string; imported_at: string }[];
}

export function PoliciesView() {
  const [policies, setPolicies] = useState<Policy[]>([]);
  const [selected, setSelected] = useState<PolicyDetail | null>(null);
  const [loading, setLoading] = useState(true);
  const [importing, setImporting] = useState(false);
  const [importRef, setImportRef] = useState("");
  const [importError, setImportError] = useState("");
  const isAdmin = currentUser.value?.role === "admin";

  const loadPolicies = () => {
    apiFetch("/api/policies")
      .then((r) => r.json())
      .then(setPolicies)
      .catch(() => setPolicies([]))
      .finally(() => setLoading(false));
  };

  useEffect(loadPolicies, []);

  const handleImport = async () => {
    if (!importRef.trim()) return;
    setImporting(true);
    setImportError("");
    try {
      const res = await apiFetch("/api/policies/import", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          oci_reference: importRef,
          title: importRef.split("/").pop()?.split(":")[0] || "Imported Policy",
          content: "---\n# Placeholder — OCI pull not yet wired\n",
        }),
      });
      if (!res.ok) {
        const text = await res.text();
        setImportError(text);
      } else {
        setImportRef("");
        loadPolicies();
      }
    } catch (e) {
      setImportError(String(e));
    } finally {
      setImporting(false);
    }
  };

  const selectPolicy = async (id: string) => {
    const res = await apiFetch(`/api/policies/${id}`);
    if (res.ok) setSelected(await res.json());
  };

  if (loading) return <div class="view-loading">Loading policies...</div>;

  return (
    <div class="policies-view">
      <div class="policies-header">
        <h2>Policies</h2>
        {isAdmin && (
          <div class="import-bar">
            <input
              type="text"
              placeholder="ghcr.io/org/policy-bundle:v1.0"
              value={importRef}
              onInput={(e) => setImportRef((e.target as HTMLInputElement).value)}
              class="import-input"
            />
            <button class="btn btn-primary" onClick={handleImport} disabled={importing}>
              {importing ? "Importing..." : "Import"}
            </button>
          </div>
        )}
        {importError && <p class="import-error">{importError}</p>}
      </div>

      {policies.length === 0 ? (
        <div class="empty-state">
          <p>No policies imported. Use the input above to import from an OCI registry.</p>
        </div>
      ) : (
        <table class="data-table">
          <thead>
            <tr>
              <th>Title</th>
              <th>Version</th>
              <th>Imported</th>
              <th>OCI Reference</th>
            </tr>
          </thead>
          <tbody>
            {policies.map((p) => (
              <tr key={p.policy_id} onClick={() => selectPolicy(p.policy_id)} class="clickable-row">
                <td>{p.title}</td>
                <td>{p.version || "—"}</td>
                <td>{fmtDate(p.imported_at)}</td>
                <td class="mono">{p.oci_reference}</td>
              </tr>
            ))}
          </tbody>
        </table>
      )}

      {selected && (
        <div class="policy-detail">
          <div class="detail-header">
            <h3>{selected.policy.title}</h3>
            <button class="btn btn-sm" onClick={() => setSelected(null)}>Close</button>
          </div>
          <div class="detail-tabs">
            <div class="detail-section">
              <h4>YAML Content</h4>
              <pre class="yaml-viewer">{selected.policy.content}</pre>
            </div>
            <div class="detail-section">
              <h4>Mapping Documents ({selected.mappings.length})</h4>
              {selected.mappings.length === 0 ? (
                <p>No mapping documents linked.</p>
              ) : (
                selected.mappings.map((m) => (
                  <details key={m.mapping_id} class="mapping-detail">
                    <summary>{m.framework} — imported {fmtDate(m.imported_at)}</summary>
                    <pre class="yaml-viewer">{m.content}</pre>
                  </details>
                ))
              )}
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
