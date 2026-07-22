package agent

import (
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestNormalizePath(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"", ""},
		{"D:\\home\\source\\go\\Agent-Resume", "d:/home/source/go/agent-resume"},
		{"d:/home/source/go/Agent-Resume/", "d:/home/source/go/agent-resume"},
		{"C:\\Users\\test\\project", "c:/users/test/project"},
	}

	for _, tt := range tests {
		got := normalizePath(tt.input)
		if runtime.GOOS == "windows" {
			if got != tt.expected {
				t.Errorf("normalizePath(%q) = %q; want %q", tt.input, got, tt.expected)
			}
		} else {
			cleanAbs, _ := filepath.Abs(tt.input)
			expectedUnix := filepath.ToSlash(filepath.Clean(cleanAbs))
			if got != expectedUnix {
				t.Errorf("normalizePath(%q) = %q; want %q", tt.input, got, expectedUnix)
			}
		}
	}
}

func TestClaudeProjectDir(t *testing.T) {
	d := &ClaudeDetector{}
	cwd := "C:\\Users\\test\\project"
	slug := d.projectSlug(cwd)
	if strings.Contains(slug, "\\") || strings.Contains(slug, ":") {
		t.Errorf("projectSlug(%q) contains invalid chars: %q", cwd, slug)
	}
}
