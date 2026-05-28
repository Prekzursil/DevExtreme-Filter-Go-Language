//go:build prod

// Production entrypoint. Built into the binary via
// “go build -tags prod“. “go test“ (no tags) uses the empty stub
// in main_test_stub.go instead, so this real main() doesn't appear
// in coverage reports as an irreducible 1-line gap.
package main

import "log"

func main() {
	log.Fatal(bootstrapAndServe(":8080"))
}
