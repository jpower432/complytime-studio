// SPDX-License-Identifier: Apache-2.0
import { useState, useEffect } from "preact/hooks";
import { navigate } from "../app";
import { fetchConfig, repoUrl } from "../api/config";
import { currentTheme, toggleTheme } from "../store/theme";
import type { UserInfo } from "../api/auth";
import { ImportOverlay } from "./import-overlay";

export function Header({
  user,
  onImportSuccess,
  chatOpen,
  onChatToggle,
}: {
  user: UserInfo;
  onImportSuccess?: () => void;
  chatOpen?: boolean;
  onChatToggle?: () => void;
}) {
  const [ghUrl, setGhUrl] = useState("");
  const [importOpen, setImportOpen] = useState(false);
  const theme = currentTheme.value;
  useEffect(() => { fetchConfig().then((cfg) => setGhUrl(repoUrl(cfg))); }, []);

  const canImport = user.role === "admin" || user.role === "writer";

  return (
    <header class="header">
      <div class="header-left">
        <h1
          class="logo"
          tabIndex={0}
          role="button"
          aria-label="Go to dashboard"
          onClick={() => navigate("dashboard")}
          onKeyDown={(e: KeyboardEvent) => {
            if (e.key === "Enter" || e.key === " ") {
              e.preventDefault();
              navigate("dashboard");
            }
          }}
        >ComplyTime Studio</h1>
        <span class="tagline">Audit Dashboard</span>
      </div>
      <nav class="header-links">
        <a href="https://gemara.openssf.org/" target="_blank" rel="noopener noreferrer" class="icon-link" title="Gemara Docs">
          <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none" stroke-linecap="round" stroke-linejoin="round"><path stroke="none" d="M0 0h24v24H0z" fill="none"/><path d="M3 19a9 9 0 0 1 9 0a9 9 0 0 1 9 0"/><path d="M3 6a9 9 0 0 1 9 0a9 9 0 0 1 9 0"/><path d="M3 6l0 13"/><path d="M12 6l0 13"/><path d="M21 6l0 13"/></svg>
        </a>
        {ghUrl && (
          <a href={ghUrl} target="_blank" rel="noopener noreferrer" class="icon-link" title="GitHub">
            <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none" stroke-linecap="round" stroke-linejoin="round"><path stroke="none" d="M0 0h24v24H0z" fill="none"/><path d="M9 19c-4.3 1.4 -4.3 -2.5 -6 -3m12 5v-3.5c0 -1 .1 -1.4 -.5 -2c2.8 -.3 5.5 -1.4 5.5 -6a4.6 4.6 0 0 0 -1.3 -3.2a4.2 4.2 0 0 0 -.1 -3.2s-1.1 -.3 -3.5 1.3a12.3 12.3 0 0 0 -6.2 0c-2.4 -1.6 -3.5 -1.3 -3.5 -1.3a4.2 4.2 0 0 0 -.1 3.2a4.6 4.6 0 0 0 -1.3 3.2c0 4.6 2.7 5.7 5.5 6c-.6 .6 -.6 1.2 -.5 2v3.5"/></svg>
          </a>
        )}
        <button class="theme-toggle icon-link" onClick={toggleTheme} title={`Switch to ${theme === "dark" ? "light" : "dark"} mode`} aria-label={`Switch to ${theme === "dark" ? "light" : "dark"} mode`}>
          {theme === "dark"
            ? <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none" stroke-linecap="round" stroke-linejoin="round"><path stroke="none" d="M0 0h24v24H0z" fill="none"/><path d="M12 12m-4 0a4 4 0 1 0 8 0a4 4 0 1 0 -8 0m-5 0h1m8 -9v1m8 8h1m-9 8v1m-6.4 -15.4l.7 .7m12.1 -.7l-.7 .7m0 11.4l.7 .7m-12.1 -.7l-.7 .7"/></svg>
            : <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none" stroke-linecap="round" stroke-linejoin="round"><path stroke="none" d="M0 0h24v24H0z" fill="none"/><path d="M12 3c.132 0 .263 0 .393 0a7.5 7.5 0 0 0 7.92 12.446a9 9 0 1 1 -8.313 -12.454z"/></svg>
          }
        </button>
        {onChatToggle && (
          <button
            class={`icon-link chat-header-toggle ${chatOpen ? "active" : ""}`}
            onClick={onChatToggle}
            title="Studio Assistant"
            aria-label="Toggle chat assistant"
            aria-expanded={chatOpen}
          >
            <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none" stroke-linecap="round" stroke-linejoin="round"><path stroke="none" d="M0 0h24v24H0z" fill="none"/><path d="M8 9h8"/><path d="M8 13h6"/><path d="M18 4a3 3 0 0 1 3 3v8a3 3 0 0 1 -3 3h-5l-5 3v-3h-2a3 3 0 0 1 -3 -3v-8a3 3 0 0 1 3 -3h12z"/></svg>
          </button>
        )}
        {canImport && (
          <>
            <button
              type="button"
              class="import-btn import-btn-icon"
              title="Import Gemara artifact"
              aria-label="Import Gemara artifact"
              onClick={() => setImportOpen(true)}
            >
              <svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path stroke="none" d="M0 0h24v24H0z" fill="none"/><path d="M4 17v2a2 2 0 0 0 2 2h12a2 2 0 0 0 2 -2v-2"/><path d="M7 9l5 -5l5 5"/><path d="M12 4l0 12"/></svg>
            </button>
            <ImportOverlay
              open={importOpen}
              onClose={() => setImportOpen(false)}
              onSuccess={onImportSuccess}
            />
          </>
        )}
        <div class="user-info">
          {user.avatar_url
            ? <img class="user-avatar" src={user.avatar_url} alt={user.name || user.login} width="24" height="24" referrerpolicy="no-referrer" />
            : <span class="user-avatar-placeholder">{(user.name || user.login || "?").charAt(0).toUpperCase()}</span>
          }
          <span class="user-login">{user.name || user.login}</span>
          <a href="/oauth2/sign_out?rd=%2Fauth%2Flogged-out" class="icon-link logout-btn" title="Sign out" aria-label="Sign out">
            <svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none" stroke-linecap="round" stroke-linejoin="round"><path stroke="none" d="M0 0h24v24H0z" fill="none"/><path d="M14 8v-2a2 2 0 0 0 -2 -2h-7a2 2 0 0 0 -2 2v12a2 2 0 0 0 2 2h7a2 2 0 0 0 2 -2v-2"/><path d="M9 12h12l-3 -3"/><path d="M18 15l3 -3"/></svg>
          </a>
        </div>
      </nav>
    </header>
  );
}
