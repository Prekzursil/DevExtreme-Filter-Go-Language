package main

import (
	"errors"
	"net/http"
	"testing"

	"transaction-filter-backend/ent"
)

func TestStartServer(t *testing.T) {
	if client == nil {
		t.Skip("ent client not initialized")
	}
	origCount := defaultSeedCount
	defer func() { defaultSeedCount = origCount }()
	defaultSeedCount = 0

	handler, err := startServer()
	if err != nil {
		t.Fatalf("startServer error: %v", err)
	}
	if handler == nil {
		t.Fatal("startServer returned nil handler")
	}
}

func TestStartServer_PrepareFails(t *testing.T) {
	orig := client
	client = nil
	defer func() { client = orig }()

	if _, err := startServer(); err == nil {
		t.Error("expected startServer to propagate prepareServer error when client is nil")
	}
}

func TestRunMain_Success(t *testing.T) {
	if client == nil {
		t.Skip("ent client not initialized")
	}
	origCount := defaultSeedCount
	defer func() { defaultSeedCount = origCount }()
	defaultSeedCount = 0

	origLAS := listenAndServe
	defer func() { listenAndServe = origLAS }()
	listenAndServe = func(addr string, h http.Handler) error { return nil }

	origClose := closeClient
	defer func() { closeClient = origClose }()
	closeClient = func() {}

	if code := runMain(); code != 0 {
		t.Errorf("runMain returned %d, want 0", code)
	}
}

func TestRunMain_PrepareError(t *testing.T) {
	orig := client
	client = nil
	defer func() { client = orig }()

	origClose := closeClient
	defer func() { closeClient = origClose }()
	closeClient = func() {}

	if code := runMain(); code != 1 {
		t.Errorf("runMain returned %d, want 1 when client is nil", code)
	}
}

func TestRunMain_ListenerError(t *testing.T) {
	if client == nil {
		t.Skip("ent client not initialized")
	}
	origCount := defaultSeedCount
	defer func() { defaultSeedCount = origCount }()
	defaultSeedCount = 0

	origLAS := listenAndServe
	defer func() { listenAndServe = origLAS }()
	listenAndServe = func(addr string, h http.Handler) error {
		return errors.New("listener refused")
	}

	origClose := closeClient
	defer func() { closeClient = origClose }()
	closeClient = func() {}

	if code := runMain(); code != 1 {
		t.Errorf("runMain returned %d, want 1 when listener errors", code)
	}
}

func TestInitOrFatal_ErrorPath(t *testing.T) {
	origOpen := entOpen
	defer func() { entOpen = origOpen }()
	entOpen = func(driverName, dataSourceName string, options ...ent.Option) (*ent.Client, error) {
		return nil, errors.New("simulated ent.Open failure")
	}

	origFatal := fatalf
	defer func() { fatalf = origFatal }()
	var captured string
	fatalf = func(format string, args ...interface{}) {
		captured = format
	}

	origClient := client
	client = nil
	defer func() { client = origClient }()

	initOrFatal()

	if captured == "" {
		t.Error("expected fatalf to be called on ent.Open error")
	}
}

func TestCloseClient_NoopWhenNil(t *testing.T) {
	origClient := client
	client = nil
	defer func() { client = origClient }()

	// Call the default closeClient body (not the mocked no-op) to cover
	// its "client != nil" guard.
	closeClient()
}

func TestMain_ExitCode(t *testing.T) {
	if client == nil {
		t.Skip("ent client not initialized")
	}
	origCount := defaultSeedCount
	defer func() { defaultSeedCount = origCount }()
	defaultSeedCount = 0

	origLAS := listenAndServe
	defer func() { listenAndServe = origLAS }()
	listenAndServe = func(addr string, h http.Handler) error { return nil }

	origClose := closeClient
	defer func() { closeClient = origClose }()
	closeClient = func() {}

	origExit := osExit
	defer func() { osExit = origExit }()
	var captured int
	osExit = func(code int) { captured = code }

	main()

	if captured != 0 {
		t.Errorf("main() exit code=%d, want 0", captured)
	}
}
