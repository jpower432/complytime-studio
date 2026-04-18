// SPDX-License-Identifier: Apache-2.0

export interface MappingReference {
  id: string;
  title: string;
  version: string;
  url: string;
  description?: string;
}

function formatRef(ref: MappingReference, indent: string): string {
  let entry = `${indent}- id: ${ref.id}\n`;
  entry += `${indent}  title: ${ref.title}\n`;
  entry += `${indent}  version: "${ref.version}"\n`;
  entry += `${indent}  url: ${ref.url}\n`;
  if (ref.description) {
    entry += `${indent}  description: |\n`;
    for (const line of ref.description.trim().split("\n")) {
      entry += `${indent}    ${line}\n`;
    }
  }
  return entry;
}

export function injectMappingRef(yaml: string, ref: MappingReference): string {
  const mrMatch = yaml.match(/^( *)mapping-references:\s*$/m);
  if (mrMatch) {
    const baseIndent = mrMatch[1];
    const entryIndent = baseIndent + "  ";
    const mrLineEnd = (mrMatch.index ?? 0) + mrMatch[0].length;

    let insertPos = mrLineEnd;
    const rest = yaml.slice(mrLineEnd);
    const entryPattern = new RegExp(`^${entryIndent}- `, "m");
    const lines = rest.split("\n");
    let offset = 0;
    let lastEntryEnd = 0;
    let inEntry = false;

    for (const line of lines) {
      const lineWithNewline = line + "\n";
      if (line.match(entryPattern) || (inEntry && line.match(new RegExp(`^${entryIndent}\\s`)))) {
        inEntry = true;
        lastEntryEnd = offset + lineWithNewline.length;
      } else if (inEntry && line.trim() === "") {
        lastEntryEnd = offset + lineWithNewline.length;
      } else if (inEntry) {
        break;
      }
      offset += lineWithNewline.length;
    }

    if (lastEntryEnd > 0) {
      insertPos = mrLineEnd + lastEntryEnd;
    } else {
      insertPos = mrLineEnd;
    }

    const newEntry = (lastEntryEnd > 0 ? "" : "\n") + formatRef(ref, entryIndent);
    return yaml.slice(0, insertPos) + newEntry + yaml.slice(insertPos);
  }

  const metaMatch = yaml.match(/^( *)metadata:\s*$/m);
  if (metaMatch) {
    const baseIndent = metaMatch[1];
    const childIndent = baseIndent + "  ";
    const entryIndent = childIndent + "  ";
    const metaLineEnd = (metaMatch.index ?? 0) + metaMatch[0].length;

    const rest = yaml.slice(metaLineEnd);
    const lines = rest.split("\n");
    let offset = 0;
    let lastChildEnd = 0;

    for (const line of lines) {
      const lineWithNewline = line + "\n";
      if (line.trim() === "" || line.match(new RegExp(`^${childIndent}\\S`)) || line.match(new RegExp(`^${childIndent} `))) {
        lastChildEnd = offset + lineWithNewline.length;
      } else if (line.trim() !== "" && !line.startsWith(childIndent)) {
        break;
      }
      offset += lineWithNewline.length;
    }

    const insertPos = metaLineEnd + lastChildEnd;
    const block = `\n${childIndent}mapping-references:\n` + formatRef(ref, entryIndent);
    return yaml.slice(0, insertPos) + block + yaml.slice(insertPos);
  }

  const block = `metadata:\n  mapping-references:\n` + formatRef(ref, "    ");
  return block + yaml;
}

export function extractMappingRefFromYaml(yaml: string): MappingReference | null {
  const idMatch = yaml.match(/^[ \t]*(?:metadata:[\s\S]*?)?id:\s*(.+)$/m);
  const versionMatch = yaml.match(/^[ \t]*version:\s*(.+)$/m);
  const titleMatch = yaml.match(/^title:\s*(.+)$/m);
  const descMatch = yaml.match(/^[ \t]*description:\s*(?:\|[\s\S]*?\n([\s\S]*?)(?=\n\S)|(.*))$/m);

  if (!titleMatch) return null;

  let id = "";
  const lines = yaml.split("\n");
  let inMetadata = false;
  for (const line of lines) {
    if (/^metadata:\s*$/.test(line)) { inMetadata = true; continue; }
    if (inMetadata && /^\s+id:\s*(.+)$/.test(line)) {
      id = line.match(/^\s+id:\s*(.+)$/)![1].trim();
      break;
    }
    if (inMetadata && /^\S/.test(line)) break;
  }

  if (!id && idMatch) {
    id = idMatch[1].trim();
  }

  if (!id) return null;

  const version = versionMatch ? versionMatch[1].trim().replace(/^["']|["']$/g, "") : "0.0.0";
  const title = titleMatch[1].trim();
  let description: string | undefined;
  if (descMatch) {
    description = (descMatch[1] || descMatch[2] || "").trim();
    if (!description) description = undefined;
  }

  return { id, title, version, url: "", description };
}
