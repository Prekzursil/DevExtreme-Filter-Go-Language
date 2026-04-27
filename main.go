// Package main is the HTTP entry point for the dynamic-table filter service.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
	"transaction-filter-backend/loghelper"
	"transaction-filter-backend/dynamictablefilter"
	"transaction-filter-backend/ent"
	"transaction-filter-backend/schematool"

	_ "transaction-filter-backend/ent/test1schema"
	_ "transaction-filter-backend/ent/test2schema"
	_ "transaction-filter-backend/ent/test3schema"

	"entgo.io/ent/dialect/sql" // Keep this for sql.Selector and potentially sql.P if needed elsewhere

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
			log.Printf("Warning: Failed to create generic adapter for %s: %v. This entity might not be filterable.", loghelper.Safe(entityName), errAdapter)
		} else {
			RegisterAdapter(entityName, adapter)
			log.Printf("Successfully registered generic adapter for entity: %s", loghelper.Safe(entityName))
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

type filterRequestBody struct {
	Entity string      `json:"entity"`
	Filter interface{} `json:"filter"`
}

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
		log.Printf("Backend: %v", err)
		http.Error(w, err.Error(), code)
		return
	}
	log.Printf("Backend: Decoded request for entity %q, filter: %+v", body.Entity, body.Filter)
	adapter, err := GetAdapter(body.Entity)
	if err != nil {
		log.Printf("Backend: Failed to get adapter for entity %q: %v", body.Entity, err)
		http.Error(w, fmt.Sprintf("No adapter for entity '%s'", body.Entity), http.StatusBadRequest)
		return
	}
	predicate, err := ParseFilterToPredicates(adapter, body.Filter)
	if err != nil {
		log.Printf("Backend: Error parsing filter for entity %q: %v", body.Entity, err)
		http.Error(w, fmt.Sprintf("Error parsing filter: %v", err), http.StatusInternalServerError)
		return
	}
	results, err := runFilterQuery(context.Background(), body.Entity, predicate)
	if err != nil {
		log.Printf("Backend: Error executing query for entity %q: %v", body.Entity, err)
		// Distinguish unsupported-entity (400) from real query errors (500)
		if strings.HasPrefix(err.Error(), "unsupported entity type:") {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		http.Error(w, fmt.Sprintf("Error executing query: %v", err), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if encErr := json.NewEncoder(w).Encode(results); encErr != nil {
		log.Printf("Backend: Error encoding response for entity %q: %v", body.Entity, encErr)
	}
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
	generateTransactions(100, ctx)
	generateTest1SchemaData(100, ctx)
	generateTest2SchemaData(100, ctx)
	generateTest3SchemaData(100, ctx)

	mux := http.NewServeMux()
	mux.HandleFunc("/filter", filterHandler)
	c := cors.New(cors.Options{
		AllowedOrigins: []string{"http://localhost:3000", "http://localhost:8080"},
		AllowedMethods: []string{"GET", "POST", "OPTIONS"},
		AllowedHeaders: []string{"Content-Type"},
	})
	handler := c.Handler(mux)

	mux.HandleFunc("/schema-editor", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "static/schema_editor.html")
	})
	mux.HandleFunc("/generate-schema-code", schematool.GenerateSchemaCodeHandler)
	mux.HandleFunc("/list-schema-definitions", schematool.ListSchemaDefinitionsHandler)
	mux.HandleFunc("/load-schema-definition", schematool.LoadSchemaDefinitionHandler)
	mux.HandleFunc("/list-filterable-entities", func(w http.ResponseWriter, r *http.Request) {
		entityNames := make([]string, 0, len(registeredAdapters))
		for name := range registeredAdapters {
			entityNames = append(entityNames, name)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(entityNames)
	})

	mux.HandleFunc("/dynamic-tables", func(w http.ResponseWriter, r *http.Request) {
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
	})
	mux.HandleFunc("/dynamic-tables/", func(w http.ResponseWriter, r *http.Request) {
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
			schema, err := dynamictablefilter.LoadTableSchema(tableName)
			if err != nil {
				log.Printf("Error loading schema for dynamic table %q: %v", tableName, err)
				if os.IsNotExist(err) {
					http.Error(w, "Schema not found for table "+tableName, http.StatusNotFound)
				} else {
					http.Error(w, "Failed to load schema for table "+tableName, http.StatusInternalServerError)
				}
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(schema)
			return
		}
		if len(pathParts) == 2 && pathParts[1] == "filter" && r.Method == http.MethodPost {
			var requestBody struct {
				Filter interface{} `json:"filter"`
			}
			decoder := json.NewDecoder(r.Body)
			if err := decoder.Decode(&requestBody); err != nil {
				log.Printf("Error decoding filter request for dynamic table %q: %v", tableName, err)
				http.Error(w, "Invalid request body", http.StatusBadRequest)
				return
			}
			schema, errSchema := dynamictablefilter.LoadTableSchema(tableName)
			if errSchema != nil {
				log.Printf("Error loading schema for dynamic table %q during filter: %v", tableName, errSchema)
				http.Error(w, "Schema not found for table "+tableName, http.StatusInternalServerError)
				return
			}
			tableData, errData := dynamictablefilter.LoadTableData(tableName)
			if errData != nil {
				log.Printf("Error loading data for dynamic table %q during filter: %v", tableName, errData)
				http.Error(w, "Data not found for table "+tableName, http.StatusInternalServerError)
				return
			}
			filteredData, errFilter := dynamictablefilter.FilterDynamicData(tableData, schema, requestBody.Filter)
			if errFilter != nil {
				log.Printf("Error filtering data for dynamic table %q: %v", tableName, errFilter)
				http.Error(w, "Error during filtering data for table "+tableName, http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(filteredData)
			return
		}
		http.NotFound(w, r)
	})

	reactAppFS := http.FileServer(http.Dir("./static/app"))
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("./static/app/static"))))
	mux.Handle("/manifest.json", reactAppFS)
	mux.Handle("/favicon.ico", reactAppFS)
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/filter") ||
			strings.HasPrefix(r.URL.Path, "/dynamic-tables") ||
			strings.HasPrefix(r.URL.Path, "/schema-editor") ||
			strings.HasPrefix(r.URL.Path, "/generate-schema-code") ||
			strings.HasPrefix(r.URL.Path, "/list-schema-definitions") ||
			strings.HasPrefix(r.URL.Path, "/load-schema-definition") ||
			strings.HasPrefix(r.URL.Path, "/list-filterable-entities") {
		}
		http.ServeFile(w, r, "./static/app/index.html")
	})

	fmt.Println("Go backend server listening on :8080")
	fmt.Println("React App (Filter UI) available at http://localhost:8080/")
	fmt.Println("Schema editor available at http://localhost:8080/schema-editor")
	log.Fatal(serveBackend(":8080", handler))
}

// serveBackend starts the backend HTTP(S) listener. If both
// ``BACKEND_TLS_CERT`` and ``BACKEND_TLS_KEY`` env vars are set, it uses
// ``ListenAndServeTLS`` (CWE-319 mitigation per Semgrep ``go.lang.security.audit.net.use-tls``).
// Otherwise it falls back to plaintext ``ListenAndServe`` for local dev.
// Production deployments should always set the TLS env vars or place a
// TLS-terminating reverse proxy in front of this binary.
func serveBackend(addr string, handler http.Handler) error {
	cert := os.Getenv("BACKEND_TLS_CERT")
	key := os.Getenv("BACKEND_TLS_KEY")
	if cert != "" && key != "" {
		return http.ListenAndServeTLS(addr, cert, key, handler)
	}
	return http.ListenAndServe(addr, handler) // nosemgrep: go.lang.security.audit.net.use-tls.use-tls — local-dev fallback path documented above
}
