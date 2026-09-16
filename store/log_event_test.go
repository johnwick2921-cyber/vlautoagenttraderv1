package store

import (
	"path/filepath"
	"testing"
	"time"
)

// P6 (ledger-close 2026-08-19) — log shipping: durable, non-blocking, pruned.

func logStore(t *testing.T) *Store {
	t.Helper()
	st, err := New(filepath.Join(t.TempDir(), "logs.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	return st
}

func waitForRows(t *testing.T, s *LogEventStore, want int) []*LogEventDB {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		rows, err := s.Recent(1000)
		if err == nil && len(rows) >= want {
			return rows
		}
		time.Sleep(20 * time.Millisecond)
	}
	rows, _ := s.Recent(1000)
	t.Fatalf("wanted %d shipped rows, have %d (async writer stalled?)", want, len(rows))
	return nil
}

// Every forensic class the last postmortem needed lands as a queryable row.
func TestLogEventClassesLand(t *testing.T) {
	st := logStore(t)
	s := st.LogEvent()
	now := time.Now().UnixMilli()

	classes := []LogEventDB{
		{TsUTC: now, Level: "warning", Component: "auto_trader_orders.go:143", TraderID: "t1",
			Message: `[trader_id=t1] ⛔ feed-gate: MNQ open_long skipped`, FieldsJSON: "{}"},
		{TsUTC: now, Level: "error", Component: "clock_health.go:70", TraderID: "",
			Message: "🚨 CLOCK CRITICAL [boot]: |drift| 116000ms exceeds C2 tolerance 60000ms", FieldsJSON: "{}"},
		{TsUTC: now, Level: "warning", Component: "client.go:220", TraderID: "",
			Message: `ai_call model=deepseek-v4-pro duration_ms=150565 ok=false timeout_source=default`, FieldsJSON: "{}"},
		{TsUTC: now, Level: "error", Component: "auto_trader_feedwatch.go:57", TraderID: "t1",
			Message: `[trader_id=t1] 🚨 FEED DOWN: no NT8 bar for 11m0s while CME is OPEN`, FieldsJSON: "{}"},
	}
	for _, ev := range classes {
		s.Enqueue(ev)
	}
	rows := waitForRows(t, s, len(classes))
	byMsg := map[string]*LogEventDB{}
	for _, r := range rows {
		byMsg[r.Message] = r
	}
	for _, want := range classes {
		got, ok := byMsg[want.Message]
		if !ok {
			t.Fatalf("class not shipped: %q", want.Message)
		}
		if got.Level != want.Level || got.TraderID != want.TraderID || got.FieldsJSON == "" {
			t.Errorf("row mismatch for %q: %+v", want.Message, got)
		}
	}
}

// The flood test: enqueueing far beyond the buffer must return fast (the
// trading path calls this inline via the logrus hook) and count drops instead
// of blocking.
func TestLogEventFloodNeverBlocks(t *testing.T) {
	st := logStore(t)
	s := st.LogEvent()
	now := time.Now().UnixMilli()

	const flood = 50_000
	start := time.Now()
	for i := 0; i < flood; i++ {
		s.Enqueue(LogEventDB{TsUTC: now, Level: "warning", Message: "flood", FieldsJSON: "{}"})
	}
	elapsed := time.Since(start)
	// 50k non-blocking enqueues are microseconds each; 2s is an extreme bound —
	// a blocking writer would take minutes on SQLite.
	if elapsed > 2*time.Second {
		t.Fatalf("flood enqueue took %s — the shipper is blocking the caller", elapsed)
	}
	full, failed := s.Dropped()
	if full == 0 {
		t.Fatal("a 50k flood over a 1024 buffer must record queue-full drops (no silent caps)")
	}
	t.Logf("flood: %s for %d enqueues, dropped_full=%d dropped_writes=%d", elapsed, flood, full, failed)
}

func TestLogEventPruneRespectsRetention(t *testing.T) {
	st := logStore(t)
	s := st.LogEvent()
	now := time.Now()

	// The writer's own day-keyed prune fires as the event days change and
	// removes the 40-day-old row automatically — only the fresh row survives.
	old := now.AddDate(0, 0, -40).UnixMilli() // beyond the 30-day default
	fresh := now.Add(-time.Hour).UnixMilli()  // well within
	s.Enqueue(LogEventDB{TsUTC: old, Level: "error", Message: "old-line", FieldsJSON: "{}"})
	s.Enqueue(LogEventDB{TsUTC: fresh, Level: "error", Message: "fresh-line", FieldsJSON: "{}"})
	rows := waitForRows(t, s, 1)
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		rows, _ = s.Recent(10)
		if len(rows) == 1 && rows[0].Message == "fresh-line" {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if len(rows) != 1 || rows[0].Message != "fresh-line" {
		t.Fatalf("daily prune must drop the 40-day row and keep the fresh one: %+v", rows)
	}

	// The manual hook removes anything older than an explicit cutoff.
	pruned, err := s.PruneOlderThan(now.Add(time.Hour).UnixMilli())
	if err != nil || pruned != 1 {
		t.Fatalf("PruneOlderThan(cutoff future) must remove the fresh row: pruned=%d err=%v", pruned, err)
	}
}
