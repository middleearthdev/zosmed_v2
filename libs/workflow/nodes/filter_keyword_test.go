package nodes_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/zosmed/zosmed/libs/workflow"
	"github.com/zosmed/zosmed/libs/workflow/nodes"
)

func buildKeywordFilter(t *testing.T, cfg string) workflow.Filter {
	t.Helper()
	fmap := workflow.FactoryMap{}
	nodes.RegisterFactories(fmap)
	factory := fmap[nodes.NodeTypeKeywordMatch]

	built, err := factory.Build(json.RawMessage(cfg))
	if err != nil {
		t.Fatalf("Build error: %v", err)
	}
	f, ok := built.(workflow.Filter)
	if !ok {
		t.Fatalf("built value does not implement workflow.Filter: %T", built)
	}
	return f
}

func TestKeywordMatch_CaseInsensitiveByDefault(t *testing.T) {
	f := buildKeywordFilter(t, `{"keywords":["MAU"]}`)

	allow, err := f.Allow(context.Background(), &workflow.RunContext{Event: workflow.Event{Text: "aku mau banget kak"}})
	if err != nil {
		t.Fatalf("Allow error: %v", err)
	}
	if !allow {
		t.Error("expected keyword match to be case-insensitive by default")
	}
}

func TestKeywordMatch_NoMatchRejects(t *testing.T) {
	f := buildKeywordFilter(t, `{"keywords":["harga"]}`)

	allow, err := f.Allow(context.Background(), &workflow.RunContext{Event: workflow.Event{Text: "keren banget kak"}})
	if err != nil {
		t.Fatalf("Allow error: %v", err)
	}
	if allow {
		t.Error("expected no match to reject")
	}
}

func TestKeywordMatch_EmptyKeywordsPermissive(t *testing.T) {
	f := buildKeywordFilter(t, `{}`)

	allow, err := f.Allow(context.Background(), &workflow.RunContext{Event: workflow.Event{Text: "apapun"}})
	if err != nil {
		t.Fatalf("Allow error: %v", err)
	}
	if !allow {
		t.Error("expected empty keyword config to be permissive (always allow)")
	}
}
