// SPDX-License-Identifier: Apache-2.0

export interface AgentSkill {
  id: string;
  name: string;
  description: string;
  tags: string[];
}

export interface AgentCard {
  name: string;
  description: string;
  url: string;
  skills: AgentSkill[];
}

export async function fetchAgents(): Promise<AgentCard[]> {
  const res = await fetch("/api/agents");
  if (!res.ok) throw new Error(`Failed to fetch agents: ${res.status}`);
  return res.json();
}
