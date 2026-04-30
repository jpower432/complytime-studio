// SPDX-License-Identifier: Apache-2.0

package auth

import "expvar"

// Auth observability counters. Accessible at /debug/vars when the default
// expvar handler is registered, or read directly by monitoring tooling.
var (
	// oidcDiscoverySuccess counts successful OIDC discovery fetches.
	oidcDiscoverySuccess = expvar.NewInt("auth_oidc_discovery_success_total")
	// oidcDiscoveryFailure counts failed OIDC discovery fetches.
	oidcDiscoveryFailure = expvar.NewInt("auth_oidc_discovery_failure_total")

	// authLoginTotal counts login initiations keyed by "success" or "error".
	authLoginTotal = expvar.NewMap("auth_login_total")
	// authCallbackTotal counts callback results: success, invalid_state,
	// token_error, verify_error, userinfo_error.
	authCallbackTotal = expvar.NewMap("auth_callback_total")
	// jwksRefreshTotal counts JWKS refresh results keyed by "success" or "error".
	jwksRefreshTotal = expvar.NewMap("auth_jwks_refresh_total")
)
