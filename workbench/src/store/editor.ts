// SPDX-License-Identifier: Apache-2.0
//
// Backward-compat layer. Components can still import editorContent, editorFilename,
// editorDefinition — they read/write the active workspace artifact.

import { computed, signal } from "@preact/signals";
import {
  activeArtifact,
  addArtifact as wsAdd,
  updateActiveContent,
  updateActiveDefinition,
} from "./workspace";
import { injectMappingRef, type MappingReference } from "../lib/yaml-inject";
import { detectDefinition, inferArtifactName } from "../lib/artifact-detect";

export type { MappingReference } from "../lib/yaml-inject";

export const editorContent = computed(() => activeArtifact.value?.yaml ?? "");
export const editorFilename = computed(() => activeArtifact.value?.name ?? "artifact.yaml");
export const editorDefinition = computed(() => activeArtifact.value?.definition ?? "#ThreatCatalog");

export interface Proposal {
  name: string;
  yaml: string;
  definition?: string;
}

export const pendingProposal = signal<Proposal | null>(null);

export function proposeArtifact(name: string, yaml: string, definition?: string) {
  pendingProposal.value = { name, yaml, definition };
}

export function applyProposal() {
  const p = pendingProposal.value;
  if (!p) return;
  const def = p.definition || detectDefinition(p.yaml) || "#ThreatCatalog";
  const resolved = p.name || inferArtifactName(p.yaml);
  wsAdd(resolved, p.yaml, def);
  pendingProposal.value = null;
}

export function dismissProposal() {
  pendingProposal.value = null;
}

export function setEditorArtifact(name: string, yaml: string, definition?: string) {
  const def = definition || detectDefinition(yaml) || "#ThreatCatalog";
  const resolved = name || inferArtifactName(yaml);
  wsAdd(resolved, yaml, def);
}

export function setEditorContent(yaml: string) {
  if (!activeArtifact.value && yaml.trim()) {
    const name = inferArtifactName(yaml) || "artifact.yaml";
    const def = detectDefinition(yaml) || "#ThreatCatalog";
    wsAdd(name, yaml, def);
    return;
  }
  updateActiveContent(yaml);
}

export function setEditorDefinition(def: string) {
  updateActiveDefinition(def);
}

export function injectMappingReference(ref: MappingReference) {
  const current = activeArtifact.value?.yaml ?? "";
  updateActiveContent(injectMappingRef(current, ref));
}
