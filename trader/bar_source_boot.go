package trader

import (
	"fmt"
	"strings"
	"time"

	"nofx/market"
	"nofx/store"
)

// BarSourceBootLine (BAR-SOURCE WAVE 2026-09-02) reports, per TF, WHERE the
// resolver actually answers from and the earliest bar it can reach — every
// value READ from the resolver at boot, never a literal (A24). A TF the
// resolver cannot answer says so rather than being omitted.
//
// Shape:
//
//	📊 bars: 1w nt8_agg via 1d since 2020-11-11 (1500) · 1d nt8 since 2020-11-11 (1500) · …
//	   · ladder(1w)=[1d 1m] native 1w EXCLUDED (Fri→Thu stamps) · retention 1m=90d coarse=forever
//
// BarSourceBootLineAfterBackfill (R1) is the SECOND print, emitted once the
// first NT8 backfill has landed. The boot print runs before the AddOn replays
// bars_historical, so on a cold cache the resolver honestly answers "own1m via
// 1m" for every TF — true of that instant, and a misleading report of
// capability. Observed twice (2026-09-02 17:51 and 18:27) with 22 years of
// weekly data already on disk.
func BarSourceBootLineAfterBackfill(r *market.BarResolver, symbol string, now time.Time) string {
	return "📊 bars after backfill: " + barSourceFields(r, symbol, now)
}

func BarSourceBootLine(r *market.BarResolver, symbol string, now time.Time) string {
	if r == nil {
		return "📊 bars: resolver unavailable — no source report"
	}
	return "📊 bars: " + barSourceFields(r, symbol, now) + " (cache cold at boot — see the 📊 bars after backfill line)"
}

// HistoryHeldBootLine (HISTORY IMPORT, wave 101) reports the imported history
// the store HOLDS, per contract × per timeframe — the boot line D7 asks for.
// READ from the store, never a literal; rows are counted by their source so a
// contract's LIVE window and its IMPORTED history are named separately.

func HistoryHeldBootLine(st *store.Store, symbol string) string {
	if st == nil || st.BarHistory() == nil {
		return "📚 history held: store unavailable — nothing read"
	}
	rows, err := st.BarHistory().HistoryHeld(symbol)
	if err != nil || len(rows) == 0 {
		return "📚 history held: none imported yet (wave 101) — the live table holds only the subscribed contract's tape"
	}
	parts := make([]string, 0, len(rows))
	for _, r := range rows {
		parts = append(parts, fmt.Sprintf("%s·%s=%d [%s→%s]", r.Contract, r.TF, r.N,
			time.UnixMilli(r.FirstMs).UTC().Format("2006-01-02"), time.UnixMilli(r.LastMs).UTC().Format("2006-01-02")))
	}
	return "📚 history held: " + strings.Join(parts, " · ")
}

// barSourceFields renders the per-TF source report both prints share.
func barSourceFields(r *market.BarResolver, symbol string, now time.Time) string {
	if r == nil {
		return "resolver unavailable — no source report"
	}
	tfs := []string{"1w", "1d", "4h", "1h", "15m", "5m", "1m"}
	parts := make([]string, 0, len(tfs))
	for _, tf := range tfs {
		s, err := r.CompletedBars(symbol, tf, 0, now.UnixMilli())
		if err != nil || len(s.Bars) == 0 {
			parts = append(parts, fmt.Sprintf("%s UNAVAILABLE", tf))
			continue
		}
		via := string(s.Source)
		if s.FromTF != tf {
			via = fmt.Sprintf("%s via %s", s.Source, s.FromTF)
		}
		parts = append(parts, fmt.Sprintf("%s %s since %s (%d)",
			tf, via, time.UnixMilli(s.EarliestMs).UTC().Format("2006-01-02"), len(s.Bars)))
	}
	excl := ""
	if why := market.ExcludedNative("1w"); why != "" {
		head := why
		if i := strings.Index(head, "."); i > 0 {
			head = head[:i]
		}
		excl = fmt.Sprintf(" · native 1w EXCLUDED from the weekly ladder: %s", head)
	}
	return fmt.Sprintf("%s · ladder(1w)=%v%s · retention 1m=%dd 5m=%dd 15m=%dd coarse=forever",
		strings.Join(parts, " · "), market.LadderFor("1w"), excl,
		store.RetentionDaysFor("1m"), store.RetentionDaysFor("5m"), store.RetentionDaysFor("15m"))
}
