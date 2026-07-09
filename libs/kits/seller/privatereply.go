package seller

import (
	"fmt"
	"net/url"
	"strings"
)

// Deferred-outbound retry (ADR-007 §2.1/§2.3 migration, formerly this file's
// own OutboundRetry/EnqueueOutboundFunc/OutboundRetryDelay): the seller kit
// now enqueues a generic outbound:send retry via nodes.EnqueueDeferredFunc /
// nodes.DeferredOutbound / nodes.DeferredRetryDelay — the same contract every
// neutral action node uses (kit.go privateReplyAction.Execute). One task, one
// handler, no seller-only duplicate (§12a-1).

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
