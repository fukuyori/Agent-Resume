package agent

import (
	"net/url"
	"os"
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

// samePath reports whether two paths identify the same filesystem location.
// os.SameFile handles symlinks and the filesystem's actual case-sensitivity.
// The normalized fallback is needed for paths recorded after a directory was
// moved or removed.
func samePath(a, b string) bool {
	if a == "" || b == "" {
		return false
	}

	aInfo, aErr := os.Stat(a)
	bInfo, bErr := os.Stat(b)
	if aErr == nil && bErr == nil {
		return os.SameFile(aInfo, bInfo)
	}

	aNorm := normalizePath(a)
	bNorm := normalizePath(b)
	if runtime.GOOS == "windows" {
		return strings.EqualFold(aNorm, bNorm)
	}
	return aNorm == bNorm
}

// fileURIPath converts a file URI stored by an agent into a native path.
func fileURIPath(raw string) string {
	u, err := url.Parse(raw)
	if err != nil || !strings.EqualFold(u.Scheme, "file") {
		return raw
	}

	p, err := url.PathUnescape(u.Path)
	if err != nil {
		p = u.Path
	}

	if u.Host != "" && !strings.EqualFold(u.Host, "localhost") {
		p = "//" + u.Host + p
	}

	if runtime.GOOS == "windows" {
		if len(p) >= 3 && p[0] == '/' && p[2] == ':' {
			p = p[1:]
		}
		p = filepath.FromSlash(p)
	}
	return p
}
