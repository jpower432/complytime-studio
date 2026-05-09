// SPDX-License-Identifier: Apache-2.0

import { apiFetch } from "./fetch";

export interface AgentCard {
  id: string;
  name: string;
  description: string;
  role?: string;
  framework?: string;
  status?: string;
  tools?: string[];
  examples?: string[];
  skills?: Array<{ id: string; name: string; description: string; tags?: string[] }>;
  model?: { provider?: string; name?: string };
}

let cached: AgentCard[] | null = null;

export async function fetchAgents(): Promise<AgentCard[]> {
  if (cached) return cached;
  try {
    const res = await apiFetch("/api/agents");
    if (!res.ok) return fallback();
    cached = await res.json();
    return cached!;
  } catch {
    return fallback();
  }
}

function fallback(): AgentCard[] {
  return [{ id: "studio-assistant", name: "Studio Assistant", description: "Compliance assistant" }];
}
