// SPDX-License-Identifier: Apache-2.0

declare global {
  interface Window {
    __STUDIO_CONFIG__?: { platformUrl?: string };
  }
}

function getPlatformUrl(): string {
  return (
    window.__STUDIO_CONFIG__?.platformUrl ||
    import.meta.env.VITE_PLATFORM_URL ||
    ""
  );
}

export function platformUrl(path: string): string {
  const base = getPlatformUrl();
  if (!base) return path;
  return base.replace(/\/+$/, "") + path;
}

export async function apiFetch(input: RequestInfo | URL, init?: RequestInit): Promise<Response> {
  const url = typeof input === "string" ? platformUrl(input) : input;
  const res = await fetch(url, init);
  if (res.status === 401) {
    window.location.href = platformUrl("/auth/login");
    throw new Error("Session expired — redirecting to login");
  }
  return res;
}
