package clean

import (
	"testing"
	"time"

	"agres/internal/session"
)

func TestSelectDefaults(t *testing.T) {
	now := time.Date(2026, 8, 27, 12, 0, 0, 0, time.UTC)
	day := 24 * time.Hour
	mk := func(id string, age time.Duration, size int64, wd string) session.Session {
		return session.Session{ID: id, Agent: session.AgentClaude, WorkDir: wd, UpdatedAt: now.Add(-age), Size: size}
	}
	sessions := []session.Session{
		mk("new1", 1*day, 1, "A"),
		mk("new2", 2*day, 1, "A"),
		mk("new3", 3*day, 1, "A"),
		mk("old-but-kept", 40*day, 1, "B"), // only session in B → kept
		mk("old1", 40*day, 1, "A"),
		mk("old2", 60*day, 1, "A"),
		mk("active", 10*time.Minute, 1, "A"),
	}
	got := Select(sessions, DefaultOptions(), now)
	ids := []string{}
	for _, s := range got {
		ids = append(ids, s.ID)
	}
	want := []string{"old2", "old1"}
	if len(ids) != len(want) {
		t.Fatalf("got %v; want %v", ids, want)
	}
	for i := range want {
		if ids[i] != want[i] {
			t.Errorf("got %v; want %v", ids, want)
		}
	}
}

func TestSelectSizeAndAgent(t *testing.T) {
	now := time.Now()
	sessions := []session.Session{
		{ID: "big-codex", Agent: session.AgentCodex, UpdatedAt: now.Add(-2 * time.Hour), Size: 20 << 20},
		{ID: "big-claude", Agent: session.AgentClaude, UpdatedAt: now.Add(-2 * time.Hour), Size: 20 << 20},
		{ID: "small-codex", Agent: session.AgentCodex, UpdatedAt: now.Add(-2 * time.Hour), Size: 1 << 20},
	}
	opts := Options{LargerThan: 10 << 20, Agent: session.AgentCodex, ActiveWindow: time.Hour}
	got := Select(sessions, opts, now)
	if len(got) != 1 || got[0].ID != "big-codex" {
		t.Errorf("got %v", got)
	}
}

func TestParseDuration(t *testing.T) {
	cases := map[string]time.Duration{"30d": 30 * 24 * time.Hour, "12h": 12 * time.Hour, "2w": 14 * 24 * time.Hour, "7": 7 * 24 * time.Hour, "0": 0}
	for in, want := range cases {
		got, err := ParseDuration(in)
		if err != nil || got != want {
			t.Errorf("ParseDuration(%q) = %v, %v; want %v", in, got, err, want)
		}
	}
	if _, err := ParseDuration("abc"); err == nil {
		t.Error("expected error")
	}
}

func TestParseSize(t *testing.T) {
	cases := map[string]int64{"10M": 10 << 20, "512K": 512 << 10, "1.5G": 3 << 29, "10MB": 10 << 20, "100": 100}
	for in, want := range cases {
		got, err := ParseSize(in)
		if err != nil || got != want {
			t.Errorf("ParseSize(%q) = %v, %v; want %v", in, got, err, want)
		}
	}
	if _, err := ParseSize("x"); err == nil {
		t.Error("expected error")
	}
}
