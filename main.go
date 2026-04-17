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
	"time"
	"transaction-filter-backend/dynamictablefilter"
	"transaction-filter-backend/ent"
	"transaction-filter-backend/schematool"

	_ "transaction-filter-backend/ent/test1schema"
	_ "transaction-filter-backend/ent/test2schema"
	_ "transaction-filter-backend/ent/test3schema"

	"entgo.io/ent/dialect/sql"

	_ "github.com/mattn/go-sqlite3"
	"github.com/rs/cors"
)

var client *ent.Client

func init() {
	var err error
	client, err = ent.Open("sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	if err != nil {
		log.Fatalf("failed opening connection to sqlite: %v", err)
	}

	entitiesToRegister := []string{"transaction", "test1schema", "test2schema", "test3schema"}
	for _, entityName := range entitiesToRegister {
		adapter, errAdapter := NewGenericEntAdapter(entityName)
		if errAdapter != nil {
			log.Printf("Warning: Failed to create generic adapter for %s: %v. This entity might not be filterable.", entityName, errAdapter)
		} else {
			RegisterAdapter(entityName, adapter)
			log.Printf("Successfully registered generic adapter for entity: %s", entityName)
		}
	}
}

type Transaction struct {
	ID       int       `json:"id"`
	Date     time.Time `json:"date"`
	Amount   float64   `json:"amount"`
	Name     string    `json:"name"`
	Location string    `json:"location"`
	Category string    `json:"category"`
	Type     string    `json:"type"`
}

func generateTransactions(count int, ctx context.Context) {
	locations := []string{"New York", "Los Angeles", "Chicago", "Houston", "Phoenix", "Philadelphia"}
	categories := []string{"Groceries", "Dining", "Food & Drink", "Income", "Shopping", "Bills", "Transportation", "Entertainment", "Housing", "Health"}
	types := []string{"Debit", "Credit"}
	for i := 0; i < count; i++ {
		baseDay := time.Now().AddDate(0, 0, -i)
		transactionDate := time.Date(baseDay.Year(), baseDay.Month(), baseDay.Day(), i%24, (i*13)%60, (i*7)%60, 0, time.UTC)
		client.Transaction.Create().
			SetAmount(float64((i+1)*10) + float64(i%10)*0.5).
			SetDate(transactionDate).
			SetName(fmt.Sprintf("Transaction %d", i+1)).
			SetLocation(locations[i%len(locations)]).
			SetCategory(categories[i%len(categories)]).
			SetType(types[i%len(types)]).
			SaveX(ctx)
	}
	log.Printf("Generated %d transactions", count)
}

func generateTest1SchemaData(count int, ctx context.Context) {
	for i := 0; i < count; i++ {
		client.Test1Schema.Create().
			SetFieldString(fmt.Sprintf("T1 String %d", i)).
			SetFieldInt(i * 100).
			SetFieldFloat(float64(i*10) + 0.55).
			SetFieldBool(i%2 == 0).
			SetFieldTime(time.Now().AddDate(0, -(i % 12), -(i % 28))).
			SetFieldText(fmt.Sprintf("This is some longer text for Test1Schema item #%d. It can contain multiple sentences.", i)).
			SaveX(ctx)
	}
	log.Printf("Generated %d Test1Schema records", count)
}

func generateTest2SchemaData(count int, ctx context.Context) {
	itemTypes := []string{"Gadget", "Widget", "Accessory", "Component", "Tool"}
	for i := 0; i < count; i++ {
		client.Test2Schema.Create().
			SetName(fmt.Sprintf("Item %c%d", 'A'+(i%26), i)).
			SetDescription(fmt.Sprintf("Detailed description of Item %c%d. Quality assured.", 'A'+(i%26), i)).
			SetQuantity(10 + (i * 3 % 50)).
			SetPrice(float64(20+(i*7%100)) + float64(i%100)/100.0).
			SetActive((i+1)%3 != 0).
			SetCreatedAt(time.Now().AddDate(0, 0, -(i * 2))).
			SetUpdatedAt(time.Now().AddDate(0, 0, -i)).
			SetItemType(itemTypes[i%len(itemTypes)]).
			SaveX(ctx)
	}
	log.Printf("Generated %d Test2Schema records", count)
}

func generateTest3SchemaData(count int, ctx context.Context) {
	tagOptions := [][]string{
		{"tech", "new", "featured"}, {"books", "classic"}, {"apparel", "sale", "cotton"},
		{"home", "decor"}, {"sports", "outdoor", "gear"},
	}
	for i := 0; i < count; i++ {
		client.Test3Schema.Create().
			SetSku(fmt.Sprintf("SKU-%04d-%c", i, 'A'+(i%26))).
			SetProductName(fmt.Sprintf("Complex Product %d", i)).
			SetShortDescription(fmt.Sprintf("Brief overview of CP%d.", i)).
			SetFullDescription(fmt.Sprintf("Extended narrative for Complex Product %d, detailing its features, benefits, and specifications. Built for performance and durability.", i)).
			SetCostPrice(float64(50+(i*12%200)) + float64(i%100)/100.0).
			SetRetailPrice(float64(100+(i*18%300)) + float64(i%100)/100.0).
			SetStockCount(50 + (i * 5 % 150)).
			SetIsActive((i)%5 != 0).
			SetPublishedAt(time.Now().AddDate(0, 0, -(i*3 + 5))).
			SetLastOrderedAt(time.Now().AddDate(0, 0, -(i*5 + 2))).
			SetTags(strings.Join(tagOptions[i%len(tagOptions)], ", ")).
			SaveX(ctx)
	}
	log.Printf("Generated %d Test3Schema records", count)
}

type filterRequest struct {
	Entity string      `json:"entity"`
	Filter interface{} `json:"filter"`
}

func filterHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Backend: filterHandler received a request")
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	req, ok := decodeFilterRequest(w, r)
	if !ok {
		return
	}
	adapter, ok := resolveAdapter(w, req.Entity)
	if !ok {
		return
	}
	predicate, err := ParseFilterToPredicates(adapter, req.Filter)
	if err != nil {
		log.Print("Backend: Error parsing filter (details suppressed for log-injection guard)")
		http.Error(w, fmt.Sprintf("Error parsing filter: %v", err), http.StatusInternalServerError)
		return
	}
	results, err := runEntityQuery(context.Background(), req.Entity, predicate)
	if err != nil {
		log.Print("Backend: Error executing query (details suppressed for log-injection guard)")
		status := http.StatusInternalServerError
		if errors.Is(err, errUnsupportedEntity) {
			status = http.StatusBadRequest
		}
		http.Error(w, fmt.Sprintf("Error executing query: %v", err), status)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

func decodeFilterRequest(w http.ResponseWriter, r *http.Request) (*filterRequest, bool) {
	var req filterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("Backend: Error decoding request body: %v", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return nil, false
	}
	if req.Entity == "" {
		log.Print("Backend: Missing 'entity' field in request body")
		http.Error(w, "Missing 'entity' field in request body", http.StatusBadRequest)
		return nil, false
	}
	log.Print("Backend: Decoded filter request")
	return &req, true
}

func resolveAdapter(w http.ResponseWriter, entity string) (EntityAdapter, bool) {
	adapter, err := GetAdapter(entity)
	if err != nil {
		log.Print("Backend: Failed to resolve adapter (entity and error suppressed for log-injection guard)")
		http.Error(w, fmt.Sprintf("No adapter for entity '%s'", entity), http.StatusBadRequest)
		return nil, false
	}
	return adapter, true
}

var errUnsupportedEntity = fmt.Errorf("unsupported entity type")

func runEntityQuery(ctx context.Context, entity string, predicate PredicateFunc) (interface{}, error) {
	apply := func(s *sql.Selector) {
		if predicate != nil {
			s.Where(predicate)
		}
	}
	switch strings.ToLower(entity) {
	case "transaction":
		return queryTransactions(ctx, predicate, apply)
	case "test1schema":
		return queryTest1Schema(ctx, predicate, apply)
	case "test2schema":
		return queryTest2Schema(ctx, predicate, apply)
	case "test3schema":
		return queryTest3Schema(ctx, predicate, apply)
	}
	log.Print("Backend: Unsupported entity type for filtering (suppressed)")
	return nil, fmt.Errorf("%w: %s", errUnsupportedEntity, entity)
}

func queryTransactions(ctx context.Context, predicate PredicateFunc, apply func(*sql.Selector)) (interface{}, error) {
	query := client.Transaction.Query()
	if predicate != nil {
		query = query.Where(apply)
	}
	rows, err := query.All(ctx)
	if err != nil {
		return nil, err
	}
	dto := make([]Transaction, len(rows))
	for i, trx := range rows {
		dto[i] = Transaction{
			ID: trx.ID, Date: trx.Date, Amount: trx.Amount, Name: trx.Name,
			Location: trx.Location, Category: trx.Category, Type: trx.Type,
		}
	}
	return dto, nil
}

func queryTest1Schema(ctx context.Context, predicate PredicateFunc, apply func(*sql.Selector)) (interface{}, error) {
	query := client.Test1Schema.Query()
	if predicate != nil {
		query = query.Where(apply)
	}
	return query.All(ctx)
}

func queryTest2Schema(ctx context.Context, predicate PredicateFunc, apply func(*sql.Selector)) (interface{}, error) {
	query := client.Test2Schema.Query()
	if predicate != nil {
		query = query.Where(apply)
	}
	return query.All(ctx)
}

func queryTest3Schema(ctx context.Context, predicate PredicateFunc, apply func(*sql.Selector)) (interface{}, error) {
	query := client.Test3Schema.Query()
	if predicate != nil {
		query = query.Where(apply)
	}
	return query.All(ctx)
}

func schemaEditorHandler(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "static/schema_editor.html")
}

func listFilterableEntitiesHandler(w http.ResponseWriter, r *http.Request) {
	entityNames := make([]string, 0, len(registeredAdapters))
	for name := range registeredAdapters {
		entityNames = append(entityNames, name)
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(entityNames)
}

func listDynamicTablesHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	tables, err := dynamictablefilter.ListDynamicTables()
	if err != nil {
		log.Printf("Error listing dynamic tables: %v", err)
		http.Error(w, "Failed to list dynamic tables", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tables)
}

func dynamicTableHandler(w http.ResponseWriter, r *http.Request) {
	tableName, action, ok := parseDynamicTablePath(w, r)
	if !ok {
		return
	}
	if action == "" && r.Method == http.MethodGet {
		http.Error(w, "Specify /schema or /filter endpoint", http.StatusBadRequest)
		return
	}
	if action == "schema" && r.Method == http.MethodGet {
		handleDynamicSchema(w, tableName)
		return
	}
	if action == "filter" && r.Method == http.MethodPost {
		handleDynamicFilter(w, r, tableName)
		return
	}
	http.NotFound(w, r)
}

func parseDynamicTablePath(w http.ResponseWriter, r *http.Request) (tableName, action string, ok bool) {
	pathParts := strings.Split(strings.TrimPrefix(r.URL.Path, "/dynamic-tables/"), "/")
	if len(pathParts) < 1 || pathParts[0] == "" {
		http.Error(w, "Table name missing", http.StatusBadRequest)
		return "", "", false
	}
	if err := dynamictablefilter.ValidateTableName(pathParts[0]); err != nil {
		log.Printf("Error: dynamic table request with invalid name: %v", err)
		http.Error(w, "Invalid table name", http.StatusBadRequest)
		return "", "", false
	}
	tableName = pathParts[0]
	if len(pathParts) >= 2 {
		action = pathParts[1]
	}
	return tableName, action, true
}

func handleDynamicSchema(w http.ResponseWriter, tableName string) {
	schema, err := dynamictablefilter.LoadTableSchema(tableName)
	if err != nil {
		log.Print("Error loading schema for dynamic table (path validated, details suppressed)")
		if errors.Is(err, os.ErrNotExist) {
			http.Error(w, "Schema not found for table "+tableName, http.StatusNotFound)
		} else {
			http.Error(w, "Failed to load schema for table "+tableName, http.StatusInternalServerError)
		}
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(schema)
}

func handleDynamicFilter(w http.ResponseWriter, r *http.Request, tableName string) {
	var requestBody struct {
		Filter interface{} `json:"filter"`
	}
	if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
		log.Print("Error decoding filter request for dynamic table (path validated, details suppressed)")
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	schema, errSchema := dynamictablefilter.LoadTableSchema(tableName)
	if errSchema != nil {
		log.Print("Error loading schema during filter (path validated, details suppressed)")
		http.Error(w, "Schema not found for table "+tableName, http.StatusInternalServerError)
		return
	}
	tableData, errData := dynamictablefilter.LoadTableData(tableName)
	if errData != nil {
		log.Print("Error loading data during filter (path validated, details suppressed)")
		http.Error(w, "Data not found for table "+tableName, http.StatusInternalServerError)
		return
	}
	filteredData, errFilter := dynamictablefilter.FilterDynamicData(tableData, schema, requestBody.Filter)
	if errFilter != nil {
		log.Print("Error filtering data for dynamic table (path validated, filter error suppressed)")
		http.Error(w, "Error during filtering data for table "+tableName, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(filteredData)
}

func serveReactAppHandler(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "./static/app/index.html")
}

func buildCORSHandler() *cors.Cors {
	return cors.New(cors.Options{
		AllowedOrigins: []string{"http://localhost:3000", "http://localhost:8080"},
		AllowedMethods: []string{"GET", "POST", "OPTIONS"},
		AllowedHeaders: []string{"Content-Type"},
	})
}

func setupMux() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/filter", filterHandler)
	mux.HandleFunc("/schema-editor", schemaEditorHandler)
	mux.HandleFunc("/generate-schema-code", schematool.GenerateSchemaCodeHandler)
	mux.HandleFunc("/list-schema-definitions", schematool.ListSchemaDefinitionsHandler)
	mux.HandleFunc("/load-schema-definition", schematool.LoadSchemaDefinitionHandler)
	mux.HandleFunc("/list-filterable-entities", listFilterableEntitiesHandler)
	mux.HandleFunc("/dynamic-tables", listDynamicTablesHandler)
	mux.HandleFunc("/dynamic-tables/", dynamicTableHandler)

	reactAppFS := http.FileServer(http.Dir("./static/app"))
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("./static/app/static"))))
	mux.Handle("/manifest.json", reactAppFS)
	mux.Handle("/favicon.ico", reactAppFS)
	mux.HandleFunc("/", serveReactAppHandler)
	return mux
}

func seedData(ctx context.Context) {
	generateTransactions(100, ctx)
	generateTest1SchemaData(100, ctx)
	generateTest2SchemaData(100, ctx)
	generateTest3SchemaData(100, ctx)
}

func main() {
	ctx := context.Background()
	if client == nil {
		log.Fatal("Ent client failed to initialize")
	}
	defer client.Close()
	if err := client.Schema.Create(ctx); err != nil {
		log.Fatalf("failed creating schema resources: %v", err)
	}
	seedData(ctx)

	handler := buildCORSHandler().Handler(setupMux())

	fmt.Println("Go backend server listening on :8080")
	fmt.Println("React App (Filter UI) available at http://localhost:8080/")
	fmt.Println("Schema editor available at http://localhost:8080/schema-editor")
	log.Fatal(http.ListenAndServe(":8080", handler))
}
