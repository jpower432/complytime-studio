// SPDX-License-Identifier: Apache-2.0

import { platformUrl } from "./fetch";

export interface UserInfo {
  login: string;
  name: string;
  avatar_url: string;
  email: string;
  role: string;
}

export async function fetchMe(): Promise<UserInfo | null> {
  const res = await fetch(platformUrl("/auth/me"), { credentials: "include" });
  if (res.status === 401) return null;
  if (!res.ok) return null;
  return res.json();
}

export function redirectToLogin(): void {
  window.location.href = platformUrl("/auth/login");
}
