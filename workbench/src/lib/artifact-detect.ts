// SPDX-License-Identifier: Apache-2.0

const METADATA_TYPE_RE = /^\s*type:\s*["']?(\w+)["']?\s*$/m;
const METADATA_BLOCK_RE = /^metadata:\s*\n((?:\s+.*\n)*)/m;

const TYPE_TO_DEFINITION: Record<string, string> = {
  ThreatCatalog: "#ThreatCatalog",
  ControlCatalog: "#ControlCatalog",
  GuidanceCatalog: "#GuidanceCatalog",
  CapabilityCatalog: "#CapabilityCatalog",
  AuditLog: "#AuditLog",
  EvaluationLog: "#EvaluationLog",
  Policy: "#Policy",
  RiskCatalog: "#RiskCatalog",
  MappingDocument: "#MappingDocument",
};

export const ALL_DEFINITIONS = Object.values(TYPE_TO_DEFINITION);

export function detectDefinition(yaml: string): string | null {
  const metaBlock = METADATA_BLOCK_RE.exec(yaml);
  if (metaBlock) {
    const typeMatch = METADATA_TYPE_RE.exec(metaBlock[1]);
    if (typeMatch && TYPE_TO_DEFINITION[typeMatch[1]]) {
      return TYPE_TO_DEFINITION[typeMatch[1]];
    }
  }
  return null;
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

export interface ExtractedArtifact { name: string; yaml: string; definition: string | null }

export function extractArtifacts(text: string): { text: string; artifacts: ExtractedArtifact[] } {
  const artifacts: ExtractedArtifact[] = [];

  // 1. Fenced code blocks (```yaml or ```)
  const cleaned = text.replace(/```(?:ya?ml)?\n([\s\S]*?)```/g, (match, yamlContent: string) => {
    if (isGemaraArtifact(yamlContent)) {
      const name = inferArtifactName(yamlContent);
      const definition = detectDefinition(yamlContent);
      artifacts.push({ name, yaml: yamlContent.trim(), definition });
      return `_[Artifact: ${name}]_`;
    }
    return match;
  });

  // 2. Raw YAML: if no fenced artifacts found, scan for inline Gemara YAML
  //    starting at a top-level key (e.g., "metadata:" or "threats:")
  if (artifacts.length === 0) {
    const rawMatch = cleaned.match(
      /^(metadata:|threats:|controls:|capabilities:|guidances:|policy:|results:|risks:|mappings:).*(?:\n(?:[ \t].*|[-#].*|$))*/m
    );
    if (rawMatch && rawMatch[0].split("\n").length >= 3) {
      const rawYaml = rawMatch[0].trim();
      if (isGemaraArtifact(rawYaml)) {
        const name = inferArtifactName(rawYaml);
        const definition = detectDefinition(rawYaml);
        artifacts.push({ name, yaml: rawYaml, definition });
        const remaining = cleaned.replace(rawMatch[0], `_[Artifact: ${name}]_`);
        return { text: remaining, artifacts };
      }
    }
  }

  return { text: cleaned, artifacts };
}
