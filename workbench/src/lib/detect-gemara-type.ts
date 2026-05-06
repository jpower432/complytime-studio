// SPDX-License-Identifier: Apache-2.0

export function detectGemaraMetadataType(content: string): string | null {
  const trimmed = content.trim();
  if (trimmed.startsWith("{")) {
    try {
      const o = JSON.parse(trimmed) as { metadata?: { type?: unknown } };
      const t = o.metadata?.type;
      if (typeof t === "string" && t.length > 0) {
        return t;
      }
    } catch {
      /* ignore */
    }
  }

  const lines = content.split(/\r?\n/);
  let inMetadata = false;
  for (let i = 0; i < lines.length; i++) {
    const stripped = lines[i].replace(/\s+$/, "");
    if (/^metadata:\s*$/.test(stripped)) {
      inMetadata = true;
      continue;
    }
    if (!inMetadata) {
      continue;
    }
    if (stripped.length > 0 && !lines[i].startsWith(" ") && !lines[i].startsWith("\t")) {
      break;
    }
    const m = stripped.match(/^\s+type:\s*['"]?([^'"\s#]+)['"]?/);
    if (m) {
      return m[1];
    }
  }
  return null;
}
