package trader

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"nofx/store"
)

// P2 (ledger-close 2026-08-19) — the stop_until pause: producer, persistence,
// expiry, and the entries-only contract.

func pauseTrader(t *testing.T) (*AutoTrader, *store.Store) {
	t.Helper()
	st, err := store.New(filepath.Join(t.TempDir(), "p.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	at := &AutoTrader{id: "tp1", exchange: "ninjatrader", store: st}
	at.config.Exchange = "ninjatrader"
	return at, st
}

func TestPauseBlocksOnlyNewEntries(t *testing.T) {
	at, _ := pauseTrader(t)

	// Unpaused → gate silent.
	if reason, paused := at.entryPaused(); paused {
		t.Fatalf("fresh trader must not be paused, got %q", reason)
	}

	until := time.Now().Add(30 * time.Minute)
	if err := at.PauseEntriesUntil(until, "test"); err != nil {
		t.Fatal(err)
	}
	reason, paused := at.entryPaused()
	if !paused {
		t.Fatal("armed pause must refuse entries")
	}
	// The dispatch's exact refusal shape: "paused until 21:30 CT (owner)".
	if !strings.HasPrefix(reason, "paused until ") || !strings.HasSuffix(reason, " CT (owner)") {
		t.Fatalf("refusal must read 'paused until HH:MM CT (owner)', got %q", reason)
	}

	// The gate is wired for open_* ONLY (position management continues): the
	// executeDecisionWithRecord switch matches open_long/open_short — pin that
	// contract at the source level like the intrade ordering guard does.
	src := readOrdersSource(t)
	gate := src[strings.Index(src, "stop_until OWNER PAUSE"):]
	gate = gate[:strings.Index(gate, "consecutive-loss")]
	if strings.Contains(gate, "close_long") || strings.Contains(gate, "close_short") {
		t.Fatal("the stop_until gate must never match close actions — closes are position management")
	}
}

func readOrdersSource(t *testing.T) string {
	t.Helper()
	b, err := os.ReadFile("auto_trader_orders.go")
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestPauseSurvivesRestart(t *testing.T) {
	at, st := pauseTrader(t)
	until := time.Now().Add(45 * time.Minute)
	if err := at.PauseEntriesUntil(until, "test"); err != nil {
		t.Fatal(err)
	}

	// A fresh AutoTrader over the SAME store (the restart) restores the pause.
	at2 := &AutoTrader{id: "tp1", exchange: "ninjatrader", store: st}
	at2.loadPersistedPause()
	if _, paused := at2.entryPaused(); !paused {
		t.Fatal("a pause must survive restart via the persisted system_config key")
	}
	got, active := at2.PauseState()
	if !active || got.UnixMilli() != until.UnixMilli() {
		t.Fatalf("restored deadline drifted: want %d got %d (active=%v)", until.UnixMilli(), got.UnixMilli(), active)
	}
}

func TestPauseExpiryAutoResumes(t *testing.T) {
	at, st := pauseTrader(t)
	at.pauseUntilMs.Store(time.Now().Add(-time.Second).UnixMilli()) // already past
	_ = st.SetSystemConfig(pauseConfigKey(at.id), "1")              // stale persisted value

	if reason, paused := at.entryPaused(); paused {
		t.Fatalf("an expired pause must auto-resume, got %q", reason)
	}
	if at.pauseUntilMs.Load() != 0 {
		t.Fatal("expiry must clear the in-memory deadline")
	}
	if raw, _ := st.GetSystemConfig(pauseConfigKey(at.id)); raw != "0" {
		t.Fatalf("expiry must clear the persisted deadline, got %q", raw)
	}
}

func TestResumeClearsImmediately(t *testing.T) {
	at, st := pauseTrader(t)
	if err := at.PauseEntriesUntil(time.Now().Add(time.Hour), "test"); err != nil {
		t.Fatal(err)
	}
	at.ResumeEntries("test")
	if _, paused := at.entryPaused(); paused {
		t.Fatal("ResumeEntries must clear the pause")
	}
	if raw, _ := st.GetSystemConfig(pauseConfigKey(at.id)); raw != "0" {
		t.Fatalf("resume must clear the persisted key, got %q", raw)
	}
	// A pause in the past is refused outright.
	if err := at.PauseEntriesUntil(time.Now().Add(-time.Minute), "test"); err == nil {
		t.Fatal("a past deadline must be rejected")
	}
}
