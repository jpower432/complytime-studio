// SPDX-License-Identifier: Apache-2.0

import { signal } from "@preact/signals";

const STORAGE_KEY = "complytime-studio-jobs";
const HISTORY_TTL_MS = 7 * 24 * 60 * 60 * 1000;

export interface ToolCall {
  name: string;
  id?: string;
  args?: Record<string, unknown>;
  result?: string;
  status: "pending" | "approved" | "rejected" | "executing" | "completed";
}

export interface Artifact { name: string; yaml: string; definition?: string }
export interface Message {
  role: string;
  content: string;
  timestamp: string;
  partial?: boolean;
  toolCall?: ToolCall;
}
export interface Job {
  id: string; taskId?: string; title: string; status: string;
  createdAt: string; updatedAt: string;
  artifacts: Artifact[]; messages: Message[];
  agentName?: string;
  contextArtifacts?: string[];
  acceptedAt?: string;
  acceptNote?: string;
}

function readStorage(): Job[] {
  try { const raw = localStorage.getItem(STORAGE_KEY); return raw ? JSON.parse(raw) : []; }
  catch { return []; }
}

function writeStorage(jobs: Job[]) {
  localStorage.setItem(STORAGE_KEY, JSON.stringify(jobs));
  jobsList.value = [...jobs];
}

export const jobsList = signal<Job[]>(readStorage());

export function loadJobs(): Job[] { return jobsList.value; }

const DEFAULT_AGENT = "studio-threat-modeler";

const ACTIVE_STATUSES = ["submitted", "working", "input-required", "ready"];
const HISTORY_STATUSES = ["accepted", "cancelled"];

export function isActiveStatus(status: string): boolean {
  return ACTIVE_STATUSES.includes(status);
}

export function isHistoryStatus(status: string): boolean {
  return HISTORY_STATUSES.includes(status);
}

export function createJob(taskId: string, title: string, agentName?: string): Job {
  const jobs = readStorage();
  const entry: Job = {
    id: taskId, title: title.length > 80 ? title.slice(0, 77) + "..." : title,
    status: "submitted", createdAt: new Date().toISOString(), updatedAt: new Date().toISOString(),
    artifacts: [], messages: [], agentName: agentName || DEFAULT_AGENT,
  };
  jobs.unshift(entry);
  writeStorage(jobs);
  return entry;
}

export function updateJob(taskId: string, updates: Partial<Job>): Job | null {
  const jobs = readStorage();
  const idx = jobs.findIndex((j) => j.id === taskId);
  if (idx === -1) return null;
  Object.assign(jobs[idx], updates, { updatedAt: new Date().toISOString() });
  writeStorage(jobs);
  return jobs[idx];
}

export function getJob(taskId: string): Job | null {
  return jobsList.value.find((j) => j.id === taskId) ?? null;
}

export function hasActiveJob(): boolean {
  return jobsList.value.some((j) => isActiveStatus(j.status));
}

export function getActiveJob(): Job | null {
  return jobsList.value.find((j) => isActiveStatus(j.status)) ?? null;
}

export function getJobAgent(job: Job): string {
  return job.agentName || DEFAULT_AGENT;
}

export function cancelJob(taskId: string): Job | null {
  return updateJob(taskId, { status: "cancelled" });
}

export function acceptJob(taskId: string, note: string): Job | null {
  return updateJob(taskId, {
    status: "accepted",
    acceptedAt: new Date().toISOString(),
    acceptNote: note,
  });
}

export function deleteJob(taskId: string): void {
  const jobs = readStorage();
  const filtered = jobs.filter((j) => j.id !== taskId);
  writeStorage(filtered);
}

export function purgeHistory(): void {
  const jobs = readStorage();
  const cutoff = Date.now() - HISTORY_TTL_MS;
  const filtered = jobs.filter((j) => {
    if (!isHistoryStatus(j.status)) return true;
    const age = j.acceptedAt || j.updatedAt;
    return new Date(age).getTime() > cutoff;
  });
  if (filtered.length !== jobs.length) writeStorage(filtered);
}

export function addArtifact(taskId: string, name: string, yaml: string, definition?: string) {
  const jobs = readStorage();
  const job = jobs.find((j) => j.id === taskId);
  if (!job) return;
  const existing = job.artifacts.findIndex((a) => a.name === name);
  if (existing >= 0) { job.artifacts[existing] = { name, yaml, definition }; }
  else { job.artifacts.push({ name, yaml, definition }); }
  job.updatedAt = new Date().toISOString();
  writeStorage(jobs);
}

export function addMessage(taskId: string, role: string, content: string, extra?: Partial<Message>) {
  const jobs = readStorage();
  const job = jobs.find((j) => j.id === taskId);
  if (!job) return;
  job.messages.push({ role, content, timestamp: new Date().toISOString(), ...extra });
  job.updatedAt = new Date().toISOString();
  writeStorage(jobs);
}

export function updateLastAgentMessage(taskId: string, content: string) {
  const jobs = readStorage();
  const job = jobs.find((j) => j.id === taskId);
  if (!job) return;
  for (let i = job.messages.length - 1; i >= 0; i--) {
    if (job.messages[i].role === "agent" && job.messages[i].partial) {
      job.messages[i].content = content;
      job.messages[i].timestamp = new Date().toISOString();
      writeStorage(jobs);
      return;
    }
  }
  job.messages.push({ role: "agent", content, timestamp: new Date().toISOString(), partial: true });
  writeStorage(jobs);
}

export function finalizeLastAgentMessage(taskId: string) {
  const jobs = readStorage();
  const job = jobs.find((j) => j.id === taskId);
  if (!job) return;
  for (let i = job.messages.length - 1; i >= 0; i--) {
    if (job.messages[i].role === "agent" && job.messages[i].partial) {
      job.messages[i].partial = false;
      writeStorage(jobs);
      return;
    }
  }
}

export function addToolCallMessage(taskId: string, toolCall: ToolCall) {
  addMessage(taskId, "tool", "", { toolCall });
}

export function updateToolCall(taskId: string, toolCallId: string, updates: Partial<ToolCall>) {
  const jobs = readStorage();
  const job = jobs.find((j) => j.id === taskId);
  if (!job) return;
  for (let i = job.messages.length - 1; i >= 0; i--) {
    const msg = job.messages[i];
    if (msg.toolCall && msg.toolCall.id === toolCallId) {
      Object.assign(msg.toolCall, updates);
      writeStorage(jobs);
      return;
    }
  }
}

export function timeAgo(isoString: string): string {
  const seconds = Math.floor((Date.now() - new Date(isoString).getTime()) / 1000);
  if (seconds < 60) return "just now";
  const minutes = Math.floor(seconds / 60);
  if (minutes < 60) return `${minutes}m ago`;
  const hours = Math.floor(minutes / 60);
  if (hours < 24) return `${hours}h ago`;
  const days = Math.floor(hours / 24);
  return `${days}d ago`;
}
