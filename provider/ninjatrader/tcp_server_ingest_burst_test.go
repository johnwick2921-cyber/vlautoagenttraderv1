package ninjatrader

import (
	"testing"
)

// ═══════════════════════════════════════════════════════════════════════════
// S-LIST CLOSER FIX3 — ingest queue cap fixtures (2026-08-27).
//
// The 2026-08-27 21:42 flood touched 1024/1024 (1 intrabar drop): the default
// cap was raised 1024 → 4096 and the peak_depth summary now also fires on the
// clean path (1-line/min heartbeat), so a zero-drop reopen still proves its
// peak. Closes are structurally impossible to drop on this path (the channel
// carries FORMING intra-bar updates; closed bars are re-derived from the cache
// tail after the drain) — the persist queue's closes_dropped counter must stay
// 0 through any burst.
//
// Fixtures: default cap = 4096 (env override still works); a synthetic
// 600-frames/s × 5s = 3000-frame burst with a STALLED drainer → zero drops
// (the capacity alone absorbs it) + peak sampled; an over-cap burst → only
// intrabar drop-oldest events, exactly (sent − cap), zero close drops.
// ═══════════════════════════════════════════════════════════════════════════

func resetIngestCounters() {
	ingestDropOld.Store(0)
	ingestDropCur.Store(0)
	ingestDropHist.Store(0)
	persistDropped.Store(0)
	persistDroppedCloses.Store(0)
	ingestPeakDepth.Store(0)
	ingestLastSum.Store(0)
}

func TestIngestQueueCapDefault4096(t *testing.T) {
	t.Setenv("INGEST_QUEUE_CAP", "")
	if got := ingestQueueCap(); got != 4096 {
		t.Fatalf("default cap = %d, want 4096 (S-list closer FIX3)", got)
	}
	t.Setenv("INGEST_QUEUE_CAP", "2048")
	if got := ingestQueueCap(); got != 2048 {
		t.Fatalf("override cap = %d, want 2048", got)
	}
	t.Setenv("INGEST_QUEUE_CAP", "0") // invalid → default
	if got := ingestQueueCap(); got != 4096 {
		t.Fatalf("zero override = %d, want 4096 fallback", got)
	}
}

// TestIngestBurst600FpsUnderCapNoDrops — a synthetic 600-frames/s × 5s burst
// (3000 frames) with a FULLY STALLED drainer: the 4096 cap alone absorbs it,
// so zero intrabar drops, zero close drops, and the peak is sampled.
func TestIngestBurst600FpsUnderCapNoDrops(t *testing.T) {
	t.Setenv("INGEST_QUEUE_CAP", "")
	resetIngestCounters()
	s := NewTCPServer(nil)
	// No drain goroutine — the queue must absorb the whole burst by capacity.
	for i := 0; i < 3000; i++ {
		s.enqueueBarUpdate("MNQ", "1m", nil)
	}
	if got := ingestDropOld.Load() + ingestDropCur.Load() + ingestDropHist.Load(); got != 0 {
		t.Fatalf("burst under cap dropped %d frame(s) — the 4096 cap must absorb 3000", got)
	}
	if got := persistDroppedCloses.Load(); got != 0 {
		t.Fatalf("closes_dropped = %d — closed bars must never be lost on this path", got)
	}
	if got := ingestPeakDepth.Load(); got < 3000 {
		t.Fatalf("peak_depth = %d, want ≥ 3000 (the burst high-water mark must be sampled)", got)
	}
	// Drain and release.
	for i := 0; i < 3000; i++ {
		<-s.barIngestCh
	}
}

// TestIngestBurstOverCapCountsIntrabarOnly — an over-cap burst drops exactly
// (sent − cap) OLDEST intra-bar frames, never the current one, and never any
// close — the honest-counter contract the summary line reports.
func TestIngestBurstOverCapCountsIntrabarOnly(t *testing.T) {
	t.Setenv("INGEST_QUEUE_CAP", "")
	resetIngestCounters()
	s := NewTCPServer(nil)
	cap := 4096
	sent := 5000
	for i := 0; i < sent; i++ {
		s.enqueueBarUpdate("MNQ", "1m", nil)
	}
	wantOld := int64(sent - cap)
	if got := ingestDropOld.Load(); got != wantOld {
		t.Fatalf("drop-oldest = %d, want %d", got, wantOld)
	}
	if got := ingestDropCur.Load(); got != 0 {
		t.Fatalf("drop-current = %d — the current frame must always land after an oldest eviction", got)
	}
	if got := persistDroppedCloses.Load(); got != 0 {
		t.Fatalf("closes_dropped = %d — closed bars must never be lost on this path", got)
	}
	if got := ingestPeakDepth.Load(); got != int64(cap) {
		t.Fatalf("peak_depth = %d, want %d (the queue filled to the cap)", got, cap)
	}
	for i := 0; i < cap; i++ {
		<-s.barIngestCh
	}
}
