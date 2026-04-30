# Tasks: Generic OIDC Authentication

- [x] Add `internal/auth/oidc.go`: `OIDCProvider`, `Discover()` with bounded retry, periodic refresh goroutine
- [x] Add `internal/auth/jwks.go`: JWKS fetch, cache with TTL, `kid`-miss refetch, `VerifyIDToken()`
- [x] Expand `Config` struct with `Provider`, `Scopes`, `RolesClaim`, `BootstrapEmails`
- [x] Add PKCE support: `code_verifier` generation, `code_challenge` in login redirect, `code_verifier` in token exchange
- [x] Remove `googleAuthURL`, `googleTokenURL`, `googleUserURL` constants
- [x] Rename `fetchGoogleUser` → `fetchUserInfo`, `googleUser` → `oidcUser`
- [x] Replace `groupsFromIDToken` with `VerifyIDToken` (cryptographic verification via JWKS)
- [x] Implement role seeding in `handleCallback` (JWT seed + bootstrap allowlist + atomic guard)
- [x] Add `email_verified` gate: unverified email cannot seed admin role
- [x] Add `sub` + `issuer` columns to users table, update `UserStore` interface
- [x] Update `cmd/gateway/main.go`: env var resolution with `OIDC_*` / `GOOGLE_*` alias + deprecation log
- [x] Add bounded startup retry for OIDC discovery (2s base, 30s cap, 5 min total)
- [x] Add periodic discovery refresh goroutine (configurable interval, default 24h)
- [x] Update Helm `values.yaml` with `auth.oidc.*` section, deprecate `auth.google.*`
- [x] Update gateway deployment template for new env var mappings
- [x] Add auth observability: metrics for discovery, login, callback, JWKS refresh
- [x] Tests: discovery retry, JWT verification (valid/invalid/expired/wrong-aud/wrong-iss), role seed, bootstrap atomicity, PKCE, env aliases
