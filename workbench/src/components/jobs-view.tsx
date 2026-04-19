// SPDX-License-Identifier: Apache-2.0
import { signal } from "@preact/signals";
import { useState, useEffect } from "preact/hooks";
import { navigate } from "../app";
import {
  jobsList, hasActiveJob, createJob, addMessage, deleteJob, updateJob,
  isActiveStatus, isHistoryStatus, timeAgo,
} from "../store/jobs";
import { allArtifacts } from "../store/workspace";
import { fetchAgents, type AgentCard } from "../api/agents";
import { StatusBadge } from "./status-badge";

const showNewDialog = signal(false);

export function JobsView() {
  const jobs = jobsList.value;
  const active = hasActiveJob();
  const activeJobs = jobs.filter((j) => isActiveStatus(j.status));
  const historyJobs = jobs.filter((j) => isHistoryStatus(j.status) || j.status === "failed");

  return (
    <div class="jobs-view">
      <div class="jobs-header">
        <h2>Jobs</h2>
        <button class="btn btn-primary" disabled={active} title={active ? "Complete the active job first" : undefined} onClick={() => (showNewDialog.value = true)}>+ New Job</button>
      </div>

      <div class="jobs-section">
        <h3 class="jobs-section-title">Active</h3>
        {activeJobs.length === 0 ? (
          <div class="empty-state">
            <h3>No active jobs</h3>
            <p>Start a job to analyze threats, author controls, or manage Gemara artifacts.</p>
          </div>
        ) : (
          activeJobs.map((j) => (
            <div key={j.id} class="job-card" onClick={() => navigate("workspace", j.id)}>
              <div class="job-card-header">
                <span class="job-card-title">{j.title}</span>
                <StatusBadge status={j.status} />
              </div>
              <div class="job-card-meta">
                <span>{j.artifacts.length} artifact{j.artifacts.length !== 1 ? "s" : ""}</span>
                <span>{timeAgo(j.updatedAt)}</span>
              </div>
            </div>
          ))
        )}
      </div>

      {historyJobs.length > 0 && (
        <div class="jobs-section jobs-section-history">
          <h3 class="jobs-section-title">Recent</h3>
          {historyJobs.map((j) => (
            <div key={j.id} class="job-card job-card-history" onClick={() => navigate("workspace", j.id)}>
              <div class="job-card-header">
                <span class="job-card-title">{j.title}</span>
                <StatusBadge status={j.status} />
                <button
                  class="btn btn-secondary btn-sm job-card-delete"
                  title="Delete"
                  onClick={(e) => { e.stopPropagation(); deleteJob(j.id); }}
                >&times;</button>
              </div>
              <div class="job-card-meta">
                {j.acceptNote && <span class="job-card-note">"{j.acceptNote}"</span>}
                <span>{timeAgo(j.acceptedAt || j.updatedAt)}</span>
              </div>
            </div>
          ))}
        </div>
      )}

      {showNewDialog.value && <NewJobDialog onClose={() => (showNewDialog.value = false)} />}
    </div>
  );
}

const DEFAULT_AGENT = "studio-threat-modeler";

const CONTEXT_SIZE_CAP = 100 * 1024;

function NewJobDialog({ onClose }: { onClose: () => void }) {
  const [error, setError] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [agents, setAgents] = useState<AgentCard[]>([]);
  const [selectedAgent, setSelectedAgent] = useState(DEFAULT_AGENT);
  const wsArtifacts = allArtifacts.value;
  const [selectedContext, setSelectedContext] = useState<Record<string, boolean>>({});
  let textareaRef: HTMLTextAreaElement | null = null;

  useEffect(() => {
    fetchAgents()
      .then((cards) => {
        setAgents(cards);
        if (cards.length > 0) setSelectedAgent(cards[0].name);
      })
      .catch(() => {});
  }, []);

  function toggleContext(name: string) {
    setSelectedContext((prev) => ({ ...prev, [name]: !prev[name] }));
  }

  function selectedContextNames(): string[] {
    return Object.entries(selectedContext).filter(([, v]) => v).map(([k]) => k);
  }

  function contextSize(): number {
    return selectedContextNames().reduce((sum, name) => {
      const a = wsArtifacts.find((w) => w.name === name);
      return sum + (a ? new TextEncoder().encode(a.yaml).length : 0);
    }, 0);
  }

  function handleSubmit() {
    const text = textareaRef?.value.trim();
    if (!text) { setError("Describe what you want the agent to do."); return; }
    const size = contextSize();
    if (size > CONTEXT_SIZE_CAP) {
      setError(`Selected context is ${Math.round(size / 1024)} KB (limit: ${CONTEXT_SIZE_CAP / 1024} KB). Deselect some artifacts.`);
      return;
    }
    setSubmitting(true);
    setError("");
    const localId = crypto.randomUUID();
    const names = selectedContextNames();
    const job = createJob(localId, text, selectedAgent);
    if (names.length > 0) updateJob(localId, { contextArtifacts: names });
    addMessage(localId, "user", text);
    onClose();
    navigate("workspace", job.id);
  }

  return (
    <div class="dialog-overlay" onClick={onClose}>
      <div class="dialog" onClick={(e) => e.stopPropagation()}>
        <h3>New Job</h3>
        {agents.length > 0 && (
          <div class="agent-picker">
            <label class="dialog-label">Specialist</label>
            {agents.map((a) => (
              <div key={a.name} class={`agent-card ${selectedAgent === a.name ? "selected" : ""}`} onClick={() => setSelectedAgent(a.name)}>
                <span class="agent-card-name">{a.name}</span>
                <span class="agent-card-desc">{a.description}</span>
              </div>
            ))}
          </div>
        )}
        <textarea ref={(el) => { textareaRef = el; if (el && !agents.length) el.focus(); }} placeholder='Describe what you want to do, e.g. "Analyze threats for github.com/kyverno/kyverno"' onKeyDown={(e) => { if (e.key === "Enter" && !e.shiftKey) { e.preventDefault(); handleSubmit(); } }} />
        {wsArtifacts.length > 0 && (
          <div class="context-picker">
            <label class="dialog-label">Include workspace artifacts as context</label>
            {wsArtifacts.map((a) => (
              <label key={a.name} class="context-picker-row">
                <input type="checkbox" checked={!!selectedContext[a.name]} onChange={() => toggleContext(a.name)} />
                <span class="artifact-name-mono">{a.name}</span>
              </label>
            ))}
            {contextSize() > CONTEXT_SIZE_CAP * 0.8 && (
              <div class="dialog-warn">Context size: {Math.round(contextSize() / 1024)} KB / {CONTEXT_SIZE_CAP / 1024} KB</div>
            )}
          </div>
        )}
        {error && <div class="dialog-error">{error}</div>}
        <div class="dialog-actions">
          <button class="btn btn-secondary" onClick={onClose}>Cancel</button>
          <button class="btn btn-primary" disabled={submitting} onClick={handleSubmit}>{submitting ? "Starting..." : "Start Job"}</button>
        </div>
      </div>
    </div>
  );
}
