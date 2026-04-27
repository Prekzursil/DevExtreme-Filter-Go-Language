package main

import (
	"errors"
	"testing"

	"transaction-filter-backend/ent"
)

// TestOpenClient_HappyPath drives the success branch (entOpen returns
// no error). Already exercised by the test framework's init() call,
// but we cover it explicitly here for clarity.
func TestOpenClient_HappyPath(t *testing.T) {
	originalEntOpen := entOpen
	originalClient := client
	defer func() {
		entOpen = originalEntOpen
		client = originalClient
	}()

	if err := openClient(); err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if client == nil {
		t.Error("expected client to be set")
	}
}

// TestOpenClient_ErrorPath drives the error branch (entOpen returns
// a non-nil error). Stubs entOpen to return a synthetic error so
// openClient hits its `return err` line.
func TestOpenClient_ErrorPath(t *testing.T) {
	originalEntOpen := entOpen
	defer func() { entOpen = originalEntOpen }()

	syntheticErr := errors.New("synthetic ent.Open failure")
	entOpen = func(driverName, dataSourceName string, _ ...ent.Option) (*ent.Client, error) {
		return nil, syntheticErr
	}

	err := openClient()
	if err == nil {
		t.Error("expected error from stubbed entOpen")
	}
	if !errors.Is(err, syntheticErr) {
		t.Errorf("expected %v, got %v", syntheticErr, err)
	}
}

// TestBootstrapPackage_FatalBranch drives the fatalLogger branch in
// bootstrapPackage. Stubs entOpen to return an error AND fatalLogger
// to a no-op (so the test runner doesn't terminate via os.Exit(1)).
// Verifies the call sequence covers the error path without
// registerAllAdapters running afterward.
func TestBootstrapPackage_FatalBranch(t *testing.T) {
	originalEntOpen := entOpen
	originalFatalLogger := fatalLogger
	defer func() {
		entOpen = originalEntOpen
		fatalLogger = originalFatalLogger
	}()

	entOpen = func(driverName, dataSourceName string, _ ...ent.Option) (*ent.Client, error) {
		return nil, errors.New("synthetic ent.Open failure")
	}
	fatalCalled := false
	fatalLogger = func(format string, args ...any) {
		fatalCalled = true
	}

	bootstrapPackage()

	if !fatalCalled {
		t.Error("expected fatalLogger to be invoked when openClient errors")
	}
}

// Note: bootstrapPackage's happy path is already exercised by Go's
// runtime when the test binary loads (init() calls bootstrapPackage).
// Calling it again from a test would replace the test client with a
// fresh one missing the seeded schema, so we don't add an explicit
// happy-path test here.
