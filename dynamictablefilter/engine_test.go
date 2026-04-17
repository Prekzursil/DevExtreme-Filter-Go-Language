package dynamictablefilter

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"transaction-filter-backend/schematool"
)

func setupTestTables(t *testing.T) string {
	t.Helper()
	tmp := t.TempDir()

	schema := TableSchema{
		EntityName: "demo",
		Fields: []schematool.SchemaFieldDefinition{
			{Name: "name", Type: "string"},
			{Name: "count", Type: "int"},
			{Name: "rate", Type: "float64"},
			{Name: "active", Type: "bool"},
			{Name: "when", Type: "time.Time"},
		},
	}

	tableDir := filepath.Join(tmp, "demo")
	if err := os.MkdirAll(tableDir, 0755); err != nil {
		t.Fatal(err)
	}
	schemaBytes, _ := json.Marshal(schema)
	if err := os.WriteFile(filepath.Join(tableDir, "schema.json"), schemaBytes, 0644); err != nil {
		t.Fatal(err)
	}

	data := []map[string]interface{}{
		{"name": "Alpha", "count": 10.0, "rate": 1.5, "active": true, "when": "2024-01-01T00:00:00Z"},
		{"name": "Beta", "count": 20.0, "rate": 2.5, "active": false, "when": "2024-06-15T12:00:00Z"},
		{"name": "Gamma", "count": 30.0, "rate": 3.5, "active": true, "when": "2024-12-31T23:59:59Z"},
	}
	dataBytes, _ := json.Marshal(data)
	if err := os.WriteFile(filepath.Join(tableDir, "data.json"), dataBytes, 0644); err != nil {
		t.Fatal(err)
	}

	return tmp
}

func TestSetGetBaseTablesPath(t *testing.T) {
	orig := GetBaseTablesPath()
	defer SetBaseTablesPath(orig)

	SetBaseTablesPath("/tmp/custom")
	if got := GetBaseTablesPath(); got != "/tmp/custom" {
		t.Errorf("got %q, want %q", got, "/tmp/custom")
	}
}

func TestLoadTableSchema(t *testing.T) {
	orig := GetBaseTablesPath()
	defer SetBaseTablesPath(orig)

	SetBaseTablesPath(setupTestTables(t))

	schema, err := LoadTableSchema("demo")
	if err != nil {
		t.Fatalf("LoadTableSchema error: %v", err)
	}
	if schema.EntityName != "demo" {
		t.Errorf("EntityName=%q, want %q", schema.EntityName, "demo")
	}
	if len(schema.Fields) != 5 {
		t.Errorf("Fields count=%d, want 5", len(schema.Fields))
	}
	if schema.FieldMap == nil {
		t.Fatal("FieldMap is nil")
	}
	if _, ok := schema.FieldMap["name"]; !ok {
		t.Error("expected 'name' in FieldMap")
	}

	if _, err := LoadTableSchema("missing"); err == nil {
		t.Error("expected error for missing table")
	}
}

func writeBrokenTable(t *testing.T, filename string, content []byte) {
	t.Helper()
	orig := GetBaseTablesPath()
	t.Cleanup(func() { SetBaseTablesPath(orig) })

	tmp := t.TempDir()
	SetBaseTablesPath(tmp)

	tableDir := filepath.Join(tmp, "broken")
	if err := os.MkdirAll(tableDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tableDir, filename), content, 0644); err != nil {
		t.Fatal(err)
	}
}

func TestLoadTableSchema_InvalidJSON(t *testing.T) {
	writeBrokenTable(t, "schema.json", []byte("not valid json"))
	if _, err := LoadTableSchema("broken"); err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestLoadTableData(t *testing.T) {
	orig := GetBaseTablesPath()
	defer SetBaseTablesPath(orig)

	SetBaseTablesPath(setupTestTables(t))

	data, err := LoadTableData("demo")
	if err != nil {
		t.Fatalf("LoadTableData error: %v", err)
	}
	if len(data) != 3 {
		t.Errorf("data count=%d, want 3", len(data))
	}

	if _, err := LoadTableData("missing"); err == nil {
		t.Error("expected error for missing table data")
	}
}

func TestLoadTableData_InvalidJSON(t *testing.T) {
	writeBrokenTable(t, "data.json", []byte("garbage"))
	if _, err := LoadTableData("broken"); err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestListDynamicTables(t *testing.T) {
	orig := GetBaseTablesPath()
	defer SetBaseTablesPath(orig)

	SetBaseTablesPath(setupTestTables(t))

	tables, err := ListDynamicTables()
	if err != nil {
		t.Fatalf("ListDynamicTables error: %v", err)
	}
	if len(tables) != 1 || tables[0] != "demo" {
		t.Errorf("tables=%v, want [demo]", tables)
	}
}

func TestListDynamicTables_MissingDir(t *testing.T) {
	orig := GetBaseTablesPath()
	defer SetBaseTablesPath(orig)

	SetBaseTablesPath("/nonexistent/path/here")

	tables, err := ListDynamicTables()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tables) != 0 {
		t.Errorf("expected empty list, got %v", tables)
	}
}

func TestEvaluateCondition_String(t *testing.T) {
	cases := []struct {
		op     string
		rec    interface{}
		filter interface{}
		want   bool
	}{
		{"=", "hello", "HELLO", true},
		{"<>", "hello", "HELLO", false},
		{"contains", "hello world", "WORLD", true},
		{"startswith", "hello world", "HELLO", true},
		{"endswith", "hello world", "world", true},
		{"notcontains", "hello world", "xyz", true},
		{"unknown", "hello", "hello", false},
	}
	for _, tc := range cases {
		got := evaluateCondition(tc.rec, tc.op, tc.filter, "string")
		if got != tc.want {
			t.Errorf("string %s %v %v: got %v, want %v", tc.op, tc.rec, tc.filter, got, tc.want)
		}
	}
}

func TestEvaluateCondition_Int(t *testing.T) {
	cases := []struct {
		op     string
		rec    interface{}
		filter interface{}
		want   bool
	}{
		{"=", 10.0, 10, true},
		{"<>", 10.0, 11, true},
		{">", 20.0, 10, true},
		{">=", 10.0, 10, true},
		{"<", 5.0, 10, true},
		{"<=", 10.0, 10, true},
		{"=", 10, 10, true},
		{"=", "notanumber", 10, false},
		{"=", 10.0, "abc", false},
	}
	for _, tc := range cases {
		got := evaluateCondition(tc.rec, tc.op, tc.filter, "int")
		if got != tc.want {
			t.Errorf("int %s %v %v: got %v, want %v", tc.op, tc.rec, tc.filter, got, tc.want)
		}
	}
}

func TestEvaluateCondition_Float(t *testing.T) {
	cases := []struct {
		op     string
		rec    interface{}
		filter interface{}
		want   bool
	}{
		{"=", 1.5, 1.5, true},
		{"<>", 1.5, 2.0, true},
		{">", 3.0, 1.0, true},
		{">=", 2.0, 2.0, true},
		{"<", 1.0, 2.0, true},
		{"<=", 2.0, 2.0, true},
		{"=", "nonnumeric", 1.5, false},
		{"=", 1.5, "abc", false},
	}
	for _, tc := range cases {
		got := evaluateCondition(tc.rec, tc.op, tc.filter, "float64")
		if got != tc.want {
			t.Errorf("float %s %v %v: got %v, want %v", tc.op, tc.rec, tc.filter, got, tc.want)
		}
	}
}

func TestEvaluateCondition_Bool(t *testing.T) {
	if !evaluateCondition(true, "=", true, "bool") {
		t.Error("true=true should be true")
	}
	if evaluateCondition(true, "=", false, "bool") {
		t.Error("true=false should be false")
	}
	if !evaluateCondition(true, "<>", false, "bool") {
		t.Error("true<>false should be true")
	}
	if evaluateCondition("notbool", "=", true, "bool") {
		t.Error("non-bool should return false")
	}
	if evaluateCondition(true, "=", "invalid", "bool") {
		t.Error("invalid filter should return false")
	}
}

func TestEvaluateCondition_Time(t *testing.T) {
	cases := []struct {
		op     string
		rec    interface{}
		filter interface{}
		want   bool
	}{
		{"=", "2024-01-01T00:00:00Z", "2024-01-01T00:00:00Z", true},
		{"<>", "2024-01-01T00:00:00Z", "2024-06-01T00:00:00Z", true},
		{">", "2024-06-01T00:00:00Z", "2024-01-01T00:00:00Z", true},
		{">=", "2024-01-01T00:00:00Z", "2024-01-01T00:00:00Z", true},
		{"<", "2024-01-01T00:00:00Z", "2024-06-01T00:00:00Z", true},
		{"<=", "2024-01-01T00:00:00Z", "2024-01-01T00:00:00Z", true},
		{"=", "notadate", "2024-01-01T00:00:00Z", false},
		{"=", "2024-01-01T00:00:00Z", "bad", false},
	}
	for _, tc := range cases {
		got := evaluateCondition(tc.rec, tc.op, tc.filter, "time.Time")
		if got != tc.want {
			t.Errorf("time %s %v %v: got %v, want %v", tc.op, tc.rec, tc.filter, got, tc.want)
		}
	}
}

func TestFilterDynamicData_NilFilter(t *testing.T) {
	data := []map[string]interface{}{{"a": 1.0}}
	got, err := FilterDynamicData(data, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 {
		t.Errorf("expected all data returned for nil filter")
	}
}

func TestFilterDynamicData_EmptyFilter(t *testing.T) {
	data := []map[string]interface{}{{"a": 1.0}}
	schema := &TableSchema{FieldMap: map[string]schematool.SchemaFieldDefinition{}}
	got, err := FilterDynamicData(data, schema, []interface{}{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 {
		t.Errorf("expected all data returned for empty filter")
	}
}

func TestFilterDynamicData_NonArray(t *testing.T) {
	_, err := FilterDynamicData(nil, nil, "not array")
	if err == nil {
		t.Error("expected error for non-array filter")
	}
}

func TestFilterDynamicData_SimpleFilter(t *testing.T) {
	orig := GetBaseTablesPath()
	defer SetBaseTablesPath(orig)
	SetBaseTablesPath(setupTestTables(t))

	schema, err := LoadTableSchema("demo")
	if err != nil {
		t.Fatal(err)
	}
	data, err := LoadTableData("demo")
	if err != nil {
		t.Fatal(err)
	}

	result, err := FilterDynamicData(data, schema, []interface{}{"name", "=", "Alpha"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 1 {
		t.Errorf("expected 1 result for name=Alpha, got %d", len(result))
	}

	result, err = FilterDynamicData(data, schema, []interface{}{"count", ">", 15})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 2 {
		t.Errorf("expected 2 results for count>15, got %d", len(result))
	}
}

func TestFilterDynamicData_NotFilter(t *testing.T) {
	orig := GetBaseTablesPath()
	defer SetBaseTablesPath(orig)
	SetBaseTablesPath(setupTestTables(t))

	schema, err := LoadTableSchema("demo")
	if err != nil {
		t.Fatal(err)
	}
	data, err := LoadTableData("demo")
	if err != nil {
		t.Fatal(err)
	}

	result, err := FilterDynamicData(data, schema, []interface{}{"!", []interface{}{"name", "=", "Alpha"}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 2 {
		t.Errorf("expected 2 results (NOT Alpha), got %d", len(result))
	}

	if _, err := FilterDynamicData(data, schema, []interface{}{"!"}); err == nil {
		t.Error("expected error for malformed NOT")
	}
	if _, err := FilterDynamicData(data, schema, []interface{}{"!", "notarray"}); err == nil {
		t.Error("expected error for NOT of non-array")
	}
}

func TestFilterDynamicData_GroupFilter(t *testing.T) {
	orig := GetBaseTablesPath()
	defer SetBaseTablesPath(orig)
	SetBaseTablesPath(setupTestTables(t))

	schema, err := LoadTableSchema("demo")
	if err != nil {
		t.Fatal(err)
	}
	data, err := LoadTableData("demo")
	if err != nil {
		t.Fatal(err)
	}

	andFilter := []interface{}{
		[]interface{}{"count", ">=", 10},
		"and",
		[]interface{}{"active", "=", true},
	}
	result, err := FilterDynamicData(data, schema, andFilter)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 2 {
		t.Errorf("expected 2 results for AND, got %d", len(result))
	}

	orFilter := []interface{}{
		[]interface{}{"name", "=", "Alpha"},
		"or",
		[]interface{}{"name", "=", "Gamma"},
	}
	result, err = FilterDynamicData(data, schema, orFilter)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 2 {
		t.Errorf("expected 2 results for OR, got %d", len(result))
	}

	badOp := []interface{}{
		[]interface{}{"name", "=", "Alpha"},
		"xor",
		[]interface{}{"name", "=", "Beta"},
	}
	if _, err := FilterDynamicData(data, schema, badOp); err == nil {
		t.Error("expected error for invalid logical op")
	}
}

func TestFilterDynamicData_MissingField(t *testing.T) {
	schema := &TableSchema{FieldMap: map[string]schematool.SchemaFieldDefinition{
		"name": {Name: "name", Type: "string"},
	}}
	data := []map[string]interface{}{{"name": "x"}}
	_, err := FilterDynamicData(data, schema, []interface{}{"doesnotexist", "=", "x"})
	if err == nil {
		t.Error("expected error for missing field")
	}
}
