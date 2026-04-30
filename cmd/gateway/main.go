// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/complytime/complytime-studio/internal/agents"
	"github.com/complytime/complytime-studio/internal/auth"
	"github.com/complytime/complytime-studio/internal/blob"
	"github.com/complytime/complytime-studio/internal/certifier"
	chclient "github.com/complytime/complytime-studio/internal/clickhouse"
	"github.com/complytime/complytime-studio/internal/config"
	"github.com/complytime/complytime-studio/internal/consts"
	"github.com/complytime/complytime-studio/internal/events"
	"github.com/complytime/complytime-studio/internal/httputil"
	"github.com/complytime/complytime-studio/internal/proxy"
	"github.com/complytime/complytime-studio/internal/publish"
	"github.com/complytime/complytime-studio/internal/registry"
	"github.com/complytime/complytime-studio/internal/store"
	"github.com/complytime/complytime-studio/internal/web"
	"github.com/complytime/complytime-studio/workbench"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})))

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	port := httputil.EnvOr("PORT", "8080")
	mux := http.NewServeMux()

	var chClient *chclient.Client
	if chAddr := os.Getenv("CLICKHOUSE_ADDR"); chAddr != "" {
		chCfg := chclient.Config{
			Addr:     chAddr,
			User:     httputil.EnvOr("CLICKHOUSE_USER", "default"),
			Password: os.Getenv("CLICKHOUSE_PASSWORD"),
		}
		maxRetries := 90
		for attempt := 1; attempt <= maxRetries; attempt++ {
			var err error
			chClient, err = chclient.New(ctx, chCfg)
			if err == nil {
				break
			}
			if attempt == maxRetries {
				slog.Error("clickhouse connection failed after retries", "error", err, "attempts", maxRetries)
				os.Exit(1)
			}
			slog.Warn("clickhouse not ready, retrying", "attempt", attempt, "error", err)
			select {
			case <-ctx.Done():
				os.Exit(1)
			case <-time.After(2 * time.Second):
			}
		}
		if err := chClient.EnsureSchema(ctx, 24); err != nil {
			slog.Error("clickhouse schema init failed", "error", err)
			os.Exit(1)
		}
		slog.Info("clickhouse ready")
	}

	var st *store.Store
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		if chClient != nil {
			if err := chClient.Ping(r.Context()); err != nil {
				http.Error(w, "clickhouse unreachable", http.StatusServiceUnavailable)
				return
			}
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	internalPort := httputil.EnvOr("INTERNAL_PORT", consts.DefaultInternalPort)
	internalMux := http.NewServeMux()

	if chClient != nil {
		st = store.New(chClient.Conn())

		var blobStore blob.BlobStore
		if cfg, ok := blob.ConfigFromEnv(); ok {
			if cfg.AccessKey == "" || cfg.SecretKey == "" {
				slog.Error("blob storage enabled but BLOB_ACCESS_KEY / BLOB_SECRET_KEY missing")
				os.Exit(1)
			}
			bs, err := blob.NewMinioBlobStore(ctx, cfg)
			if err != nil {
				slog.Error("blob storage init failed", "error", err)
				os.Exit(1)
			}
			blobStore = bs
			slog.Info("blob storage configured", "endpoint", cfg.Endpoint, "bucket", cfg.Bucket)
		}

		registryAddr := os.Getenv("REGISTRY_INSECURE")
		if err := store.PopulateCatalogsFromRegistry(ctx, st, st, st, st, registryAddr); err != nil {
			slog.Warn("catalog seed from registry failed", "error", err)
		}

		if err := store.PopulateMappingEntries(ctx, st); err != nil {
			slog.Warn("mapping entries backfill failed", "error", err)
		}
		if err := store.PopulateControls(ctx, st, st); err != nil {
			slog.Warn("controls backfill failed", "error", err)
		}
		if err := store.PopulateThreats(ctx, st, st); err != nil {
			slog.Warn("threats backfill failed", "error", err)
		}
		if err := store.PopulateRisks(ctx, st, st); err != nil {
			slog.Warn("risks backfill failed", "error", err)
		}
		if err := store.PopulateEffectiveControls(ctx, st, st, st); err != nil {
			slog.Warn("effective controls backfill failed", "error", err)
		}
		if err := store.PopulatePolicyCriteria(ctx, st, st); err != nil {
			slog.Warn("policy criteria backfill failed", "error", err)
		}
		bus, busErr := events.Connect(os.Getenv("NATS_URL"))
		if busErr != nil {
			slog.Warn("nats connection failed — event-driven posture checks disabled", "error", busErr)
		}
		if bus != nil {
			defer bus.Close()
		}

		var pub store.EvidencePublisher
		if bus != nil {
			pub = bus
		}

		stores := store.Stores{
			Policies:            st,
			Mappings:            st,
			Evidence:            st,
			Blob:                blobStore,
			AuditLogs:           st,
			DraftAuditLogs:      st,
			Requirements:        st,
			Controls:            st,
			Threats:             st,
			Risks:               st,
			Catalogs:            st,
			EvidenceAssessments: st,
			Posture:             st,
			Notifications:       st,
			Certifications:      st,
			EventPublisher:      pub,
		}
		store.Register(mux, stores)
		store.RegisterInternal(internalMux, stores)
		slog.Info("store API registered", "routes", []string{
			"/api/policies", "/api/evidence", "/api/audit-logs", "/api/mappings",
		})

		if bus != nil {
			adapter := &notificationAdapter{store: st}
			rateCache := events.NewRateCache()
			postureHandler := events.PostureCheckHandler(ctx, st, adapter, rateCache)
			postureDebouncer := events.NewDebouncer(30*time.Second, postureHandler)

			pipeline := buildCertifierPipeline()
			certAdapter := &certificationAdapter{store: st}
			certHandler := events.CertificationHandler(ctx, pipeline, certAdapter, certAdapter)
			certDebouncer := events.NewDebouncer(30*time.Second, certHandler)

			sub, err := bus.SubscribeEvidence(func(evt events.EvidenceEvent) {
				postureDebouncer.Push(evt)
				certDebouncer.Push(evt)
			})
			if err != nil {
				slog.Warn("nats subscribe failed", "error", err)
			} else if sub != nil {
				defer func() { _ = sub.Unsubscribe() }()
				slog.Info("nats evidence subscription active", "subject", events.SubjectPrefix+".>")
			}
		}
	}

	authCfg := auth.ConfigFromEnv()
	secretKey := cookieSecretKey(authCfg.ClientID != "")

	// OIDC discovery with bounded startup retry (2s base, 30s cap, 5min total).
	if authCfg.ClientID != "" {
		provider, discErr := auth.DiscoverWithRetry(ctx, authCfg.Provider.IssuerURL)
		if discErr != nil {
			slog.Error("oidc discovery failed — cannot start with auth enabled", "error", discErr)
			os.Exit(1)
		}
		authCfg.Provider = provider
		slog.Info("oidc discovery succeeded", "issuer", provider.IssuerURL)
	}

	sessionStore := auth.NewMemorySessionStore()
	authHandler, err := auth.NewHandler(authCfg, secretKey, sessionStore)
	if err != nil {
		slog.Error("auth handler init failed", "error", err)
		os.Exit(1)
	}

	// Periodic discovery refresh goroutine.
	if authCfg.ClientID != "" && authCfg.Provider != nil {
		refreshInterval := auth.ParseDiscoveryRefreshInterval()
		auth.StartRefreshLoop(ctx, authCfg.Provider.IssuerURL, refreshInterval, authHandler.UpdateProvider)
		slog.Info("oidc discovery refresh loop started", "interval", refreshInterval)
	}

	if st != nil {
		authHandler.SetUserStore(st)
		slog.Info("persistent user store enabled — first user to sign in becomes admin")
	} else {
		slog.Warn("no user store (ClickHouse not configured) — RBAC disabled, all users treated as reviewer")
	}

	authHandler.Register(mux)
	authHandler.RegisterUserAPI(mux)
	authHandler.RegisterChatHistory(mux, sessionStore)

	if apiToken := os.Getenv("STUDIO_API_TOKEN"); apiToken != "" {
		authHandler.SetAPIToken(apiToken)
		if apiToken == consts.DefaultDevAPIToken {
			slog.Warn("STUDIO_API_TOKEN is the default dev value — rotate before production use")
		}
		slog.Info("api token auth enabled for seed/CI scripts")
	}

	registerGemaraProxy(mux, os.Getenv("GEMARA_MCP_URL"))

	insecureRaw := os.Getenv("REGISTRY_INSECURE")
	insecureList := splitComma(insecureRaw)

	registry.Register(mux, registry.Options{
		MCPURL:             os.Getenv("ORAS_MCP_URL"),
		InsecureRegistries: insecureList,
	})

	publish.Register(mux, publish.Options{
		TokenProvider:      authHandler,
		InsecureRegistries: insecureList,
	})

	agentCards := agents.ParseDirectory(os.Getenv("AGENT_DIRECTORY"))
	agents.RegisterDirectory(mux, agentCards)

	if a2aProxyURL := os.Getenv("A2A_PROXY_URL"); a2aProxyURL != "" {
		agents.RegisterA2AForward(mux, a2aProxyURL)
	} else {
		agents.RegisterA2AProxy(mux, agents.Options{
			Cards:          agentCards,
			TokenProvider:  authHandler,
			KagentA2AURL:   os.Getenv("KAGENT_A2A_URL"),
			AgentNamespace: os.Getenv("KAGENT_AGENT_NAMESPACE"),
		})
		slog.Info("a2a proxy embedded in gateway (A2A_PROXY_URL not set)")
	}

	config.Register(mux, config.Options{
		Values: map[string]string{
			"github_org":             httputil.EnvOr("GITHUB_ORG", ""),
			"github_repo":            httputil.EnvOr("GITHUB_REPO", "complytime-studio"),
			"registry_insecure":      httputil.EnvOr("REGISTRY_INSECURE", ""),
			"model_provider":         httputil.EnvOr("MODEL_PROVIDER", ""),
			"model_name":             httputil.EnvOr("MODEL_NAME", ""),
			"auto_persist_artifacts": httputil.EnvOr("AUTO_PERSIST_ARTIFACTS", "true"),
		},
	})

	web.Register(mux, workbench.Assets)

	var handler http.Handler = mux
	if authCfg.ClientID != "" {
		var userStore auth.UserStore
		if st != nil {
			userStore = st
		}
		adminGuard := auth.RequireAdmin(userStore)
		handler = authHandler.Middleware(writeProtect(mux, adminGuard))
		slog.Info("auth enabled", "issuer", authCfg.Provider.IssuerURL)
	} else {
		slog.Info("auth disabled")
	}

	if origins := splitComma(os.Getenv("CORS_ORIGINS")); len(origins) > 0 {
		handler = httputil.CORS(httputil.CORSOptions{AllowedOrigins: origins})(handler)
		slog.Info("CORS enabled", "origins", origins)
	}

	handler = httputil.SecurityHeaders(handler)

	addr := net.JoinHostPort("0.0.0.0", port)
	internalAddr := net.JoinHostPort("0.0.0.0", internalPort)

	srv := &http.Server{
		Addr:           addr,
		Handler:        handler,
		ReadTimeout:    consts.ServerReadTimeout,
		WriteTimeout:   consts.ServerWriteTimeout,
		IdleTimeout:    consts.ServerIdleTimeout,
		MaxHeaderBytes: 1 << 20,
	}
	internalSrv := &http.Server{
		Addr:           internalAddr,
		Handler:        internalMux,
		ReadTimeout:    consts.ServerReadTimeout,
		WriteTimeout:   consts.ServerWriteTimeout,
		IdleTimeout:    consts.ServerIdleTimeout,
		MaxHeaderBytes: 1 << 20,
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
		_ = internalSrv.Shutdown(shutdownCtx)
	}()

	if os.Getenv("NETWORKPOLICY_ENFORCED") == "" {
		slog.Warn("NETWORKPOLICY_ENFORCED is unset — internal port has no auth; ensure a NetworkPolicy restricts access in production",
			"internal_addr", internalAddr)
	}
	go func() {
		slog.Info("internal listener starting", "addr", internalAddr)
		if err := internalSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("internal http server failed", "error", err)
			os.Exit(1)
		}
	}()

	slog.Info("gateway starting", "addr", addr)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		slog.Error("http server failed", "error", err)
		os.Exit(1)
	}
	_ = ctx
}

func registerGemaraProxy(mux *http.ServeMux, mcpURL string) {
	if mcpURL == "" {
		unavailable := httputil.UnavailableHandler("gemara-mcp unavailable")
		mux.HandleFunc("/api/validate", unavailable)
		mux.HandleFunc("/api/migrate", unavailable)
		slog.Info("gemara-mcp proxy disabled", "reason", "GEMARA_MCP_URL not set")
		return
	}

	transport := &mcp.StreamableClientTransport{Endpoint: mcpURL}
	p, err := proxy.New(transport)
	if err != nil {
		slog.Warn("gemara-mcp proxy disabled", "error", err)
		unavailable := httputil.UnavailableHandler("gemara-mcp unavailable")
		mux.HandleFunc("/api/validate", unavailable)
		mux.HandleFunc("/api/migrate", unavailable)
		return
	}

	mux.HandleFunc("/api/validate", p.ValidateHandler())
	mux.HandleFunc("/api/migrate", p.MigrateHandler())
	slog.Info("gemara-mcp proxy registered", "routes", []string{"/api/validate", "/api/migrate"})
}

func splitComma(raw string) []string {
	if raw == "" {
		return nil
	}
	var out []string
	for _, s := range strings.Split(raw, ",") {
		s = strings.TrimSpace(s)
		if s != "" {
			out = append(out, s)
		}
	}
	return out
}

// writeProtect wraps a handler so that POST/PUT/PATCH/DELETE requests to /api/*
// pass through the adminGuard middleware first. GET and non-API requests are unaffected.
// /internal/* paths are served on the internal port only — reject on the public mux.
// See docs/decisions/internal-endpoint-isolation.md.
func writeProtect(next http.Handler, adminGuard func(http.Handler) http.Handler) http.Handler {
	guarded := adminGuard(next)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/internal/") {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		if strings.HasPrefix(r.URL.Path, "/api/") && r.Method != http.MethodGet {
			if r.URL.Path == "/api/chat/history" || r.URL.Path == "/api/bootstrap" || strings.HasPrefix(r.URL.Path, "/api/a2a/") {
				next.ServeHTTP(w, r)
				return
			}
			guarded.ServeHTTP(w, r)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// notificationAdapter adapts store.Store to events.NotificationWriter.
type notificationAdapter struct {
	store interface {
		InsertNotification(ctx context.Context, n store.Notification) error
	}
}

func (a *notificationAdapter) InsertNotification(
	ctx context.Context, n events.Notification,
) error {
	return a.store.InsertNotification(ctx, store.Notification{
		NotificationID: n.NotificationID,
		Type:           n.Type,
		PolicyID:       n.PolicyID,
		Payload:        n.Payload,
	})
}

// buildCertifierPipeline constructs the day-one certifier pipeline from
// environment configuration.
func buildCertifierPipeline() *certifier.Pipeline {
	knownRegistries := make(map[string]bool)
	for _, r := range splitComma(os.Getenv("KNOWN_REGISTRIES")) {
		knownRegistries[r] = true
	}
	knownEngines := make(map[string]bool)
	for _, e := range splitComma(os.Getenv("KNOWN_ENGINES")) {
		knownEngines[e] = true
	}
	return certifier.NewPipeline(
		&certifier.SchemaCertifier{},
		&certifier.ProvenanceCertifier{KnownRegistries: knownRegistries},
		&certifier.ExecutorCertifier{KnownEngines: knownEngines},
	)
}

// certificationAdapter bridges store.Store to the events.CertificationQuerier
// and events.CertificationWriter interfaces.
type certificationAdapter struct {
	store interface {
		QueryRecentEvidence(
			ctx context.Context, policyID string, since time.Time,
		) ([]store.EvidenceRowLite, error)
		InsertCertifications(ctx context.Context, rows []store.CertificationRow) error
		UpdateEvidenceCertified(ctx context.Context, evidenceID string, certified bool) error
	}
}

func (a *certificationAdapter) QueryRecentEvidence(
	ctx context.Context, policyID string, since time.Time,
) ([]certifier.EvidenceRow, error) {
	rows, err := a.store.QueryRecentEvidence(ctx, policyID, since)
	if err != nil {
		return nil, err
	}
	out := make([]certifier.EvidenceRow, len(rows))
	for i, r := range rows {
		out[i] = certifier.EvidenceRow{
			EvidenceID:       r.EvidenceID,
			TargetID:         r.TargetID,
			RuleID:           r.RuleID,
			EvalResult:       r.EvalResult,
			ComplianceStatus: r.ComplianceStatus,
			EngineName:       r.EngineName,
			SourceRegistry:   r.SourceRegistry,
			AttestationRef:   r.AttestationRef,
			EnrichmentStatus: r.EnrichmentStatus,
			CollectedAt:      r.CollectedAt,
		}
	}
	return out, nil
}

func (a *certificationAdapter) InsertCertifications(
	ctx context.Context, rows []events.CertificationRow,
) error {
	storeRows := make([]store.CertificationRow, len(rows))
	for i, r := range rows {
		storeRows[i] = store.CertificationRow{
			EvidenceID:       r.EvidenceID,
			Certifier:        r.Certifier,
			CertifierVersion: r.CertifierVersion,
			Result:           r.Result,
			Reason:           r.Reason,
		}
	}
	return a.store.InsertCertifications(ctx, storeRows)
}

func (a *certificationAdapter) UpdateEvidenceCertified(
	ctx context.Context, evidenceID string, certified bool,
) error {
	return a.store.UpdateEvidenceCertified(ctx, evidenceID, certified)
}

// cookieSecretKey returns the 32-byte AES-256 key for session cookie encryption.
// Reads hex-encoded key from COOKIE_SECRET. Falls back to COOKIE_SIGN_KEY for
// backward compatibility. Generates an ephemeral key for development only.
func cookieSecretKey(authEnabled bool) []byte {
	raw := os.Getenv("COOKIE_SECRET")
	if raw == "" {
		raw = os.Getenv("COOKIE_SIGN_KEY")
	}
	if raw != "" {
		key, err := hex.DecodeString(raw)
		if err != nil || len(key) != 32 {
			slog.Error("COOKIE_SECRET must be 64 hex chars (32 bytes)", "hint", "openssl rand -hex 32")
			os.Exit(1)
		}
		return key
	}
	if authEnabled {
		slog.Error("COOKIE_SECRET is required when auth is enabled", "hint", "openssl rand -hex 32")
		os.Exit(1)
	}
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		slog.Error("failed to generate cookie secret", "error", err)
		os.Exit(1)
	}
	return key
}
