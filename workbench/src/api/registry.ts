// SPDX-License-Identifier: Apache-2.0

const REGISTRY_API = "/api/registry";

export interface Repository { name: string }
export interface Tag { name: string; digest?: string }
export interface ManifestLayer { mediaType: string; size: number; digest: string; annotations?: Record<string, string> }
export interface Manifest { mediaType: string; config: ManifestLayer; layers: ManifestLayer[]; annotations?: Record<string, string> }

export async function listRepositories(registryUrl: string): Promise<Repository[]> {
  const res = await fetch(`${REGISTRY_API}/repositories?registry=${encodeURIComponent(registryUrl)}`);
  if (!res.ok) throw new Error(`Failed to list repos: ${res.status}`);
  return res.json();
}

export async function listTags(reference: string): Promise<Tag[]> {
  const res = await fetch(`${REGISTRY_API}/tags?ref=${encodeURIComponent(reference)}`);
  if (!res.ok) throw new Error(`Failed to list tags: ${res.status}`);
  return res.json();
}

export async function fetchManifest(reference: string): Promise<Manifest> {
  const res = await fetch(`${REGISTRY_API}/manifest?ref=${encodeURIComponent(reference)}`);
  if (!res.ok) throw new Error(`Failed to fetch manifest: ${res.status}`);
  return res.json();
}

export async function fetchLayer(reference: string): Promise<string> {
  const res = await fetch(`${REGISTRY_API}/layer?ref=${encodeURIComponent(reference)}`);
  if (!res.ok) throw new Error(`Failed to fetch layer: ${res.status}`);
  return res.text();
}
