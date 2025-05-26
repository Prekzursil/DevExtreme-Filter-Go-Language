# DevExtreme Filter Go Language Backend

This project provides a Go backend with a dynamic filtering API designed to work with DevExtreme components, enabling a rich filtering user experience. It supports filtering data from two types of sources:
1.  **`ent`-backed entities:** Database tables managed by the `ent` ORM (using SQLite in-memory).
2.  **File-based dynamic tables:** Tables whose schema and data are defined by JSON files on the filesystem.

## Features

- **Go Backend:**
    - Serves a React application (built static assets) as the main UI.
    - API endpoint (`/filter`) for filtering `ent`-backed entities.
    - API endpoints (`/dynamic-tables/...`) for listing, loading schemas, and filtering file-based dynamic tables.
    - Supports complex, nested DevExtreme filter array syntax for both types of tables.
    - Includes pre-defined `ent` entities: `Transaction`, `Test1Schema`, `Test2Schema`, `Test3Schema`, each populated with 100 sample records on startup.
    - Includes sample file-based dynamic tables: `test1`, `test2`, `test3` (located in the `tables/` directory).
    - CORS configured for `localhost:3000` (React dev server) and `localhost:8080` (Go server).
- **React Frontend (DevExtreme based - static assets included):**
    - Dynamically lists all available entities/tables from the backend.
    - Dynamically configures DevExtreme `FilterBuilder` and `DataGrid` based on the selected entity's schema.
    - Provides a rich UI for building complex, nested filter criteria.
    - Displays filtered data in a grid.
- **Developer Schema Editor Tool:**
    - A separate web tool (`/schema-editor`) to help developers generate Go code for new `ent` schemas and their corresponding adapter templates.

## Project Structure (`transaction-filter-backend/`)

- `main.go`: Main application, HTTP handlers, data generation.
- `filterutils.go`: Core filter parsing logic for `ent` entities, adapter interface, helper functions.
- `generic_ent_adapter.go`: Provides a single, generic adapter for all `ent`-backed entities.
- `dynamictablefilter/`: Package for handling file-based dynamic tables (loading schema/data, in-memory filtering).
- `ent/`: Directory for `ent` ORM generated code and schema definitions (`ent/schema/`).
- `schema_definitions/`: JSON files representing schemas of `ent`-backed entities (used by UI and GenericEntAdapter).
- `tables/`: Directory containing subdirectories for file-based dynamic tables (e.g., `tables/test1/schema.json`, `tables/test1/data.json`).
- `static/app/`: Contains the **built static assets** of the React frontend application.
- `static/schema_editor.html`: UI for the developer schema editor tool.
- `schematool/`: Backend logic for the schema editor tool.
- `go.mod`, `go.sum`: Go module files.
- `Old Version/`: Contains the source code for the React frontend application (for reference or rebuilding).

## How to Use

### Prerequisites
- Go (version 1.24.3 or higher)
- Node.js and npm/yarn (only if you need to rebuild the React frontend from `Old Version/transaction-filter/`)

### Running the Application
1.  **Ensure React Frontend is Built and Copied (if necessary):**
    *   The Go backend expects the built static assets of the React frontend to be located in `static/app/`. These are included in the repository.
    *   If you need to rebuild or modify the React app:
        *   Navigate to `Old Version/transaction-filter/`.
        *   Run `npm install` (or `yarn install`).
        *   Run `npm run build` (or `yarn build`).
        *   Copy the contents of the build output directory (e.g., `Old Version/transaction-filter/build/`) into `static/app/`. **Ensure the `index.html` and static asset folders (like `static/css`, `static/js`) are directly under `static/app/`.**

2.  **Run the Go Backend:**
    *   Navigate to the `transaction-filter-backend/` directory (this project's root).
    *   Run the command: `go run main.go` (or `go run .`)
        *   For CGO enabled builds (SQLite):
            *   Windows: `$env:CGO_ENABLED='1'; go run main.go`
            *   Linux/macOS: `CGO_ENABLED=1 go run main.go`
    *   The server will start on `http://localhost:8080`.

3.  **Access the Application:**
    *   Open your web browser and go to `http://localhost:8080/`. This will load the React application.
    *   The developer schema editor tool is available at `http://localhost:8080/schema-editor`.

### Using the Filter UI (React App at `/`)
- Select an entity/table from the dropdown (e.g., "transaction (Ent)", "test1 (Dynamic)").
- The DevExtreme FilterBuilder will load with fields relevant to the selected entity.
- Construct your filter criteria using the UI.
- The DataGrid below will display the filtered data.

### Adding New `ent`-backed Entities
1.  Use the Schema Editor (`/schema-editor`) to generate the Go schema code for the new entity.
2.  Save the schema file to `ent/schema/`.
3.  Run `go generate ./...` in the `transaction-filter-backend/` directory.
4.  Create a JSON schema definition for the new entity in `schema_definitions/` (e.g., `newentity.json`).
5.  In `main.go`:
    *   Add a data generation function and call it in `main()`.
    *   Add the new entity's name to `entitiesToRegister` in `init()`.
    *   Add a `case` for the new entity in the `filterHandler`'s `switch` statement.
6.  Restart the Go server.

### Adding New File-Based Dynamic Tables
1.  Create a new subdirectory in `tables/` (e.g., `tables/mynewdynamictable/`).
2.  Inside this new directory, create `schema.json` and `data.json`.
3.  Restart the Go server. The new table should appear in the dropdown.

## Technologies Used
- Go 1.24.3
- Ent ORM (`entgo.io/ent`)
- SQLite (via `github.com/mattn/go-sqlite3`)
- CORS (`github.com/rs/cors`)
- React (frontend, static assets served by Go)
- DevExtreme (for UI components in the React frontend)

## Contributing

Feel free to fork the project and submit pull requests. For major changes, please open an issue first to discuss what you would like to change.

## License

This project is unlicensed.
