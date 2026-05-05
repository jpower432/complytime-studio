// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	"github.com/complytime/complytime-studio/internal/agents"
	"github.com/complytime/complytime-studio/internal/auth"
	"github.com/complytime/complytime-studio/internal/blob"
	"github.com/complytime/complytime-studio/internal/certifier"
	"github.com/complytime/complytime-studio/internal/config"
	"github.com/complytime/complytime-studio/internal/consts"
	"github.com/complytime/complytime-studio/internal/events"
	"github.com/complytime/complytime-studio/internal/httputil"
	pgstore "github.com/complytime/complytime-studio/internal/postgres"
	"github.com/complytime/complytime-studio/internal/proxy"
	"github.com/complytime/complytime-studio/internal/publish"
	"github.com/complytime/complytime-studio/internal/posture"
	"github.com/complytime/complytime-studio/internal/recommend"
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

	pgCfg, ok := pgstore.ConfigFromEnv()
	if !ok {
		slog.Error("POSTGRES_URL is required")
		os.Exit(1)
	}
	pgClient, err := pgstore.New(ctx, pgCfg)
	if err != nil {
		slog.Error("postgres connection failed", "error", err)
		os.Exit(1)
	}
	defer pgClient.Close()
	if err = pgClient.EnsureSchema(ctx); err != nil {
		slog.Error("postgres schema init failed", "error", err)
		os.Exit(1)
	}
	slog.Info("postgres ready")
	st := store.New(pgClient.Pool())

	// healthz and system-info are registered on Echo directly (below, after e is created).

	internalPort := httputil.EnvOr("INTERNAL_PORT", consts.DefaultInternalPort)

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
	registryConfig := store.LoadRegistryConfig()

	// Seed from registry in background — the Helm seed job may not have
	// pushed artifacts yet when the gateway starts (post-install hook race).
	go func() {
		const maxAttempts = 10
		const delay = 15 * time.Second
		for attempt := 1; attempt <= maxAttempts; attempt++ {
			if err := store.PopulateCatalogsFromRegistry(ctx, st, st, st, st, st, registryAddr); err != nil {
				slog.Warn("catalog seed from registry failed, will retry", "attempt", attempt, "error", err)
				select {
				case <-ctx.Done():
					return
				case <-time.After(delay):
					continue
				}
			}
			// Run dependent backfills after catalogs are seeded.
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
			slog.Info("registry seed populate complete", "attempt", attempt)
			return
		}
		slog.Warn("registry seed exhausted retries", "attempts", maxAttempts)
	}()

	if err := store.PopulateMappingEntries(ctx, st); err != nil {
		slog.Warn("mapping entries backfill failed", "error", err)
	}
	natsURL := os.Getenv("NATS_URL")
	if natsURL == "" {
		slog.Error("NATS_URL is required — event bus drives the certification pipeline")
		os.Exit(1)
	}
	bus, busErr := events.Connect(natsURL)
	if busErr != nil {
		slog.Error("nats connection failed", "error", busErr)
		os.Exit(1)
	}
	defer bus.Close()
	slog.Info("nats ready")

	var pub store.EventPublisher = bus

	var notifStore store.NotificationStore = st
	programStores := pgstore.NewProgramPG(pgClient.Pool())

	stores := store.Stores{
		Policies:            st,
		Mappings:            st,
		Evidence:            st,
		Blob:                blobStore,
		AuditLogs:           st,
		DraftAuditLogs:      st,
		Requirements:        st,
		Controls:            st,
		Guidance:            st,
		Threats:             st,
		Risks:               st,
		Catalogs:            st,
		EvidenceAssessments: st,
		Posture:             st,
		Notifications:       notifStore,
		Certifications:      st,
		EventPublisher:      pub,
		HealthChecker:       pgClient,
		Programs:            programStores,
		Jobs:                programStores,
		Inventory:           st,
		Users:               pgClient,
		Recommender:         recommend.New(pgClient.Pool()),
		Registry:            registryConfig,
	}
	slog.Info("store API registered", "routes", []string{
		"/api/policies", "/api/evidence/ingest", "/api/audit-logs", "/api/mappings",
	})

	adapter := &notificationAdapter{store: notifStore}
	rateCache := events.NewRateCache()
	postureHandler := events.PostureCheckHandler(ctx, st, adapter, rateCache)
	postureDebouncer := events.NewDebouncer(30*time.Second, postureHandler)

	pipeline := buildCertifierPipeline()
	certAdapter := &certificationAdapter{store: st}
	certHandler := events.CertificationHandler(ctx, pipeline, certAdapter, certAdapter)
	certDebouncer := events.NewDebouncer(30*time.Second, certHandler)

	sub, subErr := bus.SubscribeEvidence(func(evt events.EvidenceEvent) {
		postureDebouncer.Push(evt)
		certDebouncer.Push(evt)
	})
	if subErr != nil {
		slog.Error("nats subscribe failed", "error", subErr)
		os.Exit(1)
	}
	defer func() { _ = sub.Unsubscribe() }()
	slog.Info("nats evidence subscription active", "subject", events.SubjectEvidence+".>")

	postureEngine := posture.New(pgClient.Pool())
	postureNotifier := func(ctx context.Context, msg, severity string) error {
		return adapter.InsertNotification(ctx, "posture_change", severity, msg)
	}
	postureSub := posture.NewSubscriber(postureEngine, programStores, bus, postureNotifier)
	go func() {
		if err := postureSub.Start(ctx); err != nil && ctx.Err() == nil {
			slog.Error("posture subscriber stopped", "error", err)
		}
	}()
	slog.Info("posture subscriber started")

	draftSub, draftSubErr := bus.SubscribeDraftAuditLog(func(evt events.DraftAuditLogEvent) {
		payload := fmt.Sprintf(`{"draft_id":%q,"summary":%q}`, evt.DraftID, evt.Summary)
		if err := adapter.InsertNotification(ctx, "draft_audit_log", evt.PolicyID, payload); err != nil {
			slog.Warn("draft notification insert failed", "draft_id", evt.DraftID, "error", err)
		}
	})
	if draftSubErr != nil {
		slog.Error("nats draft subscribe failed", "error", draftSubErr)
		os.Exit(1)
	}
	defer func() { _ = draftSub.Unsubscribe() }()
	slog.Info("nats draft-audit-log subscription active", "subject", events.SubjectDraft+".>")

	apiToken := auth.APITokenFromEnv()
	authHandler := auth.NewHandler(apiToken)
	authHandler.SetUserStore(pgClient)

	chatStore := auth.NewMemoryChatStore()

	if apiToken != "" {
		if apiToken == consts.DefaultDevAPIToken {
			slog.Warn("STUDIO_API_TOKEN is the default dev value — rotate before production use")
		}
		slog.Info("api token auth enabled for seed/CI scripts")
	}
	slog.Info("auth: OAuth2 Proxy handles OIDC externally, gateway trusts X-Forwarded-* headers")

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

	// --- Echo server setup ---
	e := echo.New()
	e.HideBanner = true
	e.HidePort = true

	e.Use(middleware.Recover())
	e.Use(middleware.RequestID())
	e.Use(middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
		LogStatus: true, LogURI: true, LogMethod: true, LogLatency: true, LogError: true,
		LogValuesFunc: func(c echo.Context, v middleware.RequestLoggerValues) error {
			slog.Info("request",
				"method", v.Method, "uri", v.URI,
				"status", v.Status, "latency_ms", v.Latency.Milliseconds(),
				"error", v.Error,
			)
			return nil
		},
	}))
	e.Use(middleware.SecureWithConfig(middleware.SecureConfig{
		ContentSecurityPolicy: httputil.ContentSecurityPolicy,
		XFrameOptions:         "DENY",
		ContentTypeNosniff:    "nosniff",
		ReferrerPolicy:        "strict-origin-when-cross-origin",
	}))

	if origins := splitComma(os.Getenv("CORS_ORIGINS")); len(origins) > 0 {
		e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
			AllowOrigins:     origins,
			AllowMethods:     []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete, http.MethodOptions},
			AllowHeaders:     []string{"Content-Type", "Authorization"},
			AllowCredentials: true,
			MaxAge:           86400,
		}))
		slog.Info("CORS enabled", "origins", origins)
	}

	subsystems := map[string]pgstore.Pinger{
		"postgres": pgClient,
	}
	e.Use(echo.WrapMiddleware(pgstore.DegradedMiddleware(subsystems)))

	proxySecret := os.Getenv("PROXY_SECRET")
	if proxySecret != "" {
		slog.Info("proxy secret configured — X-Forwarded-* headers will be stripped from untrusted requests")
	} else {
		slog.Warn("PROXY_SECRET is not set — X-Forwarded-* headers are trusted from any source",
			"hint", "set PROXY_SECRET to a shared secret between OAuth2 Proxy and the gateway for production")
	}
	e.Use(auth.StripUntrustedProxyHeaders(proxySecret))

	authHandler.Register(e)
	e.Use(authHandler.Middleware())
	e.Use(writeProtect(auth.RequireWrite(pgClient)))

	e.GET("/healthz", func(c echo.Context) error {
		if err := pgClient.Ping(c.Request().Context()); err != nil {
			return c.String(http.StatusServiceUnavailable, "postgres unreachable")
		}
		return c.String(http.StatusOK, "ok")
	})

	apiGroup := e.Group("/api")
	apiGroup.Use(middleware.BodyLimit(fmt.Sprintf("%dM", consts.MaxRequestBody>>20)))
	store.Register(apiGroup, stores)
	authHandler.RegisterUserAPI(apiGroup)
	authHandler.RegisterChatHistory(apiGroup, chatStore)
	registerGemaraProxy(apiGroup, os.Getenv("GEMARA_MCP_URL"))

	apiGroup.GET("/system-info", func(c echo.Context) error {
		authProvider := "OAuth2 Proxy (external)"
		if os.Getenv("OAUTH2_PROXY_ENABLED") == "false" {
			authProvider = "none (dev mode)"
		}
		modelProvider := httputil.EnvOr("MODEL_PROVIDER", "not configured")
		modelName := httputil.EnvOr("MODEL_NAME", "")
		if modelName != "" {
			modelProvider = modelProvider + " (" + modelName + ")"
		}
		dbStatus := "connected"
		if err := pgClient.Ping(c.Request().Context()); err != nil {
			dbStatus = "unreachable"
		}
		agentNames := []string{}
		ns := os.Getenv("KAGENT_AGENT_NAMESPACE")
		if ns != "" {
			agentNames = append(agentNames, "kagent (ns: "+ns+")")
		}
		return c.JSON(http.StatusOK, map[string]any{
			"version":        httputil.EnvOr("STUDIO_VERSION", "dev"),
			"database":       "PostgreSQL — " + dbStatus,
			"auth_provider":  authProvider,
			"model_provider": modelProvider,
			"agents":         agentNames,
		})
	})

	slog.Info("api routes registered", "groups", []string{"store", "users", "gemara-proxy"})

	// Legacy API routes (agents, a2a, registry, publish, config) are registered
	// on the mux with full /api/ prefixes. The root catch-all delegates to them.
	// The SPA fallback is handled in the same catch-all for non-API paths.
	web.RegisterEchoWithMux(e, workbench.Assets, mux)

	addr := net.JoinHostPort("0.0.0.0", port)
	internalAddr := net.JoinHostPort("0.0.0.0", internalPort)

	internalE := echo.New()
	internalE.HideBanner = true
	internalE.HidePort = true
	internalE.HTTPErrorHandler = func(err error, c echo.Context) {
		if c.Response().Committed {
			return
		}
		he, ok := err.(*echo.HTTPError)
		if ok {
			msg := fmt.Sprintf("%v", he.Message)
			_ = c.JSON(he.Code, map[string]string{"error": msg})
		} else {
			slog.Error("internal handler error", "error", err, "path", c.Request().URL.Path)
			_ = c.JSON(http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		}
	}
	internalE.Use(middleware.BodyLimit(fmt.Sprintf("%dM", consts.MaxInternalRequestBody>>20)))
	store.RegisterInternal(internalE.Group(""), stores)

	internalSrv := &http.Server{
		Addr:           internalAddr,
		Handler:        internalE,
		ReadTimeout:    consts.ServerReadTimeout,
		WriteTimeout:   consts.ServerWriteTimeout,
		IdleTimeout:    consts.ServerIdleTimeout,
		MaxHeaderBytes: 1 << 20,
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = e.Shutdown(shutdownCtx)
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

	e.Server.ReadTimeout = consts.ServerReadTimeout
	e.Server.WriteTimeout = consts.ServerWriteTimeout
	e.Server.IdleTimeout = consts.ServerIdleTimeout
	e.Server.MaxHeaderBytes = 1 << 20

	slog.Info("gateway starting", "addr", addr)
	if err := e.Start(addr); err != nil && err != http.ErrServerClosed {
		slog.Error("http server failed", "error", err)
		os.Exit(1)
	}
}

func registerGemaraProxy(g *echo.Group, mcpURL string) {
	unavail := func(c echo.Context) error {
		return c.JSON(http.StatusServiceUnavailable, map[string]string{"error": "gemara-mcp unavailable"})
	}
	if mcpURL == "" {
		g.POST("/validate", unavail)
		g.POST("/migrate", unavail)
		slog.Info("gemara-mcp proxy disabled", "reason", "GEMARA_MCP_URL not set")
		return
	}

	transport := &mcp.StreamableClientTransport{Endpoint: mcpURL}
	p, err := proxy.New(transport)
	if err != nil {
		slog.Warn("gemara-mcp proxy disabled", "error", err)
		g.POST("/validate", unavail)
		g.POST("/migrate", unavail)
		return
	}

	g.POST("/validate", echo.WrapHandler(http.HandlerFunc(p.ValidateHandler())))
	g.POST("/migrate", echo.WrapHandler(http.HandlerFunc(p.MigrateHandler())))
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

// writeProtect gates POST/PUT/PATCH/DELETE on /api/* through writeGuard.
// GET and non-API requests pass through. /internal/* is rejected on the
// public port. See docs/decisions/internal-endpoint-isolation.md.
func writeProtect(writeGuard echo.MiddlewareFunc) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		guarded := writeGuard(next)
		return func(c echo.Context) error {
			path := c.Request().URL.Path
			method := c.Request().Method

			if strings.HasPrefix(path, "/internal/") {
				return c.String(http.StatusNotFound, "not found")
			}
			if strings.HasPrefix(path, "/api/") && method != http.MethodGet {
				if path == "/api/chat/history" || path == "/api/bootstrap" ||
					strings.HasPrefix(path, "/api/a2a/") ||
					(method == http.MethodPatch && strings.HasPrefix(path, "/api/notifications/") && strings.HasSuffix(path, "/read")) ||
					(method == http.MethodPost && path == "/api/audit-logs/promote") ||
					(method == http.MethodPatch && strings.HasPrefix(path, "/api/draft-audit-logs/")) {
					return next(c)
				}
				return guarded(c)
			}
			return next(c)
		}
	}
}

// notificationAdapter adapts store.NotificationStore to events.NotificationWriter.
type notificationAdapter struct {
	store store.NotificationStore
}

func (a *notificationAdapter) InsertNotification(
	ctx context.Context, notifType, policyID, payload string,
) error {
	return a.store.InsertNotification(ctx, store.Notification{
		Type:     notifType,
		PolicyID: policyID,
		Payload:  payload,
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
