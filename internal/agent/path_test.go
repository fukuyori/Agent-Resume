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

func TestSamePath(t *testing.T) {
	dir := t.TempDir()

	if !samePath(dir, filepath.Join(dir, ".")) {
		t.Fatalf("samePath should match equivalent paths: %q", dir)
	}
	if samePath(dir, filepath.Join(dir, "other")) {
		t.Fatalf("samePath should not match a different path")
	}

	if runtime.GOOS == "windows" && !samePath(dir, strings.ToUpper(dir)) {
		t.Fatalf("samePath should ignore path case on Windows")
	}
}

func TestFileURIPath(t *testing.T) {
	var uri, want string
	if runtime.GOOS == "windows" {
		uri = "file:///C:/Users/test/My%20Project"
		want = `C:\Users\test\My Project`
	} else {
		uri = "file:///Users/test/My%20Project"
		want = "/Users/test/My Project"
	}

	if got := fileURIPath(uri); got != want {
		t.Fatalf("fileURIPath(%q) = %q; want %q", uri, got, want)
	}
}
