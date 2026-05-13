# Proposal: Generic OIDC Authentication via OAuth2 Proxy

## User Story

As an enterprise platform operator, I need Studio to support my organization's identity provider (Keycloak, Okta, Azure AD, Google, Dex, Hydra) so that my team can authenticate without requiring a specific vendor's accounts.

## Problem

Authentication requires OIDC discovery, PKCE, JWKS verification, token refresh, session management, and cookie encryption. Implementing this in-process adds ~3,000 lines of security-critical code to the gateway, creates multi-replica session problems, and requires ongoing maintenance as the OIDC spec evolves.

## Solution

Deploy [OAuth2 Proxy](https://oauth2-proxy.github.io/oauth2-proxy/) as a sidecar container in the gateway pod. OAuth2 Proxy handles the full OIDC lifecycle (discovery, login, callback, PKCE, JWKS, token refresh, session cookies). The gateway receives trusted `X-Forwarded-*` headers and maintains a thin middleware (~100 lines) that reads identity from headers, upserts users on first-seen, and enforces RBAC from the `users` table.

## Scope

| In Scope | Out of Scope |
|:--|:--|
| OAuth2 Proxy sidecar in gateway pod | Multi-provider simultaneous support (one issuer per deployment) |
| Header-trust middleware in gateway | External authz engine (OPA, Casbin) |
| User upsert and role seeding from proxy headers | SAML, LDAP (IdP handles protocol translation) |
| Helm values for OAuth2 Proxy configuration | Redis session store (cookie sessions are sufficient) |
| Remove hand-rolled OIDC code (~2,800 lines deleted) | New roles beyond `admin` / `reviewer` (separate spec) |
| Dev mode (proxy disabled, anonymous access) | |
