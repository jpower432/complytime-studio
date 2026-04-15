// SPDX-License-Identifier: Apache-2.0

export function detectDefinition(yaml: string): string {
  if (/^threats:/m.test(yaml) || /^\s+threats:/m.test(yaml)) return "#ThreatCatalog";
  if (/^controls:/m.test(yaml) || /^\s+controls:/m.test(yaml)) return "#ControlCatalog";
  if (/^guidances:/m.test(yaml) || /^\s+guidances:/m.test(yaml)) return "#GuidanceCatalog";
  if (/^capabilities:/m.test(yaml) || /^\s+capabilities:/m.test(yaml)) return "#CapabilityCatalog";
  if (/^results:/m.test(yaml) || /^\s+results:/m.test(yaml)) return "#AuditLog";
  if (/^policy:/m.test(yaml) || /^\s+policy:/m.test(yaml)) return "#Policy";
  if (/^risks:/m.test(yaml) || /^\s+risks:/m.test(yaml)) return "#RiskCatalog";
  if (/^mappings:/m.test(yaml) || /^\s+mappings:/m.test(yaml)) return "#MappingDocument";
  return "#ThreatCatalog";
}

export function inferArtifactName(yaml: string): string {
  if (/^threats:/m.test(yaml)) return "threat-catalog.yaml";
  if (/^controls:/m.test(yaml)) return "control-catalog.yaml";
  if (/^capabilities:/m.test(yaml)) return "capability-catalog.yaml";
  if (/^guidances:/m.test(yaml)) return "guidance-catalog.yaml";
  if (/^results:/m.test(yaml)) return "audit-log.yaml";
  if (/^policy:/m.test(yaml)) return "policy.yaml";
  if (/^risks:/m.test(yaml)) return "risk-catalog.yaml";
  if (/^mappings:/m.test(yaml)) return "mapping-document.yaml";
  return "artifact.yaml";
}

export function isGemaraArtifact(yaml: string): boolean {
  return /^(threats|controls|capabilities|guidances|policy|metadata|results|risks|mappings):/m.test(yaml);
}

export interface ExtractedArtifact { name: string; yaml: string; definition: string }

export function extractArtifacts(text: string): { text: string; artifacts: ExtractedArtifact[] } {
  const artifacts: ExtractedArtifact[] = [];
  const cleaned = text.replace(/```ya?ml\n([\s\S]*?)```/g, (match, yamlContent: string) => {
    if (isGemaraArtifact(yamlContent)) {
      const name = inferArtifactName(yamlContent);
      const definition = detectDefinition(yamlContent);
      artifacts.push({ name, yaml: yamlContent.trim(), definition });
      return `_[Artifact: ${name}]_`;
    }
    return match;
  });
  return { text: cleaned, artifacts };
}
