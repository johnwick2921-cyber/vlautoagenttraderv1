package store

import (
	"path/filepath"
	"testing"
)

func newPictureHtfStore(t *testing.T) *Store {
	t.Helper()
	st, err := New(filepath.Join(t.TempDir(), "picture-htf.db"))
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	if err := st.MigratePictureHtf(); err != nil {
		t.Fatal(err)
	}
	return st
}

func TestPictureHtfClaimDuplicateKeyRefuses(t *testing.T) {
	st := newPictureHtfStore(t)
	key := PictureHtfOppKey("strat-a", "Sim101", "MNQ 12-26", "long", "resistance", 1789000000000, 1789100000000)
	row := &PictureHtfOpportunityDB{
		OppKey: key, TraderID: "trader-1", StrategyID: "strat-a", Account: "Sim101",
		Contract: "MNQ 12-26", Symbol: "MNQ", Direction: "long", Stage: "watching",
	}
	got, ok, err := st.PictureHtfClaim(row)
	if err != nil || !ok || got == nil {
		t.Fatalf("first claim must win: ok=%v err=%v", ok, err)
	}
	// Repeated frame / restart: the same key can never produce a second row.
	if _, ok, err := st.PictureHtfClaim(&PictureHtfOpportunityDB{OppKey: key, TraderID: "trader-1"}); err != nil || ok {
		t.Fatalf("duplicate key must refuse: ok=%v err=%v", ok, err)
	}
	// A different H1 close time is a DIFFERENT opportunity.
	key2 := PictureHtfOppKey("strat-a", "Sim101", "MNQ 12-26", "long", "resistance", 1789000000000, 1789100000001)
	if _, ok, err := st.PictureHtfClaim(&PictureHtfOpportunityDB{OppKey: key2, TraderID: "trader-1"}); err != nil || !ok {
		t.Fatalf("a new H1 close is a new opportunity: ok=%v err=%v", ok, err)
	}
}

func TestPictureHtfLifecycleAndBrokerEvidence(t *testing.T) {
	st := newPictureHtfStore(t)
	key := PictureHtfOppKey("strat-a", "Sim101", "MNQ 12-26", "short", "support", 1789000000000, 1789100000000)
	if _, ok, err := st.PictureHtfClaim(&PictureHtfOpportunityDB{OppKey: key, TraderID: "trader-1", Stage: "watching"}); err != nil || !ok {
		t.Fatalf("claim: %v", err)
	}
	if err := st.PictureHtfTransition(key, "confirmed", "h1 close below support by one tick"); err != nil {
		t.Fatalf("transition: %v", err)
	}
	if err := st.PictureHtfTransition(key, "place_pending", "entry window 0.8s, freshness 1.2s"); err != nil {
		t.Fatalf("transition: %v", err)
	}
	// Only RECEIVED broker evidence may move to filled — a fill may arrive
	// without an earlier working frame.
	if err := st.PictureHtfMarkBroker(key, "filled", "nt8-123", "filled", "", 29676.25, 0.594); err != nil {
		t.Fatalf("mark broker: %v", err)
	}
	row, ok, err := st.PictureHtfGet(key)
	if err != nil || !ok {
		t.Fatalf("get: ok=%v err=%v", ok, err)
	}
	if row.Stage != "filled" || row.FillPrice != 29676.25 || row.BrokerOrderID != "nt8-123" {
		t.Fatalf("broker evidence lost: %+v", row)
	}
	// Rejected frames keep the reason; absent reasons stay empty.
	if err := st.PictureHtfMarkBroker(key, "rejected", "nt8-123", "rejected", "reason unavailable", 0, 0); err != nil {
		t.Fatalf("mark rejected: %v", err)
	}
	row, _, _ = st.PictureHtfGet(key)
	if row.Stage != "rejected" || row.RejectReason != "reason unavailable" {
		t.Fatalf("rejected evidence wrong: %+v", row)
	}
}

func TestPictureHtfPendingByTrader(t *testing.T) {
	st := newPictureHtfStore(t)
	key := PictureHtfOppKey("strat-a", "Sim101", "MNQ 12-26", "long", "resistance", 1789000000000, 1789100000000)
	if _, ok, err := st.PictureHtfClaim(&PictureHtfOpportunityDB{OppKey: key, TraderID: "trader-1", Stage: "place_pending"}); err != nil || !ok {
		t.Fatalf("claim: %v", err)
	}
	pending, err := st.PictureHtfPendingByTrader("trader-1")
	if err != nil || len(pending) != 1 {
		t.Fatalf("pending sweep must find the ambiguous row: n=%d err=%v", len(pending), err)
	}
}

// The atomic submission claim: exactly ONE caller across both executors and
// concurrent callbacks may move confirmed → place_pending and register its
// signal. The losers must not send.
func TestPictureHtfClaimSubmissionExactlyOneWinner(t *testing.T) {
	st := newPictureHtfStore(t)
	key := PictureHtfOppKey("strat-a", "Sim101", "MNQ 12-26", "long", "resistance", 1789000000000, 1789100000000)
	if _, ok, err := st.PictureHtfClaim(&PictureHtfOpportunityDB{OppKey: key, TraderID: "trader-1", Stage: "confirmed"}); err != nil || !ok {
		t.Fatalf("claim: %v", err)
	}
	const n = 12
	type result struct {
		id  int
		won bool
	}
	results := make(chan result, n)
	for i := 0; i < n; i++ {
		go func(id int) {
			won, err := st.PictureHtfClaimSubmission(key, "sig-"+string(rune('a'+id)))
			if err != nil {
				results <- result{id: id, won: false}
				return
			}
			results <- result{id: id, won: won}
		}(i)
	}
	winners := 0
	for i := 0; i < n; i++ {
		r := <-results
		if r.won {
			winners++
		}
	}
	if winners != 1 {
		t.Fatalf("exactly one submission owner may win; got %d of %d", winners, n)
	}
	row, _, _ := st.PictureHtfGet(key)
	if row.Stage != "place_pending" || row.SignalID == "" {
		t.Fatalf("the winning signal must be registered: %+v", row)
	}
	// A second attempt (e.g. another callback, or a restart) must NOT re-win
	// while the row is place_pending — the ambiguous send blocks re-entry.
	if won, err := st.PictureHtfClaimSubmission(key, "sig-retry"); err != nil || won {
		t.Fatalf("place_pending must block a second submission until reconciled: won=%v err=%v", won, err)
	}
	// After the broker frame establishes filled, a LATER opportunity with the
	// SAME key is still blocked (the key is durable).
	if err := st.PictureHtfMarkBroker(key, "filled", "nt8-1", "filled", "", 29676.25, 1); err != nil {
		t.Fatalf("mark broker: %v", err)
	}
	if won, _ := st.PictureHtfClaimSubmission(key, "sig-again"); won {
		t.Fatalf("a filled opportunity must never re-submit")
	}
}

func TestPictureHtfStampSignalOwnership(t *testing.T) {
	st := newPictureHtfStore(t)
	row := &PictureHtfOpportunityDB{OppKey: "stamp|1", TraderID: "t1", Stage: "confirmed", Direction: "long"}
	if _, fresh, err := st.PictureHtfClaim(row); err != nil || !fresh {
		t.Fatalf("claim: %v fresh=%v", err, fresh)
	}
	if won, err := st.PictureHtfClaimSubmission(row.OppKey, "claim-sig"); err != nil || !won {
		t.Fatalf("submission ownership: won=%v err=%v", won, err)
	}
	// The owner stamps the broker signal under its claim marker.
	if err := st.PictureHtfStampSignal(row.OppKey, "claim-sig", "nt8-uuid-1"); err != nil {
		t.Fatalf("the atomic owner must be able to stamp: %v", err)
	}
	got, ok, _ := st.PictureHtfGet(row.OppKey)
	if !ok || got.SignalID != "nt8-uuid-1" || got.SubmittedAt == 0 {
		t.Fatalf("stamp must persist the broker signal + send clock: %+v", got)
	}
	// A non-owner (wrong claim marker) is refused — never silently restamped.
	if err := st.PictureHtfStampSignal(row.OppKey, "someone-elses-claim", "nt8-uuid-2"); err == nil {
		t.Fatalf("a non-owner must be refused")
	}
	got, _, _ = st.PictureHtfGet(row.OppKey)
	if got.SignalID != "nt8-uuid-1" {
		t.Fatalf("a refused stamp must not change the row: %+v", got)
	}
}
