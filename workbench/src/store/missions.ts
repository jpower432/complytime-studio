// SPDX-License-Identifier: Apache-2.0

import { signal } from "@preact/signals";

const STORAGE_KEY = "complytime-studio-missions";

export interface Artifact { name: string; yaml: string; definition: string }
export interface Message { role: string; content: string; timestamp: string }
export interface Mission {
  id: string; title: string; status: string;
  createdAt: string; updatedAt: string;
  artifacts: Artifact[]; messages: Message[];
  agentName?: string;
}

function readStorage(): Mission[] {
  try { const raw = localStorage.getItem(STORAGE_KEY); return raw ? JSON.parse(raw) : []; }
  catch { return []; }
}

function writeStorage(missions: Mission[]) {
  localStorage.setItem(STORAGE_KEY, JSON.stringify(missions));
  missionsList.value = [...missions];
}

export const missionsList = signal<Mission[]>(readStorage());

export function loadMissions(): Mission[] { return missionsList.value; }

const DEFAULT_AGENT = "studio-threat-modeler";

export function createMission(taskId: string, title: string, agentName?: string): Mission {
  const missions = readStorage();
  const entry: Mission = {
    id: taskId, title: title.length > 80 ? title.slice(0, 77) + "..." : title,
    status: "submitted", createdAt: new Date().toISOString(), updatedAt: new Date().toISOString(),
    artifacts: [], messages: [], agentName: agentName || DEFAULT_AGENT,
  };
  missions.unshift(entry);
  writeStorage(missions);
  return entry;
}

export function updateMission(taskId: string, updates: Partial<Mission>): Mission | null {
  const missions = readStorage();
  const idx = missions.findIndex((m) => m.id === taskId);
  if (idx === -1) return null;
  Object.assign(missions[idx], updates, { updatedAt: new Date().toISOString() });
  writeStorage(missions);
  return missions[idx];
}

export function getMission(taskId: string): Mission | null {
  return missionsList.value.find((m) => m.id === taskId) ?? null;
}

export function hasActiveMission(): boolean {
  return missionsList.value.some((m) => ["submitted", "working", "input-required"].includes(m.status));
}

export function getActiveMission(): Mission | null {
  return missionsList.value.find((m) => ["submitted", "working", "input-required"].includes(m.status)) ?? null;
}

export function getMissionAgent(mission: Mission): string {
  return mission.agentName || DEFAULT_AGENT;
}

export function addArtifact(taskId: string, name: string, yaml: string, definition: string) {
  const missions = readStorage();
  const mission = missions.find((m) => m.id === taskId);
  if (!mission) return;
  const existing = mission.artifacts.findIndex((a) => a.name === name);
  if (existing >= 0) { mission.artifacts[existing] = { name, yaml, definition }; }
  else { mission.artifacts.push({ name, yaml, definition }); }
  mission.updatedAt = new Date().toISOString();
  writeStorage(missions);
}

export function addMessage(taskId: string, role: string, content: string) {
  const missions = readStorage();
  const mission = missions.find((m) => m.id === taskId);
  if (!mission) return;
  mission.messages.push({ role, content, timestamp: new Date().toISOString() });
  mission.updatedAt = new Date().toISOString();
  writeStorage(missions);
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
