// SPDX-License-Identifier: Apache-2.0

import { useState, useEffect, useMemo, useRef } from "preact/hooks";
import {
  navigate,
  selectedProgramId,
  selectedProgramFilter,
  currentUser,
  viewInvalidation,
  invalidateViews,
} from "../app";
import { apiFetch } from "../api/fetch";
import { fmtDate } from "../lib/format";

interface PolicyListItem {
  policy_id: string;
  title: string;
  version?: string;
  oci_reference: string;
  imported_at: string;
}

interface PostureRow {
  policy_id: string;
  title: string;
  total_rows: number;
  passed_rows: number;
  failed_rows: number;
  other_rows: number;
  target_count: number;
  control_count: number;
  latest_evidence_at?: string;
}

interface ProgramDetail {
  id: string;
  name: string;
  guidance_catalog_id?: string | null;
  framework: string;
  applicability: string[];
  status: string;
  health?: string | null;
  description?: string | null;
  metadata: unknown;
  policy_ids: string[];
  environments: string[];
  version: number;
  green_pct: number;
  red_pct: number;
  score_pct: number;
  created_at: string;
  updated_at: string;
}

interface Recommendation {
  policy_id: string;
  policy_title: string;
  reason: string;
  mapping_strength: number;
  evidence_count: number;
  score: number;
  predicted_score_pct?: number;
  score_delta?: number;
}

function CoverageDonut({ green, red, size = 120 }: { green: number; red: number; size?: number }) {
  const uncovered = Math.max(0, 100 - green - red);
  const r = 44;
  const circumference = 2 * Math.PI * r;
  const segments = [
    { pct: green, color: "var(--success)" },
    { pct: red, color: "var(--error)" },
    { pct: uncovered, color: "var(--border)" },
  ];
  let offset = 0;
  return (
    <svg viewBox="0 0 120 120" width={size} height={size} role="img" aria-label={`${green}% pass, ${red}% fail, ${uncovered}% uncovered`}>
      {segments.map((seg, i) => {
        const dash = (seg.pct / 100) * circumference;
        const el = (
          <circle
            key={i}
            cx="60" cy="60" r={r}
            fill="none"
            stroke={seg.color}
            stroke-width="12"
            stroke-dasharray={`${dash} ${circumference - dash}`}
            stroke-dashoffset={-offset}
            transform="rotate(-90 60 60)"
          />
        );
        offset += dash;
        return el;
      })}
      <text x="60" y="56" text-anchor="middle" class="donut-center-value">{green}%</text>
      <text x="60" y="72" text-anchor="middle" class="donut-center-label">pass</text>
    </svg>
  );
}

function PolicyPostureBar({ row }: { row: PostureRow }) {
  const total = row.total_rows || 1;
  const passPct = (row.passed_rows / total) * 100;
  const failPct = (row.failed_rows / total) * 100;
  return (
    <div class="policy-posture-row">
      <span class="policy-posture-label" title={row.policy_id}>{row.title || row.policy_id}</span>
      <div class="policy-posture-bar">
        <div class="policy-posture-fill pass" style={{ width: `${passPct}%` }} />
        <div class="policy-posture-fill fail" style={{ width: `${failPct}%` }} />
      </div>
      <span class="policy-posture-pct">{Math.round(passPct)}%</span>
    </div>
  );
}

type DetailTab = "overview" | "policies" | "recommendations";

export function ProgramDetailView() {
  void viewInvalidation.value;
  const programId = selectedProgramId.value;
  const [program, setProgram] = useState<ProgramDetail | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [tab, setTab] = useState<DetailTab>("overview");
  const [allPolicies, setAllPolicies] = useState<PolicyListItem[]>([]);
  const [postureByPolicy, setPostureByPolicy] = useState<Record<string, PostureRow>>({});
  const [recommendations, setRecommendations] = useState<Recommendation[]>([]);
  const [recLoading, setRecLoading] = useState(false);
  const [recError, setRecError] = useState("");
  const [assignOpen, setAssignOpen] = useState(false);
  const [assignPick, setAssignPick] = useState("");
  const [assignBusy, setAssignBusy] = useState(false);
  const [assignErr, setAssignErr] = useState("");
  const [actionBusy, setActionBusy] = useState<string | null>(null);
  const prevProgramIdRef = useRef<string | null>(null);

  const canWrite =
    currentUser.value?.role === "admin" || currentUser.value?.role === "writer";

  useEffect(() => {
    setTab("overview");
    setRecommendations([]);
    setRecError("");
    setAssignOpen(false);
    setAssignPick("");
    setAssignErr("");
  }, [programId]);

  const loadProgram = () => {
    if (!programId) {
      setProgram(null);
      setLoading(false);
      setError("");
      return;
    }
    setError("");
    apiFetch(`/api/programs/${encodeURIComponent(programId)}`)
      .then((r) => {
        if (r.status === 404) {
          setProgram(null);
          setError("Program not found.");
          return null;
        }
        if (!r.ok) throw new Error(`HTTP ${r.status}`);
        return r.json();
      })
      .then((p: ProgramDetail | null) => {
        if (p) setProgram(p);
      })
      .catch(() => {
        setProgram(null);
        setError("Could not load program.");
      })
      .finally(() => setLoading(false));
  };

  const loadPoliciesAndPosture = () => {
    Promise.all([
      apiFetch("/api/policies")
        .then((r) => (r.ok ? r.json() : []))
        .then((rows: PolicyListItem[]) => (Array.isArray(rows) ? rows : [])),
      apiFetch("/api/posture")
        .then((r) => (r.ok ? r.json() : []))
        .then((rows: PostureRow[]) => (Array.isArray(rows) ? rows : [])),
    ]).then(([policies, postureRows]) => {
      setAllPolicies(policies);
      const m: Record<string, PostureRow> = {};
      for (const row of postureRows) {
        m[row.policy_id] = row;
      }
      setPostureByPolicy(m);
    });
  };

  const loadRecommendations = () => {
    if (!programId) return;
    setRecLoading(true);
    setRecError("");
    apiFetch(`/api/programs/${encodeURIComponent(programId)}/recommendations`)
      .then((r) => {
        if (r.status === 403) {
          setRecommendations([]);
          setRecError("writers_admins");
          return null;
        }
        if (!r.ok) throw new Error(`HTTP ${r.status}`);
        return r.json();
      })
      .then((rows: Recommendation[] | null) => {
        if (rows) setRecommendations(Array.isArray(rows) ? rows : []);
      })
      .catch(() => {
        setRecommendations([]);
        setRecError("load_failed");
      })
      .finally(() => setRecLoading(false));
  };

  useEffect(() => {
    if (!programId) {
      setProgram(null);
      setLoading(false);
      setError("");
      prevProgramIdRef.current = null;
      return;
    }
    if (prevProgramIdRef.current !== programId) {
      setLoading(true);
      prevProgramIdRef.current = programId;
    }
    loadProgram();
    loadPoliciesAndPosture();
  }, [programId, viewInvalidation.value]);

  useEffect(() => {
    if (tab === "recommendations" && programId) {
      loadRecommendations();
    }
  }, [tab, programId, viewInvalidation.value]);

  const assignedPolicies = useMemo(() => {
    if (!program?.policy_ids?.length) return [];
    const idSet = new Set(program.policy_ids);
    return allPolicies.filter((p) => idSet.has(p.policy_id));
  }, [program, allPolicies]);

  const assignCandidates = useMemo(() => {
    if (!program) return [];
    const have = new Set(program.policy_ids || []);
    return allPolicies.filter((p) => !have.has(p.policy_id));
  }, [program, allPolicies]);

  const programPostureRows = useMemo(() => {
    if (!program?.policy_ids?.length) return [];
    const idSet = new Set(program.policy_ids);
    return Object.values(postureByPolicy).filter((r) => idSet.has(r.policy_id));
  }, [program, postureByPolicy]);

  const postureAgg = useMemo(() => {
    let total = 0, passed = 0, failed = 0, targets = 0, controls = 0;
    for (const r of programPostureRows) {
      total += r.total_rows;
      passed += r.passed_rows;
      failed += r.failed_rows;
      targets += r.target_count;
      controls += r.control_count;
    }
    return { total, passed, failed, targets, controls };
  }, [programPostureRows]);

  const metadataPreview = useMemo(() => {
    if (!program?.metadata || program.metadata === null) return "";
    try {
      const s = JSON.stringify(program.metadata, null, 2);
      return s === "{}" ? "" : s;
    } catch {
      return "";
    }
  }, [program]);

  const attachPolicy = async (policyId: string) => {
    if (!programId || !policyId) return;
    setActionBusy(`attach:${policyId}`);
    try {
      const res = await apiFetch(
        `/api/programs/${encodeURIComponent(programId)}/recommendations/${encodeURIComponent(policyId)}/attach`,
        { method: "POST" },
      );
      if (!res.ok) {
        const t = await res.text();
        throw new Error(t || `HTTP ${res.status}`);
      }
      invalidateViews();
    } catch (e) {
      console.error(e);
    } finally {
      setActionBusy(null);
    }
  };

  const dismissRecommendation = async (policyId: string) => {
    if (!programId) return;
    setActionBusy(`dismiss:${policyId}`);
    try {
      const res = await apiFetch(
        `/api/programs/${encodeURIComponent(programId)}/recommendations/${encodeURIComponent(policyId)}/dismiss`,
        { method: "POST" },
      );
      if (!res.ok) {
        const t = await res.text();
        throw new Error(t || `HTTP ${res.status}`);
      }
      invalidateViews();
    } catch (e) {
      console.error(e);
    } finally {
      setActionBusy(null);
    }
  };

  const submitAssign = async () => {
    if (!program || !programId || !assignPick) return;
    setAssignBusy(true);
    setAssignErr("");
    try {
      const nextIds = [...(program.policy_ids || []), assignPick];
      const res = await apiFetch(`/api/programs/${encodeURIComponent(programId)}`, {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          ...program,
          policy_ids: nextIds,
        }),
      });
      if (res.status === 409) {
        setAssignErr("Version conflict — refresh and try again.");
        loadProgram();
        return;
      }
      if (!res.ok) {
        const j = await res.json().catch(() => ({}));
        setAssignErr((j as { error?: string }).error || `Assign failed (${res.status})`);
        return;
      }
      setAssignOpen(false);
      setAssignPick("");
      invalidateViews();
    } catch (e) {
      setAssignErr(String(e));
    } finally {
      setAssignBusy(false);
    }
  };

  if (!programId) {
    return (
      <div class="program-detail-view empty-state">
        <p>No program selected.</p>
        <button type="button" class="btn btn-primary" onClick={() => navigate("programs")}>
          Back to Programs
        </button>
      </div>
    );
  }

  if (loading) {
    return <div class="view-loading">Loading program...</div>;
  }

  if (error || !program) {
    return (
      <div class="program-detail-view empty-state">
        <p>{error || "Program unavailable."}</p>
        <button type="button" class="btn btn-primary" onClick={() => navigate("programs")}>
          Back to Programs
        </button>
      </div>
    );
  }

  return (
    <div class="program-detail-view">
      <nav class="breadcrumb" aria-label="Breadcrumb">
        <button type="button" class="breadcrumb-link" onClick={() => navigate("programs")}>
          Programs
        </button>
        <span class="breadcrumb-sep" aria-hidden="true">
          &rsaquo;
        </span>
        <span class="breadcrumb-current">{program.name}</span>
      </nav>

      <header class="program-header">
        <h2>
          {program.name}
          <span class="program-card-framework" style={{ fontSize: "11px" }}>
            {program.framework}
          </span>
          <span class={`status-badge ${statusBadgeClass(program.status)}`}>{program.status}</span>
          <span
            class={`readiness-dot ${healthClass(program.health)}`}
            title={program.health || "Health"}
            aria-label={`Health: ${program.health || "unknown"}`}
          />
        </h2>
        <div class="program-cross-links" style={{ display: "flex", gap: "8px", marginTop: "10px" }}>
          <button
            type="button"
            class="btn btn-sm"
            onClick={() => {
              const id = selectedProgramId.value;
              if (!id) return;
              selectedProgramFilter.value = id;
              navigate("policies");
            }}
          >
            Policies (this program)
          </button>
          <button
            type="button"
            class="btn btn-sm"
            onClick={() => {
              const id = selectedProgramId.value;
              if (!id) return;
              selectedProgramFilter.value = id;
              navigate("evidence");
            }}
          >
            Evidence (this program)
          </button>
        </div>
      </header>

      <div class="tab-bar" role="tablist">
        {(
          [
            ["overview", "Overview"],
            ["policies", "Policies"],
            ["recommendations", "Recommendations"],
          ] as const
        ).map(([key, label]) => (
          <button
            key={key}
            type="button"
            role="tab"
            class={`tab-btn ${tab === key ? "active" : ""}`}
            aria-selected={tab === key}
            onClick={() => setTab(key)}
          >
            {label}
          </button>
        ))}
      </div>

      {tab === "overview" && (
        <section aria-labelledby="overview-heading">
          <h3 id="overview-heading" class="settings-section-title">
            Program coverage
          </h3>

          <div class="program-overview-grid">
            <div class="program-donut-area">
              <CoverageDonut green={program.score_pct ?? 0} red={Math.max(0, 100 - (program.score_pct ?? 0))} size={140} />
            </div>

            <div class="program-metrics">
              <div class="program-metric-card">
                <span class="program-metric-value">{program.policy_ids?.length ?? 0}</span>
                <span class="program-metric-label">Policies</span>
              </div>
              <div class="program-metric-card">
                <span class="program-metric-value">{postureAgg.targets}</span>
                <span class="program-metric-label">Targets</span>
              </div>
              <div class="program-metric-card">
                <span class="program-metric-value">{postureAgg.controls}</span>
                <span class="program-metric-label">Controls</span>
              </div>
              <div class="program-metric-card">
                <span class="program-metric-value">{postureAgg.total}</span>
                <span class="program-metric-label">Evidence records</span>
              </div>
            </div>
          </div>

          {programPostureRows.length > 0 && (
            <>
              <h4 class="settings-section-title" style={{ fontSize: "14px", marginTop: "20px" }}>
                Policy posture
              </h4>
              <div class="policy-posture-list">
                {programPostureRows.map((r) => (
                  <PolicyPostureBar key={r.policy_id} row={r} />
                ))}
              </div>
            </>
          )}

          {program.environments?.length > 0 && (
            <>
              <h4 class="settings-section-title" style={{ fontSize: "14px", marginTop: "20px" }}>
                Environments
              </h4>
              <div class="program-env-chips">
                {program.environments.map((e) => (
                  <span key={e} class="filter-chip"><span class="filter-chip-label">{e}</span></span>
                ))}
              </div>
            </>
          )}

          {metadataPreview && (
            <details style={{ marginTop: "16px" }}>
              <summary style={{ cursor: "pointer", fontSize: "13px" }}>Metadata</summary>
              <pre
                class="yaml-viewer"
                style={{ marginTop: "8px", fontSize: "12px", maxHeight: "200px", overflow: "auto" }}
              >
                {metadataPreview}
              </pre>
            </details>
          )}
        </section>
      )}

      {tab === "policies" && (
        <section aria-labelledby="policies-heading">
          <div
            style={{
              display: "flex",
              alignItems: "center",
              justifyContent: "space-between",
              marginBottom: "12px",
            }}
          >
            <h3 id="policies-heading" class="settings-section-title" style={{ margin: 0 }}>
              Assigned policies
            </h3>
            {canWrite && (
              <button type="button" class="btn btn-primary btn-sm" onClick={() => setAssignOpen(true)}>
                Assign Policy
              </button>
            )}
          </div>

          {assignOpen && canWrite && (
            <div class="create-program-form" style={{ maxWidth: "480px" }}>
              {assignErr && (
                <p class="import-error" role="alert">
                  {assignErr}
                </p>
              )}
              <div class="form-group">
                <label for="assign-policy-pick">Policy</label>
                <select
                  id="assign-policy-pick"
                  value={assignPick}
                  onChange={(e) => setAssignPick((e.target as HTMLSelectElement).value)}
                >
                  <option value="">Select a policy</option>
                  {assignCandidates.map((p) => (
                    <option key={p.policy_id} value={p.policy_id}>
                      {p.title}
                    </option>
                  ))}
                </select>
              </div>
              <div class="form-actions">
                <button
                  type="button"
                  class="btn"
                  onClick={() => {
                    setAssignOpen(false);
                    setAssignPick("");
                    setAssignErr("");
                  }}
                >
                  Cancel
                </button>
                <button
                  type="button"
                  class="btn btn-primary"
                  disabled={assignBusy || !assignPick}
                  onClick={submitAssign}
                >
                  {assignBusy ? "Saving..." : "Assign"}
                </button>
              </div>
            </div>
          )}

          {assignedPolicies.length === 0 ? (
            <p class="settings-section-desc">No policies assigned to this program.</p>
          ) : (
            <table class="data-table">
              <thead>
                <tr>
                  <th>Title</th>
                  <th>Control count</th>
                  <th>Latest evidence</th>
                </tr>
              </thead>
              <tbody>
                {assignedPolicies.map((p) => {
                  const ps = postureByPolicy[p.policy_id];
                  const cc = ps?.control_count ?? 0;
                  const ev = ps?.latest_evidence_at;
                  return (
                    <tr key={p.policy_id}>
                      <td>{p.title}</td>
                      <td>{cc}</td>
                      <td>{ev ? fmtDate(ev) : "—"}</td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          )}
        </section>
      )}

      {tab === "recommendations" && (
        <section aria-labelledby="rec-heading">
          <h3 id="rec-heading" class="settings-section-title">
            Recommendations
          </h3>
          {recLoading && <p class="settings-section-desc">Loading recommendations...</p>}
          {recError === "writers_admins" && (
            <p class="settings-section-desc" role="status">
              Recommendations are visible to writers and admins.
            </p>
          )}
          {recError === "load_failed" && (
            <p class="import-error" role="alert">
              Could not load recommendations.
            </p>
          )}
          {!recLoading && !recError && recommendations.length === 0 && (
            <p class="empty-state">No recommendations available for this program.</p>
          )}
          {!recLoading &&
            !recError &&
            recommendations.map((rec) => {
              const busyAttach = actionBusy === `attach:${rec.policy_id}`;
              const busyDismiss = actionBusy === `dismiss:${rec.policy_id}`;
              const pct = Math.round(Math.min(1, Math.max(0, rec.score)) * 100);
              const mapPct = Math.round(Math.min(1, Math.max(0, rec.mapping_strength)) * 100);
              return (
                <article key={rec.policy_id} class="recommendation-card">
                  <h4>{rec.policy_title}</h4>
                  <p class="recommendation-reason">{rec.reason}</p>
                  <div class="recommendation-score">
                    <span>Score</span>
                    <div class="score-bar" aria-hidden="true">
                      <div class="score-bar-fill" style={{ width: `${pct}%` }} />
                    </div>
                    <span>{pct}%</span>
                  </div>
                  <div class="recommendation-score">
                    <span>Mapping strength</span>
                    <span>{mapPct}%</span>
                  </div>
                  {rec.predicted_score_pct != null && rec.score_delta != null && (
                    <div class="recommendation-score predicted-posture">
                      <span>Predicted posture</span>
                      <span class={rec.score_delta > 0 ? "delta-positive" : rec.score_delta < 0 ? "delta-negative" : ""}>
                        {rec.predicted_score_pct}% ({rec.score_delta > 0 ? "+" : ""}{rec.score_delta}%)
                      </span>
                    </div>
                  )}
                  {canWrite && (
                    <div class="recommendation-actions">
                      <button
                        type="button"
                        class="btn btn-primary btn-sm"
                        disabled={!!actionBusy}
                        onClick={() => attachPolicy(rec.policy_id)}
                      >
                        {busyAttach ? "..." : "Attach"}
                      </button>
                      <button
                        type="button"
                        class="btn btn-sm"
                        disabled={!!actionBusy}
                        onClick={() => dismissRecommendation(rec.policy_id)}
                      >
                        {busyDismiss ? "..." : "Dismiss"}
                      </button>
                    </div>
                  )}
                </article>
              );
            })}
        </section>
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
