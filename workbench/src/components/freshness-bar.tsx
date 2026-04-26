// SPDX-License-Identifier: Apache-2.0

import type { FreshnessBucket } from "../lib/freshness";
import type { FilterChipsState } from "./filter-chip";

interface BucketCount {
  bucket: FreshnessBucket;
  count: number;
  label: string;
}

const BUCKET_LABELS: Record<FreshnessBucket, string> = {
  current: "Current",
  aging: "Aging",
  stale: "Stale",
  "very-stale": "Very Stale",
};

export function FreshnessBar({
  buckets,
  chipState,
}: {
  buckets: Record<FreshnessBucket, number>;
  chipState: FilterChipsState;
}) {
  const total = buckets.current + buckets.aging + buckets.stale + buckets["very-stale"];
  if (total === 0) return null;

  const segments: BucketCount[] = (
    ["current", "aging", "stale", "very-stale"] as FreshnessBucket[]
  )
    .filter((b) => buckets[b] > 0)
    .map((b) => ({ bucket: b, count: buckets[b], label: BUCKET_LABELS[b] }));

  return (
    <div class="freshness-bar-container">
      <div class="freshness-bar-segments" role="img" aria-label="Evidence freshness distribution">
        {segments.map((s) => (
          <div
            key={s.bucket}
            class={`freshness-segment freshness-segment-${s.bucket}`}
            style={{ width: `${(s.count / total) * 100}%` }}
            title={s.label}
            onClick={() => chipState.add("Freshness", s.label)}
          />
        ))}
      </div>
    </div>
  );
}
