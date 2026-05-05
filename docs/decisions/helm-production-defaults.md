# Helm Production Defaults

**Status:** Accepted
**Date:** 2026-05-04

## Context

`values.yaml` line 3 stated: *"targeting kind development clusters."* Every default was dev-optimized. Production required a full override file to avoid shipping cleartext passwords, insecure cookies, and an embedded registry. This is backwards — safe defaults should be the baseline, and dev should opt in to convenience.

Non-production defaults found:

| Key | Dev Value | Risk |
|:----|:----------|:-----|
| `auth.apiToken` | `"studio-dev-token"` | Predictable static bearer token |
| `auth.oauth2Proxy.callbackUrl` | `"http://localhost:8080/..."` | Localhost callback in chart defaults |
| `auth.oauth2Proxy.cookieSecure` | `false` | HTTP cookies exposed to MITM |
| `blobStorage.useSSL` | `false` | Unencrypted blob traffic |
| `postgres.auth.password` | `"complytime-dev"` | Cleartext dev password |
| `postgres.auth.sslmode` | `disable` | No TLS to database |
| `clickhouse.auth.password` | `"complytime-dev"` | Cleartext dev password |
| `clickhouse.auth.readerPassword` | `"complytime-reader-dev"` | Cleartext dev reader password |
| `registry.enabled` | `true` | Built-in dev registry on by default |
| `registry.service.type` | `NodePort` | Exposes port on node |

## Decision

**Flip `values.yaml` to production-safe defaults. Move dev overrides into `values-dev.yaml`, layered by the Makefile.**

### `values.yaml` (production baseline)

All passwords default to `""` (forces `existingSecret`). SSL/TLS enabled. Registry disabled. Cookie security on.

### `values-dev.yaml` (dev overlay)

Restores dev convenience: cleartext passwords, `sslmode: disable`, registry enabled with `NodePort`, `cookieSecure: false`, static API token.

### Makefile

`studio-up` and `studio-template` layer `-f charts/complytime-studio/values-dev.yaml` so `make deploy` continues unchanged.

## Consequences

- Production deployments are safe by default — no silent security holes from forgotten overrides.
- `make deploy` (kind) behavior unchanged — dev overlay restores all convenience defaults.
- Extends [0007 — Default Admin & Token Hardening](default-admin-token-hardening.md) from warn-at-startup to secure-by-default.
- Template files unchanged — they already branch on `.Values` conditionally.

## Rejected Alternatives

| Approach | Why Not |
|:---------|:--------|
| Keep dev defaults, add `values-prod.yaml` | Unsafe baseline. Easy to deploy without the override and ship cleartext passwords. |
| Makefile `--set` flags instead of overlay file | 11 overrides becomes a wall of flags. Overlay file is self-documenting. |
