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

	"github.com/zosmed/zosmed/libs/platform/config"
	"github.com/zosmed/zosmed/libs/platform/db"
	platformlog "github.com/zosmed/zosmed/libs/platform/log"
	"github.com/zosmed/zosmed/libs/platform/queue"
	ptasks "github.com/zosmed/zosmed/libs/platform/tasks"
	"github.com/zosmed/zosmed/apps/worker/internal/runner"
	"github.com/zosmed/zosmed/apps/worker/internal/tasks"
)

const workerConcurrency = 10

func main() {
	log := platformlog.Logger
	ctx := context.Background()

	// ── Config ──────────────────────────────────────────────────────────────
	cfg, err := config.Load()
	if err != nil {
		log.Error("worker: config load failed", slog.String("error", err.Error()))
		os.Exit(1)
	}

	// Extra env vars for the worker (not in shared Config to avoid touching libs/platform).
	igToken := os.Getenv("IG_ACCESS_TOKEN")    // IG page access token for igapi.Client
	igAccountUserID := os.Getenv("IG_ACCOUNT_USER_ID") // IG user ID of the business account
	if igToken == "" || igAccountUserID == "" {
		log.Warn("worker: IG_ACCESS_TOKEN or IG_ACCOUNT_USER_ID not set; igapi calls will fail")
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
	r := runner.New(pool, rdb, asynqClient, cfg.WAPhone, igToken)

	// ── Register task handlers ────────────────────────────────────────────────
	ingestHandler := tasks.NewCommentIngestHandler(r, log)
	expireHandler := tasks.NewReservationExpireHandler(r, log)

	mux := asynq.NewServeMux()
	mux.HandleFunc(ptasks.TaskCommentIngest, ingestHandler.ProcessTask)
	mux.HandleFunc(ptasks.TaskReservationExpire, expireHandler.ProcessTask)

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
		srv.Shutdown()
	}()

	log.Info("worker: starting", slog.Int("concurrency", workerConcurrency))
	if err := srv.Run(mux); err != nil {
		log.Error("worker: server error", slog.String("error", err.Error()))
		os.Exit(1)
	}
}
