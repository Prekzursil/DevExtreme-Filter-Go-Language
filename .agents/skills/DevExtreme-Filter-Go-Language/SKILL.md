```markdown
# DevExtreme-Filter-Go-Language Development Patterns

> Auto-generated skill from repository analysis

## Overview
This skill covers the development patterns and conventions used in the `DevExtreme-Filter-Go-Language` repository. The codebase is written in Go, and implements logic for filtering, likely inspired by DevExtreme's filtering syntax, but adapted for Go. The repository follows clear coding conventions, a conventional commit style, and includes a recognizable testing pattern.

## Coding Conventions

### File Naming
- Use **snake_case** for all file names.
  - Example: `filter_parser.go`, `query_builder.go`

### Imports
- Use **relative import paths** within the module.
  - Example:
    ```go
    import (
        "fmt"
        "myproject/filter_utils"
    )
    ```

### Exports
- Use **named exports** for functions, types, and variables that should be accessible outside the package.
  - Example:
    ```go
    // Exported function
    func ParseFilter(input string) (Filter, error) {
        // implementation
    }
    ```

### Commit Messages
- Follow **conventional commit** style.
- Prefixes used: `fix`, `chore`
- Example:
  - `fix: handle nil pointer in filter evaluation`
  - `chore: update dependencies`

## Workflows

### Code Update and Commit
**Trigger:** When making any code changes.
**Command:** `/commit-update`

1. Make code changes following the coding conventions.
2. Write a descriptive commit message using the conventional commit style.
   - Example: `fix: correct filter parsing for nested arrays`
3. Run tests to ensure nothing is broken.
4. Commit and push your changes.

### Adding a New Filter Feature
**Trigger:** When implementing a new filter operation or logic.
**Command:** `/add-filter-feature`

1. Create a new file using snake_case if needed.
2. Implement the feature using named exports for any public functions.
3. Add or update tests in a corresponding `*.test.*` file.
4. Commit changes with a `fix:` or `chore:` prefix as appropriate.
5. Push your branch and open a pull request.

### Running Tests
**Trigger:** Before pushing changes or after implementing new features.
**Command:** `/run-tests`

1. Locate test files (pattern: `*.test.*`).
2. Use Go's testing tools to run the tests:
   ```sh
   go test ./...
   ```
3. Ensure all tests pass before committing.

## Testing Patterns

- Test files follow the pattern `*.test.*` (e.g., `filter_parser.test.go`).
- The specific testing framework is not identified, but standard Go testing practices likely apply.
- Example test function:
  ```go
  func TestParseFilter(t *testing.T) {
      result, err := ParseFilter("['field', '=', 1]")
      if err != nil {
          t.Fatalf("unexpected error: %v", err)
      }
      // assertions...
  }
  ```

## Commands
| Command             | Purpose                                         |
|---------------------|-------------------------------------------------|
| /commit-update      | Commit code changes following conventions        |
| /add-filter-feature | Add a new filter feature with tests              |
| /run-tests          | Run all test files before committing             |
```
