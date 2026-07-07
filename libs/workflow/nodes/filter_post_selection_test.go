package nodes_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/zosmed/zosmed/libs/workflow"
	"github.com/zosmed/zosmed/libs/workflow/nodes"
)

func buildPostSelectionFilter(t *testing.T, cfg string) workflow.Filter {
	t.Helper()
	fmap := workflow.FactoryMap{}
	nodes.RegisterFactories(fmap)
	built, err := fmap[nodes.NodeTypePostSelection].Build(json.RawMessage(cfg))
	if err != nil {
		t.Fatalf("Build error: %v", err)
	}
	f, ok := built.(workflow.Filter)
	if !ok {
		t.Fatalf("built value does not implement workflow.Filter: %T", built)
	}
	return f
}

func postSelectionAllow(t *testing.T, f workflow.Filter, mediaID string) bool {
	t.Helper()
	allow, err := f.Allow(context.Background(), &workflow.RunContext{Event: workflow.Event{MediaID: mediaID}})
	if err != nil {
		t.Fatalf("Allow error: %v", err)
	}
	return allow
}

func TestPostSelection_EmptyIsPermissive(t *testing.T) {
	f := buildPostSelectionFilter(t, `{}`)
	if !postSelectionAllow(t, f, "media-anything") {
		t.Error("expected empty mediaIds to allow every post/Reel")
	}
}

func TestPostSelection_MatchAllows(t *testing.T) {
	f := buildPostSelectionFilter(t, `{"mediaIds":["media-1","media-2"]}`)
	if !postSelectionAllow(t, f, "media-2") {
		t.Error("expected a media in the configured set to pass")
	}
}

func TestPostSelection_NonMatchRejects(t *testing.T) {
	f := buildPostSelectionFilter(t, `{"mediaIds":["media-1"]}`)
	if postSelectionAllow(t, f, "media-99") {
		t.Error("expected a media outside the configured set to be rejected")
	}
}
