// SPDX-License-Identifier: Apache-2.0

import { apiFetch } from "./fetch";

export interface SaveResult {
  path: string;
}

export async function saveToWorkspace(filename: string, content: string): Promise<SaveResult> {
  const res = await apiFetch("/api/workspace/save", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ filename, content }),
  });
  if (!res.ok) {
    const err = await res.json().catch(() => ({ error: res.statusText }));
    throw new Error(err.error || `Save failed: ${res.status}`);
  }
  return res.json();
}
