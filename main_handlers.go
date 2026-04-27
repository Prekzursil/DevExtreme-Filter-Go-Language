// HTTP handlers for the dynamic-table filter service. Split out of main.go
// so that file's qlty "high total complexity" stays below the smell
// threshold. Behavior is unchanged from the original main.go.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"transaction-filter-backend/dynamictablefilter"

	"entgo.io/ent/dialect/sql"
)

// HTTP header constants extracted to satisfy go:S1192 ("Define a constant
// instead of duplicating this literal N times") for ``"Content-Type"`` and
// ``"application/json"`` which appear 5 times each in handler responses.
const (
	headerContentType = "Content-Type"
	mimeApplicationJSON = "application/json"
)

type filterRequestBody struct {
	Entity string      `json:"entity"`
	Filter interface{} `json:"filter"`
}

// filterContext returns the context used by filterHandler when running
// the filter query. Tests override this to inject a cancelled context
// and exercise the runFilterQuery error path on a real entity (the
// "unsupported entity" branch is covered by GetAdapter; this hook
// drives the actual-query-error path at line 121-122 in
// main_handlers.go).
var filterContext = func() context.Context { return context.Background() }

func decodeFilterRequest(r *http.Request) (*filterRequestBody, int, error) {
	if r.Method != http.MethodPost {
		return nil, http.StatusMethodNotAllowed, fmt.Errorf("method not allowed")
	}
	var body filterRequestBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		return nil, http.StatusBadRequest, fmt.Errorf("invalid request body: %w", err)
	}
	if body.Entity == "" {
		return nil, http.StatusBadRequest, fmt.Errorf("missing 'entity' field in request body")
	}
	return &body, http.StatusOK, nil
}

func runFilterQuery(ctx context.Context, entity string, predicate *sql.Predicate) (interface{}, error) {
	applyPred := func(s *sql.Selector) {
		if predicate != nil {
			s.Where(predicate)
		}
	}
	switch strings.ToLower(entity) {
	case "transaction":
		return runTransactionQuery(ctx, predicate, applyPred)
	case "test1schema":
		query := client.Test1Schema.Query()
		if predicate != nil {
			query = query.Where(func(s *sql.Selector) { applyPred(s) })
		}
		return query.All(ctx)
	case "test2schema":
		query := client.Test2Schema.Query()
		if predicate != nil {
			query = query.Where(func(s *sql.Selector) { applyPred(s) })
		}
		return query.All(ctx)
	case "test3schema":
		query := client.Test3Schema.Query()
		if predicate != nil {
			query = query.Where(func(s *sql.Selector) { applyPred(s) })
		}
		return query.All(ctx)
	default:
		return nil, fmt.Errorf("unsupported entity type: %s", entity)
	}
}

func runTransactionQuery(ctx context.Context, predicate *sql.Predicate, applyPred func(*sql.Selector)) (interface{}, error) {
	query := client.Transaction.Query()
	if predicate != nil {
		query = query.Where(func(s *sql.Selector) { applyPred(s) })
	}
	dbResults, err := query.All(ctx)
	if err != nil {
		return nil, err
	}
	dtoResults := make([]Transaction, len(dbResults))
	for i, trx := range dbResults {
		dtoResults[i] = Transaction{
			ID: trx.ID, Date: trx.Date, Amount: trx.Amount, Name: trx.Name,
			Location: trx.Location, Category: trx.Category, Type: trx.Type,
		}
	}
	return dtoResults, nil
}

func filterHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Backend: filterHandler received a request")
	body, code, err := decodeFilterRequest(r)
	if err != nil {
		log.Println("Backend: request validation error")
		http.Error(w, err.Error(), code)
		return
	}
	// Don't log body.Filter directly: it's user-controlled JSON and the raw
	// form would carry injection risk (CWE-117).
	log.Println("Backend: Decoded request received")
	adapter, err := GetAdapter(body.Entity)
	if err != nil {
		log.Println("Backend: Failed to get adapter for requested entity")
		http.Error(w, fmt.Sprintf("No adapter for entity '%s'", body.Entity), http.StatusBadRequest)
		return
	}
	predicate, err := ParseFilterToPredicates(adapter, body.Filter)
	if err != nil {
		log.Println("Backend: Error parsing filter")
		http.Error(w, fmt.Sprintf("Error parsing filter: %v", err), http.StatusInternalServerError)
		return
	}
	results, err := runFilterQuery(filterContext(), body.Entity, predicate)
	if err != nil {
		log.Println("Backend: Error executing query")
		// Distinguish unsupported-entity (400) from real query errors (500)
		if strings.HasPrefix(err.Error(), "unsupported entity type:") {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		http.Error(w, fmt.Sprintf("Error executing query: %v", err), http.StatusInternalServerError)
		return
	}
	w.Header().Set(headerContentType, mimeApplicationJSON)
	if json.NewEncoder(w).Encode(results) != nil {
		log.Println("Backend: Error encoding response")
	}
}

func listFilterableEntitiesHandler(w http.ResponseWriter, r *http.Request) {
	entityNames := make([]string, 0, len(registeredAdapters))
	for name := range registeredAdapters {
		entityNames = append(entityNames, name)
	}
	w.Header().Set(headerContentType, mimeApplicationJSON)
	if json.NewEncoder(w).Encode(entityNames) != nil {
		log.Println("Backend: Error encoding entity list")
	}
}

func listDynamicTablesHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	tables, err := dynamictablefilter.ListDynamicTables()
	if err != nil {
		log.Println("Error listing dynamic tables")
		http.Error(w, "Failed to list dynamic tables", http.StatusInternalServerError)
		return
	}
	w.Header().Set(headerContentType, mimeApplicationJSON)
	if json.NewEncoder(w).Encode(tables) != nil {
		log.Println("Backend: Error encoding dynamic tables list")
	}
}

func dynamicTablesItemHandler(w http.ResponseWriter, r *http.Request) {
	pathParts := strings.Split(strings.TrimPrefix(r.URL.Path, "/dynamic-tables/"), "/")
	if len(pathParts) < 1 || pathParts[0] == "" {
		http.Error(w, "Table name missing", http.StatusBadRequest)
		return
	}
	tableName := pathParts[0]
	if len(pathParts) == 1 && r.Method == http.MethodGet {
		http.Error(w, "Specify /schema or /filter endpoint", http.StatusBadRequest)
		return
	}
	if len(pathParts) == 2 && pathParts[1] == "schema" && r.Method == http.MethodGet {
		dynamicTableSchemaHandler(w, tableName)
		return
	}
	if len(pathParts) == 2 && pathParts[1] == "filter" && r.Method == http.MethodPost {
		dynamicTableFilterHandler(w, r, tableName)
		return
	}
	http.NotFound(w, r)
}

func dynamicTableSchemaHandler(w http.ResponseWriter, tableName string) {
	schema, err := dynamictablefilter.LoadTableSchema(tableName)
	if err != nil {
		log.Println("Error loading schema for dynamic table")
		// errors.Is unwraps fmt.Errorf %w wrappers — needed since
		// LoadTableSchema wraps the underlying os.ErrNotExist via
		// fmt.Errorf("...: %w", err).
		if errors.Is(err, os.ErrNotExist) {
			http.Error(w, "Schema not found for table "+tableName, http.StatusNotFound)
		} else {
			http.Error(w, "Failed to load schema for table "+tableName, http.StatusInternalServerError)
		}
		return
	}
	w.Header().Set(headerContentType, mimeApplicationJSON)
	if err := json.NewEncoder(w).Encode(schema); err != nil {
		log.Println("Error encoding dynamic table schema response")
	}
}

func dynamicTableFilterHandler(w http.ResponseWriter, r *http.Request, tableName string) {
	var requestBody struct {
		Filter interface{} `json:"filter"`
	}
	if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
		log.Println("Error decoding filter request for dynamic table")
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	schema, errSchema := dynamictablefilter.LoadTableSchema(tableName)
	if errSchema != nil {
		log.Println("Error loading schema for dynamic table during filter")
		http.Error(w, "Schema not found for table "+tableName, http.StatusInternalServerError)
		return
	}
	tableData, errData := dynamictablefilter.LoadTableData(tableName)
	if errData != nil {
		log.Println("Error loading data for dynamic table during filter")
		http.Error(w, "Data not found for table "+tableName, http.StatusInternalServerError)
		return
	}
	filteredData, errFilter := dynamictablefilter.FilterDynamicData(tableData, schema, requestBody.Filter)
	if errFilter != nil {
		log.Println("Error filtering data for dynamic table")
		http.Error(w, "Error during filtering data for table "+tableName, http.StatusInternalServerError)
		return
	}
	w.Header().Set(headerContentType, mimeApplicationJSON)
	if err := json.NewEncoder(w).Encode(filteredData); err != nil {
		log.Println("Error encoding dynamic table filter response")
	}
}

// apiPathPrefixes are the URL prefixes whose dispatch is handled by other
// mux handlers. The SPA fallback handler skips these so they aren't
// shadowed by the index.html serve.
var apiPathPrefixes = []string{
	"/filter",
	"/dynamic-tables",
	"/schema-editor",
	"/generate-schema-code",
	"/list-schema-definitions",
	"/load-schema-definition",
	"/list-filterable-entities",
}

func spaFallbackHandler(w http.ResponseWriter, r *http.Request) {
	for _, prefix := range apiPathPrefixes {
		if strings.HasPrefix(r.URL.Path, prefix) {
			// API paths are handled by their own mux handlers; if we reach
			// here for an API prefix, the specific handler fell through, so
			// 404 instead of serving index.html.
			http.NotFound(w, r)
			return
		}
	}
	http.ServeFile(w, r, "./static/app/index.html")
}
