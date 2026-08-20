package agent

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestParseAntigravityTime(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name:    "RFC3339 standard",
			input:   "2026-07-22T11:40:58Z",
			wantErr: false,
		},
		{
			name:    "RFC3339 with timezone offset",
			input:   "2026-07-22T20:41:30+09:00",
			wantErr: false,
		},
		{
			name:    "RFC3339Nano with UTC subseconds",
			input:   "2026-07-22T11:40:58.1088749Z",
			wantErr: false,
		},
		{
			name:    "RFC3339Nano with timezone offset and subseconds",
			input:   "2026-07-22T20:41:30.6875769+09:00",
			wantErr: false,
		},
		{
			name:    "Invalid time string",
			input:   "invalid-time",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseAntigravityTime(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseAntigravityTime() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got.IsZero() {
				t.Errorf("parseAntigravityTime() got zero time for valid input %s", tt.input)
			}
		})
	}
}

func TestAntigravityAllProjectsIncludesWorkingDirectories(t *testing.T) {
	root := t.TempDir()
	cacheDir := filepath.Join(root, "cache")
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		t.Fatal(err)
	}

	projectA := filepath.Join(root, "project-a")
	projectB := filepath.Join(root, "project-b")
	for _, dir := range []string{projectA, projectB} {
		if err := os.Mkdir(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}

	metadata := map[string]any{
		"conversations": map[string]any{
			"session-a": map[string]any{
				"summary": map[string]any{
					"Title":         "session A",
					"WorkspaceURIs": []string{projectA},
				},
			},
			"session-b": map[string]any{
				"summary": map[string]any{
					"Title":         "session B",
					"WorkspaceURIs": []string{projectB},
				},
			},
		},
	}
	data, err := json.Marshal(metadata)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cacheDir, "conversation_metadata.json"), data, 0o600); err != nil {
		t.Fatal(err)
	}

	d := &AntigravityDetector{dir: root}
	current, err := d.ListSessions(projectA, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(current) != 1 || current[0].WorkDir != projectA {
		t.Fatalf("current-project sessions = %#v; want only project A", current)
	}

	all, err := d.ListSessions(projectA, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 2 {
		t.Fatalf("all-project sessions returned %d sessions; want 2: %#v", len(all), all)
	}
}
