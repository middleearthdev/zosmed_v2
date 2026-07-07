package nodes

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/zosmed/zosmed/libs/workflow"
)

// defaultTimeWindowTimezone is used when config.timezone is omitted. WIB
// (Western Indonesia Time) is the dominant business timezone for the
// Indonesian olshop sellers this product targets (CLAUDE.md §2).
const defaultTimeWindowTimezone = "Asia/Jakarta"

const (
	minutesPerDay  = 24 * 60
	minWeekdayNum  = 0 // time.Sunday
	maxWeekdayNum  = 6 // time.Saturday
)

// timeWindowConfig is the config shape for NodeTypeTimeWindow (CLAUDE.md §7
// "Time window — logika server"; ADR-005 §2.2). Empty config is fully
// permissive (every day, every time).
//
// Days holds time.Weekday values (0=Sunday .. 6=Saturday) AS STRINGS. The
// builder's friendly `weekdays` toggle (packages/types/src/workflow.ts) emits
// these number-strings so users never type weekday numbers themselves; Build
// parses each entry back to an int and rejects anything outside [0,6].
type timeWindowConfig struct {
	Days []string `json:"days,omitempty"`
	// StartMinute/EndMinute are minutes since local midnight [0,1439],
	// inclusive. Omitting both means no time-of-day restriction. A window
	// where StartMinute > EndMinute wraps past midnight (e.g. 22:00-06:00).
	StartMinute *int   `json:"startMinute,omitempty"`
	EndMinute   *int   `json:"endMinute,omitempty"`
	Timezone    string `json:"timezone,omitempty"`
}

// timeWindowFilter allows the run to continue only when the EVALUATION time
// (time.Now — NOT the triggering comment's timestamp, ADR-005 §2.2: "agar
// konsisten untuk 'hanya jalan saat jam kerja'") falls inside the configured
// day/minute window.
type timeWindowFilter struct {
	days        map[int]struct{} // empty = every day allowed
	hasMinutes  bool
	startMinute int
	endMinute   int
	loc         *time.Location
}

func (f *timeWindowFilter) Allow(_ context.Context, _ *workflow.RunContext) (bool, error) {
	now := time.Now().In(f.loc)

	if len(f.days) > 0 {
		if _, ok := f.days[int(now.Weekday())]; !ok {
			return false, nil
		}
	}

	if f.hasMinutes {
		minuteOfDay := now.Hour()*60 + now.Minute()
		if f.startMinute <= f.endMinute {
			if minuteOfDay < f.startMinute || minuteOfDay > f.endMinute {
				return false, nil
			}
		} else {
			// Wraps past midnight (e.g. start=22:00, end=06:00): allowed
			// UNLESS the minute falls in the gap strictly between end and start.
			if minuteOfDay > f.endMinute && minuteOfDay < f.startMinute {
				return false, nil
			}
		}
	}

	return true, nil
}

// BuildTimeWindow is the Factory.Build func for NodeTypeTimeWindow.
// Validates days (must parse to [0,6]) and minutes (must be within
// [0,1439]) at build time — a mis-configured filter fails the workflow
// compile loudly rather than silently misbehaving at runtime.
func BuildTimeWindow(cfg json.RawMessage) (any, error) {
	var c timeWindowConfig
	if len(cfg) > 0 {
		if err := json.Unmarshal(cfg, &c); err != nil {
			return nil, fmt.Errorf("nodes: time-window: parse config: %w", err)
		}
	}

	days := make(map[int]struct{}, len(c.Days))
	for _, raw := range c.Days {
		d, err := strconv.Atoi(raw)
		if err != nil || d < minWeekdayNum || d > maxWeekdayNum {
			return nil, fmt.Errorf("nodes: time-window: config.days value %q out of range [0-6]", raw)
		}
		days[d] = struct{}{}
	}

	tzName := c.Timezone
	if tzName == "" {
		tzName = defaultTimeWindowTimezone
	}
	loc, err := time.LoadLocation(tzName)
	if err != nil {
		return nil, fmt.Errorf("nodes: time-window: config.timezone %q invalid: %w", tzName, err)
	}

	f := &timeWindowFilter{days: days, loc: loc}
	if c.StartMinute != nil || c.EndMinute != nil {
		start, end := 0, minutesPerDay-1
		if c.StartMinute != nil {
			start = *c.StartMinute
		}
		if c.EndMinute != nil {
			end = *c.EndMinute
		}
		if start < 0 || start >= minutesPerDay || end < 0 || end >= minutesPerDay {
			return nil, fmt.Errorf("nodes: time-window: startMinute/endMinute must be within [0,%d]", minutesPerDay-1)
		}
		f.hasMinutes = true
		f.startMinute = start
		f.endMinute = end
	}

	return f, nil
}
