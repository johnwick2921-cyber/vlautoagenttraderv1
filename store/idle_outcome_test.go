package store

import (
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// INSTRUMENT HONESTY (owner ruling 2026-09-03) — idle_before vs outcome.
//
// The 08:11:38 cut rode a connection idle 101,212ms; the resend that succeeded
// rode one idle 34,935ms. If cuts cluster above some idle threshold the fix is
// IdleConnTimeout below it — cheap, measurable, no provider change. THE OWNER
// HAS NOT SET IT: three more cuts decide. This table is what decides.
func TestIdleOutcomeTableBucketsWithN(t *testing.T) {
	st, err := New(filepath.Join(t.TempDir(), "t.db"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	ws := st.WatchdogFires()

	at := time.Date(2026, 9, 3, 8, 11, 38, 0, time.UTC)
	rec := func(kind string, idleMs int64, reused bool, closedBy string, resolved, resendOK bool) {
		id, err := ws.Record(WatchdogFireDB{
			TraderID: "t1", At: at, Kind: kind, IdleBeforeMs: idleMs, Reused: reused,
			ClosedBy: closedBy, CallAgeMs: 283425, Bytes: 50489,
		})
		if err != nil || id == 0 {
			t.Fatalf("record: %v id=%d", err, id)
		}
		if resolved {
			if _, err := ws.ResolveLatest("t1", resendOK, 243767, "note"); err != nil {
				t.Fatalf("resolve: %v", err)
			}
		}
	}
	// two long-idle cuts, one recovered, one not
	rec("cut", 101212, true, "peer_fin", true, true)
	rec("cut", 95000, true, "peer_fin", true, false)
	// a short-idle cut
	rec("cut", 12000, true, "peer_fin", true, true)
	// a fresh-connection watchdog fire — different kind, same table
	rec("watchdog", 0, false, "local_close", false, false)

	rows, err := ws.IdleOutcomeTable()
	if err != nil {
		t.Fatalf("table: %v", err)
	}
	byBucket := map[string]IdleOutcomeRow{}
	for _, r := range rows {
		byBucket[r.Bucket] = r
	}

	long, ok := byBucket["≥60s"]
	if !ok {
		t.Fatalf("no ≥60s bucket in %+v", rows)
	}
	if long.N != 2 {
		t.Errorf("≥60s n = %d, want 2", long.N)
	}
	if long.Cuts != 2 {
		t.Errorf("≥60s cuts = %d, want 2", long.Cuts)
	}
	if long.Recovered != 1 {
		t.Errorf("≥60s recovered = %d, want 1", long.Recovered)
	}

	fresh, ok := byBucket["fresh (not reused)"]
	if !ok {
		t.Fatalf("a fresh connection must get its own bucket, not fall into 0–30s: %+v", rows)
	}
	if fresh.N != 1 || fresh.Cuts != 0 {
		t.Errorf("fresh bucket n=%d cuts=%d, want 1 and 0 (it was a watchdog fire)", fresh.N, fresh.Cuts)
	}

	// The renderer states every n and refuses a rate without one.
	out := RenderIdleOutcomeTable(rows)
	if !strings.Contains(out, "n=") {
		t.Errorf("the table prints no n:\n%s", out)
	}
	if strings.Contains(out, "NaN") || strings.Contains(out, "%\n") && !strings.Contains(out, "of ") {
		t.Errorf("a rate appears without its denominator:\n%s", out)
	}
	// And it must say out loud that nothing is decided yet.
	if !strings.Contains(out, "no threshold is set") {
		t.Errorf("the table must not read as a decision:\n%s", out)
	}
}

// An unresolved row is unresolved — never counted as a failure to recover.
func TestIdleOutcomeUnresolvedIsNotAFailure(t *testing.T) {
	st, _ := New(filepath.Join(t.TempDir(), "t.db"))
	t.Cleanup(func() { _ = st.Close() })
	ws := st.WatchdogFires()
	if _, err := ws.Record(WatchdogFireDB{
		TraderID: "t1", At: time.Now().UTC(), Kind: "cut",
		IdleBeforeMs: 90000, Reused: true, ClosedBy: "peer_fin",
	}); err != nil {
		t.Fatalf("record: %v", err)
	}
	rows, _ := ws.IdleOutcomeTable()
	for _, r := range rows {
		if r.Bucket == "≥60s" {
			if r.Recovered != 0 || r.Lost != 0 {
				t.Errorf("an unresolved row counted as an outcome: recovered=%d lost=%d", r.Recovered, r.Lost)
			}
			if r.Unresolved != 1 {
				t.Errorf("unresolved = %d, want 1 — it must be visible, not folded away", r.Unresolved)
			}
		}
	}
}
