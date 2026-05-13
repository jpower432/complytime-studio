// SPDX-License-Identifier: Apache-2.0

export interface PlatformConfig {
  github_org: string;
  github_repo: string;
  model_provider: string;
  model_name: string;
  auto_persist_artifacts: string;
}

let cached: PlatformConfig | null = null;

export async function fetchConfig(): Promise<PlatformConfig> {
  if (cached) return cached;
  const res = await fetch("/api/config");
  if (!res.ok) return { github_org: "", github_repo: "complytime-studio", model_provider: "", model_name: "", auto_persist_artifacts: "true" };
  cached = await res.json();
  return cached!;
}

export function repoUrl(cfg: PlatformConfig): string {
  return `https://github.com/${cfg.github_org}/${cfg.github_repo}`;
}
