// SPDX-License-Identifier: Apache-2.0

import { useState, useEffect, useMemo } from "preact/hooks";
import { apiFetch } from "../api/fetch";
import {
  currentUser,
  navigateToPolicy,
  selectedProgramFilter,
  updateHash,
  viewInvalidation,
  invalidateViews,
} from "../app";
import { createFilterChips } from "./filter-chip";
import { AddFilterMenu } from "./add-filter-menu";
import { ImportOverlay } from "./import-overlay";
import { fmtDate } from "../lib/format";

interface Policy {
  policy_id: string;
  title: string;
  version: string;
  oci_reference: string;
  imported_at: string;
}

interface ProgramListItem {
  id: string;
  name: string;
}

interface ProgramDetailResponse {
  id: string;
  name: string;
  policy_ids: string[];
}

export function PoliciesView() {
  void viewInvalidation.value;
  const [policies, setPolicies] = useState<Policy[]>([]);
  const [programRows, setProgramRows] = useState<ProgramListItem[]>([]);
  const [programPolicyIds, setProgramPolicyIds] = useState<Set<string> | null>(
    null,
  );
  const [chipState] = useState(() => createFilterChips());
  const [loading, setLoading] = useState(true);
  const [importOpen, setImportOpen] = useState(false);
  const canWrite =
    currentUser.value?.role === "admin" || currentUser.value?.role === "writer";

  const loadPolicies = () => {
    apiFetch("/api/policies")
      .then((r) => r.json())
      .then(setPolicies)
      .catch(() => setPolicies([]))
      .finally(() => setLoading(false));
  };

  useEffect(() => {
    loadPolicies();
  }, [viewInvalidation.value]);

  useEffect(() => {
    apiFetch("/api/programs")
      .then((r) => r.json())
      .then((data: ProgramListItem[]) =>
        setProgramRows(Array.isArray(data) ? data : []),
      )
      .catch(() => setProgramRows([]));
  }, [viewInvalidation.value]);

  useEffect(() => {
    const pid = selectedProgramFilter.value;
    if (!pid) {
      setProgramPolicyIds(null);
      return;
    }
    let cancelled = false;
    apiFetch(`/api/programs/${encodeURIComponent(pid)}`)
      .then((r) => (r.ok ? r.json() : null))
      .then((d: ProgramDetailResponse | null) => {
        if (cancelled || !d?.policy_ids) return;
        setProgramPolicyIds(new Set(d.policy_ids));
      })
      .catch(() => {
        if (!cancelled) setProgramPolicyIds(new Set());
      });
    return () => {
      cancelled = true;
    };
  }, [selectedProgramFilter.value, viewInvalidation.value]);

  const clearProgramFilter = () => {
    selectedProgramFilter.value = null;
    updateHash();
  };

  const programFilterFields = selectedProgramFilter.value
    ? []
    : [
        {
          key: "Program",
          label: "Program",
          options: () =>
            [...programRows]
              .sort((a, b) => a.name.localeCompare(b.name))
              .map((p) => ({
                value: p.id,
                label: p.name || p.id,
              })),
          pick: (id: string) => {
            selectedProgramFilter.value = id;
            updateHash();
          },
        },
      ];

  const visiblePolicies = useMemo(() => {
    if (!programPolicyIds) return policies;
    return policies.filter((p) => programPolicyIds.has(p.policy_id));
  }, [policies, programPolicyIds]);

  if (loading) return <div class="view-loading">Loading policies...</div>;

  return (
    <div class="policies-view">
      <ImportOverlay
        open={importOpen}
        onClose={() => setImportOpen(false)}
        onSuccess={invalidateViews}
      />
      <div class="policies-header">
        <h2>Policies</h2>
        <div class="policies-filter-row" style={{ display: "flex", flexWrap: "wrap", gap: "8px", alignItems: "center" }}>
          <AddFilterMenu fields={programFilterFields} chipState={chipState} />
        </div>
        {selectedProgramFilter.value && (
          <div class="filter-chips" style={{ marginTop: "8px" }}>
            <span class="filter-chip">
              <span class="filter-chip-label">
                Program:{" "}
                {programRows.find((p) => p.id === selectedProgramFilter.value)
                  ?.name || selectedProgramFilter.value}
              </span>
              <button
                type="button"
                class="filter-chip-dismiss"
                aria-label="Remove Program filter"
                onClick={clearProgramFilter}
              >
                &times;
              </button>
            </span>
          </div>
        )}
        {canWrite && (
          <div class="import-bar">
            <button type="button" class="btn btn-primary" onClick={() => setImportOpen(true)}>
              Import artifact
            </button>
          </div>
        )}
      </div>

      {policies.length === 0 ? (
        <div class="empty-state">
          <p>No policies imported. Use Import artifact to upload a Gemara policy YAML or JSON.</p>
        </div>
      ) : visiblePolicies.length === 0 ? (
        <div class="empty-state">
          <p>No policies match this program filter.</p>
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
            {visiblePolicies.map((p) => (
              <tr key={p.policy_id} onClick={() => navigateToPolicy(p.policy_id)} class="clickable-row">
                <td>{p.title}</td>
                <td>{p.version || "—"}</td>
                <td>{fmtDate(p.imported_at)}</td>
                <td class="mono">{p.oci_reference}</td>
              </tr>
            ))}
          </tbody>
        </table>
      )}

    </div>
  );
}
