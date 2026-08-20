package agent

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAiderSessionUsesHistoryDirectory(t *testing.T) {
	cwd := t.TempDir()
	history := filepath.Join(cwd, ".aider.chat.history.md")
	if err := os.WriteFile(history, []byte("# Test session\nhello\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	d := &AiderDetector{}
	sessions, err := d.ListSessions(cwd, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(sessions) != 1 {
		t.Fatalf("ListSessions returned %d sessions; want 1", len(sessions))
	}
	if sessions[0].WorkDir != cwd {
		t.Fatalf("Aider WorkDir = %q; want %q", sessions[0].WorkDir, cwd)
	}
}
