// SPDX-License-Identifier: Apache-2.0
import { navigate } from "../app";
import type { UserInfo } from "../api/auth";

export function Header({ user }: { user: UserInfo }) {
  return (
    <header class="header">
      <div class="header-left">
        <h1 class="logo" onClick={() => navigate("missions")}>ComplyTime Studio</h1>
        <span class="tagline">Gemara Artifact Workbench</span>
      </div>
      <nav class="header-links">
        <a href="https://gemara.openssf.org/" target="_blank" rel="noopener noreferrer">Gemara Docs</a>
        <a href="https://github.com/complytime/complytime-studio" target="_blank" rel="noopener noreferrer">GitHub</a>
        <div class="user-info">
          <img class="user-avatar" src={user.avatar_url} alt={user.login} width="28" height="28" />
          <span class="user-login">{user.login}</span>
          <a href="/auth/logout" class="logout-link">Logout</a>
        </div>
      </nav>
    </header>
  );
}
