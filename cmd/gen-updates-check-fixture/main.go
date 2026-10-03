// Command gen-updates-check-fixture regenerates the web-side parity fixture
// web/src/lib/api/fixtures/updates-check.json from the SAME updatescheck
// builders the updates check handler answers with. Run it from the repo root:
//
//	go run ./cmd/gen-updates-check-fixture
//
// It is never run by `go test` — tests must not write into the source tree
// (kernel/srctree_write_guard_test.go). The api/ parity test only COMPARES
// against the checked-in file and fails with this command on drift.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"vl/internal/updatescheck"
)

func main() {
	b, err := json.MarshalIndent(updatescheck.FixturePair(), "", "  ")
	if err != nil {
		fmt.Fprintln(os.Stderr, "gen-updates-check-fixture:", err)
		os.Exit(1)
	}
	b = append(b, '\n')

	path := filepath.Join("web", "src", "lib", "api", "fixtures", "updates-check.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		fmt.Fprintln(os.Stderr, "gen-updates-check-fixture:", err)
		os.Exit(1)
	}
	if err := os.WriteFile(path, b, 0o644); err != nil {
		fmt.Fprintln(os.Stderr, "gen-updates-check-fixture:", err)
		os.Exit(1)
	}
	fmt.Println("wrote", path)
}
