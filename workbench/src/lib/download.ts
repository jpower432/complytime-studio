// SPDX-License-Identifier: Apache-2.0

export function downloadYaml(content: string, filename: string): void {
  const blob = new Blob([content], { type: "application/x-yaml" });
  const url = URL.createObjectURL(blob);
  const a = document.createElement("a");
  a.href = url;
  a.download = filename;
  a.click();
  URL.revokeObjectURL(url);
}

export function auditLogFilename(policyId: string, auditStart: string): string {
  const date = auditStart ? auditStart.slice(0, 10) : new Date().toISOString().slice(0, 10);
  const slug = policyId || "audit-log";
  return `${slug}-${date}.yaml`;
}
