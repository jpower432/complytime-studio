// SPDX-License-Identifier: Apache-2.0

import { Fragment } from "preact";
import { useState, useEffect, useMemo } from "preact/hooks";
import { apiFetch } from "../api/fetch";
import {
  viewInvalidation,
  selectedPolicyId,
  selectedControlId,
  selectedEvidenceTargetId,
  selectedProgramFilter,
  updateHash,
} from "../app";
import {
  type FreshnessBucket,
  freshnessFromFrequency,
  defaultFreshnessBucket,
  freshnessRowClass,
  parsePolicyFrequencies,
} from "../lib/freshness";
import { createFilterChips } from "./filter-chip";
import { AddFilterMenu } from "./add-filter-menu";
import { fmtDateTime } from "../lib/format";
import { fetchRequirementMatrix, type RequirementRow } from "../api/requirements";

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
  certified?: boolean;
  frameworks?: string[];
  requirements?: string[];
  owner?: string;
  collected_at: string;
}

interface CertificationResult {
  evidence_id: string;
  certifier: string;
  certifier_version: string;
  result: string;
  reason: string;
  certified_at: string;
}

interface PolicyOption {
  policy_id: string;
  title: string;
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

function evidenceRowKey(r: EvidenceRecord, idx: number): string {
  return `${r.evidence_id}\t${r.collected_at}\t${idx}`;
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


function CertBadge({ certified }: { certified?: boolean }) {
  if (certified === true) {
    return (
      <span class="cert-badge cert-pass" title="Certified">&#x2713;</span>
    );
  }
  if (certified === false) {
    return (
      <span class="cert-badge cert-fail" title="Failed certification">&#x2717;</span>
    );
  }
  return <span class="cert-badge cert-pending" title="Pending">&#x2014;</span>;
}

function CertificationDetail({ evidenceId }: { evidenceId: string }) {
  const [rows, setRows] = useState<CertificationResult[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    const eid = encodeURIComponent(evidenceId);
    apiFetch(`/api/certifications?evidence_id=${eid}`)
      .then((r) => r.json())
      .then((data: CertificationResult[]) => setRows(data))
      .catch(() => setRows([]))
      .finally(() => setLoading(false));
  }, [evidenceId]);

  if (loading) return <p class="evidence-detail-muted">Loading...</p>;
  if (rows.length === 0) {
    return <p class="evidence-detail-muted">No certifications recorded.</p>;
  }

  return (
    <table class="cert-detail-table">
      <thead>
        <tr>
          <th>Certifier</th><th>Version</th>
          <th>Result</th><th>Reason</th>
        </tr>
      </thead>
      <tbody>
        {rows.map((r, i) => (
          <tr key={i}>
            <td>{r.certifier}</td>
            <td>{r.certifier_version}</td>
            <td>
              <span class={`cert-result cert-result-${r.result}`}>
                {r.result}
              </span>
            </td>
            <td>{r.reason}</td>
          </tr>
        ))}
      </tbody>
    </table>
  );
}

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
  Certification: "_certLabel",
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
  const [start, setStart] = useState("");
  const [end, setEnd] = useState("");
  const [expandedKey, setExpandedKey] = useState<string | null>(null);
  const [chipState] = useState(() => createFilterChips());
  const [policyContent, setPolicyContent] = useState("");
  const [programRows, setProgramRows] = useState<ProgramListItem[]>([]);
  const [programPolicyIds, setProgramPolicyIds] = useState<Set<string> | null>(
    null,
  );
  const [reqTextMap, setReqTextMap] = useState<Map<string, string>>(new Map());

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
    if (embedded) return;
    const tid = selectedEvidenceTargetId.value;
    if (!tid) return;
    chipState.remove("Target");
    chipState.remove("Control");
    chipState.add("Target", tid);
  }, [embedded, selectedEvidenceTargetId.value]);

  useEffect(() => {
    if (embedded) return;
    const cid = selectedControlId.value;
    if (!cid) return;
    chipState.remove("Control");
    chipState.remove("Target");
    chipState.add("Control", cid);
    selectedControlId.value = null;
  }, [embedded, selectedControlId.value]);

  useEffect(() => {
    if (!policyIdOverride) return;
    apiFetch(`/api/policies/${encodeURIComponent(policyIdOverride)}`)
      .then((r) => r.json())
      .then((d: { policy: { content?: string } }) => setPolicyContent(d.policy.content || ""))
      .catch(() => setPolicyContent(""));
  }, [policyIdOverride]);

  useEffect(() => {
    const pid = policyId || policyIdOverride;
    if (!pid) { setReqTextMap(new Map()); return; }
    fetchRequirementMatrix({ policy_id: pid })
      .then((rows: RequirementRow[]) => {
        const m = new Map<string, string>();
        for (const r of rows) m.set(r.control_id, r.requirement_text);
        setReqTextMap(m);
      })
      .catch(() => setReqTextMap(new Map()));
  }, [policyId, policyIdOverride]);

  useEffect(() => {
    apiFetch("/api/policies")
      .then((r) => r.json())
      .then((data: PolicyOption[]) => setPolicies(data))
      .catch(() => setPolicies([]));
  }, []);

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

  useEffect(() => {
    if (selectedPolicyId.value && !policyId) setPolicyId(selectedPolicyId.value);
  }, []);

  const search = () => {
    const effectivePolicyId = policyId || selectedPolicyId.value || "";
    if (effectivePolicyId) selectedPolicyId.value = effectivePolicyId;
    updateHash();
    setLoading(true);
    const params = new URLSearchParams();
    if (effectivePolicyId) params.set("policy_id", effectivePolicyId);
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
      _certLabel: r.certified === true ? "Certified" : r.certified === false ? "Uncertified" : "Pending",
    })),
    [records, freqMap]
  );

  const filteredRecords = useMemo(() => {
    const chips = chipState.filters.value;
    return recordsWithBuckets.filter((r) => {
      if (programPolicyIds && !programPolicyIds.has(r.policy_id)) return false;
      for (const [key, value] of chips) {
        if (key === "Freshness") {
          const expected = FRESHNESS_LABELS[value];
          if (expected && r._bucket !== expected) return false;
        } else if (key === "Control") {
          if (r.control_id.toLowerCase() !== value.toLowerCase()) return false;
        } else if (key === "Target") {
          const v = value.toLowerCase();
          if (
            r.target_id?.toLowerCase() !== v &&
            (r.target_name || r.target_id)?.toLowerCase() !== v
          ) return false;
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
  }, [recordsWithBuckets, chipState.filters.value, programPolicyIds]);

  const freshnessCounts = useMemo(() => {
    const counts: Record<FreshnessBucket, number> = {
      current: 0, aging: 0, stale: 0, "very-stale": 0,
    };
    for (const r of recordsWithBuckets) counts[r._bucket]++;
    return counts;
  }, [recordsWithBuckets]);

  const distinctValues = (field: keyof EvidenceRecord) => () =>
    [...new Set(records.map((r) => r[field]).filter(Boolean) as string[])].sort();

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

  const policyFilterField = embedded || policyId
    ? []
    : [
        {
          key: "Policy",
          label: "Policy",
          options: () =>
            [...policies]
              .sort((a, b) => a.title.localeCompare(b.title))
              .map((p) => ({ value: p.policy_id, label: p.title || p.policy_id })),
          pick: (id: string) => {
            setPolicyId(id);
            selectedPolicyId.value = id;
            updateHash();
          },
        },
      ];

  const filterFields = [
    ...policyFilterField,
    ...programFilterFields,
    { key: "Control", label: "Control ID", options: distinctValues("control_id") },
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
    {
      key: "Certification",
      label: "Certification",
      options: ["Certified", "Uncertified", "Pending"],
    },
  ];

  return (
    <section class="evidence-view">
      <div class="evidence-header">
        {!embedded && <h2>Evidence</h2>}
      </div>

      <div class="evidence-filters">
        <input type="date" value={start} onInput={(e) => setStart((e.target as HTMLInputElement).value)} />
        <input type="date" value={end} onInput={(e) => setEnd((e.target as HTMLInputElement).value)} />
        <AddFilterMenu fields={filterFields} chipState={chipState} />
        <button class="btn btn-primary" onClick={search}>Search</button>
      </div>

      {(chipState.filters.value.size > 0 || selectedProgramFilter.value || (policyId && !embedded)) && (
        <div class="filter-chips">
          {policyId && !embedded && (
            <span class="filter-chip" key="policy-filter">
              <span class="filter-chip-label">
                Policy:{" "}
                {policies.find((p) => p.policy_id === policyId)?.title || policyId}
              </span>
              <button
                type="button"
                class="filter-chip-dismiss"
                aria-label="Remove Policy filter"
                onClick={() => {
                  setPolicyId("");
                  selectedPolicyId.value = null;
                  setPolicyContent("");
                  updateHash();
                }}
              >
                &times;
              </button>
            </span>
          )}
          {selectedProgramFilter.value && (
            <span class="filter-chip" key="program-filter">
              <span class="filter-chip-label">
                Program:{" "}
                {programRows.find((p) => p.id === selectedProgramFilter.value)
                  ?.name || selectedProgramFilter.value}
              </span>
              <button
                type="button"
                class="filter-chip-dismiss"
                aria-label="Remove Program filter"
                onClick={() => {
                  selectedProgramFilter.value = null;
                  updateHash();
                }}
              >
                &times;
              </button>
            </span>
          )}
          {[...chipState.filters.value.entries()].map(([key, value]) => (
            <span key={key} class="filter-chip">
              <span class="filter-chip-label">{key}: {value}</span>
              <button
                type="button"
                class="filter-chip-dismiss"
                aria-label={`Remove ${key} filter`}
                onClick={() => chipState.remove(key)}
              >
                &times;
              </button>
            </span>
          ))}
        </div>
      )}

      {loading ? (
        <div class="view-loading">Querying evidence...</div>
      ) : records.length === 0 ? (
        <div class="empty-state">
          <p>No evidence found. Adjust filters or ingest evidence via Gemara artifacts.</p>
        </div>
      ) : (
        <>
          <EvidenceSummary records={filteredRecords} />
          <table class="data-table evidence-table">
            <thead>
              <tr>
                <th class="evidence-cert-col" aria-label="Certification status" />
                <th>Target</th>
                <th>Control</th>
                <th>Result</th>
                <th>Engine</th>
                <th>Collected</th>
              </tr>
            </thead>
            <tbody>
              {filteredRecords.map((r, idx) => {
                const rowKey = evidenceRowKey(r, idx);
                const open = expandedKey === rowKey;
                const certTooltip = r.certified === true
                  ? "Certified — click for details"
                  : r.certified === false
                    ? "Failed certification — click for details"
                    : "Pending certification — click for details";
                return (
                  <Fragment key={rowKey}>
                    <tr
                      data-evidence-id={r.evidence_id}
                      class={
                        `${freshnessRowClass(r._bucket)}`
                        + ` ${open ? "evidence-row-open" : ""}`
                      }
                    >
                      <td class="evidence-cert-cell">
                        <button
                          type="button"
                          class="evidence-cert-toggle"
                          aria-expanded={open}
                          title={certTooltip}
                          aria-label={certTooltip}
                          onClick={() => setExpandedKey(
                            open ? null : rowKey,
                          )}
                        >
                          <CertBadge certified={r.certified} />
                        </button>
                      </td>
                      <td title={r.target_id}>
                        {r.target_name || r.target_id}
                      </td>
                      <td title={reqTextMap.get(r.control_id) || undefined}>{r.control_id}</td>
                      <td>
                        <span class={
                          `eval-badge eval-${r.eval_result
                            ?.toLowerCase().replace(/ /g, "-")}`
                        }>
                          {r.eval_result}
                        </span>
                      </td>
                      <td>{r.engine_name || "---"}</td>
                      <td>
                        {fmtDateTime(r.collected_at)}
                      </td>
                    </tr>
                    {open && (
                      <tr
                        class="evidence-detail-row"
                        aria-label="Evidence details"
                      >
                        <td colSpan={6}>
                          <div class="evidence-detail-panel">
                            <CertificationDetail
                              evidenceId={r.evidence_id}
                            />
                            {r.source_registry?.trim() ? (
                              <SourceRegistryDetail
                                value={r.source_registry}
                              />
                            ) : (
                              <p class="evidence-detail-muted">
                                No source registry on this row.
                              </p>
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
