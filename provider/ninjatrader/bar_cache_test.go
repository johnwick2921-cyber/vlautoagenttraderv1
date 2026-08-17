// Plan 4.4 Stage 2 — BarCache tests.

package ninjatrader

import (
	"sync"
	"testing"
)

func TestBarCache_SeedAndGet(t *testing.T) {
	c := NewBarCache(0) // default max
	bars := []Bar{
		{T: 1, O: 100, H: 101, L: 99, C: 100.5, V: 10},
		{T: 2, O: 100.5, H: 102, L: 100, C: 101.5, V: 12},
	}
	c.SeedHistorical("MNQ", "1m", bars)

	got := c.Get("MNQ", "1m")
	if len(got) != 2 {
		t.Fatalf("len=%d, want 2", len(got))
	}
	if got[0].T != 1 || got[1].T != 2 {
		t.Errorf("order wrong: %+v", got)
	}
	if c.Count("MNQ", "1m") != 2 {
		t.Errorf("Count=%d, want 2", c.Count("MNQ", "1m"))
	}
}

func TestBarCache_Get_ReturnsSnapshotCopy(t *testing.T) {
	c := NewBarCache(0)
	c.SeedHistorical("MNQ", "1m", []Bar{{T: 1, C: 100}})
	snap1 := c.Get("MNQ", "1m")
	if len(snap1) != 1 {
		t.Fatalf("len=%d", len(snap1))
	}
	// Mutate snapshot — must NOT affect cache.
	snap1[0].C = 999
	snap2 := c.Get("MNQ", "1m")
	if snap2[0].C != 100 {
		t.Errorf("cache mutated via snapshot: %f", snap2[0].C)
	}
}

// TestBarCache_SeedHistorical_MergePreservesDepth is the Phase-0 regression
// guard: a reconnect recreate (or N3 re-seed) emits a bars_historical frame
// that may carry FEWER bars than the cache already holds. SeedHistorical must
// MERGE (not replace), so the deeper history survives and new bars are added.
func TestBarCache_SeedHistorical_MergePreservesDepth(t *testing.T) {
	c := NewBarCache(0)

	// Initial deep seed: bars T=1..100.
	deep := make([]Bar, 0, 100)
	for i := int64(1); i <= 100; i++ {
		deep = append(deep, Bar{T: i, C: float64(i)})
	}
	c.SeedHistorical("MNQ", "5m", deep)
	if c.Count("MNQ", "5m") != 100 {
		t.Fatalf("after deep seed Count=%d, want 100", c.Count("MNQ", "5m"))
	}

	// Reconnect recreate re-seeds a THINNER frame: only the recent tail +
	// one new bar (T=99,100,101). Must NOT wipe the deeper 1..98.
	c.SeedHistorical("MNQ", "5m", []Bar{
		{T: 99, C: 99}, {T: 100, C: 100.5}, {T: 101, C: 101},
	})
	got := c.Get("MNQ", "5m")
	if len(got) != 101 {
		t.Fatalf("after thin re-seed Count=%d, want 101 (depth preserved + 1 new)", len(got))
	}
	if got[0].T != 1 {
		t.Errorf("deeper history wiped: earliest T=%d, want 1", got[0].T)
	}
	if got[100].T != 101 {
		t.Errorf("new bar not appended: last T=%d, want 101", got[100].T)
	}
	if got[99].C != 100.5 { // T=100 overlap — incoming (freshest) wins
		t.Errorf("overlap not updated to freshest: T=100 C=%f, want 100.5", got[99].C)
	}

	// Empty re-seed (cursor-deduped reconnect frame) must preserve everything.
	c.SeedHistorical("MNQ", "5m", []Bar{})
	if c.Count("MNQ", "5m") != 101 {
		t.Errorf("empty re-seed shrank cache: Count=%d, want 101", c.Count("MNQ", "5m"))
	}

	// A DEEPER (older) re-seed merges in front — proves future deepening works.
	c.SeedHistorical("MNQ", "5m", []Bar{{T: -2, C: -2}, {T: -1, C: -1}, {T: 1, C: 1}})
	got = c.Get("MNQ", "5m")
	if got[0].T != -2 || got[1].T != -1 {
		t.Errorf("older bars not prepended: %d,%d want -2,-1", got[0].T, got[1].T)
	}
	if len(got) != 103 {
		t.Errorf("merge count wrong: %d, want 103", len(got))
	}
}

func TestBarCache_Upsert_ReplaceSameT(t *testing.T) {
	c := NewBarCache(0)
	c.SeedHistorical("MNQ", "1m", []Bar{{T: 100, C: 21500.0}})
	// Same t — in-progress update.
	c.Upsert("MNQ", "1m", []Bar{{T: 100, C: 21500.75, H: 21501.0}})

	got := c.Get("MNQ", "1m")
	if len(got) != 1 {
		t.Fatalf("len=%d, want 1 (replaced not appended)", len(got))
	}
	if got[0].C != 21500.75 || got[0].H != 21501.0 {
		t.Errorf("replace did not take effect: %+v", got[0])
	}
}

func TestBarCache_Upsert_AppendNewT(t *testing.T) {
	c := NewBarCache(0)
	c.SeedHistorical("MNQ", "1m", []Bar{{T: 100, C: 21500}})
	c.Upsert("MNQ", "1m", []Bar{{T: 160, C: 21501}})
	got := c.Get("MNQ", "1m")
	if len(got) != 2 || got[1].T != 160 {
		t.Errorf("expected append at T=160, got %+v", got)
	}
}

func TestBarCache_Upsert_IgnoreOutOfOrder(t *testing.T) {
	c := NewBarCache(0)
	c.SeedHistorical("MNQ", "1m", []Bar{{T: 100, C: 21500}, {T: 160, C: 21501}})
	// t < last.t — defensive: ignore.
	c.Upsert("MNQ", "1m", []Bar{{T: 50, C: 9999}})
	got := c.Get("MNQ", "1m")
	if len(got) != 2 {
		t.Fatalf("len=%d, expected 2 (no append)", len(got))
	}
	if got[0].T != 100 || got[1].T != 160 {
		t.Errorf("order disturbed: %+v", got)
	}
}

func TestBarCache_Upsert_MultiBar(t *testing.T) {
	// Multi-bar gotcha: a single tick can deliver multiple bars.
	c := NewBarCache(0)
	c.SeedHistorical("MNQ", "1m", []Bar{{T: 100, C: 21500}})
	c.Upsert("MNQ", "1m", []Bar{
		{T: 100, C: 21500.5}, // replace
		{T: 160, C: 21501},   // append
		{T: 220, C: 21502},   // append
	})
	got := c.Get("MNQ", "1m")
	if len(got) != 3 {
		t.Fatalf("len=%d, want 3", len(got))
	}
	if got[0].C != 21500.5 || got[1].T != 160 || got[2].T != 220 {
		t.Errorf("multi-bar upsert wrong: %+v", got)
	}
}

func TestBarCache_RingBound(t *testing.T) {
	c := NewBarCache(3) // tiny ring
	// Real prices, not zero-valued structs: a bar with no range and no volume is
	// an NT8 empty-minute placeholder and is refused at ingest (see
	// isPlaceholderBar). This test is about ring bounds, so give it real bars.
	c.SeedHistorical("MNQ", "1m", []Bar{
		{T: 1, O: 100, H: 101, L: 99, C: 100, V: 5},
		{T: 2, O: 100, H: 102, L: 99, C: 101, V: 5},
		{T: 3, O: 101, H: 103, L: 100, C: 102, V: 5},
	})
	// 4th bar — oldest must drop.
	c.Upsert("MNQ", "1m", []Bar{{T: 4, O: 102, H: 104, L: 101, C: 103, V: 5}})
	got := c.Get("MNQ", "1m")
	if len(got) != 3 {
		t.Fatalf("len=%d, want 3 (ring bounded)", len(got))
	}
	if got[0].T != 2 || got[2].T != 4 {
		t.Errorf("ring drop wrong: want [2,3,4], got %+v", got)
	}
}

func TestBarCache_Seed_TruncatesOversized(t *testing.T) {
	c := NewBarCache(2)
	c.SeedHistorical("MNQ", "1m", []Bar{ // 4 real bars, ring is 2
		{T: 1, O: 100, H: 101, L: 99, C: 100, V: 5},
		{T: 2, O: 100, H: 102, L: 99, C: 101, V: 5},
		{T: 3, O: 101, H: 103, L: 100, C: 102, V: 5},
		{T: 4, O: 102, H: 104, L: 101, C: 103, V: 5},
	})
	got := c.Get("MNQ", "1m")
	if len(got) != 2 {
		t.Fatalf("len=%d, want 2", len(got))
	}
	if got[0].T != 3 || got[1].T != 4 {
		t.Errorf("seed truncation kept wrong tail: %+v", got)
	}
}

func TestBarCache_MultipleKeys(t *testing.T) {
	c := NewBarCache(0)
	c.SeedHistorical("MNQ", "1m", []Bar{{T: 1, C: 100}})
	c.SeedHistorical("MNQ", "5m", []Bar{{T: 1, C: 200}})
	c.SeedHistorical("ES", "1m", []Bar{{T: 1, C: 4500}})

	if c.Get("MNQ", "1m")[0].C != 100 {
		t.Error("MNQ 1m wrong")
	}
	if c.Get("MNQ", "5m")[0].C != 200 {
		t.Error("MNQ 5m wrong")
	}
	if c.Get("ES", "1m")[0].C != 4500 {
		t.Error("ES 1m wrong")
	}

	keys := c.Keys()
	if len(keys) != 3 {
		t.Errorf("Keys len=%d, want 3", len(keys))
	}
}

func TestBarCache_Get_EmptyKey(t *testing.T) {
	c := NewBarCache(0)
	if got := c.Get("UNKNOWN", "1m"); got != nil {
		t.Errorf("Get unknown key should return nil, got %+v", got)
	}
}

func TestBarCache_EmptyInputsIgnored(t *testing.T) {
	c := NewBarCache(0)
	c.SeedHistorical("", "1m", []Bar{{T: 1}})  // empty symbol
	c.SeedHistorical("MNQ", "", []Bar{{T: 1}}) // empty timeframe
	c.Upsert("MNQ", "1m", nil)                 // nil bars
	c.Upsert("MNQ", "1m", []Bar{})             // empty bars
	if len(c.Keys()) != 0 {
		t.Errorf("expected no keys, got %+v", c.Keys())
	}
}

// TestBarCache_Concurrent — run under `go test -race` to validate the
// RWMutex protection. 4 writers + 4 readers banging on the same key
// briefly, race-clean.
func TestBarCache_Concurrent(t *testing.T) {
	c := NewBarCache(100)
	c.SeedHistorical("MNQ", "1m", []Bar{{T: 0, C: 21500}})
	var wg sync.WaitGroup
	const N = 50
	for w := 0; w < 4; w++ {
		wg.Add(1)
		go func(offset int) {
			defer wg.Done()
			for i := 0; i < N; i++ {
				c.Upsert("MNQ", "1m", []Bar{{T: int64(i + offset*N + 1), C: float64(21500 + i)}})
			}
		}(w)
	}
	for r := 0; r < 4; r++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < N; i++ {
				_ = c.Get("MNQ", "1m")
			}
		}()
	}
	wg.Wait()
	// We don't assert the exact final state (multi-writer ordering
	// is nondeterministic) — only that the test ran race-clean and
	// the cache holds a valid slice.
	got := c.Get("MNQ", "1m")
	if len(got) == 0 {
		t.Error("cache empty after concurrent writes")
	}
}
