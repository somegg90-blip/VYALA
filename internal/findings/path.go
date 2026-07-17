package findings

import (
	"path/filepath"
	"strings"
)

// NormalizeRelPath converts an absolute (or repo-relative) file path into a
// forward-slash, repo-root-relative path suitable for hashing into a
// finding_id and for rendering as a stable file reference in reports.
func NormalizeRelPath(repoRoot, path string) string {
	rel := path
	if filepath.IsAbs(path) {
		if r, err := filepath.Rel(repoRoot, path); err == nil {
			rel = r
		}
	}
	rel = filepath.ToSlash(rel)
	return strings.TrimPrefix(rel, "./")
}
