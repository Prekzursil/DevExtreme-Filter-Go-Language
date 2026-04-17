package dynamictablefilter

import "testing"

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
