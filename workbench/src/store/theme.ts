// SPDX-License-Identifier: Apache-2.0

import { signal } from "@preact/signals";

export type Theme = "light" | "dark";

const STORAGE_KEY = "complytime-studio-theme";

function getSystemPreference(): Theme {
  return window.matchMedia("(prefers-color-scheme: dark)").matches ? "dark" : "light";
}

function getInitialTheme(): Theme {
  const stored = localStorage.getItem(STORAGE_KEY);
  if (stored === "light" || stored === "dark") return stored;
  return getSystemPreference();
}

function applyTheme(theme: Theme) {
  document.documentElement.setAttribute("data-theme", theme);
}

export const currentTheme = signal<Theme>(getInitialTheme());
applyTheme(currentTheme.value);

export function toggleTheme() {
  const next = currentTheme.value === "dark" ? "light" : "dark";
  currentTheme.value = next;
  localStorage.setItem(STORAGE_KEY, next);
  applyTheme(next);
}
