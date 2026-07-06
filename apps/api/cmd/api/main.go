// Package main is the entry point for the Zosmed API server.
// It bootstraps configuration, database pool, Redis/asynq client, HTTP handlers,
// and the chi router, then runs the server with graceful shutdown on SIGINT/SIGTERM.
package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/zosmed/zosmed/apps/api/internal/auth"
	"github.com/zosmed/zosmed/apps/api/internal/commentorder"
	"github.com/zosmed/zosmed/apps/api/internal/connect"
	"github.com/zosmed/zosmed/apps/api/internal/enqueue"
	"github.com/zosmed/zosmed/apps/api/internal/httpx"
	"github.com/zosmed/zosmed/apps/api/internal/webhook"
	"github.com/zosmed/zosmed/apps/api/internal/workflow"
	"github.com/zosmed/zosmed/libs/igapi"
	"github.com/zosmed/zosmed/libs/platform/config"
	"github.com/zosmed/zosmed/libs/platform/db"
	"github.com/zosmed/zosmed/libs/platform/dbgen"
	platformlog "github.com/zosmed/zosmed/libs/platform/log"
	"github.com/zosmed/zosmed/libs/platform/queue"
)

func main() {
	log := platformlog.Logger
	ctx := context.Background()

	// ── Config ──────────────────────────────────────────────────────────────────
	cfg, err := config.Load()
	if err != nil {
		log.Error("api: config load failed", slog.String("error", err.Error()))
		os.Exit(1)
	}

	// ── Postgres ─────────────────────────────────────────────────────────────────
	pool, err := db.New(ctx, cfg.DBURL)
	if err != nil {
		log.Error("api: db pool failed", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer pool.Close()

	queries := dbgen.New(pool)

	// ── asynq client (for enqueueing comment:ingest tasks) ───────────────────────
	asynqClient, err := queue.NewClient(cfg.RedisURL)
	if err != nil {
		log.Error("api: asynq client failed", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer asynqClient.Close()

	enqClient := enqueue.New(asynqClient)

	// ── Instagram Login OAuth (ADR-002 §3) ───────────────────────────────────────
	oauthCfg := igapi.OAuthConfig{
		AppID:       cfg.IGAppID,
		AppSecret:   cfg.IGAppSecret,
		RedirectURI: cfg.IGRedirectURI,
	}
	connectStore := connect.NewStore(queries)
	connectHandler := connect.New(oauthCfg, cfg.IGAppSecret, connectStore, log)

	// ── Zosmed login + onboarding (ADR-003) ──────────────────────────────────────
	authStore := auth.NewStore(queries)
	authHandler := auth.New(authStore, cfg.IsProd(), log)
	requireUser := auth.RequireUser(authStore)

	// ── Build handlers ────────────────────────────────────────────────────────────
	// Account resolution is per-webhook-entry now (entry.id → GetAccountByIgUserID,
	// ADR-002 §6.1) — no more single-account env var.
	whHandler := webhook.New(queries, enqClient, cfg.IGAppSecret, cfg.IGVerifyToken, log)
	coHandler := commentorder.NewHandler(queries)

	// ── Workflow builder (ADR-004) ────────────────────────────────────────────────
	workflowStore := workflow.NewStore(pool, queries)
	workflowHandler := workflow.NewHandler(queries, workflowStore)

	// ── Wire router ───────────────────────────────────────────────────────────────
	router := httpx.NewRouter(httpx.Routes{
		WebhookChallenge:    whHandler.Challenge,
		WebhookReceive:      whHandler.Receive,
		ConnectStart:        connectHandler.Start,
		ConnectCallback:     connectHandler.Callback,
		GetCommentOrder:     coHandler.GetCommentOrder,
		GetReservation:      coHandler.GetReservation,
		CloseReservation:    coHandler.CloseReservation,
		GetSettings:         coHandler.GetSettings,
		PutSettings:         coHandler.PutSettings,
		Register:            authHandler.Register,
		Login:               authHandler.Login,
		Logout:              authHandler.Logout,
		Me:                  authHandler.Me,
		PutSegment:          authHandler.PutSegment,
		CompleteOnboarding:  authHandler.CompleteOnboarding,
		ListWorkflows:       workflowHandler.List,
		CreateWorkflow:      workflowHandler.Create,
		GetWorkflow:         workflowHandler.Get,
		SaveWorkflow:        workflowHandler.Save,
		DeleteWorkflow:      workflowHandler.Delete,
		ActivateWorkflow:    workflowHandler.Activate,
		PauseWorkflow:       workflowHandler.Pause,
		ListRunsForWorkflow: workflowHandler.ListRunsForWorkflow,
		ListRunsForAccount:  workflowHandler.ListRunsForAccount,
		RequireUser:         requireUser,
	})

	// ── HTTP server ───────────────────────────────────────────────────────────────
	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// ── Graceful shutdown ─────────────────────────────────────────────────────────
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-quit
		log.Info("api: shutdown signal received")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			log.Error("api: graceful shutdown error", slog.String("error", err.Error()))
		}
	}()

	log.Info("api: starting", slog.String("addr", srv.Addr))
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Error("api: server error", slog.String("error", err.Error()))
		os.Exit(1)
	}
	log.Info("api: stopped")
}
