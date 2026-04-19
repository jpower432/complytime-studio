// SPDX-License-Identifier: Apache-2.0

package httputil

import (
	"net"
	"net/http"
	"sync"
	"time"
)

// RateLimitOptions configures the per-IP rate limiter.
type RateLimitOptions struct {
	// RequestsPerMinute is the maximum number of requests per IP per minute.
	RequestsPerMinute int
	// PathPrefix restricts rate limiting to paths with this prefix (e.g. "/api/").
	PathPrefix string
}

type visitor struct {
	count    int
	windowAt time.Time
}

// RateLimit returns middleware that enforces per-IP request limits using
// a fixed-window counter. Stale entries are evicted every 2 minutes.
func RateLimit(opts RateLimitOptions) func(http.Handler) http.Handler {
	var mu sync.Mutex
	visitors := make(map[string]*visitor)

	go func() {
		for range time.Tick(2 * time.Minute) {
			mu.Lock()
			now := time.Now()
			for ip, v := range visitors {
				if now.Sub(v.windowAt) > 2*time.Minute {
					delete(visitors, ip)
				}
			}
			mu.Unlock()
		}
	}()

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if opts.PathPrefix != "" && len(r.URL.Path) < len(opts.PathPrefix) {
				next.ServeHTTP(w, r)
				return
			}
			if opts.PathPrefix != "" && r.URL.Path[:len(opts.PathPrefix)] != opts.PathPrefix {
				next.ServeHTTP(w, r)
				return
			}

			ip := extractIP(r)
			now := time.Now()
			windowStart := now.Truncate(time.Minute)

			mu.Lock()
			v, ok := visitors[ip]
			if !ok || v.windowAt != windowStart {
				v = &visitor{windowAt: windowStart}
				visitors[ip] = v
			}
			v.count++
			count := v.count
			mu.Unlock()

			if count > opts.RequestsPerMinute {
				w.Header().Set("Retry-After", "60")
				WriteJSON(w, http.StatusTooManyRequests, map[string]string{"error": "rate limit exceeded"})
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func extractIP(r *http.Request) string {
	if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
		if ip, _, err := net.SplitHostPort(fwd); err == nil {
			return ip
		}
		return fwd
	}
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
}
