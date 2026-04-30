# Design: Generic OIDC Authentication

## New File: `internal/auth/oidc.go`

### OIDCProvider struct

| Field | Type | Source |
|:--|:--|:--|
| `IssuerURL` | `string` | `OIDC_ISSUER_URL` env var |
| `AuthURL` | `string` | Discovery doc `authorization_endpoint` |
| `TokenURL` | `string` | Discovery doc `token_endpoint` |
| `UserInfoURL` | `string` | Discovery doc `userinfo_endpoint` |
| `JWKSURL` | `string` | Discovery doc `jwks_uri` |

### `Discover(ctx, issuerURL) (*OIDCProvider, error)`

- `GET {issuerURL}/.well-known/openid-configuration`
- Parse JSON, populate `OIDCProvider`
- Validate required fields present (`authorization_endpoint`, `token_endpoint`, `jwks_uri`)
- Normalize issuer URL (trailing slash handling per OIDC spec)
- **Startup**: bounded exponential backoff (2s base, 30s cap, 5 min total). Gateway readiness probe fails until discovery succeeds.
- **Runtime**: periodic refresh (configurable via `OIDC_DISCOVERY_REFRESH`, default `24h`). Log + alert on refresh failure but continue with cached config.

### JWKS Handling

| Concern | Approach |
|:--|:--|
| Initial fetch | At startup alongside discovery |
| Cache | In-memory with TTL (default 1h) |
| Rotation | Background refresh on TTL expiry. On-demand refetch when `kid` not found in cache. |
| Verification | RS256 + RS384 + RS512. Match `kid` from ID token header to JWKS key. |

### `VerifyIDToken(idToken string) (*IDTokenClaims, error)`

Standard OIDC ID token verification:
1. Decode header, match `kid` against cached JWKS
2. Verify signature
3. Validate `iss` == configured issuer (normalized comparison)
4. Validate `aud` contains `client_id`
5. Validate `exp` > now (with 60s clock skew leeway)
6. Validate `nbf` <= now (if present)
7. Extract claims: `sub`, `email`, `email_verified`, `name`, `picture`, groups, roles

## Modified: `internal/auth/auth.go`

### Config expansion

| Field | Type | Default | Notes |
|:--|:--|:--|:--|
| `ClientID` | `string` | — | From `OIDC_CLIENT_ID` or `GOOGLE_CLIENT_ID` |
| `ClientSecret` | `string` | — | From `OIDC_CLIENT_SECRET` or `GOOGLE_CLIENT_SECRET` |
| `CallbackURL` | `string` | `http://localhost:8080/auth/callback` | |
| `Provider` | `*OIDCProvider` | — | From `Discover()` |
| `Scopes` | `string` | `openid email profile` | From `OIDC_SCOPES` |
| `RolesClaim` | `string` | `""` (disabled) | Dot-path into ID token, e.g. `realm_access.roles` |
| `BootstrapEmails` | `[]string` | `[]` (any user) | From `OIDC_BOOTSTRAP_EMAILS` |

### Removed

- `googleAuthURL`, `googleTokenURL`, `googleUserURL` constants
- `fetchGoogleUser` → renamed `fetchUserInfo` (takes URL param)
- `googleUser` → renamed `oidcUser`
- `groupsFromIDToken` → replaced by `VerifyIDToken` (cryptographic verification)

### handleLogin changes

- Redirect URL uses `cfg.Provider.AuthURL`
- Scopes from `cfg.Scopes`
- Add S256 PKCE: generate `code_verifier`, store in state cookie, send `code_challenge` + `code_challenge_method`

### exchangeCode changes

- Posts to `cfg.Provider.TokenURL`
- Include `code_verifier` from state cookie for PKCE

### handleCallback — role seeding

```
1. Exchange code (with PKCE verifier)
2. Verify ID token via JWKS (iss, aud, exp, signature)
3. Fetch userinfo from Provider.UserInfoURL
4. Extract roles from verified ID token claims (via RolesClaim dot-path)
5. UpsertUser (keyed on sub + issuer, email as display field)
6. existingUser, err = GetUser(sub)
7. IF err (new user):
   a. IF BootstrapEmails is non-empty AND email not in list → default reviewer
   b. ELSE IF roles contains "admin" → SetRole(admin), log role-change (jwt-seed)
   c. ELSE IF CountAdmins() == 0 → atomic INSERT-if-zero-admins, log role-change (first-admin)
   d. ELSE → default reviewer
8. IF no err (existing user): skip role mutation (DB authoritative)
```

### Bootstrap hardening

- `OIDC_BOOTSTRAP_EMAILS`: comma-separated email allowlist for first-admin promotion
- When set: only listed emails can become admin via bootstrap or JWT seed
- When empty: existing behavior preserved, with deprecation warning in logs ("configure OIDC_BOOTSTRAP_EMAILS for production")
- Transactional guard: `INSERT INTO users ... WHERE (SELECT count(*) FROM users WHERE role = 'admin') = 0` — atomic, no race

### Identity model

| Field | Purpose | Uniqueness |
|:--|:--|:--|
| `sub` | Stable identity from IdP (scoped to issuer) | Primary key candidate |
| `email` | Display, notifications, human-readable lookup | Not unique across issuers |
| `email_verified` | Gate trust on email for role decisions | Must be `true` for admin seed |

Users table gains `sub TEXT NOT NULL` and `issuer TEXT NOT NULL` columns. `email` remains for display and backward compat. Uniqueness constraint on `(sub, issuer)`.

## Environment Variables

| Variable | Default | Notes |
|:--|:--|:--|
| `OIDC_ISSUER_URL` | `""` | Must expose `/.well-known/openid-configuration` |
| `OIDC_CLIENT_ID` | `""` | Falls back to `GOOGLE_CLIENT_ID` (deprecated) |
| `OIDC_CLIENT_SECRET` | `""` | Falls back to `GOOGLE_CLIENT_SECRET` (deprecated) |
| `OIDC_CALLBACK_URL` | `http://localhost:8080/auth/callback` | |
| `OIDC_SCOPES` | `openid email profile` | Space-separated |
| `OIDC_ROLES_CLAIM` | `""` | Dot-path into verified ID token claims |
| `OIDC_BOOTSTRAP_EMAILS` | `""` | Comma-separated admin bootstrap allowlist |
| `OIDC_DISCOVERY_REFRESH` | `24h` | Periodic refresh interval for discovery doc |
| `GOOGLE_CLIENT_ID` | `""` | **Deprecated** — triggers Google issuer default |
| `GOOGLE_CLIENT_SECRET` | `""` | **Deprecated** |
| `GOOGLE_CALLBACK_URL` | `""` | **Deprecated** |

### Deprecation behavior

When `GOOGLE_*` vars are set and `OIDC_*` equivalents are empty:
1. Log warning: `"GOOGLE_* auth vars are deprecated — migrate to OIDC_* (removal planned for v2.0)"`
2. Map: `OIDC_ISSUER_URL` = `https://accounts.google.com`, client ID/secret/callback from `GOOGLE_*`
3. Single code path — no branching based on which vars were set

## Helm Values

```yaml
auth:
  oidc:
    issuerUrl: ""
    clientId: ""
    clientSecret: ""
    callbackUrl: "http://localhost:8080/auth/callback"
    scopes: "openid email profile"
    rolesClaim: ""
    bootstrapEmails: []
    discoveryRefresh: "24h"
  google:
    clientId: ""          # deprecated
    clientSecret: ""      # deprecated
    callbackUrl: ""       # deprecated
  cookieSecretName: ""
  admins: []
  apiToken: "dev-seed-token"
```

## Observability

| Metric | Type | Notes |
|:--|:--|:--|
| `auth_oidc_discovery_success_total` | Counter | Successful discovery fetches |
| `auth_oidc_discovery_failure_total` | Counter | Failed discovery fetches |
| `auth_login_total{result}` | Counter | success / error |
| `auth_callback_total{result}` | Counter | success / invalid_state / token_error / verify_error |
| `auth_jwks_refresh_total{result}` | Counter | success / error |

Alert on: sustained discovery failure, callback error rate spike, zero successful logins in window.

## Tests

| Test | Validates |
|:--|:--|
| `TestDiscover_Success` | Discovery parses all required fields |
| `TestDiscover_RetryOnTransientFailure` | Bounded backoff succeeds after transient 503 |
| `TestDiscover_FailAfterMaxRetries` | Fatal error after retry budget exhausted |
| `TestVerifyIDToken_ValidSignature` | RS256 verification passes with correct JWKS key |
| `TestVerifyIDToken_InvalidSignature` | Rejects tampered token |
| `TestVerifyIDToken_ExpiredToken` | Rejects expired token |
| `TestVerifyIDToken_WrongAudience` | Rejects token for different client |
| `TestVerifyIDToken_WrongIssuer` | Rejects token from wrong issuer |
| `TestCallback_RoleSeedFromJWT` | Verified ID token with admin role → user seeded as admin |
| `TestCallback_BootstrapEmailAllowlist` | Non-allowlisted email cannot become first admin |
| `TestCallback_AtomicBootstrap` | Concurrent first logins → exactly one admin |
| `TestCallback_ExistingUserNoOverride` | Returning user → JWT roles ignored |
| `TestCallback_UnverifiedEmailNoAdmin` | `email_verified: false` → cannot seed admin |
| `TestEnvAliases_GoogleDeprecated` | `GOOGLE_CLIENT_ID` set → deprecation warning + Google issuer default |
| `TestPKCE_CodeChallenge` | Login sends S256 code_challenge, callback includes code_verifier |
