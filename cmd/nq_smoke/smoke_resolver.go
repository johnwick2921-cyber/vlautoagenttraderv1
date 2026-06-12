package main

import (
	"fmt"
	"os"
	"regexp"

	"github.com/joho/godotenv"

	"nofx/provider/databento"
)

// runResolverSmoke resolves the continuous symbol NQ.c.0 to today's specific
// CME contract code (e.g. NQM6 / NQU6 / NQZ6 / NQH7) via Databento symbology.
// Skips cleanly when DATABENTO_API_KEY is not set.
func runResolverSmoke() {
	_ = godotenv.Load() // .env from the working directory (run from the repo root)
	apiKey := os.Getenv("DATABENTO_API_KEY")
	if apiKey == "" {
		fmt.Println("SKIP resolver: DATABENTO_API_KEY not set")
		return
	}

	client := databento.NewClient("", apiKey)
	resolved, err := client.ResolveContinuous("NQ.c.0")
	if err != nil {
		fmt.Printf("FAIL resolver: %v\n", err)
		os.Exit(1)
	}

	// Expected CME format: 2-3 letters (root) + 1 month code (H/M/U/Z) + 1 digit (year).
	// Examples: NQM6, NQU6, NQZ6, NQH7, MNQM6.
	pattern := regexp.MustCompile(`^[A-Z]{2,3}[HMUZ]\d$`)
	if !pattern.MatchString(resolved) {
		fmt.Printf("FAIL resolver: unexpected contract code %q (want pattern like NQM6)\n", resolved)
		os.Exit(1)
	}

	fmt.Printf("OK resolver: NQ.c.0 -> %s\n", resolved)
}
