// SPDX-License-Identifier: Apache-2.0

import { signal, computed } from "@preact/signals";
import { detectDefinition, inferArtifactName } from "../lib/artifact-detect";

const STORAGE_KEY = "complytime-studio-workspace";
const CAPACITY_WARN_RATIO = 0.8;
const ESTIMATED_QUOTA = 5 * 1024 * 1024;

export interface WorkspaceArtifact {
  name: string;
  yaml: string;
  definition: string;
}

interface WorkspaceState {
  artifacts: Record<string, WorkspaceArtifact>;
  activeName: string | null;
}

function readStorage(): WorkspaceState {
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    if (!raw) return { artifacts: {}, activeName: null };
    return JSON.parse(raw) as WorkspaceState;
  } catch {
    return { artifacts: {}, activeName: null };
  }
}

function writeStorage(state: WorkspaceState) {
  const json = JSON.stringify(state);
  localStorage.setItem(STORAGE_KEY, json);
  checkCapacity(json.length);
}

export const capacityWarning = signal("");

function checkCapacity(currentBytes: number) {
  const ratio = currentBytes / ESTIMATED_QUOTA;
  if (ratio >= CAPACITY_WARN_RATIO) {
    const pct = Math.round(ratio * 100);
    capacityWarning.value = `Workspace using ~${pct}% of local storage. Consider publishing or downloading artifacts.`;
  } else {
    capacityWarning.value = "";
  }
}

const _state = signal<WorkspaceState>(readStorage());

function persist() {
  writeStorage(_state.value);
}

function mutate(fn: (s: WorkspaceState) => WorkspaceState) {
  _state.value = fn({ ..._state.value, artifacts: { ..._state.value.artifacts } });
  persist();
}

export const allArtifacts = computed<WorkspaceArtifact[]>(() =>
  Object.values(_state.value.artifacts),
);

export const activeArtifactName = computed<string | null>(() => _state.value.activeName);

export const activeArtifact = computed<WorkspaceArtifact | null>(() => {
  const name = _state.value.activeName;
  if (!name) return null;
  return _state.value.artifacts[name] ?? null;
});

export function addArtifact(name: string, yaml: string, definition?: string) {
  const def = definition || detectDefinition(yaml) || "#ThreatCatalog";
  const resolved = name || inferArtifactName(yaml);
  mutate((s) => {
    s.artifacts[resolved] = { name: resolved, yaml, definition: def };
    s.activeName = resolved;
    return s;
  });
}

export function removeArtifact(name: string) {
  mutate((s) => {
    delete s.artifacts[name];
    if (s.activeName === name) {
      const keys = Object.keys(s.artifacts);
      s.activeName = keys.length > 0 ? keys[keys.length - 1] : null;
    }
    return s;
  });
}

export function activateArtifact(name: string) {
  if (!_state.value.artifacts[name]) return;
  mutate((s) => {
    s.activeName = name;
    return s;
  });
}

export function updateActiveContent(yaml: string) {
  const name = _state.value.activeName;
  if (!name || !_state.value.artifacts[name]) return;
  mutate((s) => {
    s.artifacts[name] = { ...s.artifacts[name], yaml };
    return s;
  });
}

export function updateActiveDefinition(def: string) {
  const name = _state.value.activeName;
  if (!name || !_state.value.artifacts[name]) return;
  mutate((s) => {
    s.artifacts[name] = { ...s.artifacts[name], definition: def };
    return s;
  });
}

export function getArtifactByName(name: string): WorkspaceArtifact | null {
  return _state.value.artifacts[name] ?? null;
}

export function renameArtifact(oldName: string, newName: string) {
  const trimmed = newName.trim();
  if (!trimmed || trimmed === oldName) return;
  if (_state.value.artifacts[trimmed]) return;
  mutate((s) => {
    const artifact = s.artifacts[oldName];
    if (!artifact) return s;
    delete s.artifacts[oldName];
    s.artifacts[trimmed] = { ...artifact, name: trimmed };
    if (s.activeName === oldName) s.activeName = trimmed;
    return s;
  });
}

export function getAllArtifactNames(): string[] {
  return Object.keys(_state.value.artifacts);
}
