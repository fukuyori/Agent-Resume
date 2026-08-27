package agent

import (
	"errors"
	"os"
)

// removePaths deletes every path (file or directory tree). Missing paths are
// ignored; other errors are collected and returned together.
func removePaths(paths []string) error {
	var errs []error
	for _, p := range paths {
		if err := os.RemoveAll(p); err != nil && !os.IsNotExist(err) {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}
