package main

import "fmt"

// runAllSmokes runs each subcommand in sequence. Each smoke handles its own
// exit semantics — a hard failure inside a smoke will os.Exit(1) and skip the
// remainder. SKIP behaviour (e.g. missing DATABENTO_API_KEY) is non-fatal.
func runAllSmokes() {
	smokes := []struct {
		name string
		fn   func()
	}{
		{"databento", runDatabentoSmoke},
		{"resolver", runResolverSmoke},
		{"prompt", runPromptSmoke},
		{"roundtrip", runRoundtripSmoke},
	}
	ran := make([]string, 0, len(smokes))
	for _, s := range smokes {
		fmt.Printf("\n=== %s ===\n", s.name)
		s.fn()
		ran = append(ran, s.name)
	}
	fmt.Printf("\n=== summary ===\n")
	for _, r := range ran {
		fmt.Printf("  ran: %s\n", r)
	}
}
