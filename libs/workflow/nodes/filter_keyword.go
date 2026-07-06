package nodes

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/zosmed/zosmed/libs/workflow"
)

// keywordMatchConfig is the config shape for NodeTypeKeywordMatch.
// caseInsensitive defaults to true (olshop comments are rarely consistent
// about casing) when omitted from config.
type keywordMatchConfig struct {
	Keywords        []string `json:"keywords"`
	CaseInsensitive *bool    `json:"caseInsensitive,omitempty"`
}

// keywordMatchFilter allows the run to continue only when rc.Event.Text
// contains at least one configured keyword (substring match).
type keywordMatchFilter struct {
	keywords        []string
	caseInsensitive bool
}

func (f *keywordMatchFilter) Allow(_ context.Context, rc *workflow.RunContext) (bool, error) {
	if len(f.keywords) == 0 {
		// No keywords configured — permissive default: don't block the run.
		return true, nil
	}
	text := rc.Event.Text
	if f.caseInsensitive {
		text = strings.ToLower(text)
	}
	for _, kw := range f.keywords {
		if strings.TrimSpace(kw) == "" {
			continue
		}
		k := kw
		if f.caseInsensitive {
			k = strings.ToLower(k)
		}
		if strings.Contains(text, k) {
			return true, nil
		}
	}
	return false, nil
}

// BuildKeywordMatch is the Factory.Build func for NodeTypeKeywordMatch.
func BuildKeywordMatch(cfg json.RawMessage) (any, error) {
	var c keywordMatchConfig
	if len(cfg) > 0 {
		if err := json.Unmarshal(cfg, &c); err != nil {
			return nil, fmt.Errorf("nodes: keyword-match: parse config: %w", err)
		}
	}
	caseInsensitive := true
	if c.CaseInsensitive != nil {
		caseInsensitive = *c.CaseInsensitive
	}
	return &keywordMatchFilter{keywords: c.Keywords, caseInsensitive: caseInsensitive}, nil
}
