package nodes

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/zosmed/zosmed/libs/workflow"
)

// outboundWebhookTimeout bounds every outbound POST (ADR-005 §2.4).
const outboundWebhookTimeout = 5 * time.Second

// outboundWebhookSignatureHeader carries the optional HMAC-SHA256 signature
// of the JSON body, hex-encoded (ADR-005 §2.4 step 3). This is Zosmed's OWN
// outbound signature for the receiver to verify — unrelated to Meta's
// X-Hub-Signature-256, which verifies INBOUND webhooks from Meta (CLAUDE.md §4.0).
const outboundWebhookSignatureHeader = "X-Zosmed-Signature-256"

// outboundWebhookConfig is the config shape for NodeTypeOutboundWebhook
// (CLAUDE.md §7 "Webhook (keluar)"; ADR-005 §2.4). Secret is not part of the
// ADR-005 §2.4 sketch but is required here to actually compute the optional
// HMAC signature — it is the caller's OWN shared secret for verifying the
// signed payload on their receiving end, unrelated to the Meta App Secret
// used to verify inbound webhooks (§4.0).
type outboundWebhookConfig struct {
	URL              string `json:"url"`
	IncludeSignature bool   `json:"includeSignature,omitempty"`
	Secret           string `json:"secret,omitempty"`
}

// outboundWebhookPayload is the JSON body posted to config.url (ADR-005 §2.4 step 2).
type outboundWebhookPayload struct {
	Event     string            `json:"event"`
	AccountID string            `json:"account_id"`
	From      string            `json:"from"`
	Text      string            `json:"text"`
	MediaID   string            `json:"media_id"`
	Vars      map[string]string `json:"vars"`
}

// httpDoer is the minimal subset of *http.Client this action needs —
// extracted so tests can inject a fake without a real network call or
// tripping the SSRF guard (which correctly rejects httptest's loopback URLs).
type httpDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

// outboundWebhookAction POSTs a JSON payload to a user-controlled URL — NOT
// an Instagram outbound call (CLAUDE.md §7 "Webhook (keluar)"): rc.Gate is
// NOT consulted (that gate exists solely for IG quota, §10 one-door is about
// IG sends specifically), but the URL is guarded against SSRF
// (loopback/link-local/private hosts rejected at Build time, ADR-005 R6) and
// every call is time-bounded.
//
// Guardrail note: URL validation happens in BuildOutboundWebhook, not here.
// Because the compiler builds a fresh node instance from Compile every
// single event (compile-per-event, ADR-004 R1), this still re-validates on
// every send — there is no stale, once-at-startup check to worry about.
type outboundWebhookAction struct {
	url              string
	includeSignature bool
	secret           string
	client           httpDoer
}

// BuildOutboundWebhook is the Factory.Build func for NodeTypeOutboundWebhook.
func BuildOutboundWebhook(cfg json.RawMessage) (any, error) {
	var c outboundWebhookConfig
	if len(cfg) > 0 {
		if err := json.Unmarshal(cfg, &c); err != nil {
			return nil, fmt.Errorf("nodes: outbound-webhook: parse config: %w", err)
		}
	}
	if strings.TrimSpace(c.URL) == "" {
		return nil, fmt.Errorf("nodes: outbound-webhook: config.url is required")
	}
	if err := validateOutboundWebhookURL(c.URL); err != nil {
		return nil, fmt.Errorf("nodes: outbound-webhook: %w", err)
	}
	if c.IncludeSignature && strings.TrimSpace(c.Secret) == "" {
		return nil, fmt.Errorf("nodes: outbound-webhook: config.secret is required when includeSignature is true")
	}
	return &outboundWebhookAction{
		url:              c.URL,
		includeSignature: c.IncludeSignature,
		secret:           c.Secret,
		client:           &http.Client{Timeout: outboundWebhookTimeout},
	}, nil
}

// Execute posts the JSON payload. Non-2xx responses (and transport errors)
// are reported in ActionResult.Detail but do NOT fail the run (ADR-005 §2.4
// step 4) — a misbehaving third-party endpoint must not break the rest of
// the workflow.
func (a *outboundWebhookAction) Execute(ctx context.Context, rc *workflow.RunContext) (workflow.ActionResult, error) {
	body := outboundWebhookPayload{
		Event:     rc.Event.Source,
		AccountID: rc.Event.AccountID,
		From:      rc.Event.FromUsername,
		Text:      rc.Event.Text,
		MediaID:   rc.Event.MediaID,
		Vars:      rc.Vars,
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return workflow.ActionResult{}, fmt.Errorf("nodes: outbound-webhook: marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.url, bytes.NewReader(payload))
	if err != nil {
		return workflow.ActionResult{}, fmt.Errorf("nodes: outbound-webhook: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if a.includeSignature {
		req.Header.Set(outboundWebhookSignatureHeader, signOutboundWebhookPayload(a.secret, payload))
	}

	resp, err := a.client.Do(req)
	if err != nil {
		// A network failure to a user-controlled endpoint is reported, not
		// fatal to the run (same tolerance as a non-2xx response below).
		return workflow.ActionResult{Detail: fmt.Sprintf("webhook gagal terkirim: %v", err)}, nil
	}
	defer func() {
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
	}()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return workflow.ActionResult{Detail: fmt.Sprintf("webhook non-2xx (status=%d)", resp.StatusCode)}, nil
	}
	return workflow.ActionResult{Detail: fmt.Sprintf("webhook terkirim (status=%d)", resp.StatusCode)}, nil
}

func signOutboundWebhookPayload(secret string, payload []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

// validateOutboundWebhookURL enforces ADR-005 R6's minimal anti-SSRF guard:
// http/https only, and the host must not resolve to a loopback, link-local,
// or private (RFC1918/ULA) address. No domain allowlist — R6 explicitly
// accepted this reduced scope for the MVP.
func validateOutboundWebhookURL(raw string) error {
	u, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("invalid url: %w", err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("url scheme must be http or https, got %q", u.Scheme)
	}
	host := u.Hostname()
	if host == "" {
		return fmt.Errorf("url has no host")
	}
	if strings.EqualFold(host, "localhost") {
		return fmt.Errorf("url host %q is not allowed (loopback)", host)
	}

	ips, err := net.LookupIP(host)
	if err != nil {
		// A literal IP always resolves without a real DNS query; a genuine
		// resolution failure here means the URL is currently unusable anyway.
		return fmt.Errorf("cannot resolve host %q: %w", host, err)
	}
	for _, ip := range ips {
		if isDisallowedOutboundIP(ip) {
			return fmt.Errorf("url host %q resolves to a disallowed address (%s)", host, ip)
		}
	}
	return nil
}

// isDisallowedOutboundIP rejects loopback, link-local, and private
// (RFC1918/ULA) addresses — the minimal anti-SSRF guard (ADR-005 R6).
func isDisallowedOutboundIP(ip net.IP) bool {
	return ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() ||
		ip.IsPrivate() || ip.IsUnspecified()
}
