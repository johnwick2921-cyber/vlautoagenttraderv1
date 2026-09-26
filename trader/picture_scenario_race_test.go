package trader

import (
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"nofx/kernel"
	ntwire "nofx/provider/ninjatrader"
	"nofx/store"
	ntTrader "nofx/trader/ninjatrader"
)

// ── W-EXEC-TRUTH W5 (Builder B) — ONE ENTRY when a Picture scenario races ────
//
// Two producers, one account, one entry:
//
//	R1  a Picture scenario and a planner scenario in ONE armed pass → the
//	    in-pass latch (placedThisPass) lets exactly one frame out; the other
//	    row is refused and KEPT (D9 — never cancelled across sources)
//	R2  a Picture scenario's pass and an AI decision's send (plan_mode
//	    advisory — non-strict) released at the broker's entry permit at the
//	    same instant → the ONE entry latch lets exactly one frame out; the
//	    loser is latched
//
// R2's barrier is W0's raceAtPermit pattern, copied here (the shared
// duplicate-sequence file is W4's to change first — C3): the permit holds each
// caller until both have arrived, then releases them together, so the latch's
// per-key send mutex is what is tested, not goroutine scheduling. The latch
// reads the WALL clock (production wiring), so its books are seeded at the
// wall instant; the armed pass reads its pinned clock, where that book's age
// is negative — fresh — exactly as in the duplicate-sequence fixture.

// picRaceAtPermit is raceAtPermit (entry_duplicate_sequences_test.go), copied.
func picRaceAtPermit(t *testing.T, nt *ntTrader.TCPTrader) *atomic.Int32 {
	t.Helper()
	var arrived atomic.Int32
	release := make(chan struct{})
	nt.SetEntryPermit(func() (func(), bool) {
		if arrived.Add(1) == 2 {
			close(release)
		}
		select {
		case <-release:
		case <-time.After(20 * time.Second): // bound: a producer that never arrives cannot hang the test
		}
		return MaintenanceEntryPermit()
	})
	return &arrived
}

// picLatchRefusals is the entry latch's refusal count (the TCPTrader carries
// no trader id in these fixtures, so the latch counts under "").
func picLatchRefusals() int {
	n := 0
	for _, r := range []string{"book_unverifiable", "working_entry_or_position", "ledger_unreadable", "ledger_open", "queued_entry", "recent_send"} {
		n += gateBlocks("", "one_entry_latch:"+r)
	}
	return n
}

func picWaitBounded(t *testing.T, wg *sync.WaitGroup, d time.Duration) {
	t.Helper()
	done := make(chan struct{})
	go func() { wg.Wait(); close(done) }()
	select {
	case <-done:
	case <-time.After(d):
		t.Fatalf("the racing producers did not return within %s", d)
	}
}

// R1 — one pass, a Picture scenario and a planner scenario both inside their
// zones: ONE frame, whichever the pass reaches first; the other row stays
// armed (refused one_contract_placed_this_pass, counted — not cancelled).
func TestPictureAndPlannerScenarioInOnePassYieldOneFrame(t *testing.T) {
	for _, pictureFirst := range []bool{false, true} {
		name := "planner first"
		if pictureFirst {
			name = "picture first"
		}
		t.Run(name, func(t *testing.T) {
			r, epoch := newPicRig(t, "w5b-race-pass-"+strconv.FormatBool(pictureFirst), nil)
			s1 := plannerZone("S1", 99.5, 100.5, 100)
			p1 := picScenario("P1", "opp-race-pass", r.now, epoch, picDefault)
			if pictureFirst {
				picPlan(r, p1, s1)
			} else {
				picPlan(r, s1, p1)
			}
			before := gateBlocks(r.at.id, "arm_one_contract_placed_this_pass")
			picPass(r, 0, 100.25)
			sigs, _ := r.drain()
			if len(sigs) != 1 {
				t.Fatalf("one pass, two sources inside: exactly ONE frame, got %d: %+v", len(sigs), sigs)
			}
			limitOnly(t, sigs)
			first, second := r.row("S1"), r.row("P1")
			if pictureFirst {
				first, second = second, first
			}
			if first.SignalID != sigs[0].SignalID {
				t.Fatalf("the first row reached in the pass placed: %+v vs %+v", first, sigs[0])
			}
			if second.State != store.StateArmed || second.SignalID != "" {
				t.Fatalf("the other source's row is refused and KEPT (D9), never cancelled: %+v", second)
			}
			if gateBlocks(r.at.id, "arm_one_contract_placed_this_pass") != before+1 {
				t.Fatal("the in-pass refusal is counted")
			}
		})
	}
}

// R2 — a Picture scenario's pass and an AI decision (advisory, non-strict)
// meet at the broker's entry permit at the same instant → exactly ONE frame;
// the loser is refused by the entry latch (counted). Run several rounds: a
// latch without its per-key mutex lets both through in most of them.
func TestPictureScenarioAndAIDecisionRaceYieldOneFrame(t *testing.T) {
	stubOutboundHTTP(t) // the AI send half makes no outbound call; belt and braces
	for i := 0; i < 6; i++ {
		t.Run("round "+strconv.Itoa(i), func(t *testing.T) {
			withMaintenanceDir(t)
			r, epoch := newPicRig(t, "w5b-race-ai-"+strconv.Itoa(i), nil)
			picPlan(r, picScenario("P1", "opp-race-ai", r.now, epoch, picDefault))
			nt, ok := r.at.trader.(*ntTrader.TCPTrader)
			if !ok {
				t.Fatal("fixture: an NT8 TCP trader")
			}
			r.at.positionFirstSeenTime = map[string]int64{}
			r.at.config.NinjaTraderSymbol = "MNQ"
			// NewAutoTrader's TCPTrader wiring (both calls pinned there).
			wireNT8Maintenance(r.at, nt)
			wireNT8EntryLatch(r.at, nt)
			if !nt.EntryLatchWired() {
				t.Fatal("fixture: the latch must be wired as production wires it")
			}
			r.srv.OrderSnapshots().PutAt(ntwire.OrderSnapshotPayload{Account: "Sim101", Orders: []ntwire.NT8Order{}}, time.Now())
			r.setTape(zoneTape(100.25, r.now, 0))
			arrived := picRaceAtPermit(t, nt)
			before := picLatchRefusals()

			var wg sync.WaitGroup
			var aiErr error
			rec := &store.DecisionAction{Action: "open_long", Symbol: "MNQ"}
			wg.Add(2)
			go func() {
				defer wg.Done()
				d := &kernel.Decision{Action: "open_long", Symbol: "MNQ", Leverage: 1, Confidence: 70, StopLoss: 99, TakeProfit: 106}
				aiErr = r.at.executeOpenLongWithRecord(d, rec)
			}()
			go func() { defer wg.Done(); r.at.maybeManageArmedOrdersAt(nil, r.now) }()
			picWaitBounded(t, &wg, 30*time.Second)

			if n := arrived.Load(); n != 2 {
				t.Fatalf("fixture: both producers must reach the broker's entry permit, got %d", n)
			}
			sigs, _ := r.drain()
			if len(sigs) != 1 {
				t.Fatalf("Picture scenario ‖ AI decision: exactly ONE frame, got %d: %+v", len(sigs), sigs)
			}
			aiSent := aiErr == nil && rec.Error == ""
			picSent := r.row("P1").SignalID != ""
			if aiSent == picSent {
				t.Fatalf("exactly one producer may send (ai sent=%v err=%v rec=%q, picture sent=%v)", aiSent, aiErr, rec.Error, picSent)
			}
			if picSent {
				limitOnly(t, sigs)
				if !ntTrader.IsEntryLatched(aiErr) {
					t.Fatalf("the losing AI open must be refused by the entry latch: %v", aiErr)
				}
			} else if row := r.row("P1"); row.State != store.StateArmed {
				t.Fatalf("the losing Picture row stays armed (a refusal, never a send): %+v", row)
			}
			if picLatchRefusals() <= before {
				t.Fatal("the loser must have been refused by the entry latch (counted)")
			}
		})
	}
}
