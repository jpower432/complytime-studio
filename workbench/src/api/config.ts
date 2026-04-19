// SPDX-License-Identifier: Apache-2.0

export interface PlatformConfig {
  github_org: string;
  github_repo: string;
}

let cached: PlatformConfig | null = null;

export async function fetchConfig(): Promise<PlatformConfig> {
  if (cached) return cached;
  const res = await fetch("/api/config");
  if (!res.ok) return { github_org: "", github_repo: "complytime-studio" };
  cached = await res.json();
  return cached!;
}

export function repoUrl(cfg: PlatformConfig): string {
  return `https://github.com/${cfg.github_org}/${cfg.github_repo}`;
}
