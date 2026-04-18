// SPDX-License-Identifier: Apache-2.0

export async function apiFetch(input: RequestInfo | URL, init?: RequestInit): Promise<Response> {
  const res = await fetch(input, init);
  if (res.status === 401) {
    window.location.href = "/auth/login";
    throw new Error("Session expired — redirecting to login");
  }
  return res;
}
