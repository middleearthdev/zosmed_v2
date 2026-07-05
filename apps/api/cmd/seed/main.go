// Command seed populates a dev database with a demo Zosmed user, a demo
// (dummy-token) Instagram account already linked and "connected", and a
// small comment-to-order catalog — so login, onboarding, and Comment-to-Order
// can all be exercised end-to-end without a real Instagram OAuth round-trip
// (ADR-003 §7).
//
// Usage (from repo root):
//
//	go run ./apps/api/cmd/seed          # segment/onboarding left NULL (test the onboarding flow)
//	go run ./apps/api/cmd/seed -complete # segment=seller + onboarding already completed
//
// Idempotent: safe to run repeatedly (upsert by email / ig_user_id / (account,media) / (post,code)).
// Refuses to run when APP_ENV=prod — this seed uses dummy, clearly-fake credentials
// and must never touch a production database.
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/zosmed/zosmed/apps/api/internal/auth"
	"github.com/zosmed/zosmed/apps/api/internal/connect"
	"github.com/zosmed/zosmed/libs/platform/config"
	"github.com/zosmed/zosmed/libs/platform/db"
	"github.com/zosmed/zosmed/libs/platform/dbgen"
	platformlog "github.com/zosmed/zosmed/libs/platform/log"
	"github.com/zosmed/zosmed/libs/platform/uuidx"
)

// Demo credentials — deliberately fake/obvious, never real secrets (ADR-003 §7).
const (
	demoEmail    = "demo@zosmed.test"
	demoPassword = "demo12345"

	demoIgUserID    = "SEED-IG-0001"
	demoHandle      = "olshop.aurora"
	demoDisplayName = "Aurora Olshop"
	demoAccessToken = "SEED-DUMMY-TOKEN-not-a-real-ig-token"

	demoMediaID = "SEED-MEDIA-0001"
	demoCaption = "Drop koleksi baru! Komen KEEP + kode buat checkout ya kak 🛍️"
)

func main() {
	complete := flag.Bool("complete", false, "seed the demo user as already onboarded (segment=seller, onboarding complete)")
	flag.Parse()

	log := platformlog.Logger
	ctx := context.Background()

	cfg, err := config.Load()
	if err != nil {
		log.Error("seed: config load failed", slog.String("error", err.Error()))
		os.Exit(1)
	}
	if cfg.IsProd() {
		log.Error("seed: refusing to run with APP_ENV=prod — this seed uses dummy credentials and must never touch production data")
		os.Exit(1)
	}

	pool, err := db.New(ctx, cfg.DBURL)
	if err != nil {
		log.Error("seed: db pool failed", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer pool.Close()

	queries := dbgen.New(pool)
	authStore := auth.NewStore(queries)
	connectStore := connect.NewStore(queries)

	user, err := seedUser(ctx, authStore, log)
	if err != nil {
		log.Error("seed: user", slog.String("error", err.Error()))
		os.Exit(1)
	}

	account, err := seedAccount(ctx, connectStore, user, log)
	if err != nil {
		log.Error("seed: account", slog.String("error", err.Error()))
		os.Exit(1)
	}

	post, err := seedCatalog(ctx, queries, account, log)
	if err != nil {
		log.Error("seed: catalog", slog.String("error", err.Error()))
		os.Exit(1)
	}

	if err := seedProducts(ctx, queries, post, log); err != nil {
		log.Error("seed: products", slog.String("error", err.Error()))
		os.Exit(1)
	}

	if *complete {
		if err := completeOnboarding(ctx, authStore, user, log); err != nil {
			log.Error("seed: complete onboarding", slog.String("error", err.Error()))
			os.Exit(1)
		}
	}

	fmt.Println("── Zosmed dev seed complete ──────────────────────────────────")
	fmt.Printf("Login:    %s / %s\n", demoEmail, demoPassword)
	fmt.Printf("Account:  @%s (status=connected, dummy token — no real Instagram API calls will work)\n", demoHandle)
	if *complete {
		fmt.Println("Onboarding: already completed (segment=seller) — demo user lands on /dashboard")
	} else {
		fmt.Println("Onboarding: NOT completed — demo user lands on /onboarding to exercise the full flow")
	}
	fmt.Println("───────────────────────────────────────────────────────────────")
}

// seedUser upserts the demo login user by email. Idempotent: a second run
// finds the existing row and skips re-hashing/re-inserting.
func seedUser(ctx context.Context, store *auth.Store, log *slog.Logger) (dbgen.AppUser, error) {
	if existing, err := store.UserByEmail(ctx, demoEmail); err == nil {
		log.Info("seed: demo user already exists, skipping create", slog.String("email", demoEmail))
		return existing, nil
	}

	hash, err := auth.HashPassword(demoPassword)
	if err != nil {
		return dbgen.AppUser{}, fmt.Errorf("hash demo password: %w", err)
	}
	user, err := store.CreateUser(ctx, demoEmail, hash)
	if err != nil {
		return dbgen.AppUser{}, fmt.Errorf("create demo user: %w", err)
	}
	log.Info("seed: created demo user", slog.String("email", demoEmail))
	return user, nil
}

// seedAccount upserts the demo Instagram account (status=connected, dummy
// token) linked to the demo user. UpsertAccountFromOAuth is idempotent by
// ig_user_id, so re-running this refreshes the row rather than duplicating it.
func seedAccount(ctx context.Context, store *connect.Store, user dbgen.AppUser, log *slog.Logger) (dbgen.Account, error) {
	acc, err := store.UpsertAccount(ctx, connect.UpsertAccountParams{
		IgUserID:       demoIgUserID,
		Handle:         demoHandle,
		DisplayName:    demoDisplayName,
		AccessToken:    demoAccessToken,
		TokenType:      "bearer",
		Scopes:         []string{"instagram_business_basic", "instagram_business_manage_comments", "instagram_business_manage_messages"},
		TokenExpiresAt: time.Now().Add(60 * 24 * time.Hour),
		UserID:         uuidx.Format(user.ID),
	})
	if err != nil {
		return dbgen.Account{}, fmt.Errorf("upsert demo account: %w", err)
	}
	log.Info("seed: upserted demo account", slog.String("handle", demoHandle), slog.String("ig_user_id", demoIgUserID))
	return acc, nil
}

// seedCatalog registers one demo post/Reel eligible for comment-to-order.
func seedCatalog(ctx context.Context, q *dbgen.Queries, account dbgen.Account, log *slog.Logger) (dbgen.CatalogPost, error) {
	post, err := q.UpsertCatalogPost(ctx, dbgen.UpsertCatalogPostParams{
		AccountID: account.ID,
		IgMediaID: demoMediaID,
		Caption:   demoCaption,
		Active:    true,
	})
	if err != nil {
		return dbgen.CatalogPost{}, fmt.Errorf("upsert demo catalog post: %w", err)
	}
	log.Info("seed: upserted demo catalog post", slog.String("ig_media_id", demoMediaID))
	return post, nil
}

// seedProducts registers two demo keep/C products on the catalog post.
func seedProducts(ctx context.Context, q *dbgen.Queries, post dbgen.CatalogPost, log *slog.Logger) error {
	demoProducts := []struct {
		code       string
		name       string
		priceIDR   int64
		stockTotal int32
	}{
		{"C1", "Kaos Oversize Aurora - Hitam", 89_000, 20},
		{"C2", "Kaos Oversize Aurora - Putih", 89_000, 20},
	}

	for _, p := range demoProducts {
		if _, err := q.UpsertProduct(ctx, dbgen.UpsertProductParams{
			CatalogPostID: post.ID,
			Code:          p.code,
			Name:          p.name,
			PriceIdr:      p.priceIDR,
			StockTotal:    p.stockTotal,
		}); err != nil {
			return fmt.Errorf("upsert demo product %s: %w", p.code, err)
		}
	}
	log.Info("seed: upserted demo products", slog.Int("count", len(demoProducts)))
	return nil
}

// completeOnboarding is used by the -complete flag to fast-forward a demo
// user straight to a "fully onboarded" state (segment=seller + stamped
// onboarding_completed_at), for testing screens that assume onboarding is done.
func completeOnboarding(ctx context.Context, store *auth.Store, user dbgen.AppUser, log *slog.Logger) error {
	if _, err := store.SetSegment(ctx, user.ID, "seller"); err != nil {
		return fmt.Errorf("set demo user segment: %w", err)
	}
	if _, err := store.CompleteOnboarding(ctx, user.ID); err != nil {
		return fmt.Errorf("complete demo user onboarding: %w", err)
	}
	log.Info("seed: marked demo user onboarding complete", slog.String("segment", "seller"))
	return nil
}
