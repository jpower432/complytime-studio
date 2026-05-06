# Design: Generic OIDC Authentication via OAuth2 Proxy

## Architecture

```
Browser → OAuth2 Proxy (:4180) → Gateway (:8080) → PostgreSQL
                                                  → NATS
```

OAuth2 Proxy runs as a sidecar in the gateway pod. All external traffic enters through the proxy on port 4180. The proxy authenticates users via OIDC, then forwards requests to the gateway on localhost:8080 with identity headers injected.

## OAuth2 Proxy Sidecar

### Responsibilities

| Concern | Handled by |
|:--|:--|
| OIDC discovery | OAuth2 Proxy (`--oidc-issuer-url`) |
| Login redirect | OAuth2 Proxy (`/oauth2/start`) |
| Callback + code exchange | OAuth2 Proxy (`/oauth2/callback`) |
| PKCE (S256) | OAuth2 Proxy (`--code-challenge-method=S256`) |
| JWKS fetch + rotation | OAuth2 Proxy (automatic) |
| ID token verification | OAuth2 Proxy (iss, aud, exp, signature) |
| Session cookies | OAuth2 Proxy (encrypted, configurable TTL) |
| Logout | OAuth2 Proxy (`/oauth2/sign_out`) |

### Headers Injected

| Header | OIDC Claim | Example |
|:--|:--|:--|
| `X-Forwarded-Email` | `email` | `alice@example.com` |
| `X-Forwarded-User` | `sub` | `auth0\|abc123` |
| `X-Forwarded-Preferred-Username` | `preferred_username` | `alice` |
| `X-Forwarded-Groups` | `groups` | `admins,engineering` |
| `X-Forwarded-Access-Token` | access token | (opaque) |

### Provider Compatibility

OAuth2 Proxy supports any OIDC-compliant provider via `--provider=oidc`. Keycloak has a dedicated `--provider=keycloak-oidc` with role/group mapping. Tested providers: Keycloak, Dex, Hydra, Okta, Azure AD, Google.

### Unauthenticated Paths

Routes skipped from authentication via `--skip-auth-route`:

| Pattern | Reason |
|:--|:--|
| `GET=/healthz` | Kubernetes probes |
| `POST=/internal/*` | Cluster-internal endpoints (NetworkPolicy-protected) |

## Gateway Middleware

### Rewritten `internal/auth/auth.go`

The `Handler` struct shrinks to:

```go
type Handler struct {
    apiToken string
    users    UserStore
}
```

`NewHandler(apiToken string)` — no cipher, no OIDC config, no session store.

### `Middleware()` — header-trust

1. Skip non-API paths and `/api/config`
2. Check `STUDIO_API_TOKEN` bypass (CI/scripts)
3. Read `X-Forwarded-Email` — if empty, return 401
4. Build `Session` from `X-Forwarded-*` headers
5. Upsert user on first-seen (sub + issuer from headers)
6. Seed role from `X-Forwarded-Groups` if user is new
7. Inject `Session` into request context

### `TokenFromRequest()` — access token forwarding

Reads `X-Forwarded-Access-Token` header instead of decrypting session cookies. Used by the A2A proxy and publish modules to forward tokens to upstream services.

### Routes

| Route | Owner | Notes |
|:--|:--|:--|
| `/oauth2/start` | OAuth2 Proxy | Login redirect |
| `/oauth2/callback` | OAuth2 Proxy | OIDC callback |
| `/oauth2/sign_out` | OAuth2 Proxy | Logout |
| `/auth/me` | Gateway | Returns user info from `users` table |
| `/api/users` | Gateway | Admin user management |
| `/api/users/:email/role` | Gateway | Role assignment |
| `/api/setup-status` | Gateway | Bootstrap check |
| `/api/bootstrap` | Gateway | First-admin promotion |

### Removed Routes

`/auth/login`, `/auth/callback`, `/auth/logout` — owned by OAuth2 Proxy.

## Deleted Code (~2,800 lines)

| File | Lines | Reason |
|:--|:--|:--|
| `internal/auth/oidc.go` | 155 | Discovery, retry — proxy handles |
| `internal/auth/jwks.go` | 391 | JWKS fetch/cache/verify — proxy handles |
| `internal/auth/session_store.go` | 122 | OIDC sessions — proxy handles. ChatStore moves to `chat_store.go` |
| `internal/auth/oidc_test.go` | 662 | Tests for deleted code |
| `internal/auth/auth_test.go` | 515 | Tests for deleted code |
| Most of `internal/auth/auth.go` | ~500 | Login, callback, cookie encrypt/decrypt |

## Helm Configuration

### Sidecar in `templates/gateway.yaml`

The OAuth2 Proxy container is conditionally added to the gateway pod when `auth.oauth2Proxy.enabled` is true.

### `values.yaml`

```yaml
auth:
  oauth2Proxy:
    enabled: true
    image:
      repository: quay.io/oauth2-proxy/oauth2-proxy
      tag: "v7.15.1"
    provider: oidc
    issuerUrl: ""
    clientId: ""
    secretName: studio-oauth-credentials
    secretKey: client-secret
    callbackUrl: "http://localhost:4180/oauth2/callback"
    scopes: "openid email profile groups"
    cookieSecure: false
    emailDomains: ["*"]
    allowedGroups: []
  apiToken: ""
```

### Service Routing

When proxy is enabled, the gateway Service targets port 4180 (proxy). When disabled, it targets port 8080 (gateway direct) for dev mode.

## Dev Mode

When `auth.oauth2Proxy.enabled: false`:
- Sidecar omitted from pod spec
- Service points to gateway :8080
- No `X-Forwarded-Email` header → middleware falls through to anonymous
- Existing `writeProtect` conditional handles the anonymous path

## Observability

| Metric | Type | Notes |
|:--|:--|:--|
| `auth_request_total{result}` | Counter | authenticated / anonymous / api_token |
| `auth_user_upsert_total` | Counter | first-seen user registrations |

OAuth2 Proxy exposes its own `/metrics` endpoint with OIDC-level metrics (login success/failure, token refresh, etc.).
