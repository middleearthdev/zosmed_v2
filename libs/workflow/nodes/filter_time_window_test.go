package nodes_test

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/zosmed/zosmed/libs/workflow"
	"github.com/zosmed/zosmed/libs/workflow/nodes"
)

func buildTimeWindowFilter(t *testing.T, cfg string) (workflow.Filter, error) {
	t.Helper()
	fmap := workflow.FactoryMap{}
	nodes.RegisterFactories(fmap, nil)
	built, err := fmap[nodes.NodeTypeTimeWindow].Build(json.RawMessage(cfg))
	if err != nil {
		return nil, err
	}
	f, ok := built.(workflow.Filter)
	if !ok {
		t.Fatalf("built value does not implement workflow.Filter: %T", built)
	}
	return f, nil
}

func timeWindowAllow(t *testing.T, f workflow.Filter) bool {
	t.Helper()
	allow, err := f.Allow(context.Background(), &workflow.RunContext{})
	if err != nil {
		t.Fatalf("Allow error: %v", err)
	}
	return allow
}

func TestTimeWindow_EmptyIsPermissive(t *testing.T) {
	f, err := buildTimeWindowFilter(t, `{}`)
	if err != nil {
		t.Fatalf("Build error: %v", err)
	}
	if !timeWindowAllow(t, f) {
		t.Error("expected empty config to allow at any time")
	}
}

func TestTimeWindow_DayIncludingTodayAllows(t *testing.T) {
	today := int(time.Now().In(time.UTC).Weekday())
	// Use WITA-agnostic assertion: allow both today and neighbours so a run
	// straddling midnight in the default tz can't flake. We only assert the
	// "today is in the set" path allows.
	cfg := fmt.Sprintf(`{"days":["%d","%d","%d"],"timezone":"UTC"}`, (today+6)%7, today, (today+1)%7)
	f, err := buildTimeWindowFilter(t, cfg)
	if err != nil {
		t.Fatalf("Build error: %v", err)
	}
	if !timeWindowAllow(t, f) {
		t.Error("expected a day set including today (UTC) to allow")
	}
}

func TestTimeWindow_DayExcludingTodayRejects(t *testing.T) {
	// Build a set of the four days that are neither today nor its neighbours,
	// so the assertion holds regardless of tz-induced day boundaries.
	today := int(time.Now().In(time.UTC).Weekday())
	excluded := map[int]struct{}{today: {}, (today + 1) % 7: {}, (today + 6) % 7: {}}
	var days []string
	for d := 0; d <= 6; d++ {
		if _, skip := excluded[d]; !skip {
			days = append(days, fmt.Sprintf("%q", fmt.Sprint(d)))
		}
	}
	cfg := fmt.Sprintf(`{"days":[%s],"timezone":"UTC"}`, joinCSV(days))
	f, err := buildTimeWindowFilter(t, cfg)
	if err != nil {
		t.Fatalf("Build error: %v", err)
	}
	if timeWindowAllow(t, f) {
		t.Error("expected a day set excluding today (UTC) to reject")
	}
}

func TestTimeWindow_InvalidDayRejectedAtBuild(t *testing.T) {
	if _, err := buildTimeWindowFilter(t, `{"days":["9"]}`); err == nil {
		t.Fatal("expected Build error for weekday out of [0,6]")
	}
}

func TestTimeWindow_InvalidTimezoneRejectedAtBuild(t *testing.T) {
	if _, err := buildTimeWindowFilter(t, `{"timezone":"Not/AZone"}`); err == nil {
		t.Fatal("expected Build error for an invalid IANA timezone")
	}
}

func TestTimeWindow_InvalidMinuteRejectedAtBuild(t *testing.T) {
	if _, err := buildTimeWindowFilter(t, `{"startMinute":-1}`); err == nil {
		t.Fatal("expected Build error for startMinute out of [0,1439]")
	}
}

func joinCSV(items []string) string {
	out := ""
	for i, s := range items {
		if i > 0 {
			out += ","
		}
		out += s
	}
	return out
}
