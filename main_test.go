package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"agres/internal/session"
)

func TestParseArgs(t *testing.T) {
	tests := []struct {
		name            string
		args            []string
		wantLimit       int
		wantSearchLimit *int
		wantSearch      string
		wantAll         bool
		wantVersion     bool
		wantHelp        bool
		wantErr         bool
	}{
		{
			name:        "default limit",
			args:        []string{},
			wantLimit:   10,
			wantVersion: false,
			wantHelp:    false,
			wantErr:     false,
		},
		{
			name:      "all projects short flag",
			args:      []string{"-a"},
			wantLimit: 10,
			wantAll:   true,
		},
		{
			name:      "all projects long flag with limit",
			args:      []string{"--all", "--limit", "25"},
			wantLimit: 25,
			wantAll:   true,
		},
		{
			name:        "positional limit",
			args:        []string{"20"},
			wantLimit:   20,
			wantVersion: false,
			wantHelp:    false,
			wantErr:     false,
		},
		{
			name:        "flag -n limit",
			args:        []string{"-n", "15"},
			wantLimit:   15,
			wantVersion: false,
			wantHelp:    false,
			wantErr:     false,
		},
		{
			name:        "flag -n= limit",
			args:        []string{"-n=25"},
			wantLimit:   25,
			wantVersion: false,
			wantHelp:    false,
			wantErr:     false,
		},
		{
			name:        "flag --limit",
			args:        []string{"--limit", "30"},
			wantLimit:   30,
			wantVersion: false,
			wantHelp:    false,
			wantErr:     false,
		},
		{
			name:        "flag --limit=",
			args:        []string{"--limit=35"},
			wantLimit:   35,
			wantVersion: false,
			wantHelp:    false,
			wantErr:     false,
		},
		{
			name:        "version flag -v",
			args:        []string{"-v"},
			wantLimit:   10,
			wantVersion: true,
			wantHelp:    false,
			wantErr:     false,
		},
		{
			name:        "version flag --version",
			args:        []string{"--version"},
			wantLimit:   10,
			wantVersion: true,
			wantHelp:    false,
			wantErr:     false,
		},
		{
			name:        "help flag -h",
			args:        []string{"-h"},
			wantLimit:   10,
			wantVersion: false,
			wantHelp:    true,
			wantErr:     false,
		},
		{
			name:        "help flag --help",
			args:        []string{"--help"},
			wantLimit:   10,
			wantVersion: false,
			wantHelp:    true,
			wantErr:     false,
		},
		{
			name:       "search short flag",
			args:       []string{"-s", "Authentication Error"},
			wantLimit:  10,
			wantSearch: "Authentication Error",
		},
		{
			name:       "search inline long flag",
			args:       []string{"--search=認証エラー", "--search-limit=500", "--limit", "20"},
			wantLimit:  20,
			wantSearch: "認証エラー",
			wantSearchLimit: func() *int {
				v := 500
				return &v
			}(),
		},
		{
			name:       "unlimited search",
			args:       []string{"--search", "error", "--search-limit", "0"},
			wantLimit:  10,
			wantSearch: "error",
			wantSearchLimit: func() *int {
				v := 0
				return &v
			}(),
		},
		{
			name:    "invalid negative limit",
			args:    []string{"-5"},
			wantErr: true,
		},
		{
			name:    "invalid non-integer positional arg",
			args:    []string{"foo"},
			wantErr: true,
		},
		{
			name:    "missing value for -n",
			args:    []string{"-n"},
			wantErr: true,
		},
		{
			name:    "invalid value for -n",
			args:    []string{"-n", "abc"},
			wantErr: true,
		},
		{
			name:    "zero value for -n",
			args:    []string{"-n", "0"},
			wantErr: true,
		},
		{
			name:    "missing search keyword",
			args:    []string{"--search"},
			wantErr: true,
		},
		{
			name:    "empty search keyword",
			args:    []string{"--search=   "},
			wantErr: true,
		},
		{
			name:    "negative search limit",
			args:    []string{"--search", "error", "--search-limit", "-1"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts, err := parseArgs(tt.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseArgs() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}
			if opts.limit != tt.wantLimit {
				t.Errorf("parseArgs() limit = %v, want %v", opts.limit, tt.wantLimit)
			}
			wantSearchLimit := 100
			if tt.wantSearchLimit != nil {
				wantSearchLimit = *tt.wantSearchLimit
			}
			if opts.searchLimit != wantSearchLimit {
				t.Errorf("parseArgs() searchLimit = %v, want %v", opts.searchLimit, wantSearchLimit)
			}
			if opts.search != tt.wantSearch {
				t.Errorf("parseArgs() search = %q, want %q", opts.search, tt.wantSearch)
			}
			if opts.allProjects != tt.wantAll {
				t.Errorf("parseArgs() allProjects = %v, want %v", opts.allProjects, tt.wantAll)
			}
			if opts.showVersion != tt.wantVersion {
				t.Errorf("parseArgs() showVersion = %v, want %v", opts.showVersion, tt.wantVersion)
			}
			if opts.showHelp != tt.wantHelp {
				t.Errorf("parseArgs() showHelp = %v, want %v", opts.showHelp, tt.wantHelp)
			}
		})
	}
}

type fakeSearcher struct {
	matches map[string]bool
	err     error
}

func (f fakeSearcher) SearchSession(s session.Session, keyword string) (bool, error) {
	if f.err != nil {
		return false, f.err
	}
	return f.matches[s.ID], nil
}

func TestFilterSessionsAppliesSearchLimitBeforeResultLimit(t *testing.T) {
	sessions := []session.Session{
		{ID: "newest", Agent: session.AgentCodex, Title: "first"},
		{ID: "second", Agent: session.AgentCodex, Title: "second"},
		{ID: "older-match", Agent: session.AgentCodex, Title: "third"},
	}
	searchers := map[session.Agent]session.Searcher{
		session.AgentCodex: fakeSearcher{matches: map[string]bool{"older-match": true}},
	}

	limited, err := filterSessions(sessions, "needle", 2, searchers)
	if err != nil {
		t.Fatal(err)
	}
	if len(limited) != 0 {
		t.Fatalf("limited search returned %#v; want no matches", limited)
	}

	unlimited, err := filterSessions(sessions, "needle", 0, searchers)
	if err != nil {
		t.Fatal(err)
	}
	if len(unlimited) != 1 || unlimited[0].ID != "older-match" {
		t.Fatalf("unlimited search returned %#v; want older-match", unlimited)
	}
}

func TestFilterSessionsMatchesTitlesCaseInsensitively(t *testing.T) {
	sessions := []session.Session{{ID: "title", Agent: session.AgentCodex, Title: "Authentication Error"}}
	got, err := filterSessions(sessions, "authentication", 100, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("filterSessions returned %#v; want title match", got)
	}
}

func TestResumeCommandUsesOriginalWorkingDirectory(t *testing.T) {
	workDir := t.TempDir()
	selected := session.Session{
		WorkDir:   workDir,
		ResumeCmd: []string{"agent-command", "--resume", "session-id"},
	}

	cmd, err := resumeCommand(selected)
	if err != nil {
		t.Fatal(err)
	}
	if cmd.Dir != workDir {
		t.Fatalf("resumeCommand() Dir = %q; want %q", cmd.Dir, workDir)
	}
}

func TestResumeCommandRejectsMissingWorkingDirectory(t *testing.T) {
	selected := session.Session{
		WorkDir:   filepath.Join(t.TempDir(), "missing"),
		ResumeCmd: []string{"agent-command", "--resume", "session-id"},
	}

	if _, err := resumeCommand(selected); err == nil {
		t.Fatal("resumeCommand() succeeded for a missing working directory")
	}

	selected.WorkDir = ""
	if _, err := resumeCommand(selected); err == nil {
		t.Fatal("resumeCommand() succeeded without a working directory")
	}

	filePath := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(filePath, []byte("not a directory"), 0o600); err != nil {
		t.Fatal(err)
	}
	selected.WorkDir = filePath
	if _, err := resumeCommand(selected); err == nil {
		t.Fatal("resumeCommand() succeeded with a file as working directory")
	}
}

func TestParseCleanArgs(t *testing.T) {
	opts, err := parseCleanArgs([]string{"-a", "--older-than", "7d", "--larger-than=10M", "--keep", "0", "--agent", "codex", "-y", "--dry-run"})
	if err != nil {
		t.Fatal(err)
	}
	if !opts.allProjects || !opts.yes || !opts.dryRun || opts.Keep != 0 || opts.Agent != "codex" ||
		opts.OlderThan != 7*24*time.Hour || opts.LargerThan != 10<<20 {
		t.Errorf("unexpected options: %+v", opts)
	}

	defaults, err := parseCleanArgs(nil)
	if err != nil || defaults.OlderThan != 30*24*time.Hour || defaults.Keep != 3 || defaults.ActiveWindow != time.Hour {
		t.Errorf("unexpected defaults: %+v (%v)", defaults, err)
	}

	for _, bad := range [][]string{{"--older-than"}, {"--agent", "aider"}, {"--keep", "-1"}, {"--bogus"}} {
		if _, err := parseCleanArgs(bad); err == nil {
			t.Errorf("expected error for %v", bad)
		}
	}
}
