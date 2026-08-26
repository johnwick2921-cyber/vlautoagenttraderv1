package store

import (
	"path/filepath"
	"testing"
)

func TestDigestStore(t *testing.T) {
	st, err := New(filepath.Join(t.TempDir(), "t.db"))
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	dg := st.Digest()

	// session digests (idempotent).
	wrote, _ := dg.SaveIfAbsent(&DigestDB{TraderID: "t1", Symbol: "MNQ", TradeDate: "2026-08-14", Session: "NY", Kind: "session", Text: "NY digest"})
	if !wrote {
		t.Fatalf("first session digest should write")
	}
	again, _ := dg.SaveIfAbsent(&DigestDB{TraderID: "t1", Symbol: "MNQ", TradeDate: "2026-08-14", Session: "NY", Kind: "session", Text: "changed"})
	if again {
		t.Fatalf("duplicate session digest must be a no-op")
	}
	sd, _ := dg.SessionDigests("t1", "MNQ", "2026-08-14")
	if len(sd) != 1 || sd[0] != "NY digest" {
		t.Fatalf("session digests = %v", sd)
	}

	// dailies newest-first.
	for _, d := range []string{"2026-08-11", "2026-08-13", "2026-08-12"} {
		dg.SaveIfAbsent(&DigestDB{TraderID: "t1", Symbol: "MNQ", TradeDate: d, Kind: "daily", Text: d})
	}
	dailies, _ := dg.RecentDailies("t1", "MNQ", 7)
	if len(dailies) != 3 || dailies[0] != "2026-08-13" {
		t.Fatalf("recent dailies newest-first wrong: %v", dailies)
	}
}

// W16/R4 — THE TWO-TRADER RACE.
//
// Identity used to be (symbol, trade_date, session, kind) — session-GLOBAL —
// while the Text holds ONE trader's entries and realized P&L. With two day-plan
// traders on the same symbol, whichever goroutine won SaveIfAbsent wrote the
// digest that BOTH then read and fed to their next planner read: one trader's
// day silently became the other's memory.
func TestW16DigestIsScopedPerTrader(t *testing.T) {
	st, err := New(filepath.Join(t.TempDir(), "d.db"))
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	dg := st.Digest()

	const symbol, date, sess = "MNQ", "2026-08-17", "NY"
	row := func(trader, text string) *DigestDB {
		return &DigestDB{TraderID: trader, Symbol: symbol, TradeDate: date, Session: sess, Kind: "session", Text: text}
	}

	// Both traders close the same session and each writes its OWN digest.
	if wrote, _ := dg.SaveIfAbsent(row("traderA", "A: 2 entries · realized +120.5")); !wrote {
		t.Fatal("trader A's digest must write")
	}
	if wrote, _ := dg.SaveIfAbsent(row("traderB", "B: 1 entry · realized -40.0")); !wrote {
		t.Fatal("trader B's digest must ALSO write — it used to lose the race and inherit A's numbers")
	}

	// Each reads back only its own.
	a, _ := dg.SessionDigests("traderA", symbol, date)
	b, _ := dg.SessionDigests("traderB", symbol, date)
	if len(a) != 1 || a[0] != "A: 2 entries · realized +120.5" {
		t.Fatalf("trader A read %v — must see only its own digest", a)
	}
	if len(b) != 1 || b[0] != "B: 1 entry · realized -40.0" {
		t.Fatalf("trader B read %v — must see only its own digest", b)
	}

	// Idempotency still holds PER TRADER.
	if again, _ := dg.SaveIfAbsent(row("traderA", "A: rewritten")); again {
		t.Fatal("a second write for the same trader/session must be a no-op")
	}
	a2, _ := dg.SessionDigests("traderA", symbol, date)
	if a2[0] != "A: 2 entries · realized +120.5" {
		t.Fatalf("the original text must survive, got %q", a2[0])
	}

	// Dailies are scoped too.
	dg.SaveIfAbsent(&DigestDB{TraderID: "traderA", Symbol: symbol, TradeDate: date, Kind: "daily", Text: "A daily"})
	dg.SaveIfAbsent(&DigestDB{TraderID: "traderB", Symbol: symbol, TradeDate: date, Kind: "daily", Text: "B daily"})
	da, _ := dg.RecentDailies("traderA", symbol, 7)
	if len(da) != 1 || da[0] != "A daily" {
		t.Fatalf("trader A dailies = %v, want only its own", da)
	}

	// A digest that cannot say WHOSE P&L it reports is refused outright.
	if _, err := dg.SaveIfAbsent(&DigestDB{Symbol: symbol, TradeDate: date, Kind: "daily", Text: "orphan"}); err == nil {
		t.Fatal("an unscoped digest must be REFUSED — the next planner read would consume it as fact")
	}
}
