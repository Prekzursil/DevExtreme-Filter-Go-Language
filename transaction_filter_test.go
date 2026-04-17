package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	"transaction-filter-backend/ent"

	_ "github.com/mattn/go-sqlite3"
)

var testClient *ent.Client

func TestMain(m *testing.M) {
	var err error
	testClient, err = ent.Open("sqlite3", "file:ent_test_main?mode=memory&cache=shared&_fk=1")
	if err != nil {
		log.Fatalf("failed opening connection to sqlite: %v", err)
	}
	defer testClient.Close()

	if err := testClient.Schema.Create(context.Background()); err != nil {
		log.Fatalf("failed creating schema resources: %v", err)
	}

	originalClient := client
	client = testClient

	generateTestTransactions(testClient, 50)

	code := m.Run()

	client = originalClient
	os.Exit(code)
}

func generateTestTransactions(c *ent.Client, count int) {
	locations := []string{"Testville", "Sampleburg", "Demo City", "Alpha Town", "Beta Village"}
	categories := []string{"Test Cat A", "Sample Cat B", "Demo Cat C", "Alpha Cat D", "Beta Cat E"}
	types := []string{"Test Debit", "Sample Credit"}

	for i := 0; i < count; i++ {
		base := time.Date(2024, time.January, 1, 0, 0, 0, 0, time.UTC).AddDate(0, 0, i)
		d := time.Date(base.Year(), base.Month(), base.Day(), i%24, (i*13)%60, (i*7)%60, 0, time.UTC)
		if _, err := c.Transaction.Create().
			SetAmount(float64((i%10 + 1) * 100)).
			SetDate(d).
			SetName(fmt.Sprintf("Test Trans %d", i)).
			SetLocation(locations[i%len(locations)]).
			SetCategory(categories[i%len(categories)]).
			SetType(types[i%len(types)]).
			Save(context.Background()); err != nil {
			log.Fatalf("failed generating test transaction %d: %v", i, err)
		}
	}
}

type filterCase struct {
	name          string
	filterInput   interface{}
	expectError   bool
}

func parseFilterCase(t *testing.T, tc filterCase) {
	t.Helper()
	adapter, err := GetAdapter("transaction")
	if err != nil {
		t.Skipf("no transaction adapter: %v", err)
	}
	_, err = ParseFilterToPredicates(adapter, tc.filterInput)
	if tc.expectError && err == nil {
		t.Errorf("%s: expected error, got nil", tc.name)
	}
	if !tc.expectError && err != nil {
		t.Errorf("%s: unexpected error: %v", tc.name, err)
	}
}

func TestFilterTransactions_NilFilter(t *testing.T) {
	parseFilterCase(t, filterCase{name: "nil filter", filterInput: nil})
}

func TestFilterTransactions_EmptyArray(t *testing.T) {
	parseFilterCase(t, filterCase{name: "empty array", filterInput: []interface{}{}})
}

func TestFilterTransactions_SimpleEquality(t *testing.T) {
	parseFilterCase(t, filterCase{
		name:        "amount equals",
		filterInput: []interface{}{"amount", "=", 100.0},
	})
}

func TestFilterTransactions_GreaterThan(t *testing.T) {
	parseFilterCase(t, filterCase{
		name:        "amount > 500",
		filterInput: []interface{}{"amount", ">", 500.0},
	})
}

func TestFilterTransactions_Between(t *testing.T) {
	parseFilterCase(t, filterCase{
		name:        "amount between",
		filterInput: []interface{}{"amount", "between", []interface{}{200.0, 400.0}},
	})
}

func TestFilterTransactions_Contains(t *testing.T) {
	parseFilterCase(t, filterCase{
		name:        "name contains",
		filterInput: []interface{}{"name", "contains", "Trans 1"},
	})
}

func TestFilterTransactions_ComplexGroup(t *testing.T) {
	parseFilterCase(t, filterCase{
		name: "complex group",
		filterInput: []interface{}{
			[]interface{}{
				[]interface{}{"name", "contains", "Trans 0"},
				"or",
				[]interface{}{"name", "contains", "Trans 1"},
			},
			"and",
			[]interface{}{"amount", "=", 100.0},
		},
	})
}

func TestFilterTransactions_NonExistentField(t *testing.T) {
	parseFilterCase(t, filterCase{
		name:        "non-existent field",
		filterInput: []interface{}{"nonexistentfield", "=", "value"},
		expectError: true,
	})
}

func TestFilterTransactions_MismatchedOperators(t *testing.T) {
	parseFilterCase(t, filterCase{
		name: "more operators than predicates",
		filterInput: []interface{}{
			[]interface{}{"amount", "=", 100.0},
			"and",
			[]interface{}{"amount", "=", 200.0},
			"and",
		},
		expectError: true,
	})
}

func TestFilterTransactions_RealQuery(t *testing.T) {
	adapter, err := GetAdapter("transaction")
	if err != nil {
		t.Skipf("no transaction adapter: %v", err)
	}

	pred, err := ParseFilterToPredicates(adapter, []interface{}{"amount", "=", 100.0})
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if pred == nil {
		t.Fatal("expected non-nil predicate")
	}
}
