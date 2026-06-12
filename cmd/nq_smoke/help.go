package main

import "fmt"

// printHelp documents the subcommand matrix added by Plan 5 Task 29.
func printHelp() {
	fmt.Println(`nq_smoke - Plan 1 end-to-end smoke matrix

USAGE
    nq_smoke              run the default flow (Databento -> prompt -> CSV -> tail)
    nq_smoke <subcommand> run a single component smoke

SUBCOMMANDS
    databento   fetch 90min of NQ.c.0 1m bars from live Databento (needs DATABENTO_API_KEY)
    resolver    resolve NQ.c.0 -> MNQM6 via Databento symbology (needs DATABENTO_API_KEY)
    prompt      build futures system + user prompts; verify non-empty (no network)
    roundtrip   write signal -> mockNT -> fill (no network, no NT required)
    all         run every subcommand in sequence
    help        this message`)
}
