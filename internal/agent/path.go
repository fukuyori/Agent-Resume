package agent

import (
	"path/filepath"
	"runtime"
	"strings"
)

// normalizePath converts a path to a clean absolute path for comparison.
// On Windows, it handles case-insensitivity, slashes, and volume names.
func normalizePath(p string) string {
	if p == "" {
		return ""
	}
	abs, err := filepath.Abs(p)
	if err != nil {
		abs = p
	}
	cleaned := filepath.Clean(abs)
	slashed := filepath.ToSlash(cleaned)
	if runtime.GOOS == "windows" {
		return strings.ToLower(slashed)
	}
	return slashed
}
