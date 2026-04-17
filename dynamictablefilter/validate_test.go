package dynamictablefilter

import (
	"errors"
	"os"
	"testing"
)

func TestListDynamicTables_ReadDirError(t *testing.T) {
	origRead := fsReadDir
	defer func() { fsReadDir = origRead }()
	fsReadDir = func(dirname string) ([]os.FileInfo, error) {
		return nil, errors.New("permission denied")
	}

	if _, err := ListDynamicTables(); err == nil {
		t.Error("expected error when ReadDir fails")
	}
}

func TestSetReadDirForTesting_NilRestore(t *testing.T) {
	// Passing nil should keep the current hook intact but still return
	// a valid restore callback.
	restore := SetReadDirForTesting(nil)
	if restore == nil {
		t.Fatal("expected non-nil restore function")
	}
	restore()
}

func TestSetReadDirForTesting_NonNilReplacesHook(t *testing.T) {
	var called bool
	restore := SetReadDirForTesting(func(dir string) ([]os.FileInfo, error) {
		called = true
		return nil, nil
	})
	defer restore()

	// Trigger the hook via ListDynamicTables to verify it's wired up.
	_, _ = ListDynamicTables()
	if !called {
		t.Error("expected injected read-dir to be called")
	}
}


func TestValidateTableName(t *testing.T) {
	cases := []struct {
		name    string
		in      string
		wantErr bool
	}{
		{"simple", "widgets", false},
		{"with digits", "table1", false},
		{"with hyphen", "my-table", false},
		{"with underscore", "my_table", false},
		{"mixed case", "MyTable", false},
		{"empty", "", true},
		{"path traversal", "../etc/passwd", true},
		{"slash", "foo/bar", true},
		{"newline", "foo\nbar", true},
		{"dot", "foo.json", true},
		{"space", "foo bar", true},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateTableName(tc.in)
			if (err != nil) != tc.wantErr {
				t.Errorf("ValidateTableName(%q)=%v, wantErr=%v", tc.in, err, tc.wantErr)
			}
		})
	}
}

func TestLoadTableSchema_InvalidName(t *testing.T) {
	if _, err := LoadTableSchema("../../etc"); err == nil {
		t.Error("expected error for invalid name")
	}
}

func TestLoadTableData_InvalidName(t *testing.T) {
	if _, err := LoadTableData("../../etc"); err == nil {
		t.Error("expected error for invalid name")
	}
}
