// SPDX-License-Identifier: Apache-2.0

import { useState, useEffect, useMemo } from "preact/hooks";
import { navigateToProgram, currentUser, viewInvalidation, invalidateViews } from "../app";
import { apiFetch } from "../api/fetch";
import { cardKeyHandler } from "../lib/a11y";
import { ImportOverlay } from "./import-overlay";

interface Program {
  id: string;
  name: string;
  framework: string;
  status: string;
  health?: string | null;
  policy_ids: string[];
  environments: string[];
}

interface CatalogRow {
  catalog_id: string;
  catalog_type: string;
  title: string;
}

type StatusFilter = "all" | "intake" | "active" | "monitoring" | "renewal" | "closed";

export function ProgramsView() {
  void viewInvalidation.value;
  const [programs, setPrograms] = useState<Program[]>([]);
  const [catalogs, setCatalogs] = useState<CatalogRow[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [statusFilter, setStatusFilter] = useState<StatusFilter>("all");
  const [showCreate, setShowCreate] = useState(false);
  const [formName, setFormName] = useState("");
  const [formFramework, setFormFramework] = useState("");
  const [formCatalogId, setFormCatalogId] = useState("");
  const [formApplicability, setFormApplicability] = useState("");
  const [formDescription, setFormDescription] = useState("");
  const [formError, setFormError] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [importGuidanceOpen, setImportGuidanceOpen] = useState(false);

  const canWrite =
    currentUser.value?.role === "admin" || currentUser.value?.role === "writer";

  const loadPrograms = () => {
    setError("");
    apiFetch("/api/programs")
      .then((r) => {
        if (!r.ok) throw new Error(`HTTP ${r.status}`);
        return r.json();
      })
      .then((rows: Program[]) => setPrograms(Array.isArray(rows) ? rows : []))
      .catch(() => {
        setPrograms([]);
        setError("Could not load programs.");
      })
      .finally(() => setLoading(false));
  };

  const loadCatalogs = () => {
    apiFetch("/api/catalogs?type=GuidanceCatalog")
      .then((r) => (r.ok ? r.json() : []))
      .then((rows: CatalogRow[]) => setCatalogs(Array.isArray(rows) ? rows : []))
      .catch(() => setCatalogs([]));
  };

  useEffect(() => {
    setLoading(true);
    loadPrograms();
    loadCatalogs();
  }, [viewInvalidation.value]);

  const filtered = useMemo(() => {
    if (statusFilter === "all") return programs;
    return programs.filter((p) => (p.status || "").toLowerCase() === statusFilter);
  }, [programs, statusFilter]);

  const resetForm = () => {
    setFormName("");
    setFormFramework("");
    setFormCatalogId("");
    setFormApplicability("");
    setFormDescription("");
    setFormError("");
  };

  const submitCreate = async () => {
    const name = formName.trim();
    const framework = formFramework.trim();
    if (!name || !framework) {
      setFormError("Name and framework are required.");
      return;
    }
    setFormError("");
    setSubmitting(true);
    try {
      const applicability = formApplicability
        .split(",")
        .map((s) => s.trim())
        .filter(Boolean);
      const body: Record<string, unknown> = {
        name,
        framework,
        applicability,
      };
      if (formDescription.trim()) {
        body.description = formDescription.trim();
      }
      if (formCatalogId) {
        body.guidance_catalog_id = formCatalogId;
      }
      const res = await apiFetch("/api/programs", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body),
      });
      if (!res.ok) {
        const j = await res.json().catch(() => ({}));
        setFormError((j as { error?: string }).error || `Create failed (${res.status})`);
        return;
      }
      const created = (await res.json()) as Program;
      resetForm();
      setShowCreate(false);
      invalidateViews();
      if (created?.id) {
        navigateToProgram(created.id);
      }
    } catch (e) {
      setFormError(String(e));
    } finally {
      setSubmitting(false);
    }
  };

  if (loading) {
    return <div class="view-loading">Loading programs...</div>;
  }

  return (
    <div class="programs-view">
      <ImportOverlay
        open={importGuidanceOpen}
        onClose={() => setImportGuidanceOpen(false)}
        expectedArtifactType="GuidanceCatalog"
        onSuccess={invalidateViews}
      />
      <div class="programs-header">
        <div>
          <h2>Programs</h2>
        </div>
        <div style={{ display: "flex", alignItems: "center", gap: "12px", flexWrap: "wrap" }}>
          <label class="sr-only" htmlFor="program-status-filter">
            Filter by status
          </label>
          <select
            id="program-status-filter"
            class="programs-status-filter"
            value={statusFilter}
            onChange={(e) => setStatusFilter((e.target as HTMLSelectElement).value as StatusFilter)}
          >
            <option value="all">All statuses</option>
            <option value="intake">Intake</option>
            <option value="active">Active</option>
            <option value="monitoring">Monitoring</option>
            <option value="renewal">Renewal</option>
            <option value="closed">Closed</option>
          </select>
          {canWrite && (
            <>
              <button
                type="button"
                class="btn btn-secondary"
                onClick={() => setImportGuidanceOpen(true)}
              >
                Import Guidance
              </button>
              <button type="button" class="btn btn-primary" onClick={() => setShowCreate((v) => !v)}>
                {showCreate ? "Cancel" : "New Program"}
              </button>
            </>
          )}
        </div>
      </div>

      {error && (
        <p class="import-error" role="alert">
          {error}
        </p>
      )}

      {showCreate && canWrite && (
        <div class="create-program-form">
          {formError && (
            <p class="import-error" role="alert">
              {formError}
            </p>
          )}
          <div class="form-group">
            <label for="prog-name">Name</label>
            <input
              id="prog-name"
              type="text"
              value={formName}
              onInput={(e) => setFormName((e.target as HTMLInputElement).value)}
              required
            />
          </div>
          <div class="form-group">
            <label for="prog-framework">Framework</label>
            <input
              id="prog-framework"
              type="text"
              placeholder="FedRAMP, PCI-DSS, ISO 27001"
              value={formFramework}
              onInput={(e) => setFormFramework((e.target as HTMLInputElement).value)}
              required
            />
          </div>
          <div class="form-group">
            <label for="prog-catalog">Guidance catalog</label>
            <select
              id="prog-catalog"
              value={formCatalogId}
              onChange={(e) => setFormCatalogId((e.target as HTMLSelectElement).value)}
            >
              <option value="">None</option>
              {catalogs.map((c) => (
                <option key={c.catalog_id} value={c.catalog_id}>
                  {c.title || c.catalog_id}
                </option>
              ))}
            </select>
          </div>
          <div class="form-group">
            <label for="prog-app">Applicability</label>
            <input
              id="prog-app"
              type="text"
              placeholder="Comma-separated tags"
              value={formApplicability}
              onInput={(e) => setFormApplicability((e.target as HTMLInputElement).value)}
            />
          </div>
          <div class="form-group">
            <label for="prog-desc">Description</label>
            <textarea
              id="prog-desc"
              value={formDescription}
              onInput={(e) => setFormDescription((e.target as HTMLTextAreaElement).value)}
            />
          </div>
          <div class="form-actions">
            <button type="button" class="btn" onClick={() => { resetForm(); setShowCreate(false); }}>
              Cancel
            </button>
            <button type="button" class="btn btn-primary" disabled={submitting} onClick={submitCreate}>
              {submitting ? "Creating..." : "Create"}
            </button>
          </div>
        </div>
      )}

      {filtered.length === 0 && !error ? (
        programs.length === 0 ? (
          <div class="empty-state">
            <p>
              No compliance programs. Click &apos;New Program&apos; to create one from a guidance
              framework.
            </p>
          </div>
        ) : (
          <div class="empty-state">
            <p>No programs match this status filter.</p>
          </div>
        )
      ) : (
        <div class="programs-grid">
          {filtered.map((p) => (
            <article
              key={p.id}
              class="program-card"
              tabIndex={0}
              role="button"
              onClick={() => navigateToProgram(p.id)}
              onKeyDown={cardKeyHandler(() => navigateToProgram(p.id))}
            >
              <header style={{ display: "flex", justifyContent: "space-between", gap: "8px" }}>
                <h3>{p.name}</h3>
                <span
                  class={`readiness-dot ${
                    healthClass(p.health)
                  }`}
                  title={p.health || "Health unknown"}
                  aria-hidden="true"
                />
              </header>
              <span class="program-card-framework">{p.framework}</span>
              <div style={{ marginTop: "8px" }}>
                <span class={`status-badge ${statusBadgeClass(p.status)}`}>{p.status || "—"}</span>
              </div>
              <div class="program-card-meta">
                <span>{(p.policy_ids?.length ?? 0)} policies</span>
                <span>
                  {p.environments?.length
                    ? p.environments.join(", ")
                    : "No environments"}
                </span>
              </div>
            </article>
          ))}
        </div>
      )}
    </div>
  );
}

function healthClass(health?: string | null): string {
  const h = (health || "").toLowerCase();
  if (h === "green" || h === "good") return "readiness-green";
  if (h === "yellow" || h === "warning" || h === "amber") return "readiness-yellow";
  if (h === "red" || h === "bad" || h === "critical") return "readiness-red";
  return "readiness-gray";
}

function statusBadgeClass(status: string): string {
  const s = (status || "").toLowerCase();
  if (s === "intake") return "status-intake";
  if (s === "active") return "status-active";
  if (s === "monitoring") return "status-monitoring";
  if (s === "renewal") return "status-renewal";
  if (s === "closed") return "status-closed";
  return "status-intake";
}
