// SPDX-License-Identifier: Apache-2.0

export const STALE_THRESHOLD_DAYS = 30;
const MS_PER_DAY = 86_400_000;

export type FreshnessBucket = "current" | "aging" | "stale" | "very-stale";

export const FREQUENCY_TO_DAYS: Record<string, number> = {
  daily: 1,
  weekly: 7,
  monthly: 30,
  quarterly: 90,
  annually: 365,
  "on-demand": Infinity,
};

export function frequencyToDays(frequency: string): number {
  return FREQUENCY_TO_DAYS[frequency.toLowerCase()] ?? STALE_THRESHOLD_DAYS;
}

export function ageDays(iso: string | undefined): number {
  if (!iso) return Infinity;
  return (Date.now() - new Date(iso).getTime()) / MS_PER_DAY;
}

export function isStale(iso: string | undefined): boolean {
  return ageDays(iso) > STALE_THRESHOLD_DAYS;
}

export function freshnessClass(iso: string | undefined): string {
  const days = ageDays(iso);
  if (!isFinite(days)) return "freshness-none";
  if (days <= 7) return "freshness-current";
  if (days <= STALE_THRESHOLD_DAYS) return "freshness-aging";
  return "freshness-stale";
}

export function freshnessFromFrequency(collectedAt: string, cycleDays: number): FreshnessBucket {
  if (!isFinite(cycleDays) || cycleDays === Infinity) return "current";
  const days = ageDays(collectedAt);
  if (!isFinite(days)) return "very-stale";
  if (days <= cycleDays) return "current";
  if (days <= cycleDays * 2) return "aging";
  if (days <= cycleDays * 3) return "stale";
  return "very-stale";
}

export function freshnessRowClass(bucket: FreshnessBucket): string {
  return `freshness-row-${bucket}`;
}

export function defaultFreshnessBucket(collectedAt: string): FreshnessBucket {
  const days = ageDays(collectedAt);
  if (days <= 7) return "current";
  if (days <= 30) return "aging";
  if (days <= 90) return "stale";
  return "very-stale";
}

export function parsePolicyFrequencies(contentYaml: string): Map<string, number> {
  const freqMap = new Map<string, number>();
  const lines = contentYaml.split("\n");
  let currentReqId = "";
  for (const line of lines) {
    const reqMatch = /requirement-id:\s*(\S+)/.exec(line);
    if (reqMatch) currentReqId = reqMatch[1];
    const freqMatch = /frequency:\s*(\S+)/.exec(line);
    if (freqMatch && currentReqId) {
      freqMap.set(currentReqId, frequencyToDays(freqMatch[1]));
    }
  }
  return freqMap;
}

export function relativeTime(iso: string): string {
  const diff = Date.now() - new Date(iso).getTime();
  if (diff < 0) return "just now";
  const mins = Math.floor(diff / 60_000);
  if (mins < 1) return "just now";
  if (mins < 60) return `${mins}m ago`;
  const hrs = Math.floor(mins / 60);
  if (hrs < 24) return `${hrs}h ago`;
  const days = Math.floor(hrs / 24);
  if (days < 30) return `${days}d ago`;
  const months = Math.floor(days / 30);
  return `${months}mo ago`;
}
