// SPDX-License-Identifier: Apache-2.0

import { Fragment } from "preact";
import { useState, useEffect, useRef, useMemo } from "preact/hooks";
import { apiFetch } from "../api/fetch";
import { currentUser, viewInvalidation, selectedPolicyId, updateHash } from "../app";
import {
  type FreshnessBucket,
  freshnessFromFrequency,
  defaultFreshnessBucket,
  freshnessRowClass,
  parsePolicyFrequencies,
} from "../lib/freshness";
import { createFilterChips, FilterChips } from "./filter-chip";
import { FreshnessBar } from "./freshness-bar";
import { AddFilterMenu } from "./add-filter-menu";

interface EvidenceRecord {
  evidence_id: string;
  policy_id: string;
  target_id: string;
  target_name?: string;
  target_type?: string;
  target_env?: string;
  control_id: string;
  rule_id: string;
  eval_result: string;
  engine_name?: string;
  engine_version?: string;
  requirement_id?: string;
  plan_id?: string;
  confidence?: string;
  compliance_status?: string;
  enrichment_status?: string;
  attestation_ref?: string;
  source_registry?: string;
  blob_ref?: string;
  frameworks?: string[];
  requirements?: string[];
  owner?: string;
  collected_at: string;
}

interface PolicyOption {
  policy_id: string;
  title: string;
}

function evidenceRowKey(r: EvidenceRecord): string {
  return `${r.evidence_id}\t${r.collected_at}`;
}

function SourceRegistryDetail({ value }: { value: string }) {
  const t = value.trim();
  if (!t) return null;
  const isHTTP = t.startsWith("https://") || t.startsWith("http://");
  const copy = async () => {
    try {
      await navigator.clipboard.writeText(t);
    } catch {
      /* clipboard unavailable */
    }
  };
  return (
    <div class="evidence-detail-registry">
      <span class="evidence-detail-label">Source registry</span>
      <div class="evidence-detail-registry-value">
        {isHTTP ? (
          <a href={t} target="_blank" rel="noopener noreferrer">{t}</a>
        ) : (
          <code class="evidence-registry-code">{t}</code>
        )}
        <button type="button" class="btn btn-xs" onClick={copy} aria-label="Copy source registry">
          Copy
        </button>
      </div>
    </div>
  );
}

const MAX_FILE_SIZE = 50 * 1024 * 1024;
const ALLOWED_TYPES = [
  "image/png", "image/jpeg", "image/webp",
  "application/pdf", "text/plain", "text/csv", "application/gzip",
];

function EvidenceSummary({ records }: { records: EvidenceRecord[] }) {
  const total = records.length;
  const passed = records.filter((r) => r.eval_result?.toLowerCase() === "passed").length;
  const passRate = total > 0 ? Math.round((passed / total) * 100) : 0;
  const engines = new Set(records.map((r) => r.engine_name).filter(Boolean)).size;
  const targets = new Set(records.map((r) => r.target_id)).size;
  const failed = records.filter((r) => r.eval_result?.toLowerCase() === "failed").length;
  const other = total - passed - failed;

  return (
    <div class="evidence-summary">
      <span class="summary-stat">{total} records</span>
      <span class="summary-stat">{passRate}% pass</span>
      <span class="summary-stat">{engines} engines</span>
      <span class="summary-stat">{targets} targets</span>
      <div class="posture-bar">
        {total > 0 && (
          <>
            <div class="bar-pass" style={{ width: `${(passed / total) * 100}%` }} />
            <div class="bar-fail" style={{ width: `${(failed / total) * 100}%` }} />
            <div class="bar-other" style={{ width: `${(other / total) * 100}%` }} />
          </>
        )}
      </div>
    </div>
  );
}

const CHIP_FIELD_MAP: Record<string, string> = {
  Target: "target_name_or_id",
  Result: "eval_result",
  Engine: "engine_name",
  "Compliance Status": "compliance_status",
  Owner: "owner",
  "Enrichment Status": "enrichment_status",
};

const FRESHNESS_LABELS: Record<string, FreshnessBucket> = {
  Current: "current",
  Aging: "aging",
  Stale: "stale",
  "Very Stale": "very-stale",
};

export function EvidenceView({ policyIdOverride, initialTargetFilter, initialControlFilter }: {
  policyIdOverride?: string;
  initialTargetFilter?: string;
  initialControlFilter?: string;
} = {}) {
  const embedded = !!policyIdOverride;
  const [records, setRecords] = useState<EvidenceRecord[]>([]);
  const [policies, setPolicies] = useState<PolicyOption[]>([]);
  const [loading, setLoading] = useState(false);
  const [policyId, setPolicyId] = useState(policyIdOverride || "");
  const [controlId, setControlId] = useState("");
  const [start, setStart] = useState("");
  const [end, setEnd] = useState("");
  const [uploadStatus, setUploadStatus] = useState("");
  const [uploadWarnings, setUploadWarnings] = useState<string[]>([]);
  const [showUpload, setShowUpload] = useState(false);
  const [showManual, setShowManual] = useState(false);
  const [expandedKey, setExpandedKey] = useState<string | null>(null);
  const fileRef = useRef<HTMLInputElement>(null);
  const attachRef = useRef<HTMLInputElement>(null);
  const [chipState] = useState(() => createFilterChips());
  const [policyContent, setPolicyContent] = useState("");

  useEffect(() => {
    if (initialTargetFilter) {
      chipState.remove("Target");
      chipState.remove("Control");
      chipState.add("Target", initialTargetFilter);
    }
  }, [initialTargetFilter]);

  useEffect(() => {
    if (initialControlFilter) {
      chipState.remove("Control");
      chipState.remove("Target");
      chipState.add("Control", initialControlFilter);
    }
  }, [initialControlFilter]);

  useEffect(() => {
    if (!policyIdOverride) return;
    apiFetch(`/api/policies/${encodeURIComponent(policyIdOverride)}`)
      .then((r) => r.json())
      .then((d: { policy: { content?: string } }) => setPolicyContent(d.policy.content || ""))
      .catch(() => setPolicyContent(""));
  }, [policyIdOverride]);

  useEffect(() => {
    apiFetch("/api/policies")
      .then((r) => r.json())
      .then((data: PolicyOption[]) => setPolicies(data))
      .catch(() => setPolicies([]));
  }, []);

  useEffect(() => {
    if (selectedPolicyId.value && !policyId) setPolicyId(selectedPolicyId.value);
  }, []);

  const search = () => {
    if (policyId) selectedPolicyId.value = policyId;
    updateHash();
    setLoading(true);
    const params = new URLSearchParams();
    if (policyId) params.set("policy_id", policyId);
    if (controlId) params.set("control_id", controlId);
    if (start) params.set("start", start);
    if (end) params.set("end", end);
    params.set("limit", "200");

    apiFetch(`/api/evidence?${params}`)
      .then((r) => r.json())
      .then(setRecords)
      .catch(() => setRecords([]))
      .finally(() => setLoading(false));
  };

  useEffect(search, []);
  useEffect(search, [viewInvalidation.value]);

  const handleUpload = async () => {
    const file = fileRef.current?.files?.[0];
    if (!file) return;
    setUploadStatus("Uploading...");
    setUploadWarnings([]);
    const formData = new FormData();
    formData.append("file", file);
    try {
      const res = await apiFetch("/api/evidence/upload", { method: "POST", body: formData });
      const data = await res.json();
      setUploadStatus(`Imported ${data.inserted} rows, ${data.failed} failed`);
      if (data.warnings?.length) setUploadWarnings(data.warnings);
      search();
    } catch (e) {
      setUploadStatus(`Upload failed: ${e}`);
    }
  };

  const [manual, setManual] = useState({
    policy_id: "", target_id: "", control_id: "", rule_id: "",
    eval_result: "Passed", requirement_id: "", plan_id: "",
    engine_name: "", confidence: "", compliance_status: "",
  });

  const handleManualSubmit = async () => {
    setUploadStatus("Submitting...");
    setUploadWarnings([]);

    const attachment = attachRef.current?.files?.[0];
    if (attachment) {
      if (attachment.size > MAX_FILE_SIZE) {
        setUploadStatus(`File too large (max ${MAX_FILE_SIZE / 1024 / 1024} MiB)`);
        return;
      }
      if (!ALLOWED_TYPES.includes(attachment.type)) {
        setUploadStatus(`File type ${attachment.type} not allowed`);
        return;
      }
    }

    const row = { ...manual, collected_at: new Date().toISOString() };

    try {
      let res: Response;
      if (attachment) {
        const formData = new FormData();
        formData.set("data", JSON.stringify([row]));
        formData.set("file", attachment, attachment.name);
        res = await apiFetch("/api/evidence", { method: "POST", body: formData });
      } else {
        res = await apiFetch("/api/evidence", {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify([row]),
        });
      }
      const text = await res.text();
      let payload: { inserted?: number; errors?: string[] } = {};
      try {
        payload = JSON.parse(text) as typeof payload;
      } catch {
        /* non-JSON error body */
      }
      if (!res.ok) {
        const msg = payload.errors?.length ? payload.errors.join("; ") : text;
        setUploadStatus(`Submit failed: ${msg}`);
        return;
      }
      if (payload.errors?.length) {
        setUploadStatus(`Validation failed: ${payload.errors.join("; ")}`);
        return;
      }
      setUploadStatus(`Inserted ${payload.inserted ?? 0} row(s)`);
      setManual({
        policy_id: "", target_id: "", control_id: "", rule_id: "",
        eval_result: "Passed", requirement_id: "", plan_id: "",
        engine_name: "", confidence: "", compliance_status: "",
      });
      if (attachRef.current) attachRef.current.value = "";
      search();
    } catch (e) {
      setUploadStatus(`Submit failed: ${e}`);
    }
  };

  const updateManual = (field: string) => (e: Event) => {
    setManual({ ...manual, [field]: (e.target as HTMLInputElement).value });
  };

  const freqMap = useMemo(() => parsePolicyFrequencies(policyContent), [policyContent]);

  const computeBucket = (r: EvidenceRecord): FreshnessBucket => {
    const reqId = r.requirement_id || r.control_id;
    const cycleDays = freqMap.get(reqId);
    if (cycleDays !== undefined) return freshnessFromFrequency(r.collected_at, cycleDays);
    return defaultFreshnessBucket(r.collected_at);
  };

  const recordsWithBuckets = useMemo(() =>
    records.map((r) => ({
      ...r,
      _bucket: computeBucket(r),
      target_name_or_id: r.target_name || r.target_id,
    })),
    [records, freqMap]
  );

  const filteredRecords = useMemo(() => {
    const chips = chipState.filters.value;
    return recordsWithBuckets.filter((r) => {
      for (const [key, value] of chips) {
        if (key === "Freshness") {
          const expected = FRESHNESS_LABELS[value];
          if (expected && r._bucket !== expected) return false;
        } else if (key === "Control") {
          if (r.control_id.toLowerCase() !== value.toLowerCase()) return false;
        } else {
          const fieldName = CHIP_FIELD_MAP[key];
          if (fieldName) {
            const actual = String((r as Record<string, unknown>)[fieldName] ?? "");
            if (actual.toLowerCase() !== value.toLowerCase()) return false;
          }
        }
      }
      return true;
    });
  }, [recordsWithBuckets, chipState.filters.value]);

  const freshnessCounts = useMemo(() => {
    const counts: Record<FreshnessBucket, number> = {
      current: 0, aging: 0, stale: 0, "very-stale": 0,
    };
    for (const r of recordsWithBuckets) counts[r._bucket]++;
    return counts;
  }, [recordsWithBuckets]);

  const distinctValues = (field: keyof EvidenceRecord) => () =>
    [...new Set(records.map((r) => r[field]).filter(Boolean) as string[])].sort();

  const filterFields = [
    { key: "Target", label: "Target", options: distinctValues("target_name") },
    { key: "Result", label: "Result", options: ["Passed", "Failed", "Unknown"] },
    {
      key: "Compliance Status",
      label: "Compliance Status",
      options: [
        "Compliant", "Non-Compliant", "Exempt",
        "Not Applicable", "Unknown",
      ],
    },
    { key: "Engine", label: "Engine", options: distinctValues("engine_name") },
    { key: "Owner", label: "Owner", options: distinctValues("owner") },
    {
      key: "Enrichment Status",
      label: "Enrichment Status",
      options: distinctValues("enrichment_status"),
    },
  ];

  return (
    <section class="evidence-view">
      <div class="evidence-header">
        {!embedded && <h2>Evidence</h2>}
        {!embedded && currentUser.value?.role === "admin" && (
          <button class="btn btn-sm" onClick={() => setShowUpload(!showUpload)}>
            {showUpload ? "Hide Upload" : "Upload Evidence"}
          </button>
        )}
      </div>

      <div class="evidence-filters">
        {!embedded && (
          <select value={policyId} data-policy-id={policyId || ""} onChange={(e) => setPolicyId((e.target as HTMLSelectElement).value)}>
            <option value="">All Policies</option>
            {policies.map((p) => (
              <option key={p.policy_id} value={p.policy_id}>{p.title}</option>
            ))}
          </select>
        )}
        <input placeholder="Control ID" value={controlId} onInput={(e) => setControlId((e.target as HTMLInputElement).value)} />
        <input type="date" value={start} onInput={(e) => setStart((e.target as HTMLInputElement).value)} />
        <input type="date" value={end} onInput={(e) => setEnd((e.target as HTMLInputElement).value)} />
        <AddFilterMenu fields={filterFields} chipState={chipState} />
        <button class="btn btn-primary" onClick={search}>Search</button>
      </div>

      <FilterChips state={chipState} />

      {showUpload && (
        <div class="evidence-upload">
          <div class="upload-toggle">
            <button class={`btn btn-xs ${!showManual ? "btn-primary" : ""}`} onClick={() => setShowManual(false)}>CSV Upload</button>
            <button class={`btn btn-xs ${showManual ? "btn-primary" : ""}`} onClick={() => setShowManual(true)}>Manual Entry</button>
          </div>

          {!showManual ? (
            <div class="upload-file-row">
              <input type="file" ref={fileRef} accept=".csv,.json" />
              <button class="btn btn-secondary" onClick={handleUpload}>Upload</button>
            </div>
          ) : (
            <div class="manual-entry-form">
              <div class="manual-entry-row">
                <select value={manual.policy_id} onChange={updateManual("policy_id")}>
                  <option value="">Policy *</option>
                  {policies.map((p) => (
                    <option key={p.policy_id} value={p.policy_id}>{p.title}</option>
                  ))}
                </select>
                <input placeholder="Target ID *" value={manual.target_id} onInput={updateManual("target_id")} />
                <input placeholder="Control ID *" value={manual.control_id} onInput={updateManual("control_id")} />
              </div>
              <div class="manual-entry-row">
                <input placeholder="Rule ID" value={manual.rule_id} onInput={updateManual("rule_id")} />
                <select value={manual.eval_result} onChange={updateManual("eval_result")}>
                  <option value="Passed">Passed</option>
                  <option value="Failed">Failed</option>
                  <option value="Unknown">Unknown</option>
                </select>
                <select value={manual.compliance_status} onChange={updateManual("compliance_status")}>
                  <option value="">Compliance Status</option>
                  <option value="Compliant">Compliant</option>
                  <option value="Non-Compliant">Non-Compliant</option>
                  <option value="Exempt">Exempt</option>
                  <option value="Not Applicable">Not Applicable</option>
                  <option value="Unknown">Unknown</option>
                </select>
              </div>
              <details class="manual-extras">
                <summary>Optional fields</summary>
                <div class="manual-entry-row">
                  <input placeholder="Requirement ID" value={manual.requirement_id} onInput={updateManual("requirement_id")} />
                  <input placeholder="Plan ID" value={manual.plan_id} onInput={updateManual("plan_id")} />
                  <input placeholder="Engine Name" value={manual.engine_name} onInput={updateManual("engine_name")} />
                  <input placeholder="Confidence" value={manual.confidence} onInput={updateManual("confidence")} />
                </div>
                <div class="file-attachment">
                  <label>Attachment (max 50 MiB):</label>
                  <input type="file" ref={attachRef} accept=".png,.jpg,.jpeg,.webp,.pdf,.txt,.csv,.gz" />
                </div>
              </details>
              <button
                class="btn btn-secondary"
                onClick={handleManualSubmit}
                disabled={!manual.policy_id || !manual.target_id || !manual.control_id}
              >
                Submit
              </button>
            </div>
          )}

          {uploadStatus && <span class="upload-status">{uploadStatus}</span>}
          {uploadWarnings.length > 0 && (
            <div class="upload-warnings">
              {uploadWarnings.map((w, i) => <div key={i} class="warning-msg">{w}</div>)}
            </div>
          )}
        </div>
      )}

      {loading ? (
        <div class="view-loading">Querying evidence...</div>
      ) : records.length === 0 ? (
        <div class="empty-state">
          <p>No evidence found. Adjust filters or upload evidence.</p>
        </div>
      ) : (
        <>
          <FreshnessBar buckets={freshnessCounts} chipState={chipState} />
          <EvidenceSummary records={filteredRecords} />
          <table class="data-table evidence-table">
            <thead>
              <tr>
                <th class="evidence-expand-col" aria-hidden="true" />
                <th>Target</th>
                <th>Control</th>
                <th>Result</th>
                <th>Engine</th>
                <th>Collected</th>
              </tr>
            </thead>
            <tbody>
              {filteredRecords.map((r) => {
                const rowKey = evidenceRowKey(r);
                const open = expandedKey === rowKey;
                return (
                  <Fragment key={rowKey}>
                    <tr data-evidence-id={r.evidence_id} class={`${freshnessRowClass(r._bucket)} ${open ? "evidence-row-open" : ""}`}>
                      <td class="evidence-expand-cell">
                        <button
                          type="button"
                          class="btn btn-xs"
                          aria-expanded={open}
                          aria-label={open ? "Hide evidence details" : "Show evidence details"}
                          onClick={() => setExpandedKey(open ? null : rowKey)}
                        >
                          {open ? "−" : "+"}
                        </button>
                      </td>
                      <td title={r.target_id}>{r.target_name || r.target_id}</td>
                      <td>{r.control_id}</td>
                      <td><span class={`eval-badge eval-${r.eval_result?.toLowerCase().replace(/ /g, "-")}`}>{r.eval_result}</span></td>
                      <td>{r.engine_name || "---"}</td>
                      <td>{new Date(r.collected_at).toLocaleString()}</td>
                    </tr>
                    {open && (
                      <tr class="evidence-detail-row" aria-label="Evidence details">
                        <td colSpan={6}>
                          <div class="evidence-detail-panel">
                            {r.source_registry?.trim() ? (
                              <SourceRegistryDetail value={r.source_registry} />
                            ) : (
                              <p class="evidence-detail-muted">No source registry on this row.</p>
                            )}
                          </div>
                        </td>
                      </tr>
                    )}
                  </Fragment>
                );
              })}
            </tbody>
          </table>
        </>
      )}
    </section>
  );
}
