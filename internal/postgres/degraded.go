// SPDX-License-Identifier: Apache-2.0

package postgres

import (
	"context"
	"net/http"
	"strings"
	"sync"
	"time"
)

const degradedHeader = "X-Studio-Degraded"

// Pinger abstracts a health check for any backing store.
type Pinger interface {
	Ping(ctx context.Context) error
}

type pingResult struct {
	degraded []string
	checkedAt time.Time
}

// DegradedMiddleware checks subsystem health and sets the X-Studio-Degraded
// header when a backing store is unavailable. Results are cached for 5 seconds
// to avoid pinging on every HTTP request.
func DegradedMiddleware(subsystems map[string]Pinger) func(http.Handler) http.Handler {
	const cacheTTL = 5 * time.Second

	var (
		mu     sync.RWMutex
		cached pingResult
	)

	check := func(ctx context.Context) []string {
		mu.RLock()
		if time.Since(cached.checkedAt) < cacheTTL {
			result := cached.degraded
			mu.RUnlock()
			return result
		}
		mu.RUnlock()

		mu.Lock()
		defer mu.Unlock()
		if time.Since(cached.checkedAt) < cacheTTL {
			return cached.degraded
		}

		var degraded []string
		for name, p := range subsystems {
			if p == nil {
				continue
			}
			if err := p.Ping(ctx); err != nil {
				degraded = append(degraded, name)
			}
		}
		cached = pingResult{degraded: degraded, checkedAt: time.Now()}
		return degraded
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if degraded := check(r.Context()); len(degraded) > 0 {
				w.Header().Set(degradedHeader, strings.Join(degraded, ","))
			}
			next.ServeHTTP(w, r)
		})
	}
}
