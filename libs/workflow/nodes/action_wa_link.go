package nodes

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/zosmed/zosmed/libs/workflow"
)

// Event.Raw keys populated by apps/worker/internal/tasks/comment_ingest.go for
// EVERY comment event, regardless of which workflow ends up running (ADR-004
// §4.2/§4.4.3: "konteks runtime tetap ditulis handler ingest"). The string
// values MUST stay identical to seller.RawKeyIgUserID / seller.RawKeyCommentAt
// — duplicated here as literals (not imported) so this neutral package never
// depends on libs/kits/seller (§9 guardrail). This is a shared wire-key
// convention, not shared logic, so the small duplication is acceptable
// (§12a-4 rule of three).
const (
	rawKeyIgUserID  = "ig_user_id"
	rawKeyCommentAt = "comment_at"
)

// defaultWaLinkTemplate is the Indonesian olshop-style default reply text
// when no custom template is configured (CLAUDE.md §12 — default copy in
// Bahasa Indonesia). Placeholders: {nama}, {wa_link}.
const defaultWaLinkTemplate = "Halo kak {nama}! Yuk lanjut ngobrol di WhatsApp ya: {wa_link}"

// waLinkKind identifies this action's outbound kind to the safety gate: a
// PRIVATE reply DELIVERED as a DM (POST /{ig-id}/messages, recipient
// {comment_id}), so it counts against the DM caps (§4c: 200/hr→Queue,
// 1000/day) plus the ≤7-day window — same Kind as seller.privateReplyAction.
// Duplicated as a literal (not imported from libs/safety) — MUST stay
// identical to safety.KindPrivateReply / seller's "private-reply" literal.
const waLinkKind = "private-reply"

// sendWhatsAppLinkConfig is the config shape for NodeTypeSendWhatsAppLink.
type sendWhatsAppLinkConfig struct {
	Phone    string `json:"phone"`              // E.164 without '+', e.g. "6281234567890"
	Template string `json:"template,omitempty"` // placeholders: {nama}, {wa_link}
}

// sendWhatsAppLinkAction sends exactly ONE private reply containing a
// prefilled wa.me link (CLAUDE.md §4c "1 private reply per comment"; §8.1.1
// "handoff ke WhatsApp — nol API eksternal, murni URL ber-encode").
type sendWhatsAppLinkAction struct {
	phone    string
	template string

	// enqueueDeferred schedules an outbound:send retry when the gate returns
	// DecisionQueue (ADR-007 §2.1); nil falls back to report-only.
	enqueueDeferred EnqueueDeferredFunc
}

// newSendWhatsAppLinkFactory returns the Factory.Build func for
// NodeTypeSendWhatsAppLink, binding enqueueDeferred (ADR-007 §3.7) into every
// built instance.
func newSendWhatsAppLinkFactory(enqueueDeferred EnqueueDeferredFunc) func(json.RawMessage) (any, error) {
	return func(cfg json.RawMessage) (any, error) {
		var c sendWhatsAppLinkConfig
		if len(cfg) > 0 {
			if err := json.Unmarshal(cfg, &c); err != nil {
				return nil, fmt.Errorf("nodes: send-whatsapp-link: parse config: %w", err)
			}
		}
		if strings.TrimSpace(c.Phone) == "" {
			return nil, fmt.Errorf("nodes: send-whatsapp-link: config.phone is required")
		}
		tmpl := c.Template
		if strings.TrimSpace(tmpl) == "" {
			tmpl = defaultWaLinkTemplate
		}
		return &sendWhatsAppLinkAction{phone: c.Phone, template: tmpl, enqueueDeferred: enqueueDeferred}, nil
	}
}

// Execute sends the private reply. Guardrail (§10 one-door): rc.Gate.Allow is
// called BEFORE rc.Sender is ever touched. Decision handling mirrors
// seller.privateReplyAction exactly:
//   - Allow  -> send via rc.Sender.SendPrivateReply
//   - Queue  -> enqueue a generic outbound:send retry (ADR-007 #3), or report
//     deferred only if enqueueDeferred is nil (no reservation state to hold
//     here, unlike the seller kit)
//   - Reject -> skipped, reported only
func (a *sendWhatsAppLinkAction) Execute(ctx context.Context, rc *workflow.RunContext) (workflow.ActionResult, error) {
	nama := rc.Event.FromUsername
	igUserID := rawString(rc.Event.Raw, rawKeyIgUserID)

	waLink := buildWaMeLink(a.phone, nama)
	replyText := strings.NewReplacer("{nama}", nama, "{wa_link}", waLink).Replace(a.template)
	commentAt := rawTime(rc.Event.Raw, rawKeyCommentAt)

	req := workflow.OutboundReq{
		AccountID:    rc.Event.AccountID,
		Kind:         waLinkKind,
		TargetUserID: rc.Event.FromID,
		TriggerKey:   rc.Event.ObjectID,
		CommentID:    rc.Event.ObjectID,
		CommentAt:    commentAt,
		PostID:       rc.Event.MediaID,
	}

	// Guardrail F (§10 one-door): gate check BEFORE any igapi call.
	d, err := rc.Gate.Allow(ctx, req)
	if err != nil {
		return workflow.ActionResult{}, fmt.Errorf("nodes: send-whatsapp-link: gate: %w", err)
	}

	switch d.Action {
	case workflow.DecisionAllow:
		if err := rc.Sender.SendPrivateReply(ctx, igUserID, rc.Event.ObjectID, replyText); err != nil {
			return workflow.ActionResult{}, fmt.Errorf("nodes: send-whatsapp-link: send: %w", err)
		}
		return workflow.ActionResult{Detail: "private reply dengan link WhatsApp terkirim"}, nil

	case workflow.DecisionQueue:
		// §10 overflow → antre, bukan ditolak (ADR-007 #3).
		if a.enqueueDeferred == nil {
			return workflow.ActionResult{
				Detail: fmt.Sprintf("gate=queue (%s); outbound ditunda (belum ada retry generik)", d.Reason),
			}, nil
		}
		def := DeferredOutbound{
			AccountID:    rc.Event.AccountID,
			Kind:         waLinkKind,
			IgUserID:     igUserID,
			TargetUserID: rc.Event.FromID,
			ObjectID:     rc.Event.ObjectID,
			Text:         replyText,
			CommentAt:    commentAt,
			PostID:       rc.Event.MediaID,
			TriggerKey:   rc.Event.ObjectID,
			Deadline:     DeadlineFrom(commentAt, PrivateReplyWindow), // §4c: 7 days from the comment
		}
		if err := a.enqueueDeferred(ctx, def, DeferredRetryDelay); err != nil {
			return workflow.ActionResult{}, fmt.Errorf("nodes: send-whatsapp-link: enqueue deferred: %w", err)
		}
		return workflow.ActionResult{
			Detail: fmt.Sprintf("gate=queue (%s); outbound diantre untuk retry", d.Reason),
		}, nil

	default: // DecisionReject
		return workflow.ActionResult{
			Detail: fmt.Sprintf("gate=reject (%s); tidak dikirim", d.Reason),
		}, nil
	}
}

// buildWaMeLink returns a prefilled wa.me deep link for WhatsApp handoff.
// phone must be E.164 without '+' (e.g. "6281234567890"). Pure URL
// construction — no external API call (§4b.6/§8.1.1).
func buildWaMeLink(phone, nama string) string {
	text := fmt.Sprintf("Halo, saya %s dari Instagram", nama)
	return "https://wa.me/" + phone + "?text=" + url.QueryEscape(text)
}

// ── Event.Raw helpers (local copies; see comment on the rawKey* constants) ───

func rawString(raw map[string]any, key string) string {
	if v, ok := raw[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func rawTime(raw map[string]any, key string) time.Time {
	if v, ok := raw[key]; ok {
		if t, ok := v.(time.Time); ok {
			return t
		}
	}
	return time.Time{}
}
