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

	"github.com/zosmed/zosmed/apps/api/internal/commentorder"
	"github.com/zosmed/zosmed/apps/api/internal/enqueue"
	"github.com/zosmed/zosmed/apps/api/internal/httpx"
	"github.com/zosmed/zosmed/apps/api/internal/webhook"
	"github.com/zosmed/zosmed/libs/platform/config"
	"github.com/zosmed/zosmed/libs/platform/db"
	"github.com/zosmed/zosmed/libs/platform/dbgen"
	seller "github.com/zosmed/zosmed/libs/kits/seller"
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

	// ── Account resolution (MVP single-account) ─────────────────────────────────
	// Assumption: one business account per deployment in MVP phase.
	// IG_ACCOUNT_ID must be the Postgres UUID from the account table.
	// The webhook entry.id field (IG user ID string) is NOT the same as this UUID.
	// Production: replace with a DB lookup from the account table using the IG user ID
	// extracted from the webhook entry.id.
	accountIDStr := os.Getenv("IG_ACCOUNT_ID")
	accountID, err := seller.ParseUUID(accountIDStr)
	if err != nil {
		// Non-fatal at startup: the server still handles REST endpoints.
		// Webhooks will fail the catalog lookup and log a debug message.
		log.Warn("api: IG_ACCOUNT_ID missing or invalid — webhook ingest disabled",
			slog.String("raw", accountIDStr),
			slog.String("error", err.Error()),
		)
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

	// ── Build handlers ────────────────────────────────────────────────────────────
	whHandler := webhook.New(queries, enqClient, cfg.MetaAppSecret, cfg.MetaVerifyToken, accountID, log)
	coHandler := commentorder.NewHandler(queries)

	// ── Wire router ───────────────────────────────────────────────────────────────
	router := httpx.NewRouter(httpx.Routes{
		WebhookChallenge: whHandler.Challenge,
		WebhookReceive:   whHandler.Receive,
		GetCommentOrder:  coHandler.GetCommentOrder,
		GetReservation:   coHandler.GetReservation,
		CloseReservation: coHandler.CloseReservation,
		GetSettings:      coHandler.GetSettings,
		PutSettings:      coHandler.PutSettings,
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
