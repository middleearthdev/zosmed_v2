package seller

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"
)

// OutboundRetry carries the context needed to re-attempt a private reply that
// the safety gate deferred (MAJOR-2). The concrete enqueue (asynq) lives in
// apps/worker/runner so the seller kit stays free of an asynq import.
type OutboundRetry struct {
	AccountID     string
	IgUserID      string
	CommentID     string
	TargetUserID  string
	ReservationID string
	ReplyText     string
	PostID        string
	TriggerKey    string
	CommentAt     time.Time
}

// EnqueueOutboundFunc schedules a private-reply retry after delay. Injected into
// RegisterNodes; nil disables retry (Queue outbound is simply reported deferred).
type EnqueueOutboundFunc func(ctx context.Context, r OutboundRetry, delay time.Duration) error

// OutboundRetryDelay is how long a deferred private reply waits before the
// outbound:send handler re-checks the safety gate (MAJOR-2). One minute gives
// the per-hour DM quota room to recover without holding the reservation too long
// (default hold is 5 minutes).
const OutboundRetryDelay = time.Minute

// defaultReplyTemplate is the Indonesian olshop-style private reply template.
// Used when no custom template is configured (§12 copy — all default text in BI
// gaya olshop). Placeholders: {nama}, {kode}, {produk}, {wa_link}.
const defaultReplyTemplate = "Halo kak {nama}! Kode *{kode}* untuk *{produk}* sudah dicatat ya ✅ Lanjut ke WhatsApp buat closing: {wa_link}"

// BuildWaLink returns a prefilled wa.me deep link for WhatsApp handoff.
// phone must be in E.164 format without '+' (e.g. "6281234567890").
// nama, kode, produk are substituted into the prefilled text message.
//
// Guardrail §8.1.1 / §4b.6: pure URL construction — no external API call.
// {nama} MUST come from the webhook payload (FromUsername) only — never scraped.
func BuildWaLink(phone, nama, kode, produk string) string {
	text := fmt.Sprintf("Halo, saya %s mau konfirmasi order %s untuk %s", nama, kode, produk)
	return "https://wa.me/" + phone + "?text=" + url.QueryEscape(text)
}

// BuildPrivateReplyText fills tmpl with {nama}, {kode}, {produk}, {wa_link}.
// Falls back to defaultReplyTemplate when tmpl is empty.
// The reply includes the wa.me link so the ENTIRE message fits in one outbound
// (guardrail A: one outbound per comment — no separate DM blast).
func BuildPrivateReplyText(tmpl, nama, kode, produk, waLink string) string {
	if strings.TrimSpace(tmpl) == "" {
		tmpl = defaultReplyTemplate
	}
	r := strings.NewReplacer(
		"{nama}", nama,
		"{kode}", kode,
		"{produk}", produk,
		"{wa_link}", waLink,
	)
	return r.Replace(tmpl)
}
