// SPDX-License-Identifier: Apache-2.0

export interface UserInfo {
  login: string;
  name: string;
  avatar_url: string;
  email: string;
  role: string;
}

export async function fetchMe(): Promise<UserInfo | null> {
  const res = await fetch("/auth/me");
  if (res.status === 401) return null;
  if (!res.ok) return null;
  return res.json();
}

export function redirectToLogin(): void {
  window.location.href = "/oauth2/sign_in";
}
