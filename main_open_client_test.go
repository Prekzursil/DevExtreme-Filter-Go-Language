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
