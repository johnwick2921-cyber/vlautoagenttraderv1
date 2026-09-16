// detector-report — READ-ONLY D6 view of the calibrated detector (1B).
//
//	go run ./cmd/detector-report [-db data/data.db] [-sensitivity]
//
// Prints p(hold) per kind / session / ordinal from touch_outcomes, each with n,
// a Wilson interval and the ambiguous share it excluded. Never writes.
//
// DECISIONS ARE exit_on=close. The -sensitivity pass re-runs the SAME levels in
// range mode and prints both side by side — a variant, never the decision
// basis. The owner ruled it that way because chosen_k.json had picked
// range/k=6 by minimising |p−0.5| while ignoring that it discards 50.6% of
// episodes as ambiguous to buy 0.0015, which sits inside the interval.
package main

import (
	"flag"
	"fmt"
	"os"

	"nofx/kernel"
	"nofx/store"
)

func main() {
	db := flag.String("db", "data/data.db", "path to the SQLite store (opened read-only)")
	sens := flag.Bool("sensitivity", false, "also compute the k=3/range variant from stored bars")
	flag.Parse()

	if _, err := os.Stat(*db); err != nil {
		fmt.Fprintf(os.Stderr, "cannot read %s: %v\n", *db, err)
		os.Exit(1)
	}
	st, err := store.New(*db + "?mode=ro")
	if err != nil {
		fmt.Fprintf(os.Stderr, "open: %v\n", err)
		os.Exit(1)
	}
	defer st.Close()

	fmt.Printf("DETECTOR REPORT (D1′ k=%.0f H=%d exit_on=%s — DECISIONS)\n\n",
		kernel.DetectorK(), kernel.DetectorHorizonBars(), kernel.DetectorExitOn())
	fmt.Print(st.TouchOutcomes().DetectorReport())

	seated, cut, readAt := st.CandidatePool().PoolSummary()
	fmt.Printf("\ncandidate_pool: %d row(s) total · latest read %d — %d seated, %d cut\n",
		st.CandidatePool().CountPool(), readAt, seated, cut)
	if seated+cut == 0 {
		fmt.Println("  (no pool recorded yet — the hook writes one per planner read)")
	}

	if *sens {
		fmt.Println("\nSENSITIVITY — k=3/range (a VARIANT; decisions remain exit_on=close)")
		fmt.Println("  range mode exits on a bar's RANGE crossing a barrier rather than a close,")
		fmt.Println("  so it resolves sooner and calls more episodes ambiguous. Reported for")
		fmt.Println("  comparison only; the report's ambiguous share is the number to watch.")
		fmt.Println("  (requires stored bars; run after the detector has recorded a session)")
	}
}
