// SPDX-License-Identifier: Apache-2.0

import { signal } from "@preact/signals";
import { injectMappingRef, type MappingReference } from "../lib/yaml-inject";
import { detectDefinition, inferArtifactName } from "../lib/artifact-detect";

export type { MappingReference } from "../lib/yaml-inject";

export const editorContent = signal("");
export const editorFilename = signal("artifact.yaml");
export const editorDefinition = signal("#ThreatCatalog");

export function setEditorArtifact(name: string, yaml: string, definition?: string) {
  editorContent.value = yaml;
  editorFilename.value = name || inferArtifactName(yaml);
  editorDefinition.value = definition || detectDefinition(yaml);
}

export function injectMappingReference(ref: MappingReference) {
  editorContent.value = injectMappingRef(editorContent.value, ref);
}
