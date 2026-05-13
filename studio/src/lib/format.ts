// SPDX-License-Identifier: Apache-2.0

/**
 * Format a date string as "27 Apr 2026".
 */
export function fmtDate(value: string | Date): string {
  const d = typeof value === "string" ? new Date(value) : value;
  if (isNaN(d.getTime())) return "—";
  return d.toLocaleDateString("en-GB", {
    day: "numeric",
    month: "short",
    year: "numeric",
  });
}

/**
 * Format a date string as "27 Apr 2026, 14:30".
 */
export function fmtDateTime(value: string | Date): string {
  const d = typeof value === "string" ? new Date(value) : value;
  if (isNaN(d.getTime())) return "—";
  return d.toLocaleDateString("en-GB", {
    day: "numeric",
    month: "short",
    year: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  });
}

type NameEntry = { email: string; name?: string };
let nameRegistry: NameEntry[] = [];

/**
 * Register known email→name mappings (e.g. from currentUser or users list).
 * Call on login and whenever the users list refreshes.
 */
export function registerNames(entries: NameEntry[]) {
  nameRegistry = entries;
}

/**
 * Resolve an email or identifier to a human-readable display name.
 * Checks the name registry first, then falls back to email parsing.
 */
export function displayName(value: string | undefined | null): string {
  if (!value) return "System";
  const known = nameRegistry.find((e) => e.email === value);
  if (known?.name) return known.name;
  if (!value.includes("@")) {
    return value.charAt(0).toUpperCase() + value.slice(1);
  }
  return value
    .split("@")[0]
    .split(/[._-]/)
    .map((w) => w.charAt(0).toUpperCase() + w.slice(1))
    .join(" ");
}
