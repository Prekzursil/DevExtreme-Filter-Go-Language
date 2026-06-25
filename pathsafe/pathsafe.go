// Package pathsafe centralises the CWE-22 (path-injection) containment check
// shared by the schema and dynamic-table file loaders. Both call sites join a
// user-supplied name onto a trusted base directory and must prove, via a
// filepath.Rel containment check, that the resolved path cannot escape that
// base before reaching os.ReadFile/os.WriteFile (gosec G304 / Sonar
// gosecurity:S2083 / CodeQL go/path-injection).
package pathsafe

import (
	"fmt"
	"path/filepath"
	"strings"
)

// AbsFunc resolves a path to its absolute form. It matches filepath.Abs and
// exists so callers can inject filepath.Abs failures in tests (which on real
// systems only happen if os.Getwd fails — virtually untestable without
// process-level state corruption).
type AbsFunc func(string) (string, error)

// Contain joins base and candidate, resolves both to cleaned absolute paths
// via absFn, and returns the cleaned candidate only when it stays inside base.
// The supplied candidate is expected to already be base joined with the
// user-controlled parts; label is the human-readable value reported in the
// "escapes base directory" error so callers can surface the offending name.
func Contain(absFn AbsFunc, base, candidate, label string) (string, error) {
	cleanedBase, err := absFn(filepath.Clean(base))
	if err != nil {
		return "", fmt.Errorf("failed to resolve base path: %w", err)
	}
	cleanedCandidate, err := absFn(filepath.Clean(candidate))
	if err != nil {
		return "", fmt.Errorf("failed to resolve candidate path: %w", err)
	}
	rel, err := filepath.Rel(cleanedBase, cleanedCandidate)
	if err != nil || strings.HasPrefix(rel, "..") || rel == ".." {
		return "", fmt.Errorf("path escapes base directory: %s", label)
	}
	return cleanedCandidate, nil
}
