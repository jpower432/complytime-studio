// SPDX-License-Identifier: Apache-2.0

import { useMemo, useState, useEffect } from "preact/hooks";
import { apiFetch } from "../api/fetch";
import { relativeTime } from "../lib/freshness";
import {
  navigate,
  navigateToPolicy,
  selectedPolicyId,
  selectedEvidenceTargetId,
  viewInvalidation,
} from "../app";
import { createFilterChips, FilterChips } from "./filter-chip";
import { AddFilterMenu } from "./add-filter-menu";

export interface InventoryViewProps {
  policyIdOverride?: string;
}

interface InventoryItem {
  target_id: string;
  target_type: string;
  environment: string;
  policy_count: number;
  pass_count: number;
  fail_count: number;
  latest_evidence: string;
}

interface PolicyOption {
  policy_id: string;
  title: string;
}

interface ProgramRow {
  id: string;
  name: string;
}

type SortKey =
  | "target_id"
  | "target_type"
  | "environment"
  | "policy_count"
  | "pass_count"
  | "fail_count"
  | "latest_evidence";

const FILTER_KEYS = {
  policy: "Policy",
  program: "Program",
  targetType: "Target Type",
  environment: "Environment",
} as const;

function buildInventoryQuery(chips: Map<string, string>): string {
  const params = new URLSearchParams();
  const policyId = chips.get(FILTER_KEYS.policy);
  const programId = chips.get(FILTER_KEYS.program);
  const targetType = chips.get(FILTER_KEYS.targetType);
  const environment = chips.get(FILTER_KEYS.environment);
  if (policyId) params.set("policy_id", policyId);
  if (programId) params.set("program_id", programId);
  if (targetType) params.set("target_type", targetType);
  if (environment) params.set("environment", environment);
  return params.toString();
}

function sortInventory(
  rows: InventoryItem[],
  key: SortKey,
  dir: "asc" | "desc",
): InventoryItem[] {
  const mul = dir === "asc" ? 1 : -1;
  return [...rows].sort((a, b) => {
    let cmp = 0;
    if (key === "latest_evidence") {
      const ta = new Date(a.latest_evidence).getTime();
      const tb = new Date(b.latest_evidence).getTime();
      cmp = (Number.isFinite(ta) ? ta : 0) - (Number.isFinite(tb) ? tb : 0);
    } else if (typeof a[key] === "number" && typeof b[key] === "number") {
      cmp = (a[key] as number) - (b[key] as number);
    } else {
      cmp = String(a[key]).localeCompare(String(b[key]));
    }
    return cmp * mul;
  });
}

function SortHeader({
  label,
  sortKey,
  activeKey,
  dir,
  onSort,
}: {
  label: string;
  sortKey: SortKey;
  activeKey: SortKey;
  dir: "asc" | "desc";
  onSort: (k: SortKey) => void;
}) {
  const active = activeKey === sortKey;
  const ind = active ? (dir === "asc" ? "▲" : "▼") : "";
  return (
    <th
      class="sort-header"
      scope="col"
      onClick={() => onSort(sortKey)}
    >
      {label}
      {active && <span class="sort-indicator">{ind}</span>}
    </th>
  );
}

function InlinePostureBar({ pass, fail }: { pass: number; fail: number }) {
  const total = pass + fail;
  if (total <= 0) {
    return <span class="inline-bar" aria-hidden="true" />;
  }
  const pw = `${(pass / total) * 100}%`;
  const fw = `${(fail / total) * 100}%`;
  return (
    <span class="inline-bar" aria-hidden="true">
      <span class="inline-bar-pass" style={{ width: pw }} />
      <span class="inline-bar-fail" style={{ width: fw }} />
    </span>
  );
}

function formatEvidenceAge(iso: string): string {
  const t = new Date(iso).getTime();
  if (!iso || !Number.isFinite(t)) return "—";
  return relativeTime(iso);
}

export function InventoryView({ policyIdOverride }: InventoryViewProps = {}) {
  const [items, setItems] = useState<InventoryItem[]>([]);
  const [policies, setPolicies] = useState<PolicyOption[]>([]);
  const [programs, setPrograms] = useState<ProgramRow[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [sortKey, setSortKey] = useState<SortKey>("target_id");
  const [sortDir, setSortDir] = useState<"asc" | "desc">("asc");
  const [searchTerm, setSearchTerm] = useState("");
  const [chipState] = useState(() => createFilterChips());
  const inventoryFilterMap = chipState.filters.value;

  useEffect(() => {
    if (!policyIdOverride) return;
    const cur = chipState.filters.value.get(FILTER_KEYS.policy);
    if (cur !== policyIdOverride) {
      chipState.remove(FILTER_KEYS.policy);
      chipState.add(FILTER_KEYS.policy, policyIdOverride);
    }
  }, [policyIdOverride]);

  useEffect(() => {
    apiFetch("/api/policies")
      .then((r) => r.json())
      .then((data: PolicyOption[]) => setPolicies(Array.isArray(data) ? data : []))
      .catch(() => setPolicies([]));
  }, [viewInvalidation.value]);

  useEffect(() => {
    apiFetch("/api/programs")
      .then((r) => r.json())
      .then((data: ProgramRow[]) => setPrograms(Array.isArray(data) ? data : []))
      .catch(() => setPrograms([]));
  }, [viewInvalidation.value]);

  useEffect(() => {
    setLoading(true);
    setError(null);
    const q = buildInventoryQuery(chipState.filters.value);
    const path = q ? `/api/inventory?${q}` : "/api/inventory";
    apiFetch(path)
      .then((r) => {
        if (!r.ok) throw new Error(`HTTP ${r.status}`);
        return r.json() as Promise<InventoryItem[]>;
      })
      .then((data) => setItems(Array.isArray(data) ? data : []))
      .catch(() => {
        setItems([]);
        setError("Could not load inventory.");
      })
      .finally(() => setLoading(false));
  }, [viewInvalidation.value, inventoryFilterMap]);

  const policyOptions = useMemo(() => {
    return [...policies]
      .sort((a, b) => a.title.localeCompare(b.title))
      .map((p) => p.policy_id);
  }, [policies]);

  const programOptions = useMemo(() => {
    return [...programs].sort((a, b) => a.name.localeCompare(b.name)).map((p) => p.id);
  }, [programs]);

  const distinctTypes = useMemo(() => {
    const s = new Set<string>();
    for (const it of items) {
      if (it.target_type) s.add(it.target_type);
    }
    return [...s].sort();
  }, [items]);

  const distinctEnvs = useMemo(() => {
    const s = new Set<string>();
    for (const it of items) {
      if (it.environment) s.add(it.environment);
    }
    return [...s].sort();
  }, [items]);

  const filterFields = [
    { key: FILTER_KEYS.policy, label: "Policy", options: policyOptions },
    { key: FILTER_KEYS.program, label: "Program", options: programOptions },
    {
      key: FILTER_KEYS.targetType,
      label: "Target Type",
      options: distinctTypes,
    },
    {
      key: FILTER_KEYS.environment,
      label: "Environment",
      options: distinctEnvs,
    },
  ];

  const filteredItems = useMemo(() => {
    if (!searchTerm.trim()) return items;
    const term = searchTerm.toLowerCase();
    return items.filter((it) =>
      it.target_id.toLowerCase().includes(term) ||
      it.target_type?.toLowerCase().includes(term) ||
      it.environment?.toLowerCase().includes(term),
    );
  }, [items, searchTerm]);

  const sortedItems = useMemo(
    () => sortInventory(filteredItems, sortKey, sortDir),
    [filteredItems, sortKey, sortDir],
  );

  const onSort = (k: SortKey) => {
    if (sortKey === k) {
      setSortDir((d) => (d === "asc" ? "desc" : "asc"));
    } else {
      setSortKey(k);
      setSortDir("asc");
    }
  };

  const goToEvidence = (targetId: string) => {
    const policyFromChip = chipState.filters.value.get(FILTER_KEYS.policy);
    if (policyFromChip) {
      selectedPolicyId.value = policyFromChip;
    } else if (policyIdOverride) {
      selectedPolicyId.value = policyIdOverride;
    }
    selectedEvidenceTargetId.value = targetId;
    navigate("evidence");
  };

  const goToPolicies = () => {
    const policyFromChip = chipState.filters.value.get(FILTER_KEYS.policy);
    if (policyFromChip) {
      navigateToPolicy(policyFromChip, "requirements");
      return;
    }
    if (policyIdOverride) {
      navigateToPolicy(policyIdOverride, "requirements");
      return;
    }
    navigate("policies");
  };

  return (
    <section class="inventory-view-standalone">
      <header class="inventory-header">
        <h2>Inventory</h2>
        <span class="inventory-count">
          {loading ? "…" : `${filteredItems.length} ${filteredItems.length === 1 ? "target" : "targets"}`}
        </span>
      </header>

      <div class="evidence-filters inventory-filter-row">
        <input
          class="inventory-search"
          type="search"
          placeholder="Search targets..."
          value={searchTerm}
          onInput={(e) => setSearchTerm((e.target as HTMLInputElement).value)}
        />
        <AddFilterMenu fields={filterFields} chipState={chipState} />
      </div>

      <FilterChips state={chipState} />

      {error && (
        <div class="empty-state" role="alert">
          <p>{error}</p>
        </div>
      )}

      {loading ? (
        <div class="view-loading">Loading inventory...</div>
      ) : error ? null : sortedItems.length === 0 ? (
        <div class="empty-state">
          <p>No inventory targets found. Import policies and ingest evidence to see targets here.</p>
        </div>
      ) : (
        <table class="data-table" role="grid">
          <thead>
            <tr>
              <SortHeader
                label="Target ID"
                sortKey="target_id"
                activeKey={sortKey}
                dir={sortDir}
                onSort={onSort}
              />
              <SortHeader
                label="Type"
                sortKey="target_type"
                activeKey={sortKey}
                dir={sortDir}
                onSort={onSort}
              />
              <SortHeader
                label="Environment"
                sortKey="environment"
                activeKey={sortKey}
                dir={sortDir}
                onSort={onSort}
              />
              <SortHeader
                label="Policies"
                sortKey="policy_count"
                activeKey={sortKey}
                dir={sortDir}
                onSort={onSort}
              />
              <SortHeader
                label="Pass"
                sortKey="pass_count"
                activeKey={sortKey}
                dir={sortDir}
                onSort={onSort}
              />
              <SortHeader
                label="Fail"
                sortKey="fail_count"
                activeKey={sortKey}
                dir={sortDir}
                onSort={onSort}
              />
              <SortHeader
                label="Latest Evidence"
                sortKey="latest_evidence"
                activeKey={sortKey}
                dir={sortDir}
                onSort={onSort}
              />
            </tr>
          </thead>
          <tbody>
            {sortedItems.map((row) => (
              <tr key={row.target_id}>
                <td>
                  <button
                    type="button"
                    class="btn btn-sm btn-secondary inventory-target-id"
                    title={row.target_id}
                    onClick={() => goToEvidence(row.target_id)}
                  >
                    {row.target_id}
                  </button>
                  <InlinePostureBar pass={row.pass_count} fail={row.fail_count} />
                </td>
                <td>{row.target_type || "—"}</td>
                <td>{row.environment || "—"}</td>
                <td>
                  <button
                    type="button"
                    class="btn btn-sm btn-secondary"
                    onClick={goToPolicies}
                  >
                    {row.policy_count}
                  </button>
                </td>
                <td style={{ color: "var(--success)" }}>{row.pass_count}</td>
                <td style={{ color: "var(--error)" }}>{row.fail_count}</td>
                <td>{formatEvidenceAge(row.latest_evidence)}</td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </section>
  );
}
