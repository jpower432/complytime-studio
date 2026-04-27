// SPDX-License-Identifier: Apache-2.0

import { signal, type Signal } from "@preact/signals";

export type FilterMap = Map<string, string>;

export interface FilterChipsState {
  filters: Signal<FilterMap>;
  add: (key: string, value: string) => void;
  remove: (key: string) => void;
  clear: () => void;
  has: (key: string) => boolean;
  matches: (record: Record<string, unknown>, fieldMap: Record<string, string>) => boolean;
}

export function createFilterChips(): FilterChipsState {
  const filters = signal<FilterMap>(new Map());

  const add = (key: string, value: string) => {
    const next = new Map(filters.value);
    next.set(key, value);
    filters.value = next;
  };

  const remove = (key: string) => {
    const next = new Map(filters.value);
    next.delete(key);
    filters.value = next;
  };

  const clear = () => {
    filters.value = new Map();
  };

  const has = (key: string) => filters.value.has(key);

  const matches = (record: Record<string, unknown>, fieldMap: Record<string, string>) => {
    for (const [chipKey, chipValue] of filters.value) {
      const recordField = fieldMap[chipKey];
      if (!recordField) continue;
      const actual = String(record[recordField] ?? "");
      if (actual.toLowerCase() !== chipValue.toLowerCase()) return false;
    }
    return true;
  };

  return { filters, add, remove, clear, has, matches };
}

export function FilterChips({ state }: { state: FilterChipsState }) {
  const entries = [...state.filters.value.entries()];
  if (entries.length === 0) return null;

  return (
    <div class="filter-chips">
      {entries.map(([key, value]) => (
        <span key={key} class="filter-chip">
          <span class="filter-chip-label">{key}: {value}</span>
          <button
            type="button"
            class="filter-chip-dismiss"
            aria-label={`Remove ${key} filter`}
            onClick={() => state.remove(key)}
          >
            &times;
          </button>
        </span>
      ))}
    </div>
  );
}
