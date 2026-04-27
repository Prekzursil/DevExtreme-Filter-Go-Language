//go:build !prod

// Test stub for the runtime entrypoint. ``go test`` compiles this file
// (no build tags supplied) so the test binary has a ``main`` symbol to
// satisfy the linker — but the function body is empty, so it doesn't
// drag the per-package coverage total down with an unreachable
// log.Fatal(bootstrapAndServe(...)) line.
//
// The real entry is in main_prod.go (built via ``go build -tags prod``).
package main

func main() {}
