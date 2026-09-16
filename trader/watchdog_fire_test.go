package trader

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	"nofx/store"
)

// The fire log records what the ruling asked for — call age, bytes — and the
// resend outcome attaches to the OPEN row afterwards.
func TestWatchdogFireRecordAndResolve(t *testing.T) {
	st, err := store.New(filepath.Join(t.TempDir(), "w.db"))
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	defer st.Close()
	w := st.WatchdogFires()

	// The real 2026-09-02 20:50:44 fire.
	id, err := w.Record(store.WatchdogFireDB{
		TraderID: "t1", Mode: "post", GapMs: 30000, LimitMs: 30000,
		CallAgeMs: 376135, Bytes: 60034,
	})
	if err != nil || id == 0 {
		t.Fatalf("record: id=%d err=%v", id, err)
	}
	rows, _ := w.Recent(10)
	if len(rows) != 1 || rows[0].Resolved {
		t.Fatalf("a fresh fire must be UNRESOLVED, not a false success: %+v", rows)
	}
	if rows[0].Bytes != 60034 || rows[0].CallAgeMs != 376135 {
		t.Fatalf("call age and bytes must survive the round trip: %+v", rows[0])
	}

	ok, err := w.ResolveLatest("t1", true, 41200, "resend landed")
	if err != nil || !ok {
		t.Fatalf("resolve: ok=%v err=%v", ok, err)
	}
	rows, _ = w.Recent(10)
	if !rows[0].Resolved || !rows[0].ResendOK || rows[0].ResendMs != 41200 {
		t.Fatalf("the resend outcome must attach: %+v", rows[0])
	}
	// Nothing open → honest false, not an error and not a stray write.
	if again, err := w.ResolveLatest("t1", false, 1, "x"); err != nil || again {
		t.Fatalf("with nothing open, resolve must report false: %v %v", again, err)
	}
	// Another trader's fire is never claimed.
	_, _ = w.Record(store.WatchdogFireDB{TraderID: "t2", Mode: "post"})
	if got, _ := w.ResolveLatest("t1", true, 1, "x"); got {
		t.Fatal("t1 must not resolve t2's fire")
	}
}

// The week's table separates recovered from unresolved, and never prints an
// unresolved row as a success.
func TestWatchdogFireTable(t *testing.T) {
	at := time.Date(2026, 9, 2, 20, 50, 44, 0, time.UTC)
	rows := []store.WatchdogFireDB{
		{At: at, Mode: "post", GapMs: 30000, CallAgeMs: 376135, Bytes: 60034, Resolved: true, ResendOK: true, ResendMs: 41200},
		{At: at.Add(time.Hour), Mode: "post", GapMs: 30000, CallAgeMs: 120000, Bytes: 900, Resolved: true, ResendOK: false, ResendMs: 8000, ResendNote: "transport"},
		{At: at.Add(2 * time.Hour), Mode: "pre", GapMs: 600000, CallAgeMs: 600000, Bytes: 0},
	}
	tbl := WatchdogFireTable(rows)
	for _, want := range []string{"3 recorded", "376.1", "60034", "ok in 41.2s", "FAILED after 8.0s (transport)", "UNRESOLVED", "3 fire(s), 1 recovered"} {
		if !strings.Contains(tbl, want) {
			t.Fatalf("table missing %q:\n%s", want, tbl)
		}
	}
	if WatchdogFireTable(nil) != "⏱ watchdog fires: none recorded" {
		t.Fatal("an empty week must say so, not print an empty table")
	}
}
