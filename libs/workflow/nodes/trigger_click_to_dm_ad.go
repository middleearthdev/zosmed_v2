package nodes

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/zosmed/zosmed/libs/workflow"
)

// clickToDMAdConfig is the config shape for NodeTypeClickToDMAd (ADR-006
// §2 table). adRef is an OPTIONAL filter: when set, the trigger only matches
// ad-referral events whose Raw[ad_ref] equals it; empty (the default) matches
// every ad-referral event.
type clickToDMAdConfig struct {
	AdRef string `json:"adRef,omitempty"`
}

// clickToDMAdTrigger fires for an ad-referral event: Source==dm AND
// Raw[event_subtype]=="ad-referral" (ADR-006 §2.1). This is a Click-to-DM ad
// entry point — the user clicked an ad and started the conversation, a
// legitimate percakapan-yang-sah start (§4a "Click-to-DM ad"; §4b.6 does NOT
// apply — this is not a system-initiated blast).
type clickToDMAdTrigger struct {
	adRef string
}

func (t *clickToDMAdTrigger) Match(_ context.Context, e workflow.Event) bool {
	if e.Source != workflow.SourceDM || rawString(e.Raw, rawKeyEventSubtype) != subtypeAdReferral {
		return false
	}
	if t.adRef != "" && rawString(e.Raw, rawKeyAdRef) != t.adRef {
		return false
	}
	return true
}

// BuildClickToDMAd is the Factory.Build func for NodeTypeClickToDMAd.
func BuildClickToDMAd(cfg json.RawMessage) (any, error) {
	var c clickToDMAdConfig
	if len(cfg) > 0 {
		if err := json.Unmarshal(cfg, &c); err != nil {
			return nil, fmt.Errorf("nodes: click-to-dm-ad: parse config: %w", err)
		}
	}
	return &clickToDMAdTrigger{adRef: strings.TrimSpace(c.AdRef)}, nil
}
