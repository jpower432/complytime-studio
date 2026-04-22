# Session Token Storage: Cookie vs Server-Side

**Status:** Proposed
**Date:** 2026-04-21

## Context

The gateway stores the full Google OAuth access token inside the AES-GCM encrypted session cookie (`studio_session`). On A2A requests, the gateway extracts this token from the cookie and injects it as `Authorization: Bearer` on proxied requests to agent pods and MCP servers.

This works but couples token confidentiality entirely to cookie encryption. If the AES-GCM key is compromised — or ephemeral on restart because `COOKIE_SECRET` was unset — every active user's Google access token is recoverable from captured cookies. The blast radius of a key compromise is all user tokens, not just session continuity.

Additionally, the access token is never refreshed. Google access tokens expire after ~1 hour. The session cookie has an 8-hour `ExpiresAt`, so the embedded token becomes stale well before the session expires. Proxied requests to MCP servers silently carry an expired token.

## Options

### Option A: Server-side session store (recommended)

The cookie carries only a random session ID. The gateway maps session IDs to tokens in a server-side store.

**Store candidates:**

| Store | Pros | Cons |
|:--|:--|:--|
| In-memory (sync.Map) | Zero dependencies, fast | Lost on restart, no cross-replica sharing |
| ClickHouse | Already deployed | Writes on every login, OLAP engine is wrong fit for session lookups |
| Redis / Valkey | Purpose-built for sessions, TTL support, shared across replicas | New dependency |
| Kubernetes Secret / ConfigMap | No new infra | Not designed for high-frequency reads, RBAC complexity |

**Implementation sketch:**

```
Login:    generate session_id → store {session_id: {token, user, expires}} → set cookie(session_id)
Request:  read cookie(session_id) → lookup store → inject Bearer token
Logout:   delete store entry → clear cookie
Expiry:   TTL on store entry = sessionMaxAge (8h)
```

**Session struct change:**

```go
// Cookie payload — only the session ID
type CookiePayload struct {
    SessionID string `json:"sid"`
}

// Server-side — not in the cookie
type ServerSession struct {
    AccessToken string
    Login       string
    Name        string
    AvatarURL   string
    Email       string
    ExpiresAt   int64
}
```

**Consequences:**
- Cookie compromise reveals a random session ID, not a Google access token.
- Key rotation invalidates session IDs (cheap) instead of exposing tokens.
- Enables future token refresh — the server-side store can update the access token without rewriting the cookie.
- Requires a store that survives restarts for production. In-memory is acceptable for single-replica dev.

### Option B: Keep token in cookie, add refresh (status quo + patch)

Keep the current AES-GCM encrypted cookie but add offline refresh tokens.

**Consequences:**
- Simpler — no new store infrastructure.
- Token compromise blast radius unchanged. A captured cookie still contains a valid (or refreshable) Google token.
- Refresh tokens in cookies increase cookie size and require `access_type=offline` in the OAuth flow, which prompts users for consent on every login unless `prompt=consent` is cached.

## Recommendation

**Option A with in-memory store**, graduating to Redis/Valkey when multi-replica or HA is required.

The in-memory store is zero-dependency and sufficient for single-replica `kind` deployments. The interface should abstract the store so swapping to Redis is a configuration change:

```go
type SessionStore interface {
    Put(ctx context.Context, id string, sess ServerSession) error
    Get(ctx context.Context, id string) (*ServerSession, error)
    Delete(ctx context.Context, id string) error
}
```

## Migration Path

1. Add `SessionStore` interface and in-memory implementation.
2. Refactor `auth.Handler` to use `SessionStore` instead of embedding tokens in cookies.
3. Cookie payload shrinks to `{sid}` — existing cookies are invalidated (users re-login once).
4. Add `sessionStore` Helm value (`memory` | `redis`) with `memory` as default.
5. Redis implementation when multi-replica gateway is deployed.

## Rejected Approaches

| Approach | Why Not |
|:--|:--|
| ClickHouse as session store | Wrong engine type. ReplacingMergeTree is optimized for analytics, not point lookups with TTL. |
| Kubernetes Secrets | Not designed for session-frequency reads. RBAC grants are coarse. |
| JWT (stateless, no store) | Still embeds claims client-side. Doesn't solve the token-in-cookie problem — moves it to a different envelope. |

## Open Questions

- Should the session store support token refresh (store refresh_token, rotate access_token on expiry)?
- Is the 8-hour session TTL appropriate, or should it match the Google access token lifetime (~1 hour) with transparent refresh?
- Should session invalidation be exposed as an admin API for the RBAC work?

## Related

- [Authorization Model](authorization-model.md) — RBAC enforcement depends on session data
- [Backend Architecture](backend-architecture.md) — gateway owns the auth chokepoint
- `internal/auth/auth.go` — current implementation
