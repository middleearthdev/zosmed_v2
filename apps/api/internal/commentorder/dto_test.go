package commentorder

import (
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/zosmed/zosmed/libs/platform/dbgen"
)

// ── formatPriceLabel ──────────────────────────────────────────────────────────

func TestFormatPriceLabel(t *testing.T) {
	tests := []struct {
		price int64
		want  string
	}{
		{0, "Rp —"},
		{-1000, "Rp —"},
		{500, "Rp 500"},
		{999, "Rp 999"},
		{1_000, "Rp 1rb"},
		{5_000, "Rp 5rb"},
		{189_000, "Rp 189rb"},
		{999_000, "Rp 999rb"},
		{1_000_000, "Rp 1jt"},
		{2_000_000, "Rp 2jt"},
		{1_500_000, "Rp 1.5jt"},
		{1_250_000, "Rp 1.2jt"},
		{10_000_000, "Rp 10jt"},
	}

	for _, tc := range tests {
		got := formatPriceLabel(tc.price)
		if got != tc.want {
			t.Errorf("formatPriceLabel(%d) = %q, want %q", tc.price, got, tc.want)
		}
	}
}

// ── countdownLabel ────────────────────────────────────────────────────────────

func TestCountdownLabel(t *testing.T) {
	tests := []struct {
		status dbgen.ReservationStatus
		want   string
	}{
		{dbgen.ReservationStatusReserved, "—"},
		{dbgen.ReservationStatusWaitingPay, "—"},
		{dbgen.ReservationStatusClosedWa, "✓ closed"},
		{dbgen.ReservationStatusExpiredReleased, "— released"},
	}

	for _, tc := range tests {
		got := countdownLabel(tc.status)
		if got != tc.want {
			t.Errorf("countdownLabel(%q) = %q, want %q", tc.status, got, tc.want)
		}
	}
}

// ── agoLabel ─────────────────────────────────────────────────────────────────

func TestAgoLabel_Invalid(t *testing.T) {
	got := agoLabel(pgtype.Timestamptz{Valid: false})
	if got != "—" {
		t.Errorf("agoLabel(invalid) = %q, want %q", got, "—")
	}
}

func TestAgoLabel_JustNow(t *testing.T) {
	ts := pgtype.Timestamptz{Time: time.Now().Add(-10 * time.Second), Valid: true}
	got := agoLabel(ts)
	if got != "baru saja" {
		t.Errorf("agoLabel(<1min) = %q, want %q", got, "baru saja")
	}
}

func TestAgoLabel_Minutes(t *testing.T) {
	ts := pgtype.Timestamptz{Time: time.Now().Add(-5 * time.Minute), Valid: true}
	got := agoLabel(ts)
	if got != "5 mnt lalu" {
		t.Errorf("agoLabel(5min) = %q, want %q", got, "5 mnt lalu")
	}
}

func TestAgoLabel_Hours(t *testing.T) {
	ts := pgtype.Timestamptz{Time: time.Now().Add(-3 * time.Hour), Valid: true}
	got := agoLabel(ts)
	if got != "3 jam lalu" {
		t.Errorf("agoLabel(3hr) = %q, want %q", got, "3 jam lalu")
	}
}

func TestAgoLabel_Days(t *testing.T) {
	ts := pgtype.Timestamptz{Time: time.Now().Add(-48 * time.Hour), Valid: true}
	got := agoLabel(ts)
	if got != "2 hari lalu" {
		t.Errorf("agoLabel(2days) = %q, want %q", got, "2 hari lalu")
	}
}

// ── formatCommentCount ────────────────────────────────────────────────────────

func TestFormatCommentCount(t *testing.T) {
	tests := []struct {
		n    int32
		want string
	}{
		{0, "0 komentar"},
		{1, "1 komentar"},
		{123, "123 komentar"},
	}
	for _, tc := range tests {
		got := formatCommentCount(tc.n)
		if got != tc.want {
			t.Errorf("formatCommentCount(%d) = %q, want %q", tc.n, got, tc.want)
		}
	}
}

// ── mapStats ──────────────────────────────────────────────────────────────────

func TestMapStats_Keys(t *testing.T) {
	row := dbgen.GetCommentOrderStatsRow{
		TotalDetected:   10,
		ReservedNow:     3,
		WaitingPay:      1,
		ClosedWa:        5,
		ExpiredReleased: 2,
	}
	stats := mapStats(row)
	if len(stats) != 4 {
		t.Fatalf("expected 4 stats, got %d", len(stats))
	}

	wantKeys := []string{"code-detected", "reserved-now", "closed-wa", "expired"}
	for i, s := range stats {
		if s.Key != wantKeys[i] {
			t.Errorf("stats[%d].Key = %q, want %q", i, s.Key, wantKeys[i])
		}
	}

	// Spot-check values.
	if stats[0].Value != "10" {
		t.Errorf("code-detected value: want %q, got %q", "10", stats[0].Value)
	}
	if stats[2].Value != "5" {
		t.Errorf("closed-wa value: want %q, got %q", "5", stats[2].Value)
	}
}
