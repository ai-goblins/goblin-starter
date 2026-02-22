package main

import (
	"encoding/json"
	"fmt"
	"time"

	sdk "github.com/ai-goblins/goblin-sdk"
)

// ── Arguments ─────────────────────────────────────────────────────────────────

// goblinArgs holds the blueprint-declared configuration for this goblin.
// All fields are optional and fall back to sensible defaults.
type goblinArgs struct {
	// Name is the recipient's name, used to personalise the greeting.
	// Default: "friend"
	Name string `json:"name"`

	// EarliestHour is the earliest hour (UTC, 0–23) the salutation may be sent.
	// Default: 8
	EarliestHour int `json:"earliest_hour"`

	// LatestHour is the latest hour (UTC, 0–23, exclusive) the salutation may be sent.
	// Must be greater than EarliestHour.
	// Default: 20
	LatestHour int `json:"latest_hour"`
}

func parseArgs(raw map[string]any) (goblinArgs, error) {
	a := goblinArgs{Name: "friend", EarliestHour: 8, LatestHour: 20}

	if v, ok := raw["name"].(string); ok && v != "" {
		a.Name = v
	}
	if v, ok := raw["earliest_hour"].(float64); ok {
		a.EarliestHour = int(v)
	}
	if v, ok := raw["latest_hour"].(float64); ok {
		a.LatestHour = int(v)
	}

	if a.LatestHour <= a.EarliestHour {
		return goblinArgs{}, fmt.Errorf(
			"latest_hour (%d) must be greater than earliest_hour (%d)",
			a.LatestHour, a.EarliestHour,
		)
	}
	return a, nil
}

// ── State ─────────────────────────────────────────────────────────────────────

// goblinState tracks what the goblin has sent and when it plans to send next.
type goblinState struct {
	// LastSentDate is the UTC date (YYYY-MM-DD) of the most recent salutation.
	// Empty on first run.
	LastSentDate string `json:"last_sent_date,omitempty"`

	// ScheduledFor is the UTC datetime (YYYY-MM-DDTHH:MM) the goblin has chosen
	// to send today's salutation. Repicked at the start of each new day.
	ScheduledFor string `json:"scheduled_for,omitempty"`
}

func parseState(raw map[string]any) (goblinState, error) {
	var s goblinState
	data, err := json.Marshal(raw)
	if err != nil {
		return goblinState{}, fmt.Errorf("marshal state: %w", err)
	}
	if err := json.Unmarshal(data, &s); err != nil {
		return goblinState{}, fmt.Errorf("unmarshal state: %w", err)
	}
	return s, nil
}

func saveState(s goblinState) map[string]any {
	data, _ := json.Marshal(s)
	var m map[string]any
	_ = json.Unmarshal(data, &m)
	return m
}

// ── Core logic ────────────────────────────────────────────────────────────────

// run is the goblin's business logic.
//
// It is separated from main so it can be unit-tested without WASM or the SDK.
// Dependencies on the current time and randomness are injected so tests are
// fully deterministic.
//
// Behaviour:
//  1. If the salutation has already been sent today → skip.
//  2. If no send time has been chosen for today yet → pick one at random within
//     the configured window, persist it, and skip (will send when the time comes).
//  3. If the chosen send time has not yet arrived → skip.
//  4. If the chosen send time has arrived → send the salutation and reset state.
func run(input sdk.Input, now time.Time, randIntn func(int) int) (sdk.Output, error) {
	args, err := parseArgs(input.Arguments)
	if err != nil {
		return sdk.Output{}, fmt.Errorf("parse arguments: %w", err)
	}

	state, err := parseState(input.State)
	if err != nil {
		return sdk.Output{}, fmt.Errorf("parse state: %w", err)
	}

	today := now.UTC().Format("2006-01-02")

	// Already sent today — nothing to do.
	if state.LastSentDate == today {
		return sdk.Output{Skip: true, State: saveState(state)}, nil
	}

	// No send time chosen for today yet — pick one and wait.
	if state.ScheduledFor == "" || len(state.ScheduledFor) < 10 || state.ScheduledFor[:10] != today {
		hour := args.EarliestHour + randIntn(args.LatestHour-args.EarliestHour)
		minute := randIntn(60)
		state.ScheduledFor = fmt.Sprintf("%sT%02d:%02d", today, hour, minute)
		return sdk.Output{Skip: true, State: saveState(state)}, nil
	}

	// Send time chosen but not yet reached — keep waiting.
	scheduledAt, err := time.Parse("2006-01-02T15:04", state.ScheduledFor)
	if err != nil {
		return sdk.Output{}, fmt.Errorf("parse scheduled_for %q: %w", state.ScheduledFor, err)
	}
	if now.UTC().Before(scheduledAt) {
		return sdk.Output{Skip: true, State: saveState(state)}, nil
	}

	// Time to send.
	return sdk.Output{
		Data: map[string]any{
			"name":         args.Name,
			"time_of_day":  timeOfDay(now.UTC().Hour()),
		},
		State: saveState(goblinState{LastSentDate: today}),
		Skip:  false,
	}, nil
}

// timeOfDay returns a human-readable part of the day for the given UTC hour.
func timeOfDay(hour int) string {
	switch {
	case hour < 12:
		return "morning"
	case hour < 17:
		return "afternoon"
	default:
		return "evening"
	}
}
