package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/fiber/v2/middleware/requestid"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gorm.io/gorm"

	"wa-saas/backend/internal/config"
	"wa-saas/backend/internal/database"
	"wa-saas/backend/internal/handler"
	"wa-saas/backend/internal/middleware"
	"wa-saas/backend/internal/model"
	"wa-saas/backend/internal/service"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "config: %v\n", err)
		os.Exit(1)
	}

	log := newLogger(cfg.LogLevel)
	defer func() { _ = log.Sync() }()

	db, err := database.NewPostgres(cfg.DatabaseURL, log)
	if err != nil {
		log.Fatal("postgres", zap.Error(err))
	}
	if err := db.AutoMigrate(
		&model.User{},
		&model.Workspace{},
		&model.WorkspaceMember{},
		&model.RefreshToken{},
		&model.WhatsAppInstance{},
		&model.Conversation{},
		&model.Message{},
		&model.WebhookMessage{},
		&model.MediaTempToken{},
		&model.AIAgent{},
		&model.Flow{},
		&model.Campaign{},
	); err != nil {
		log.Fatal("migrate", zap.Error(err))
	}

	rdb, err := database.NewRedis(cfg.RedisURL)
	if err != nil {
		log.Fatal("redis client", zap.Error(err))
	}

	var ev *service.EvolutionClient
	if cfg.WhatsAppProvider == "evolution" {
		ev = service.NewEvolutionClient(cfg.EvolutionBaseURL, cfg.EvolutionAPIKey)
	} else {
		log.Info("whatsapp sem Evolution REST",
			zap.String("provider", cfg.WhatsAppProvider),
			zap.String("hint", "auto-reply grava só em DB se não houver Evolution; use WHATSAPP_PROVIDER=evolution para envio"),
		)
	}
	llm, err := service.NewLLM(cfg)
	if err != nil {
		log.Fatal("llm", zap.Error(err))
	}
	if llm != nil {
		log.Info("llm ativo", zap.String("provider", service.ProviderName(llm)))
	} else if cfg.AutoReplyEnabled {
		log.Info("auto-resposta ligada sem LLM global (.env); usa agente por workspace ou defina GEMINI/OPENAI_API_KEY como fallback")
	}

	// WriteTimeout tem de exceder o maior context.WithTimeout dos handlers (ex.: sync-contacts 120s).
	// Caso contrário o fasthttp devolve 408 Request Timeout enquanto o handler ainda corre.
	// ReadTimeout generoso para WebSocket (ping ~25s) e corpos lentos.
	app := fiber.New(fiber.Config{
		AppName:      "wa-saas-api",
		ReadTimeout:  120 * time.Second,
		WriteTimeout: 180 * time.Second,
		BodyLimit:    40 * 1024 * 1024, // uploads de mídia (multipart)
		ErrorHandler: errorHandler(log),
	})

	app.Use(recover.New())
	app.Use(requestid.New())
	app.Use(logger.New())
	app.Use(cors.New(cors.Config{
		AllowOrigins:     joinOrigins(cfg.CORSAllowOrigins),
		AllowMethods:     "GET,POST,PATCH,PUT,DELETE,OPTIONS",
		AllowHeaders:     "Origin, Content-Type, Accept, Authorization, X-Internal-API-Key, X-Webhook-Secret",
		AllowCredentials: true,
		// Sem isto, em pedidos cross-origin (ex.: Vite :5173 → API :8088) o browser oculta
		// Content-Type para valores como audio/ogg — o axios recebe Blob sem tipo e o <audio> falha.
		ExposeHeaders: "Content-Type, Content-Length, Content-Disposition",
	}))

	app.Get("/health", handler.HandleHealth(handler.HealthDeps{DB: db, Redis: rdb}))
	app.Get("/media/temp/:token", handler.HandlePublicMediaTemp(db, log))
	app.Get("/api/v1/meta", handler.HandleMeta(cfg))

	// WebhookAuth na rota (não no Group): assim :instance_id está disponível em c.Params para validar o token da instância.
	wh := app.Group("/webhooks", limiter.New(limiter.Config{
		Max:        180,
		Expiration: 1 * time.Minute,
	}))
	wh.Post("/whatsapp/:instance_id",
		middleware.WebhookAuth(cfg, db, log),
		handler.HandleWhatsAppWebhook(handler.WebhookDeps{
			Log:       log,
			DB:        db,
			Redis:     rdb,
			Cfg:       cfg,
			Evolution: ev,
			LLM:       llm,
		}))

	auth := app.Group("/api/v1/auth", limiter.New(limiter.Config{Max: 60, Expiration: time.Minute}))
	auth.Post("/register", handler.HandleRegister(log, db, cfg))
	auth.Post("/login", handler.HandleLogin(log, db, cfg))
	auth.Post("/refresh", handler.HandleRefresh(log, db, cfg))
	auth.Post("/logout", handler.HandleLogout(log, db))

	// JWT por rota: app.Group("/api/v1", JWT) ou api.Use(JWT) no prefix /api/v1 aplica JWT a
	// /api/v1/internal/* e quebra POST /api/v1/internal/evolution/send (só chave interna).
	authMW := middleware.RequireAuth(cfg)
	api := app.Group("/api/v1")
	api.Get("/auth/me", authMW, handler.HandleMe(db))
	api.Patch("/auth/me", authMW, handler.HandlePatchMe(log, db, cfg))

	api.Get("/instances", authMW, handler.HandleListInstances(log, db, cfg, ev))
	api.Post("/instances", authMW, handler.HandleCreateInstance(log, db, cfg, ev))
	api.Post("/instances/import", authMW, handler.HandleImportInstance(log, db, cfg, ev))
	api.Get("/instances/:id", authMW, handler.HandleGetInstance(db, cfg, ev))
	api.Delete("/instances/:id", authMW, handler.HandleDeleteInstance(log, db, cfg, ev))
	api.Get("/instances/:id/qrcode", authMW, handler.HandleInstanceQRCode(log, db, cfg, ev))
	api.Post("/instances/:id/sync-webhook", authMW, handler.HandleSyncInstanceWebhook(log, db, cfg, ev))
	api.Post("/instances/:id/sync-chats", authMW, handler.HandleSyncChatHistory(log, db, rdb, cfg, ev))
	api.Post("/instances/:id/reconcile-inbox", authMW, handler.HandleReconcileInboxWebhooks(log, db, rdb))
	api.Post("/instances/:id/sync-contacts", authMW, handler.HandleSyncWhatsAppContacts(log, db, rdb, cfg, ev))

	api.Get("/conversations", authMW, handler.HandleListConversations(db))
	api.Post("/conversations", authMW, handler.HandleCreateConversation(db, rdb))
	api.Delete("/conversations/:id", authMW, handler.HandleDeleteConversation(db, rdb))
	api.Get("/conversations/:id/messages", authMW, handler.HandleListMessages(db))
	api.Get("/conversations/:id/messages/:message_id/attachment", authMW, handler.HandleGetMessageAttachment(log, db, cfg, ev))
	api.Post("/conversations/:id/messages", authMW, handler.HandleSendMessage(log, db, rdb, cfg, ev))
	api.Post("/conversations/:id/messages/media", authMW, handler.HandleSendConversationMedia(log, db, rdb, cfg, ev))

	api.Get("/contacts", authMW, handler.HandleListContacts(db))
	api.Get("/analytics/overview", authMW, handler.HandleAnalyticsOverview(db))
	api.Get("/analytics/timeseries", authMW, handler.HandleAnalyticsTimeseries(db))
	api.Get("/workspace", authMW, handler.HandleGetWorkspace(db))
	api.Patch("/workspace", authMW, handler.HandlePatchWorkspace(db))

	api.Get("/agents", authMW, handler.HandleListAgents(db, cfg))
	api.Post("/elevenlabs/outbound-call", authMW, handler.HandleElevenLabsOutboundCall(cfg, log))
	api.Post("/agents", authMW, handler.HandleCreateAgent(log, db, cfg))
	api.Patch("/agents/:id", authMW, handler.HandlePatchAgent(log, db, cfg))
	api.Delete("/agents/:id", authMW, handler.HandleDeleteAgent(db, cfg))
	api.Get("/agents/:id/voice-preview", authMW, handler.HandleGetAgentVoicePreview(db, cfg))
	api.Post("/agents/:id/test", authMW, handler.HandleTestAgent(db, cfg))
	api.Get("/flows", authMW, handler.HandleListFlows(db))
	api.Post("/flows", authMW, handler.HandleCreateFlow(log, db))
	api.Get("/campaigns", authMW, handler.HandleListCampaigns(db))
	api.Post("/campaigns", authMW, handler.HandleCreateCampaign(log, db))
	api.Get("/kanban/board", authMW, handler.HandleKanbanBoard(db))
	api.Patch("/kanban/cards/:conversation_id", authMW, handler.HandleKanbanMoveCard(db))

	app.Get("/ws", handler.WebSocketRoute(cfg, log, rdb))

	internal := app.Group("/api/v1/internal", middleware.InternalAPIKey(cfg.InternalAPIKey))
	internal.Post("/evolution/send", handler.HandleEvolutionSend(log, cfg, ev))

	addr := ":" + cfg.HTTPPort
	log.Info("listening",
		zap.String("addr", addr),
		zap.String("env", cfg.Env),
		zap.String("public_webhook_base_url", cfg.PublicWebhookBaseURL),
		zap.String("public_media_base_url", cfg.PublicMediaBaseURL),
		zap.String("whatsapp_provider", cfg.WhatsAppProvider),
		zap.String("evolution_base_url", cfg.EvolutionBaseURL),
	)

	go func() {
		t := time.NewTicker(5 * time.Minute)
		defer t.Stop()
		for range t.C {
			service.PurgeExpiredMediaTokens(db, log)
		}
	}()

	go func() {
		if err := app.Listen(addr); err != nil {
			log.Fatal("listen", zap.Error(err))
		}
	}()

	shutdown(app, db, rdb, log)
}

func joinOrigins(origins []string) string {
	if len(origins) == 0 {
		return ""
	}
	out := origins[0]
	for i := 1; i < len(origins); i++ {
		out += "," + origins[i]
	}
	return out
}

func errorHandler(log *zap.Logger) fiber.ErrorHandler {
	return func(c *fiber.Ctx, err error) error {
		code := fiber.StatusInternalServerError
		if e, ok := err.(*fiber.Error); ok {
			code = e.Code
		}
		// 408: quase sempre WriteTimeout/ReadTimeout vs handler longo (Evolution sync) ou WS idle.
		log.Warn("http error",
			zap.Int("status", code),
			zap.String("method", c.Method()),
			zap.String("path", c.Path()),
			zap.Error(err),
		)
		return handler.JSONError(c, code, "internal_error", err.Error(), nil)
	}
}

func newLogger(level string) *zap.Logger {
	var zapLevel zapcore.Level
	switch level {
	case "debug":
		zapLevel = zapcore.DebugLevel
	case "warn":
		zapLevel = zapcore.WarnLevel
	case "error":
		zapLevel = zapcore.ErrorLevel
	default:
		zapLevel = zapcore.InfoLevel
	}
	zcfg := zap.NewProductionConfig()
	zcfg.Level = zap.NewAtomicLevelAt(zapLevel)
	zl, err := zcfg.Build()
	if err != nil {
		zl = zap.NewNop()
	}
	return zl
}

func shutdown(app *fiber.App, db *gorm.DB, rdb *redis.Client, log *zap.Logger) {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	// WebSocket e pedidos longos podem exiger mais tempo; 15s gerava "context deadline exceeded" ao Ctrl+C.
	const shutdownTimeout = 45 * time.Second
	if err := app.ShutdownWithTimeout(shutdownTimeout); err != nil {
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) ||
			strings.Contains(strings.ToLower(err.Error()), "deadline exceeded") {
			log.Warn("fiber shutdown: tempo esgotado ao fechar conexões (normal com WS/clientes ativos)",
				zap.Duration("timeout", shutdownTimeout),
				zap.Error(err))
		} else {
			log.Error("fiber shutdown", zap.Error(err))
		}
	}
	if sqlDB, err := db.DB(); err == nil {
		_ = sqlDB.Close()
	}
	if rdb != nil {
		_ = rdb.Close()
	}
	log.Info("shutdown complete")
}
