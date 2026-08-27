// Package clean selects sessions for deletion based on age, size and
// per-project retention rules.
package clean

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"agres/internal/session"
)

// Options controls which sessions are selected.
type Options struct {
	OlderThan  time.Duration // zero disables the age condition
	LargerThan int64         // bytes; zero disables the size condition
	Agent      session.Agent // empty matches all agents
	Keep       int           // most recent sessions to always keep per project
	// ActiveWindow protects sessions updated within this duration, on the
	// assumption they may belong to a running agent.
	ActiveWindow time.Duration
}

// DefaultOptions are used by `agres clean` when no flags are given.
func DefaultOptions() Options {
	return Options{
		OlderThan:    30 * 24 * time.Hour,
		Keep:         3,
		ActiveWindow: time.Hour,
	}
}

// Select returns the sessions that satisfy every condition in opts, sorted
// oldest first. Sessions without any deletable path (and not owned by a
// Cleaner) are the caller's responsibility to exclude.
func Select(sessions []session.Session, opts Options, now time.Time) []session.Session {
	// Rank sessions per project so the newest Keep can be protected.
	byProject := make(map[string][]int)
	for i, s := range sessions {
		key := string(s.Agent) + "\x00" + strings.ToLower(s.WorkDir)
		byProject[key] = append(byProject[key], i)
	}
	protected := make(map[int]bool)
	for _, idx := range byProject {
		sort.Slice(idx, func(a, b int) bool {
			return sessions[idx[a]].UpdatedAt.After(sessions[idx[b]].UpdatedAt)
		})
		for n, i := range idx {
			if n < opts.Keep {
				protected[i] = true
			}
		}
	}

	var out []session.Session
	for i, s := range sessions {
		if protected[i] {
			continue
		}
		if opts.Agent != "" && s.Agent != opts.Agent {
			continue
		}
		if opts.ActiveWindow > 0 && now.Sub(s.UpdatedAt) < opts.ActiveWindow {
			continue
		}
		if opts.OlderThan > 0 && now.Sub(s.UpdatedAt) < opts.OlderThan {
			continue
		}
		if opts.LargerThan > 0 && s.Size < opts.LargerThan {
			continue
		}
		out = append(out, s)
	}
	sort.Slice(out, func(a, b int) bool { return out[a].UpdatedAt.Before(out[b].UpdatedAt) })
	return out
}

var durationRe = regexp.MustCompile(`^(\d+)\s*([dhwm]?)$`)

// ParseDuration accepts "30d", "12h", "2w", "3m" (months of 30 days) and a
// bare number (days). "0" or "0d" disables the condition.
func ParseDuration(s string) (time.Duration, error) {
	m := durationRe.FindStringSubmatch(strings.TrimSpace(strings.ToLower(s)))
	if m == nil {
		return 0, fmt.Errorf("invalid duration %q (use e.g. 30d, 12h, 2w)", s)
	}
	n, err := strconv.Atoi(m[1])
	if err != nil {
		return 0, err
	}
	unit := 24 * time.Hour
	switch m[2] {
	case "h":
		unit = time.Hour
	case "w":
		unit = 7 * 24 * time.Hour
	case "m":
		unit = 30 * 24 * time.Hour
	}
	return time.Duration(n) * unit, nil
}

var sizeRe = regexp.MustCompile(`^(\d+(?:\.\d+)?)\s*([kmg]?)b?$`)

// ParseSize accepts "10M", "512K", "1.5G", "10MB" or a bare byte count.
func ParseSize(s string) (int64, error) {
	m := sizeRe.FindStringSubmatch(strings.TrimSpace(strings.ToLower(s)))
	if m == nil {
		return 0, fmt.Errorf("invalid size %q (use e.g. 10M, 512K, 1G)", s)
	}
	v, err := strconv.ParseFloat(m[1], 64)
	if err != nil {
		return 0, err
	}
	mult := 1.0
	switch m[2] {
	case "k":
		mult = 1024
	case "m":
		mult = 1024 * 1024
	case "g":
		mult = 1024 * 1024 * 1024
	}
	return int64(v * mult), nil
}

// FormatSize renders a byte count compactly.
func FormatSize(n int64) string {
	switch {
	case n < 1024:
		return fmt.Sprintf("%dB", n)
	case n < 1024*1024:
		return fmt.Sprintf("%.1fK", float64(n)/1024)
	case n < 1024*1024*1024:
		return fmt.Sprintf("%.1fM", float64(n)/(1024*1024))
	default:
		return fmt.Sprintf("%.1fG", float64(n)/(1024*1024*1024))
	}
}
