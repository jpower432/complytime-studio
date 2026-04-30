# Tasks: Generic OIDC Authentication

- [ ] Add `internal/auth/oidc.go`: `OIDCProvider`, `Discover()` with bounded retry, periodic refresh goroutine
- [ ] Add `internal/auth/jwks.go`: JWKS fetch, cache with TTL, `kid`-miss refetch, `VerifyIDToken()`
- [ ] Expand `Config` struct with `Provider`, `Scopes`, `RolesClaim`, `BootstrapEmails`
- [ ] Add PKCE support: `code_verifier` generation, `code_challenge` in login redirect, `code_verifier` in token exchange
- [ ] Remove `googleAuthURL`, `googleTokenURL`, `googleUserURL` constants
- [ ] Rename `fetchGoogleUser` → `fetchUserInfo`, `googleUser` → `oidcUser`
- [ ] Replace `groupsFromIDToken` with `VerifyIDToken` (cryptographic verification via JWKS)
- [ ] Implement role seeding in `handleCallback` (JWT seed + bootstrap allowlist + atomic guard)
- [ ] Add `email_verified` gate: unverified email cannot seed admin role
- [ ] Add `sub` + `issuer` columns to users table, update `UserStore` interface
- [ ] Update `cmd/gateway/main.go`: env var resolution with `OIDC_*` / `GOOGLE_*` alias + deprecation log
- [ ] Add bounded startup retry for OIDC discovery (2s base, 30s cap, 5 min total)
- [ ] Add periodic discovery refresh goroutine (configurable interval, default 24h)
- [ ] Update Helm `values.yaml` with `auth.oidc.*` section, deprecate `auth.google.*`
- [ ] Update gateway deployment template for new env var mappings
- [ ] Add auth observability: metrics for discovery, login, callback, JWKS refresh
- [ ] Tests: discovery retry, JWT verification (valid/invalid/expired/wrong-aud/wrong-iss), role seed, bootstrap atomicity, PKCE, env aliases
