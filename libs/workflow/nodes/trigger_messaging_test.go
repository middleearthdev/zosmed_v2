package nodes_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/zosmed/zosmed/libs/workflow"
	"github.com/zosmed/zosmed/libs/workflow/nodes"
)

// buildTrigger builds nodeType's Trigger via the registered factory.
func buildTrigger(t *testing.T, nodeType, cfg string) workflow.Trigger {
	t.Helper()
	fmap := workflow.FactoryMap{}
	nodes.RegisterFactories(fmap)
	factory, ok := fmap[nodeType]
	if !ok {
		t.Fatalf("node_type %q not registered", nodeType)
	}
	built, err := factory.Build(json.RawMessage(cfg))
	if err != nil {
		t.Fatalf("Build error: %v", err)
	}
	tr, ok := built.(workflow.Trigger)
	if !ok {
		t.Fatalf("built value does not implement workflow.Trigger: %T", built)
	}
	return tr
}

// eventWithSubtype builds a minimal Source=dm Event carrying the given
// event_subtype raw key (ADR-006 §2 wire convention).
func eventWithSubtype(subtype string) workflow.Event {
	return workflow.Event{
		Source: workflow.SourceDM,
		Raw:    map[string]any{"event_subtype": subtype},
	}
}

// ── dm-received ─────────────────────────────────────────────────────────────

func TestDMReceived_MatchesPlainDM(t *testing.T) {
	tr := buildTrigger(t, nodes.NodeTypeDMReceived, `{}`)
	if !tr.Match(context.Background(), eventWithSubtype("dm")) {
		t.Fatal("expected match on subtype=dm")
	}
}

func TestDMReceived_NoMatchOtherSubtypes(t *testing.T) {
	tr := buildTrigger(t, nodes.NodeTypeDMReceived, `{}`)
	for _, st := range []string{"story-reply", "story-mention", "ad-referral", ""} {
		if tr.Match(context.Background(), eventWithSubtype(st)) {
			t.Errorf("dm-received matched subtype=%q, want no match", st)
		}
	}
}

func TestDMReceived_NoMatchNonDMSource(t *testing.T) {
	tr := buildTrigger(t, nodes.NodeTypeDMReceived, `{}`)
	e := workflow.Event{Source: workflow.SourceComment, Raw: map[string]any{"event_subtype": "dm"}}
	if tr.Match(context.Background(), e) {
		t.Fatal("dm-received must not match Source=comment even with subtype=dm raw key")
	}
}

// ── story-reply ──────────────────────────────────────────────────────────────

func TestStoryReply_MatchesOwnSubtypeOnly(t *testing.T) {
	tr := buildTrigger(t, nodes.NodeTypeStoryReply, `{}`)
	if !tr.Match(context.Background(), eventWithSubtype("story-reply")) {
		t.Fatal("expected match on subtype=story-reply")
	}
	for _, st := range []string{"dm", "story-mention", "ad-referral"} {
		if tr.Match(context.Background(), eventWithSubtype(st)) {
			t.Errorf("story-reply matched subtype=%q, want no match", st)
		}
	}
}

// ── story-mention (ADR-006 koreksi B0: messaging attachment, opens window) ──

func TestStoryMention_MatchesOwnSubtypeOnly(t *testing.T) {
	tr := buildTrigger(t, nodes.NodeTypeStoryMention, `{}`)
	if !tr.Match(context.Background(), eventWithSubtype("story-mention")) {
		t.Fatal("expected match on subtype=story-mention")
	}
	for _, st := range []string{"dm", "story-reply", "ad-referral"} {
		if tr.Match(context.Background(), eventWithSubtype(st)) {
			t.Errorf("story-mention matched subtype=%q, want no match", st)
		}
	}
}

func TestStoryMention_SourceMustBeDM(t *testing.T) {
	tr := buildTrigger(t, nodes.NodeTypeStoryMention, `{}`)
	e := workflow.Event{Source: workflow.SourceStory, Raw: map[string]any{"event_subtype": "story-mention"}}
	if tr.Match(context.Background(), e) {
		t.Fatal("story-mention must require Source==dm, never SourceStory (ADR-006 koreksi B0 point 4)")
	}
}

// ── click-to-dm-ad ────────────────────────────────────────────────────────────

func TestClickToDMAd_MatchesAdReferral(t *testing.T) {
	tr := buildTrigger(t, nodes.NodeTypeClickToDMAd, `{}`)
	if !tr.Match(context.Background(), eventWithSubtype("ad-referral")) {
		t.Fatal("expected match on subtype=ad-referral with no adRef filter configured")
	}
}

func TestClickToDMAd_NoMatchOtherSubtypes(t *testing.T) {
	tr := buildTrigger(t, nodes.NodeTypeClickToDMAd, `{}`)
	for _, st := range []string{"dm", "story-reply", "story-mention"} {
		if tr.Match(context.Background(), eventWithSubtype(st)) {
			t.Errorf("click-to-dm-ad matched subtype=%q, want no match", st)
		}
	}
}

func TestClickToDMAd_AdRefFilterMatches(t *testing.T) {
	tr := buildTrigger(t, nodes.NodeTypeClickToDMAd, `{"adRef":"campaign-42"}`)
	e := workflow.Event{
		Source: workflow.SourceDM,
		Raw:    map[string]any{"event_subtype": "ad-referral", "ad_ref": "campaign-42"},
	}
	if !tr.Match(context.Background(), e) {
		t.Fatal("expected match when configured adRef equals Raw[ad_ref]")
	}
}

func TestClickToDMAd_AdRefFilterRejectsMismatch(t *testing.T) {
	tr := buildTrigger(t, nodes.NodeTypeClickToDMAd, `{"adRef":"campaign-42"}`)
	e := workflow.Event{
		Source: workflow.SourceDM,
		Raw:    map[string]any{"event_subtype": "ad-referral", "ad_ref": "campaign-other"},
	}
	if tr.Match(context.Background(), e) {
		t.Fatal("expected no match when Raw[ad_ref] differs from configured adRef")
	}
}
