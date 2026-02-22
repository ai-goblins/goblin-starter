package main

import (
	"fmt"
	"testing"
	"time"

	sdk "github.com/ai-goblins/goblin-sdk"
)

// ── helpers ───────────────────────────────────────────────────────────────────

// fixedRand always returns the same value, making random behaviour deterministic in tests.
func fixedRand(v int) func(int) int {
	return func(_ int) int { return v }
}

// at parses a UTC datetime string "2006-01-02T15:04" into a time.Time.
func at(s string) time.Time {
	t, err := time.Parse("2006-01-02T15:04", s)
	if err != nil {
		panic(fmt.Sprintf("at(%q): %v", s, err))
	}
	return t.UTC()
}

func inputWith(args map[string]any, state map[string]any) sdk.Input {
	if args == nil {
		args = map[string]any{}
	}
	if state == nil {
		state = map[string]any{}
	}
	return sdk.Input{Arguments: args, State: state}
}

// ── parseArgs ─────────────────────────────────────────────────────────────────

func TestParseArgs_Defaults(t *testing.T) {
	a, err := parseArgs(map[string]any{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if a.Name != "friend" {
		t.Errorf("Name = %q, want %q", a.Name, "friend")
	}
	if a.EarliestHour != 8 {
		t.Errorf("EarliestHour = %d, want 8", a.EarliestHour)
	}
	if a.LatestHour != 20 {
		t.Errorf("LatestHour = %d, want 20", a.LatestHour)
	}
}

func TestParseArgs_CustomValues(t *testing.T) {
	a, err := parseArgs(map[string]any{
		"name":          "Alice",
		"earliest_hour": float64(9),
		"latest_hour":   float64(17),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if a.Name != "Alice" {
		t.Errorf("Name = %q, want %q", a.Name, "Alice")
	}
	if a.EarliestHour != 9 {
		t.Errorf("EarliestHour = %d, want 9", a.EarliestHour)
	}
	if a.LatestHour != 17 {
		t.Errorf("LatestHour = %d, want 17", a.LatestHour)
	}
}

func TestParseArgs_InvalidWindow(t *testing.T) {
	cases := []struct {
		name     string
		earliest int
		latest   int
	}{
		{"equal", 10, 10},
		{"inverted", 20, 8},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := parseArgs(map[string]any{
				"earliest_hour": float64(tc.earliest),
				"latest_hour":   float64(tc.latest),
			})
			if err == nil {
				t.Error("expected error, got nil")
			}
		})
	}
}

// ── timeOfDay ─────────────────────────────────────────────────────────────────

func TestTimeOfDay(t *testing.T) {
	tests := []struct {
		hour int
		want string
	}{
		{0, "morning"},
		{7, "morning"},
		{11, "morning"},
		{12, "afternoon"},
		{16, "afternoon"},
		{17, "evening"},
		{23, "evening"},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("hour_%d", tt.hour), func(t *testing.T) {
			got := timeOfDay(tt.hour)
			if got != tt.want {
				t.Errorf("timeOfDay(%d) = %q, want %q", tt.hour, got, tt.want)
			}
		})
	}
}

// ── run ───────────────────────────────────────────────────────────────────────

func TestRun_AlreadySentToday_Skips(t *testing.T) {
	now := at("2026-02-22T14:00")
	input := inputWith(
		map[string]any{"name": "Alice"},
		map[string]any{"last_sent_date": "2026-02-22", "scheduled_for": "2026-02-22T10:00"},
	)

	out, err := run(input, now, fixedRand(0))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !out.Skip {
		t.Error("expected Skip=true when already sent today")
	}
}

func TestRun_FirstRun_PicksScheduleAndSkips(t *testing.T) {
	// fixedRand(2) → hour offset=2, minute=2 → send at EarliestHour+2:02 = 10:02
	now := at("2026-02-22T08:00")
	input := inputWith(map[string]any{"name": "Alice"}, nil)

	out, err := run(input, now, fixedRand(2))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !out.Skip {
		t.Error("expected Skip=true on first run (no schedule yet)")
	}
	if out.State["scheduled_for"] != "2026-02-22T10:02" {
		t.Errorf("scheduled_for = %v, want 2026-02-22T10:02", out.State["scheduled_for"])
	}
}

func TestRun_ScheduledTimeNotYetReached_Skips(t *testing.T) {
	now := at("2026-02-22T09:00")
	input := inputWith(
		map[string]any{"name": "Alice"},
		map[string]any{"scheduled_for": "2026-02-22T14:30"},
	)

	out, err := run(input, now, fixedRand(0))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !out.Skip {
		t.Error("expected Skip=true when scheduled time not yet reached")
	}
}

func TestRun_ScheduledTimeReached_Sends(t *testing.T) {
	now := at("2026-02-22T14:30")
	input := inputWith(
		map[string]any{"name": "Alice"},
		map[string]any{"scheduled_for": "2026-02-22T14:30"},
	)

	out, err := run(input, now, fixedRand(0))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Skip {
		t.Error("expected Skip=false when scheduled time reached")
	}
	if out.Data["name"] != "Alice" {
		t.Errorf("data.name = %v, want Alice", out.Data["name"])
	}
	if out.Data["time_of_day"] != "afternoon" {
		t.Errorf("data.time_of_day = %v, want afternoon", out.Data["time_of_day"])
	}
	if out.State["last_sent_date"] != "2026-02-22" {
		t.Errorf("state.last_sent_date = %v, want 2026-02-22", out.State["last_sent_date"])
	}
	// scheduled_for should be cleared after sending.
	if _, hasSchedule := out.State["scheduled_for"]; hasSchedule {
		t.Error("scheduled_for should be absent from state after sending")
	}
}

func TestRun_AfterSending_NewDayPicksNewSchedule(t *testing.T) {
	// Simulate a new day after having sent yesterday.
	now := at("2026-02-23T08:05")
	input := inputWith(
		map[string]any{"name": "Alice"},
		map[string]any{"last_sent_date": "2026-02-22"},
	)

	out, err := run(input, now, fixedRand(0))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !out.Skip {
		t.Error("expected Skip=true — new day, schedule just picked")
	}
	sched, _ := out.State["scheduled_for"].(string)
	if len(sched) < 10 || sched[:10] != "2026-02-23" {
		t.Errorf("scheduled_for = %q, expected date prefix 2026-02-23", sched)
	}
}

func TestRun_ScheduleStaleFromYesterday_RepicksForToday(t *testing.T) {
	// scheduled_for is from yesterday — must be repicked for today.
	now := at("2026-02-23T09:00")
	input := inputWith(
		map[string]any{"name": "Alice"},
		map[string]any{"scheduled_for": "2026-02-22T14:00"},
	)

	out, err := run(input, now, fixedRand(0))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !out.Skip {
		t.Error("expected Skip=true — must repick schedule for new day")
	}
	sched, _ := out.State["scheduled_for"].(string)
	if len(sched) < 10 || sched[:10] != "2026-02-23" {
		t.Errorf("scheduled_for = %q, expected today's date prefix", sched)
	}
}

func TestRun_DefaultName_UsedWhenArgMissing(t *testing.T) {
	now := at("2026-02-22T15:00")
	input := inputWith(
		nil, // no arguments
		map[string]any{"scheduled_for": "2026-02-22T14:00"},
	)

	out, err := run(input, now, fixedRand(0))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Data["name"] != "friend" {
		t.Errorf("data.name = %v, want friend (default)", out.Data["name"])
	}
}

func TestRun_InvalidArgs_ReturnsError(t *testing.T) {
	now := at("2026-02-22T10:00")
	input := inputWith(
		map[string]any{"earliest_hour": float64(20), "latest_hour": float64(8)},
		nil,
	)

	_, err := run(input, now, fixedRand(0))
	if err == nil {
		t.Error("expected error for invalid hour window, got nil")
	}
}

func TestRun_ScheduleRespectsWindow(t *testing.T) {
	// With earliest=9, latest=17 and fixedRand returning the max offset (7),
	// the scheduled hour should be within [9, 17).
	now := at("2026-02-22T08:00")
	input := inputWith(
		map[string]any{"earliest_hour": float64(9), "latest_hour": float64(17)},
		nil,
	)

	out, err := run(input, now, fixedRand(7))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	sched, _ := out.State["scheduled_for"].(string)
	var schedTime time.Time
	schedTime, err = time.Parse("2006-01-02T15:04", sched)
	if err != nil {
		t.Fatalf("parse scheduled_for %q: %v", sched, err)
	}
	if h := schedTime.Hour(); h < 9 || h >= 17 {
		t.Errorf("scheduled hour %d outside window [9, 17)", h)
	}
}
