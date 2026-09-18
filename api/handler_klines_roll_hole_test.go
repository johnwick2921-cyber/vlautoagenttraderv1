// W-CHART-ROLL-HOLE (2026-09-17) — the chart across the Sept→Dec roll, at the
// PRODUCTION call site: the real router, the real JWT middleware, a real sqlite
// store carrying the LIVE store's shape (read 2026-09-17 17:10 CT, read-only):
//
//	MNQ 09-26 live 5m  : 09-06 17:00 → 09-14 09:55 CT, continuous inside CME hours
//	MNQ 12-26 strays   : historical_import 09-07 12:00 · 09-08 16:00 · 09-09 16:00 ·
//	                     09-10 16:00 (one bar per day, wave-101 snapshots);
//	                     historical 09-10 22:10–22:30 (5) · 09-11 15:55 (1) ·
//	                     09-14 08:50–09:20 (7) — NT8 served ~2000 bars of 12-26
//	                     history at subscribe, and the bars PK (symbol, tf,
//	                     open_time_ms) has no contract, so a 12-26 row landed
//	                     wherever 09-26 held no row.
//	MNQ 12-26 live 5m  : 09-14 10:00 CT onward (FirstLiveOn = the real roll).
//
// The owner's screenshot (2026-09-17 16:40 CT): candles missing for several
// days across the roll with a sparse stray candle or two in the gap. The
// stray 12-26 rows OLDER than the roll pulled the chart boundary back and the
// prior-contract reader then excluded every 09-26 row after that point.
package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"nofx/auth"
	"nofx/manager"
	"nofx/market"
	"nofx/store"
)

var rollCT = time.FixedZone("CT", -5*3600)

func rollAt(mo, d, h, m int) int64 {
	return time.Date(2026, time.Month(mo), d, h, m, 0, 0, rollCT).UnixMilli()
}

// rollFixtureBars builds the live shape above. Sept bars carry the September
// scale (~29,000), Dec bars the December scale (+292), so a stray is visible
// on price alone. Returns the rows and the ring (Dec live bars only — the ring
// is contract-pure).
func rollFixtureBars() (rows []store.BarHistoryDB, ring []market.Kline) {
	const fiveMin = int64(5 * 60_000)
	taken := map[int64]bool{}
	add := func(contract, source string, fromMs, toMs int64, px float64) {
		for t := fromMs; t <= toMs; t += fiveMin {
			if taken[t] {
				continue
			}
			taken[t] = true
			rows = append(rows, store.BarHistoryDB{Symbol: "MNQ", TF: "5m", OpenTimeMs: t,
				O: px, H: px + 10, L: px - 10, C: px + 5, V: 1, Contract: contract, Source: source})
		}
	}
	// Dec strays FIRST so they take their slots (INSERT-OR-IGNORE order in the
	// live store: the 12-26 history arrived at subscribe, before the 09-26 rows
	// that would have held those times were ever written).
	dec := 29292.0
	for _, t := range []int64{rollAt(9, 7, 12, 0), rollAt(9, 8, 16, 0), rollAt(9, 9, 16, 0), rollAt(9, 10, 16, 0)} {
		add("MNQ 12-26", store.BarSourceHistoricalImport, t, t, dec)
	}
	add("MNQ 12-26", store.BarSourceHistorical, rollAt(9, 10, 22, 10), rollAt(9, 10, 22, 30), dec)
	add("MNQ 12-26", store.BarSourceHistorical, rollAt(9, 11, 15, 55), rollAt(9, 11, 15, 55), dec)
	add("MNQ 12-26", store.BarSourceHistorical, rollAt(9, 14, 8, 50), rollAt(9, 14, 9, 20), dec)
	// Sept live: Sunday 09-06 17:00 → Friday 09-11 15:55 (holes where a stray
	// sits), then Sunday 09-13 17:00 → Monday 09-14 09:55.
	sep := 29000.0
	add("MNQ 09-26", store.BarSourceLive, rollAt(9, 6, 17, 0), rollAt(9, 11, 15, 55), sep)
	add("MNQ 09-26", store.BarSourceLive, rollAt(9, 13, 17, 0), rollAt(9, 14, 9, 55), sep)
	// Dec live from the roll, through Wednesday 09-16 23:55.
	add("MNQ 12-26", store.BarSourceLive, rollAt(9, 14, 10, 0), rollAt(9, 16, 23, 55), dec)
	// LatestContract (the handler's "what contract are we on" fallback) reads
	// the 1m rung: one 1m Dec live row at the newest time answers it.
	rows = append(rows, store.BarHistoryDB{Symbol: "MNQ", TF: "1m", OpenTimeMs: rollAt(9, 16, 23, 59),
		O: dec, H: dec + 10, L: dec - 10, C: dec + 5, V: 1, Contract: "MNQ 12-26", Source: store.BarSourceLive})
	// The ring: the newest 500 Dec LIVE bars, as the provider serves them.
	for _, r := range rows {
		if r.TF == "5m" && r.Contract == "MNQ 12-26" && r.Source == store.BarSourceLive {
			ring = append(ring, market.Kline{OpenTime: r.OpenTimeMs, Open: r.O, High: r.H, Low: r.L, Close: r.C, Volume: r.V})
		}
	}
	sort.Slice(ring, func(i, j int) bool { return ring[i].OpenTime < ring[j].OpenTime })
	if len(ring) > 500 {
		ring = ring[len(ring)-500:]
	}
	return rows, ring
}

// newRollHoleServer seeds the fixture into a temp store and returns the real
// Server plus a Bearer token.
func newRollHoleServer(t *testing.T) (*Server, string) {
	t.Helper()
	orig := market.FuturesBarsProvider
	t.Cleanup(func() { market.FuturesBarsProvider = orig })

	st, err := store.New(filepath.Join(t.TempDir(), "roll.db"))
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	if err := st.BarHistory().Migrate(); err != nil {
		t.Fatal(err)
	}
	rows, ring := rollFixtureBars()
	var imports, live []store.BarHistoryDB
	for _, r := range rows {
		if r.Source == store.BarSourceHistoricalImport {
			imports = append(imports, r)
		} else {
			live = append(live, r)
		}
	}
	if err := st.BarHistory().InsertBars(live); err != nil {
		t.Fatal(err)
	}
	if _, _, err := st.BarHistory().ImportBars(imports); err != nil {
		t.Fatal(err)
	}
	market.FuturesBarsProvider = func(symbol, tf string, count int) []market.Kline {
		if symbol != "MNQ" || tf != "5m" {
			return nil
		}
		out := ring
		if count > 0 && len(out) > count {
			out = out[len(out)-count:]
		}
		return append([]market.Kline(nil), out...)
	}

	auth.SetJWTSecret("roll-hole-test-secret")
	tok, err := auth.GenerateJWT("u-roll", "roll@test")
	if err != nil {
		t.Fatalf("jwt: %v", err)
	}
	return NewServer(manager.NewTraderManager(), st, nil, "127.0.0.1", 0), tok
}

type rollDayKey struct {
	day      string
	contract string
}

// rollPerDay counts klines per (CT day, contract), and renders the table the
// PR body quotes.
func rollPerDay(ks []market.Kline) (map[rollDayKey]int, string) {
	counts := map[rollDayKey]int{}
	for _, k := range ks {
		day := time.UnixMilli(k.OpenTime).In(rollCT).Format("2006-01-02")
		counts[rollDayKey{day, k.Contract}]++
	}
	keys := make([]rollDayKey, 0, len(counts))
	for k := range counts {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].day != keys[j].day {
			return keys[i].day < keys[j].day
		}
		return keys[i].contract < keys[j].contract
	})
	var b strings.Builder
	for _, k := range keys {
		fmt.Fprintf(&b, "  %s  %-10s %4d\n", k.day, k.contract, counts[k])
	}
	return counts, b.String()
}

func rollGetKlines(t *testing.T, s *Server, tok string, limit int) []market.Kline {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/klines?symbol=MNQ&interval=5m&limit=%d&exchange=ninjatrader", limit), nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	rec := httptest.NewRecorder()
	s.router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /api/klines: %d %s", rec.Code, rec.Body.String())
	}
	var ks []market.Kline
	if err := json.Unmarshal(rec.Body.Bytes(), &ks); err != nil {
		t.Fatalf("decode: %v", err)
	}
	return ks
}

func TestKlinesRouteRequiresJWT(t *testing.T) {
	s, _ := newRollHoleServer(t)
	req := httptest.NewRequest(http.MethodGet, "/api/klines?symbol=MNQ&interval=5m&limit=10&exchange=ninjatrader", nil)
	rec := httptest.NewRecorder()
	s.router.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("no JWT must be 401, got %d", rec.Code)
	}
}

// The pin: 5m, limit 1500, through the production route. Sept rows fill every
// slot before the roll (09-14 10:00 CT), Dec rows every slot after it, and NOT
// ONE Dec kline sits before the roll. Per-day counts are continuous across the
// roll (the only empty day is Saturday 09-12).
func TestKlinesAcrossRollNoHoleNoStrays(t *testing.T) {
	s, tok := newRollHoleServer(t)
	ks := rollGetKlines(t, s, tok, 1500)
	counts, table := rollPerDay(ks)
	t.Logf("per-day klines (CT day, contract, n), %d klines:\n%s", len(ks), table)

	boundary := rollAt(9, 14, 10, 0)
	for _, k := range ks {
		if k.Contract == "" {
			t.Fatalf("unlabelled kline at %d", k.OpenTime)
		}
		if k.Contract == "MNQ 12-26" && k.OpenTime < boundary {
			t.Errorf("STRAY: a 12-26 kline before the roll at %s", time.UnixMilli(k.OpenTime).In(rollCT).Format("01-02 15:04"))
		}
		if k.Contract == "MNQ 09-26" && k.OpenTime >= boundary {
			t.Errorf("a 09-26 kline after the roll at %s", time.UnixMilli(k.OpenTime).In(rollCT).Format("01-02 15:04"))
		}
	}
	for i := 1; i < len(ks); i++ {
		if ks[i].OpenTime <= ks[i-1].OpenTime {
			t.Fatalf("not time-ordered at %d", i)
		}
	}
	// Per-day continuity across the roll. Expected from the fixture: 09-11 =
	// 192 slots (00:00–15:55) minus the 15:55 stray slot = 191 Sept; 09-13 =
	// 84 Sept; 09-14 = 120 slots minus the 7 stray slots = 113 Sept + 168 Dec
	// (10:00–23:55); 09-15/09-16 = 288 Dec each.
	want := map[rollDayKey]int{
		{"2026-09-11", "MNQ 09-26"}: 191,
		{"2026-09-13", "MNQ 09-26"}: 84,
		{"2026-09-14", "MNQ 09-26"}: 113,
		{"2026-09-14", "MNQ 12-26"}: 168,
		{"2026-09-15", "MNQ 12-26"}: 288,
		{"2026-09-16", "MNQ 12-26"}: 288,
	}
	for k, n := range want {
		if counts[k] != n {
			t.Errorf("%s %s: got %d klines, want %d", k.day, k.contract, counts[k], n)
		}
	}
	// 09-10: Sept holds every slot except the 16:00 import slot and the five
	// 22:10–22:30 replay slots; with 1500 asked the series reaches back into
	// 09-10, so the day is present and at least the evening is full.
	if counts[rollDayKey{"2026-09-10", "MNQ 09-26"}] < 100 {
		t.Errorf("09-10 09-26: got %d klines, the day before the strays must be drawn", counts[rollDayKey{"2026-09-10", "MNQ 09-26"}])
	}
	if len(ks) != 1500 {
		t.Errorf("served %d klines, want the full ask of 1500 (the store holds more than that)", len(ks))
	}
}
