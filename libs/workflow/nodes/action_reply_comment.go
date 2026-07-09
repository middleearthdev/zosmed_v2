package nodes

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/zosmed/zosmed/libs/workflow"
)

// defaultReplyCommentTemplate is the Indonesian olshop-style default public
// reply when no custom template is configured (CLAUDE.md §12 — default copy
// in Bahasa Indonesia). Placeholder: {nama}.
const defaultReplyCommentTemplate = "Halo kak {nama}, makasih ya udah komen! Kita cek dulu sebentar ya 🙏"

// replyCommentKind identifies this action's outbound kind to the safety gate:
// a PUBLIC comment reply (POST /{comment-id}/replies, CLAUDE.md §7 "Reply
// comment"), metered against the comment-reply caps (§4c: 750/hr,
// 30/post/5min) — NOT the DM caps used by private-reply/DM. Duplicated as a
// literal (not imported from libs/safety) so this neutral package never
// imports libs/safety directly; the runner's gateAdapter forwards Kind
// verbatim to safety.OutboundReq (ADR-004 §1), so this string MUST stay
// identical to safety.KindCommentReply.
const replyCommentKind = "comment-reply"

// replyCommentConfig is the config shape for NodeTypeReplyComment.
type replyCommentConfig struct {
	Template string `json:"template,omitempty"`
}

// replyCommentAction posts exactly ONE public reply to the triggering
// comment (CLAUDE.md §7 "Reply comment", §4a ALLOW). Distinct from
// send-whatsapp-link/send-dm, which are private (DM) — this action stays
// public and never touches the DM/private-reply quota.
type replyCommentAction struct {
	template string

	// enqueueDeferred schedules an outbound:send retry when the gate returns
	// DecisionQueue (ADR-007 §2.1); nil falls back to report-only.
	enqueueDeferred EnqueueDeferredFunc
}

// newReplyCommentFactory returns the Factory.Build func for
// NodeTypeReplyComment, binding enqueueDeferred (ADR-007 §3.7) into every
// built instance.
func newReplyCommentFactory(enqueueDeferred EnqueueDeferredFunc) func(json.RawMessage) (any, error) {
	return func(cfg json.RawMessage) (any, error) {
		var c replyCommentConfig
		if len(cfg) > 0 {
			if err := json.Unmarshal(cfg, &c); err != nil {
				return nil, fmt.Errorf("nodes: reply-comment: parse config: %w", err)
			}
		}
		tmpl := c.Template
		if strings.TrimSpace(tmpl) == "" {
			tmpl = defaultReplyCommentTemplate
		}
		return &replyCommentAction{template: tmpl, enqueueDeferred: enqueueDeferred}, nil
	}
}

// Execute posts the public reply. Guardrail (§10 one-door): rc.Gate.Allow is
// called BEFORE rc.Sender is ever touched, using Kind=comment-reply (public
// comment-reply caps, distinct from the DM caps used by
// send-whatsapp-link/seller.private-reply). Decision handling mirrors
// sendWhatsAppLinkAction/seller.privateReplyAction exactly:
//   - Allow  -> send via rc.Sender.ReplyToComment
//   - Queue  -> enqueue a generic outbound:send retry (ADR-007 #3), or report
//     deferred only if enqueueDeferred is nil
//   - Reject -> skipped, reported only
func (a *replyCommentAction) Execute(ctx context.Context, rc *workflow.RunContext) (workflow.ActionResult, error) {
	nama := rc.Event.FromUsername
	replyText := strings.ReplaceAll(a.template, "{nama}", nama)
	commentAt := rawTime(rc.Event.Raw, rawKeyCommentAt)

	req := workflow.OutboundReq{
		AccountID:    rc.Event.AccountID,
		Kind:         replyCommentKind,
		TargetUserID: rc.Event.FromID,
		TriggerKey:   rc.Event.ObjectID,
		CommentID:    rc.Event.ObjectID,
		CommentAt:    commentAt,
		PostID:       rc.Event.MediaID,
	}

	// Guardrail (§10 one-door): gate check BEFORE any igapi call.
	d, err := rc.Gate.Allow(ctx, req)
	if err != nil {
		return workflow.ActionResult{}, fmt.Errorf("nodes: reply-comment: gate: %w", err)
	}

	switch d.Action {
	case workflow.DecisionAllow:
		if err := rc.Sender.ReplyToComment(ctx, rc.Event.ObjectID, replyText); err != nil {
			return workflow.ActionResult{}, fmt.Errorf("nodes: reply-comment: send: %w", err)
		}
		return workflow.ActionResult{Detail: "balasan komentar publik terkirim"}, nil

	case workflow.DecisionQueue:
		// §10 overflow → antre, bukan ditolak (ADR-007 #3).
		if a.enqueueDeferred == nil {
			return workflow.ActionResult{
				Detail: fmt.Sprintf("gate=queue (%s); balasan komentar ditunda (belum ada retry generik)", d.Reason),
			}, nil
		}
		def := DeferredOutbound{
			AccountID:    rc.Event.AccountID,
			Kind:         replyCommentKind,
			IgUserID:     rawString(rc.Event.Raw, rawKeyIgUserID),
			TargetUserID: rc.Event.FromID,
			ObjectID:     rc.Event.ObjectID,
			Text:         replyText,
			CommentAt:    commentAt,
			PostID:       rc.Event.MediaID,
			TriggerKey:   rc.Event.ObjectID,
			// comment-reply has no §4c hard window (window.go) — use the fixed TTL.
			Deadline: DeadlineFrom(commentAt, DeferredCommentReplyTTL),
		}
		if err := a.enqueueDeferred(ctx, def, DeferredRetryDelay); err != nil {
			return workflow.ActionResult{}, fmt.Errorf("nodes: reply-comment: enqueue deferred: %w", err)
		}
		return workflow.ActionResult{
			Detail: fmt.Sprintf("gate=queue (%s); balasan komentar diantre untuk retry", d.Reason),
		}, nil

	default: // DecisionReject
		return workflow.ActionResult{
			Detail: fmt.Sprintf("gate=reject (%s); balasan komentar tidak dikirim", d.Reason),
		}, nil
	}
}
