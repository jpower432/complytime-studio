// SPDX-License-Identifier: Apache-2.0

import { useState } from "preact/hooks";
import type { FilterChipsState } from "./filter-chip";

interface FilterField {
  key: string;
  label: string;
  options: string[] | (() => string[]);
}

export function AddFilterMenu({
  fields,
  chipState,
}: {
  fields: FilterField[];
  chipState: FilterChipsState;
}) {
  const [open, setOpen] = useState(false);
  const [selecting, setSelecting] = useState<FilterField | null>(null);

  const available = fields.filter((f) => !chipState.has(f.key));
  if (available.length === 0 && !selecting) return null;

  const handleFieldPick = (field: FilterField) => {
    setSelecting(field);
  };

  const handleValuePick = (value: string) => {
    if (selecting) {
      chipState.add(selecting.key, value);
    }
    setSelecting(null);
    setOpen(false);
  };

  const cancel = () => {
    setSelecting(null);
    setOpen(false);
  };

  if (selecting) {
    const opts = typeof selecting.options === "function" ? selecting.options() : selecting.options;
    return (
      <div class="add-filter-wrapper">
        <div class="add-filter-value-row">
          <select
            autoFocus
            onChange={(e) => {
              const v = (e.target as HTMLSelectElement).value;
              if (v) handleValuePick(v);
            }}
            onBlur={cancel}
          >
            <option value="">{selecting.label}...</option>
            {opts.map((o) => (
              <option key={o} value={o}>{o}</option>
            ))}
          </select>
        </div>
      </div>
    );
  }

  return (
    <div class="add-filter-wrapper">
      <button
        class="btn btn-sm btn-secondary"
        onClick={() => setOpen(!open)}
        type="button"
      >
        + Filter
      </button>
      {open && (
        <div class="add-filter-menu" onMouseLeave={() => setOpen(false)}>
          {available.map((f) => (
            <button
              key={f.key}
              class="add-filter-option"
              type="button"
              onClick={() => handleFieldPick(f)}
            >
              {f.label}
            </button>
          ))}
        </div>
      )}
    </div>
  );
}
