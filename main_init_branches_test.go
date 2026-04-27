package main

import (
	"testing"
)

// TestRegisterOneAdapter_HappyPath drives the "Successfully registered"
// branch (line 42 in main.go) for an entity that has a real schema file
// in schema_definitions/.
func TestRegisterOneAdapter_HappyPath(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("registerOneAdapter panicked: %v", r)
		}
	}()
	// "transaction" has a schema file in schema_definitions/ (created
	// during the project setup). NewGenericEntAdapter succeeds, the
	// happy-path branch runs.
	registerOneAdapter("transaction")
}

// TestRegisterOneAdapter_WarningPath drives the previously-unreachable
// "Warning: Failed to create generic adapter" branch by passing an
// entity name that has no schema_definitions/<name>.json file.
// NewGenericEntAdapter returns an error, the warning branch runs.
func TestRegisterOneAdapter_WarningPath(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("registerOneAdapter panicked on missing entity: %v", r)
		}
	}()
	registerOneAdapter("totally-nonexistent-entity-no-schema-file")
}

// TestRegisterAllAdapters_HappyPath exercises the loop dispatcher.
func TestRegisterAllAdapters_HappyPath(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("registerAllAdapters panicked: %v", r)
		}
	}()
	registerAllAdapters([]string{"transaction"})
}

// TestRegisterAllAdapters_MixedNames covers both happy and warning
// paths in a single call.
func TestRegisterAllAdapters_MixedNames(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("registerAllAdapters panicked: %v", r)
		}
	}()
	registerAllAdapters([]string{
		"transaction",
		"totally-nonexistent-entity-no-schema-file",
	})
}
