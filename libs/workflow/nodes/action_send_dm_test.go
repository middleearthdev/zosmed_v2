package nodes_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/zosmed/zosmed/libs/workflow"
	"github.com/zosmed/zosmed/libs/workflow/nodes"
)

func buildSendDMAction(t *testing.T, cfg string) workflow.Action {
	t.Helper()
	fmap := workflow.FactoryMap{}
	nodes.RegisterFactories(fmap)
	factory := fmap[nodes.NodeTypeSendDM]

	built, err := factory.Build(json.RawMessage(cfg))
	if err != nil {
		t.Fatalf("Build error: %v", err)
	}
	a, ok := built.(workflow.Action)
	if !ok {
		t.Fatalf("built value does not implement workflow.Action: %T", built)
	}
	return a
}

// TestSendDM_NoWindow_SkipsWithoutTouchingGateOrSender is the CRITICAL §10 +
// ADR-006 R4 guard: when Raw[last_interaction_at] is absent (e.g. a
// comment-triggered flow, which never populates this key), send-dm must skip
// LOCALLY — never call rc.Gate.Allow, never call rc.Sender.SendDM.
func TestSendDM_NoWindow_SkipsWithoutTouchingGateOrSender(t *testing.T) {
	action := buildSendDMAction(t, `{}`)

	gater := &recordingGater{decision: workflow.Decision{Action: workflow.DecisionAllow}}
	sender := &recordingSender{}
	rc := workflow.NewRunContext(workflow.Event{
		Source:       workflow.SourceComment, // comment-triggered: no window
		FromUsername: "rina",
		FromID:       "user-1",
		ObjectID:     "comment-1",
		// Raw deliberately omits last_interaction_at.
	}, sender, gater)

	res, err := action.Execute(context.Background(), rc)
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}
	if gater.called {
		t.Fatal("gate must NOT be consulted when last_interaction_at is absent (guard-before-gate)")
	}
	if sender.dmSendCalled {
		t.Fatal("SendDM must NOT be called when last_interaction_at is absent")
	}
	if !strings.Contains(res.Detail, "tidak ada window") {
		t.Errorf("expected skip detail to mention the missing window, got %q", res.Detail)
	}
}

// TestSendDM_ZeroTimeWindow_Skips proves the guard checks IsZero(), not mere
// presence: a zero time.Time value in Raw must also skip.
func TestSendDM_ZeroTimeWindow_Skips(t *testing.T) {
	action := buildSendDMAction(t, `{}`)

	gater := &recordingGater{decision: workflow.Decision{Action: workflow.DecisionAllow}}
	sender := &recordingSender{}
	rc := workflow.NewRunContext(workflow.Event{
		Source: workflow.SourceDM,
		FromID: "user-1",
		Raw:    map[string]any{"last_interaction_at": time.Time{}},
	}, sender, gater)

	if _, err := action.Execute(context.Background(), rc); err != nil {
		t.Fatalf("Execute error: %v", err)
	}
	if gater.called || sender.dmSendCalled {
		t.Fatal("zero time.Time must be treated as absent — gate/sender must not be touched")
	}
}

// TestSendDM_AllowSendsDM verifies the §10 one-door happy path: gate
// consulted with Kind=dm BEFORE SendDM, and SendDM fires on DecisionAllow.
// Uses a custom {nama} template so substitution is exercised (the default
// copy no longer references {nama} — it is blank on the messaging surface).
func TestSendDM_AllowSendsDM(t *testing.T) {
	action := buildSendDMAction(t, `{"template":"Halo {nama}!"}`)

	gater := &recordingGater{decision: workflow.Decision{Action: workflow.DecisionAllow}}
	sender := &recordingSender{}
	last := time.Now().Add(-1 * time.Hour)
	rc := workflow.NewRunContext(workflow.Event{
		Source:       workflow.SourceDM,
		AccountID:    "acc-1",
		ObjectID:     "msg-1",
		FromID:       "user-1",
		FromUsername: "rina",
		Raw: map[string]any{
			"ig_user_id":          "biz-1",
			"last_interaction_at": last,
		},
	}, sender, gater)

	res, err := action.Execute(context.Background(), rc)
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}
	if !gater.called {
		t.Fatal("gate was not consulted — guardrail §10 violated")
	}
	if gater.lastReq.Kind != "dm" {
		t.Errorf("gate req.Kind = %q, want %q (safety.KindDM)", gater.lastReq.Kind, "dm")
	}
	if !gater.lastReq.CommentAt.Equal(last) {
		t.Errorf("gate req.CommentAt = %v, want %v (from last_interaction_at)", gater.lastReq.CommentAt, last)
	}
	if gater.lastReq.PostID != "" {
		t.Errorf("gate req.PostID = %q, want empty (send-dm is never a comment-reply)", gater.lastReq.PostID)
	}
	if !sender.dmSendCalled {
		t.Fatal("expected SendDM to be called on DecisionAllow")
	}
	if sender.dmIgUserID != "biz-1" {
		t.Errorf("SendDM igUserID = %q, want %q", sender.dmIgUserID, "biz-1")
	}
	if sender.dmTargetUser != "user-1" {
		t.Errorf("SendDM targetUserID = %q, want %q", sender.dmTargetUser, "user-1")
	}
	if !strings.Contains(sender.dmSentText, "rina") {
		t.Errorf("sent text = %q, want it to contain {nama} substitution", sender.dmSentText)
	}
	if res.Detail == "" {
		t.Error("expected a non-empty ActionResult.Detail")
	}
}

func TestSendDM_RejectSkipsSend(t *testing.T) {
	action := buildSendDMAction(t, `{}`)

	gater := &recordingGater{decision: workflow.Decision{Action: workflow.DecisionReject, Reason: "messaging window 24j lewat"}}
	sender := &recordingSender{}
	rc := workflow.NewRunContext(workflow.Event{
		Source: workflow.SourceDM,
		FromID: "user-1",
		Raw:    map[string]any{"last_interaction_at": time.Now()},
	}, sender, gater)

	if _, err := action.Execute(context.Background(), rc); err != nil {
		t.Fatalf("Execute error: %v", err)
	}
	if sender.dmSendCalled {
		t.Fatal("SendDM must NOT be called when the gate rejects (§10)")
	}
}

func TestSendDM_QueueSkipsSend(t *testing.T) {
	action := buildSendDMAction(t, `{}`)

	gater := &recordingGater{decision: workflow.Decision{Action: workflow.DecisionQueue, Reason: "overflow"}}
	sender := &recordingSender{}
	rc := workflow.NewRunContext(workflow.Event{
		Source: workflow.SourceDM,
		FromID: "user-1",
		Raw:    map[string]any{"last_interaction_at": time.Now()},
	}, sender, gater)

	if _, err := action.Execute(context.Background(), rc); err != nil {
		t.Fatalf("Execute error: %v", err)
	}
	if sender.dmSendCalled {
		t.Fatal("SendDM must NOT be called when the gate returns Queue (§10)")
	}
}

// TestSendDM_DefaultTemplate_UsedWhenConfigEmpty verifies the default
// Indonesian olshop-style copy is applied when config.template is omitted. The
// default deliberately carries no {nama} placeholder (blank on the messaging
// surface — ADR-006 R6), so the check is on the default copy itself.
func TestSendDM_DefaultTemplate_UsedWhenConfigEmpty(t *testing.T) {
	action := buildSendDMAction(t, `{}`)

	gater := &recordingGater{decision: workflow.Decision{Action: workflow.DecisionAllow}}
	sender := &recordingSender{}
	rc := workflow.NewRunContext(workflow.Event{
		Source:       workflow.SourceDM,
		FromID:       "user-1",
		FromUsername: "budi",
		Raw:          map[string]any{"last_interaction_at": time.Now()},
	}, sender, gater)

	if _, err := action.Execute(context.Background(), rc); err != nil {
		t.Fatalf("Execute error: %v", err)
	}
	if !strings.Contains(sender.dmSentText, "Makasih udah chat") {
		t.Errorf("expected default olshop copy, got %q", sender.dmSentText)
	}
	if strings.Contains(sender.dmSentText, "{nama}") {
		t.Errorf("default copy must not leave an unrendered {nama} placeholder, got %q", sender.dmSentText)
	}
}

// TestSendDM_CustomTemplate verifies config.template overrides the default.
func TestSendDM_CustomTemplate(t *testing.T) {
	action := buildSendDMAction(t, `{"template":"Halo {nama}, ada promo nih!"}`)

	gater := &recordingGater{decision: workflow.Decision{Action: workflow.DecisionAllow}}
	sender := &recordingSender{}
	rc := workflow.NewRunContext(workflow.Event{
		Source:       workflow.SourceDM,
		FromID:       "user-1",
		FromUsername: "sari",
		Raw:          map[string]any{"last_interaction_at": time.Now()},
	}, sender, gater)

	if _, err := action.Execute(context.Background(), rc); err != nil {
		t.Fatalf("Execute error: %v", err)
	}
	if sender.dmSentText != "Halo sari, ada promo nih!" {
		t.Errorf("sent text = %q, want custom template with {nama} substituted", sender.dmSentText)
	}
}
