package pathsafe

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"
)

func TestContain(t *testing.T) {
	base := "/srv/data"
	absErr := errors.New("synthetic abs failure")

	// identity resolves a cleaned path to a deterministic absolute form
	// without touching the working directory, so tests stay hermetic.
	identity := func(p string) (string, error) {
		if filepath.IsAbs(p) {
			return p, nil
		}
		return filepath.Join("/abs", p), nil
	}

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
			absFn:     identity,
			base:      base,
			candidate: filepath.Join(base, "items", "schema.json"),
			label:     "items/schema.json",
			wantPath:  "/srv/data/items/schema.json",
		},
		{
			name:      "escapes base",
			absFn:     identity,
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
			name: "candidate abs fails",
			absFn: func() AbsFunc {
				calls := 0
				return func(p string) (string, error) {
					calls++
					if calls == 1 {
						return p, nil
					}
					return "", absErr
				}
			}(),
			base:      base,
			candidate: filepath.Join(base, "x"),
			label:     "x",
			wantErr:   "failed to resolve candidate path",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Contain(tc.absFn, tc.base, tc.candidate, tc.label)
			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("Contain() error = %v, want substring %q", err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("Contain() unexpected error: %v", err)
			}
			if got != tc.wantPath {
				t.Errorf("Contain() = %q, want %q", got, tc.wantPath)
			}
		})
	}
}
