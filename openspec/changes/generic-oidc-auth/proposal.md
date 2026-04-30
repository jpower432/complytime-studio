# Proposal: Generic OIDC Authentication

## User Story

As an enterprise platform operator, I need Studio to support my organization's identity provider (Keycloak, Okta, Azure AD, Google) so that my team can authenticate without requiring a specific vendor's accounts.

## Problem

Gateway auth is hardcoded to Google OAuth. Organizations using Keycloak, Red Hat SSO, Okta, or Azure AD cannot deploy Studio for their teams.

## Solution

Replace hardcoded Google OAuth URLs with a generic OIDC discovery flow. The gateway fetches `/.well-known/openid-configuration` from a configurable issuer URL, caching authorization, token, userinfo, and JWKS endpoints. ID tokens are cryptographically verified (JWKS + standard claims). Role seeding uses JWT claims on first login with DB override afterward. Google env vars are kept as deprecated aliases with a removal timeline.

## Scope

| In Scope | Out of Scope |
|:--|:--|
| OIDC discovery with bounded retry + periodic refresh | Multi-provider simultaneous support (one issuer per deployment) |
| JWKS-based ID token verification (`iss`, `aud`, `exp`, `nbf`, signature) | External authz engine |
| S256 PKCE for authorization code flow | SAML, LDAP, non-OIDC protocols |
| Role seeding from configurable JWT claim path | New roles beyond `admin` / `reviewer` (separate spec) |
| Bootstrap hardening (email allowlist for first admin) | Redis/Valkey session store (separate spec) |
| `sub`-keyed stable identity alongside email | |
| `GOOGLE_*` env vars as deprecated aliases | |
| Helm values restructure (`auth.oidc.*`) | |
