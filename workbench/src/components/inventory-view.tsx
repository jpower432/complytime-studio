// SPDX-License-Identifier: Apache-2.0

import { useState, useEffect } from "preact/hooks";
import { apiFetch } from "../api/fetch";

interface EvidenceRow {
  target_id: string;
  target_name?: string;
  control_id: string;
  eval_result: string;
}

interface TargetSummary {
  targetId: string;
  name: string;
  total: number;
  passed: number;
  failed: number;
  other: number;
}

interface ControlSummary {
  controlId: string;
  total: number;
  passed: number;
  passRate: number;
}

function groupByTarget(rows: EvidenceRow[]): TargetSummary[] {
  const map = new Map<string, TargetSummary>();
  for (const r of rows) {
    let entry = map.get(r.target_id);
    if (!entry) {
      entry = { targetId: r.target_id, name: r.target_name || r.target_id, total: 0, passed: 0, failed: 0, other: 0 };
      map.set(r.target_id, entry);
    }
    entry.total++;
    const result = r.eval_result?.toLowerCase();
    if (result === "passed") entry.passed++;
    else if (result === "failed") entry.failed++;
    else entry.other++;
  }
  return [...map.values()].sort((a, b) => b.total - a.total);
}

function groupByControl(rows: EvidenceRow[]): ControlSummary[] {
  const map = new Map<string, { total: number; passed: number }>();
  for (const r of rows) {
    let entry = map.get(r.control_id);
    if (!entry) {
      entry = { total: 0, passed: 0 };
      map.set(r.control_id, entry);
    }
    entry.total++;
    if (r.eval_result?.toLowerCase() === "passed") entry.passed++;
  }
  return [...map.entries()]
    .map(([controlId, s]) => ({
      controlId,
      total: s.total,
      passed: s.passed,
      passRate: s.total > 0 ? Math.round((s.passed / s.total) * 100) : 0,
    }))
    .sort((a, b) => a.controlId.localeCompare(b.controlId));
}

export function InventoryView({ policyIdOverride, onTargetClick, onControlClick }: {
  policyIdOverride?: string;
  onTargetClick?: (targetId: string, targetName: string) => void;
  onControlClick?: (controlId: string) => void;
} = {}) {
  const [rows, setRows] = useState<EvidenceRow[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    if (!policyIdOverride) { setLoading(false); return; }
    const params = new URLSearchParams({ policy_id: policyIdOverride, limit: "1000" });
    apiFetch(`/api/evidence?${params}`)
      .then((r) => r.json())
      .then(setRows)
      .catch(() => setRows([]))
      .finally(() => setLoading(false));
  }, [policyIdOverride]);

  if (loading) return <div class="view-loading">Loading inventory...</div>;
  if (rows.length === 0) {
    return (
      <div class="empty-state">
        <p>No evidence found for this policy. Inventory is derived from evidence records.</p>
      </div>
    );
  }

  const targets = groupByTarget(rows);
  const controls = groupByControl(rows);

  return (
    <section class="inventory-view">
      <div class="inventory-grid">
        <div class="inventory-section">
          <h3>Targets ({targets.length})</h3>
          <div class="inventory-list">
            {targets.map((t) => (
              <div
                key={t.targetId}
                class={`inventory-item ${onTargetClick ? "inventory-item-clickable" : ""}`}
                onClick={onTargetClick ? () => onTargetClick(t.targetId, t.name) : undefined}
              >
                <span class="inventory-item-name" title={t.targetId}>{t.name}</span>
                <span class="inventory-item-stats">{t.total} records</span>
                <div class="posture-bar" style={{ width: "80px" }}>
                  {t.total > 0 && (
                    <>
                      <div class="bar-pass" style={{ width: `${(t.passed / t.total) * 100}%` }} />
                      <div class="bar-fail" style={{ width: `${(t.failed / t.total) * 100}%` }} />
                      <div class="bar-other" style={{ width: `${(t.other / t.total) * 100}%` }} />
                    </>
                  )}
                </div>
              </div>
            ))}
          </div>
        </div>

        <div class="inventory-section">
          <h3>Controls ({controls.length})</h3>
          <div class="inventory-list">
            {controls.map((c) => (
              <div
                key={c.controlId}
                class={`inventory-item ${onControlClick ? "inventory-item-clickable" : ""}`}
                onClick={onControlClick ? () => onControlClick(c.controlId) : undefined}
              >
                <span class="inventory-item-name mono">{c.controlId}</span>
                <span class="inventory-item-stats">{c.total} records</span>
                <span class="inventory-item-stats">{c.passRate}% pass</span>
              </div>
            ))}
          </div>
        </div>
      </div>
    </section>
  );
}
