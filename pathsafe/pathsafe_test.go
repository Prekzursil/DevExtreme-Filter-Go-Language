package pathsafe

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"
)

// identityAbs resolves a path to an absolute form without touching the
// working directory, so table-driven cases that call it stay hermetic.
func identityAbs(p string) (string, error) {
	if filepath.IsAbs(p) {
		return p, nil
	}
	return filepath.Join("/abs", p), nil
}

// failOnSecondCall returns an AbsFunc that succeeds on the first call (returns
// the path unchanged) and returns errSynth on every subsequent call. This
// exercises the "candidate abs fails" branch of Contain without requiring
// os.Getwd to fail in the real process.
func failOnSecondCall(errSynth error) AbsFunc {
	calls := 0
	return func(p string) (string, error) {
		calls++
		if calls == 1 {
			return p, nil
		}
		return "", errSynth
	}
}

// checkContainError asserts that Contain returned a non-nil error containing
// wantSubstr, then returns. It is extracted so the caller (runContainCase) has
// no nested if/else chains that inflate cognitive complexity.
func checkContainError(t *testing.T, err error, wantSubstr string) {
	t.Helper()
	if err == nil || !strings.Contains(err.Error(), wantSubstr) {
		t.Fatalf("Contain() error = %v, want substring %q", err, wantSubstr)
	}
}

// runContainCase executes a single Contain table-driven case and reports
// failures via t. Splitting assertion logic into a named helper keeps
// TestContain below the cognitive-complexity threshold.
func runContainCase(t *testing.T, absFn AbsFunc, base, candidate, label, wantErr, wantPath string) {
	t.Helper()
	got, err := Contain(absFn, base, candidate, label)
	if wantErr != "" {
		checkContainError(t, err, wantErr)
		return
	}
	if err != nil {
		t.Fatalf("Contain() unexpected error: %v", err)
	}
	if got != wantPath {
		t.Errorf("Contain() = %q, want %q", got, wantPath)
	}
}

func TestContain(t *testing.T) {
	base := "/srv/data"
	absErr := errors.New("synthetic abs failure")

	tests := []struct {
		name      string
		absFn     AbsFunc
		base      string
		candidate string
		label     string
		wantErr   string // substring; "" means success
		wantPath  string // only checked on success
	}{
		{
			name:      "contained",
			absFn:     identityAbs,
			base:      base,
			candidate: filepath.Join(base, "items", "schema.json"),
			label:     "items/schema.json",
			wantPath:  "/srv/data/items/schema.json",
		},
		{
			name:      "escapes base",
			absFn:     identityAbs,
			base:      base,
			candidate: filepath.Join(base, "..", "..", "etc", "passwd"),
			label:     "../../etc/passwd",
			wantErr:   "path escapes base directory",
		},
		{
			name:      "base abs fails",
			absFn:     func(string) (string, error) { return "", absErr },
			base:      base,
			candidate: filepath.Join(base, "x"),
			label:     "x",
			wantErr:   "failed to resolve base path",
		},
		{
			name:      "candidate abs fails",
			absFn:     failOnSecondCall(absErr),
			base:      base,
			candidate: filepath.Join(base, "x"),
			label:     "x",
			wantErr:   "failed to resolve candidate path",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			runContainCase(t, tc.absFn, tc.base, tc.candidate, tc.label, tc.wantErr, tc.wantPath)
		})
	}
}
