package nodes

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/zosmed/zosmed/libs/workflow"
)

// sendDMKind identifies this action's outbound kind to the safety gate: a
// standard free-form DM (POST /{ig-user-id}/messages, recipient {id}, §4a),
// metered against the DM caps (200/hr→Queue, 1000/day) and the 24h messaging
// window (§4c) — as opposed to KindPrivateReply/KindCommentReply. Duplicated
// as a literal (not imported from libs/safety, §9 guardrail) — MUST stay
// identical to safety.KindDM (ADR-006 R7).
const sendDMKind = "dm"

// defaultSendDMTemplate is the Indonesian olshop-style default DM text when no
// custom template is configured (CLAUDE.md §12 default copy). It deliberately
// does NOT use {nama}: the messaging surface never carries the contact's
// username (webhook sets FromUsername="" — ADR-006 R6), so a {nama} default
// would always render "Halo kak !" with a blank name. Custom templates may
// still opt into {nama}; a blank name only collapses the trailing space then.
const defaultSendDMTemplate = "Halo kak! Makasih udah chat kita ya 😊"

// sendDMConfig is the config shape for NodeTypeSendDM (ADR-006 §2 table).
type sendDMConfig struct {
	Template string `json:"template,omitempty"`
}

// sendDMAction sends a standard free-form DM within the 24h messaging window
// (CLAUDE.md §7 "Send DM"; ADR-006 §2.3). Meaningful ONLY on flows where the
// triggering event carries an open messaging window (DM/story/ad-referral) —
// on a comment-triggered flow it always skips (ADR-006 R4), since a comment
// never opens the messaging window (§4c).
type sendDMAction struct {
	template string
}

// BuildSendDM is the Factory.Build func for NodeTypeSendDM.
func BuildSendDM(cfg json.RawMessage) (any, error) {
	var c sendDMConfig
	if len(cfg) > 0 {
		if err := json.Unmarshal(cfg, &c); err != nil {
			return nil, fmt.Errorf("nodes: send-dm: parse config: %w", err)
		}
	}
	tmpl := c.Template
	if strings.TrimSpace(tmpl) == "" {
		tmpl = defaultSendDMTemplate
	}
	return &sendDMAction{template: tmpl}, nil
}

// Execute sends the DM. Guardrails (ADR-006 §2.3/§9, reviewer-enforced):
//
//  1. GUARD PRESENCE first: if Raw[last_interaction_at] is absent/zero, skip
//     WITHOUT touching rc.Gate or rc.Sender. This must happen before the gate
//     because safety.checkWindow allows a zero CommentAt for Kind=dm (the
//     caller is responsible for window tracking) — without this guard,
//     send-dm on a comment-triggered flow would incorrectly pass the gate.
//  2. §10 ONE-DOOR: rc.Gate.Allow(Kind="dm") BEFORE rc.Sender.SendDM.
//  3. Dedupe (account, user, message-id) is enforced inside the gate — not a
//     blast (§4b.6): exactly one DM per (contact, triggering message).
//  4. No PostID is set — send-dm is never a comment-reply, so the per-post/
//     5-min counter does not apply.
func (a *sendDMAction) Execute(ctx context.Context, rc *workflow.RunContext) (workflow.ActionResult, error) {
	last := rawTime(rc.Event.Raw, rawKeyLastInteractionAt)
	if last.IsZero() {
		// GUARD (ADR-006 R4): no open messaging window — e.g. a comment-triggered
		// flow, which never populates last_interaction_at. Skip locally, never
		// touch the gate or the Sender.
		return workflow.ActionResult{Detail: "skip: tidak ada window 24 jam terbuka"}, nil
	}

	// {nama} substitution. The messaging surface usually has no username
	// (FromUsername=="" — ADR-006 R6); when blank, drop the placeholder (and an
	// adjacent space) instead of leaving an empty slot like "Halo kak !".
	nama := rc.Event.FromUsername
	var text string
	if nama == "" {
		text = strings.ReplaceAll(a.template, "{nama} ", "")
		text = strings.ReplaceAll(text, "{nama}", "")
	} else {
		text = strings.ReplaceAll(a.template, "{nama}", nama)
	}
	igUserID := rawString(rc.Event.Raw, rawKeyIgUserID)

	req := workflow.OutboundReq{
		AccountID:    rc.Event.AccountID,
		Kind:         sendDMKind,
		TargetUserID: rc.Event.FromID,
		TriggerKey:   rc.Event.ObjectID,
		CommentAt:    last, // gate enforces 24h freshness from here (§4c)
		PostID:       "",   // never a comment-reply — per-post/5min N/A
	}

	// Guardrail (§10 one-door): gate check BEFORE any igapi call.
	d, err := rc.Gate.Allow(ctx, req)
	if err != nil {
		return workflow.ActionResult{}, fmt.Errorf("nodes: send-dm: gate: %w", err)
	}

	switch d.Action {
	case workflow.DecisionAllow:
		if err := rc.Sender.SendDM(ctx, igUserID, rc.Event.FromID, text); err != nil {
			return workflow.ActionResult{}, fmt.Errorf("nodes: send-dm: send: %w", err)
		}
		return workflow.ActionResult{Detail: "DM terkirim"}, nil

	case workflow.DecisionQueue:
		return workflow.ActionResult{
			Detail: fmt.Sprintf("gate=queue (%s); DM ditunda (belum ada retry generik)", d.Reason),
		}, nil

	default: // DecisionReject
		return workflow.ActionResult{
			Detail: fmt.Sprintf("gate=reject (%s); DM tidak dikirim", d.Reason),
		}, nil
	}
}
