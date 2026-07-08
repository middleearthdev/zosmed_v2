// Package main is the entry point for the Zosmed asynq worker.
// It bootstraps all infrastructure (DB, Redis, asynq), creates the wired Runner,
// registers task handlers, and runs the server with graceful shutdown.
package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/hibiken/asynq"
	"github.com/redis/go-redis/v9"

	"github.com/zosmed/zosmed/apps/worker/internal/runner"
	"github.com/zosmed/zosmed/apps/worker/internal/tasks"
	"github.com/zosmed/zosmed/libs/igapi"
	"github.com/zosmed/zosmed/libs/platform/config"
	"github.com/zosmed/zosmed/libs/platform/db"
	platformlog "github.com/zosmed/zosmed/libs/platform/log"
	"github.com/zosmed/zosmed/libs/platform/queue"
	ptasks "github.com/zosmed/zosmed/libs/platform/tasks"
)

const workerConcurrency = 10

// tokenRefreshCron runs the token refresh sweep every 6 hours (ADR-002 §5).
// Long-lived tokens live ~60 days, so this cadence comfortably catches every
// account before its 7-day refresh-lead window (see tasks.refreshLeadWindow).
const tokenRefreshCron = "0 */6 * * *"

// reservationReconcileCron runs the expired-reservation backstop every minute
// (MAJOR-3b). Frequent because it releases held stock; the default hold is 5
// minutes, so a 1-minute sweep keeps worst-case over-hold small.
const reservationReconcileCron = "@every 1m"

func main() {
	log := platformlog.Logger
	ctx := context.Background()

	// ── Config ──────────────────────────────────────────────────────────────
	cfg, err := config.Load()
	if err != nil {
		log.Error("worker: config load failed", slog.String("error", err.Error()))
		os.Exit(1)
	}

	// ── Postgres ─────────────────────────────────────────────────────────────
	pool, err := db.New(ctx, cfg.DBURL)
	if err != nil {
		log.Error("worker: db pool failed", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer pool.Close()

	// ── Redis (shared by asynq + safety gate) ────────────────────────────────
	redisOpt, err := redis.ParseURL(cfg.RedisURL)
	if err != nil {
		log.Error("worker: parse redis URL", slog.String("error", err.Error()))
		os.Exit(1)
	}
	rdb := redis.NewClient(redisOpt)
	defer rdb.Close()

	// ── asynq client (for enqueueing reservation:expire tasks) ───────────────
	asynqClient, err := queue.NewClient(cfg.RedisURL)
	if err != nil {
		log.Error("worker: asynq client failed", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer asynqClient.Close()

	// ── Wire runner (engine + safety gate + seller kit) ──────────────────────
	// Per-account IG tokens are looked up from Postgres inside task handlers
	// (ADR-002 §6.2) — the Runner itself no longer holds a static IG token.
	r := runner.New(pool, rdb, asynqClient, cfg.WAPhone)

	// ── Instagram Login OAuth (for the token refresh sweep, ADR-002 §5) ──────
	oauthCfg := igapi.OAuthConfig{
		AppID:       cfg.IGAppID,
		AppSecret:   cfg.IGAppSecret,
		RedirectURI: cfg.IGRedirectURI,
	}

	// ── Register task handlers ────────────────────────────────────────────────
	ingestHandler := tasks.NewCommentIngestHandler(r, log)
	dmIngestHandler := tasks.NewDMIngestHandler(r, log)
	expireHandler := tasks.NewReservationExpireHandler(r, log)
	tokenRefreshHandler := tasks.NewTokenRefreshHandler(r.DB, oauthCfg, log)
	reconcileHandler := tasks.NewReservationReconcileHandler(r.DB, r.Svc, log)
	outboundHandler := tasks.NewOutboundSendHandler(
		r.DB, r.Gate, r.Svc,
		func(token string) tasks.PrivateReplySender { return igapi.New(token) },
		log,
	)

	mux := asynq.NewServeMux()
	mux.HandleFunc(ptasks.TaskCommentIngest, ingestHandler.ProcessTask)
	mux.HandleFunc(ptasks.TaskDMIngest, dmIngestHandler.ProcessTask)
	mux.HandleFunc(ptasks.TaskReservationExpire, expireHandler.ProcessTask)
	mux.HandleFunc(ptasks.TaskTokenRefreshSweep, tokenRefreshHandler.ProcessTask)
	mux.HandleFunc(ptasks.TaskReservationReconcile, reconcileHandler.ProcessTask)
	mux.HandleFunc(ptasks.TaskOutboundSend, outboundHandler.ProcessTask)

	// ── asynq scheduler (periodic tasks; ADR-002 §5 — first scheduler in the
	// worker, future periodic tasks e.g. reservation reconcile register onto
	// this same instance rather than creating a second one) ─────────────────
	scheduler, err := queue.NewScheduler(cfg.RedisURL)
	if err != nil {
		log.Error("worker: asynq scheduler failed", slog.String("error", err.Error()))
		os.Exit(1)
	}
	if _, err := scheduler.Register(tokenRefreshCron, asynq.NewTask(ptasks.TaskTokenRefreshSweep, nil)); err != nil {
		log.Error("worker: register token refresh sweep failed", slog.String("error", err.Error()))
		os.Exit(1)
	}
	if _, err := scheduler.Register(reservationReconcileCron, asynq.NewTask(ptasks.TaskReservationReconcile, nil)); err != nil {
		log.Error("worker: register reservation reconcile failed", slog.String("error", err.Error()))
		os.Exit(1)
	}

	// ── asynq server ──────────────────────────────────────────────────────────
	srv, err := queue.NewServer(cfg.RedisURL, workerConcurrency)
	if err != nil {
		log.Error("worker: asynq server failed", slog.String("error", err.Error()))
		os.Exit(1)
	}

	// ── Graceful shutdown ─────────────────────────────────────────────────────
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-quit
		log.Info("worker: shutdown signal received")
		scheduler.Shutdown()
		srv.Shutdown()
	}()

	go func() {
		log.Info("worker: scheduler starting",
			slog.String("token_refresh_cron", tokenRefreshCron),
			slog.String("reservation_reconcile_cron", reservationReconcileCron),
		)
		if err := scheduler.Run(); err != nil {
			log.Error("worker: scheduler error", slog.String("error", err.Error()))
		}
	}()

	log.Info("worker: starting", slog.Int("concurrency", workerConcurrency))
	if err := srv.Run(mux); err != nil {
		log.Error("worker: server error", slog.String("error", err.Error()))
		os.Exit(1)
	}
}
