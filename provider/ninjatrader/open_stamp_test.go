package ninjatrader

import "testing"

// CANONICAL TIME CONTRACT (chart-timestamp dispatch, 2026-08-19): a Bar's T in
// the cache is the bar's OPEN time, epoch ms UTC. NT8 delivers CLOSE stamps
// (proven live: the forming 5m bar covering 01:30–01:35 CT arrived stamped
// 01:35 at 01:31), and the conversion happens once, at ingest, so REST, SSE,
// kernel and charts all read the same corrected value.

func TestIngestConvertsCloseStampsToOpenStamps(t *testing.T) {
	for _, tc := range []struct {
		tf   string
		dur  int64
		wire int64 // NT8 close stamp
		want int64 // stored open stamp
	}{
		{"1m", 60_000, 1_787_121_060_000, 1_787_121_000_000},
		{"5m", 300_000, 1_787_121_300_000, 1_787_121_000_000},
		{"15m", 900_000, 1_787_121_900_000, 1_787_121_000_000},
		{"1h", 3_600_000, 1_787_124_600_000, 1_787_121_000_000},
	} {
		c := NewBarCache(10)
		c.SeedHistorical("MNQ", tc.tf, []Bar{{T: tc.wire, O: 1, H: 2, L: 0.5, C: 1.5, V: 9}})
		got := c.Get("MNQ", tc.tf)
		if len(got) != 1 || got[0].T != tc.want {
			t.Errorf("%s: wire close-stamp %d must store as open-stamp %d, got %+v",
				tc.tf, tc.wire, tc.want, got)
		}
		// The streaming path applies the SAME conversion (multi-instance rule).
		c2 := NewBarCache(10)
		c2.Upsert("MNQ", tc.tf, []Bar{{T: tc.wire, O: 1, H: 2, L: 0.5, C: 1.5, V: 9}})
		if got := c2.Get("MNQ", tc.tf); len(got) != 1 || got[0].T != tc.want {
			t.Errorf("%s upsert: want open-stamp %d, got %+v", tc.tf, tc.want, got)
		}
	}
}

// Seed and update for the SAME bar (historical close-stamp then live update)
// land on one T — the conversion cannot split a bar into two.
func TestSeedAndUpdateAgreeOnTheSameBar(t *testing.T) {
	c := NewBarCache(10)
	c.SeedHistorical("MNQ", "5m", []Bar{{T: 1_787_121_300_000, C: 10, O: 1, H: 11, L: 1, V: 1}})
	c.Upsert("MNQ", "5m", []Bar{{T: 1_787_121_300_000, C: 12, O: 1, H: 12, L: 1, V: 2}})
	got := c.Get("MNQ", "5m")
	if len(got) != 1 || got[0].T != 1_787_121_000_000 || got[0].C != 12 {
		t.Fatalf("seed+update must converge on one open-stamped bar with the freshest close, got %+v", got)
	}
}
