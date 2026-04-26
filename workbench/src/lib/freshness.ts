// SPDX-License-Identifier: Apache-2.0

export const STALE_THRESHOLD_DAYS = 30;
const MS_PER_DAY = 86_400_000;

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

export function evidenceRecencyClass(collectedAt: string): string {
  const days = ageDays(collectedAt);
  if (days <= 7) return "recency-current";
  if (days <= 30) return "recency-aging";
  if (days <= 90) return "recency-stale";
  return "recency-very-stale";
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
