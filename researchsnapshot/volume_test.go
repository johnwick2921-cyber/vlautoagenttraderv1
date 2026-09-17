package researchsnapshot

import (
	"context"
	"os"
	"strings"
	"sync"
	"testing"
	"time"
)

// B1: the OFF note must never print when Start() was simply not called yet.
func TestVolumeBootLineNABeforeStart(t *testing.T) {
	startCalled.Store(false) // fresh-process semantics: Start never ran
	Install(nil)
	if !strings.Contains(CurrentBootLine(), "n/a") {
		t.Fatalf("boot line must read n/a when Start() was never called; got %q", CurrentBootLine())
	}
}

type captureSink struct {
	mu    sync.Mutex
	facts []Fact
}

func (s *captureSink) Save(_ context.Context, facts []Fact) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.facts = append(s.facts, facts...)
	return nil
}

type lineCapture struct {
	mu    sync.Mutex
	lines []string
}

func (l *lineCapture) add(s string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.lines = append(l.lines, s)
}

func (l *lineCapture) snapshot() []string {
	l.mu.Lock()
	defer l.mu.Unlock()
	out := make([]string, len(l.lines))
	copy(out, l.lines)
	return out
}

// RED 1+4: the recorder narrates per-fact at INFO (13.2M lines/day). The volume
// contract: ONE rollup line per RESEARCH_LOG_EVERY_S (default 60s), carrying
// rows-per-object, the real dropped count, and the queue depth — never a
// per-fact line.
func TestVolumeRollupNotPerFact(t *testing.T) {
	t.Setenv("RESEARCH_LOG_EVERY_S", "60")
	sink := &captureSink{}
	log := &lineCapture{}
	r := NewRecorder(sink, 128, log.add)
	defer r.Close()

	now := time.Date(2026, 9, 16, 12, 0, 0, 0, time.UTC)
	clock := func() time.Time { return now }
	for i := 0; i < 20; i++ {
		f := NewFact("market", "test", nil, Clocks{})
		if !r.offerWithClock(clock, "market", func() []Fact { return []Fact{f} }) {
			t.Fatalf("offer %d refused", i)
		}
	}
	r.drop("queue full: market") // 3 synthetic drops for the dropped= assertion
	r.drop("queue full: market")
	r.drop("queue full: market")
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := r.Flush(ctx); err != nil {
		t.Fatalf("flush: %v", err)
	}
	time.Sleep(50 * time.Millisecond) // let the rollup timer/drain settle

	lines := log.snapshot()
	perFact := 0
	rollup := ""
	for _, l := range lines {
		if strings.Contains(l, "research snapshot written:") {
			perFact++
		}
		if strings.Contains(l, "research snapshot rollup") {
			rollup = l
		}
	}
	if perFact > 0 {
		t.Fatalf("per-fact narration must be gone: %d line(s) still contain 'research snapshot written:' (first: %q)", perFact, firstContaining(lines, "research snapshot written:"))
	}
	if rollup == "" {
		t.Fatal("no rollup line emitted; want one per RESEARCH_LOG_EVERY_S with rows/objects/drops/queue")
	}
	if !strings.Contains(rollup, "rows=20") {
		t.Fatalf("rollup must carry the row count; got %q", rollup)
	}
	if !strings.Contains(rollup, "market=20") {
		t.Fatalf("rollup must carry rows per object; got %q", rollup)
	}
	if !strings.Contains(rollup, "drops=3") {
		t.Fatalf("rollup must carry the REAL dropped count (3 forced); got %q", rollup)
	}
	if !strings.Contains(rollup, "queue=0") {
		t.Fatalf("rollup must carry the queue depth; got %q", rollup)
	}
}

// RED 2: drop notices must reach the WARN-level func passed from main.go,
// rate-limited to ONE line per minute carrying the coalesced delta — not one
// INFO line per drop.
func TestVolumeDropNoticeWarnRateLimited(t *testing.T) {
	sink := &captureSink{}
	warn := &lineCapture{}
	r := NewRecorder(sink, 1, warn.add)
	defer r.Close()

	for i := 0; i < 5; i++ {
		r.drop("queue full: market")
	}
	deadline := time.Now().Add(2 * time.Second)
	var lines []string
	for time.Now().Before(deadline) {
		lines = warn.snapshot()
		if len(lines) > 0 {
			break
		}
		time.Sleep(25 * time.Millisecond)
	}
	// settle the coalescing window: no further lines may appear for a burst
	time.Sleep(250 * time.Millisecond)
	lines = warn.snapshot()
	dropLines := 0
	joined := ""
	for _, l := range lines {
		if strings.Contains(l, "research snapshot dropped") {
			dropLines++
			joined = l
		}
	}
	if dropLines == 0 {
		t.Fatal("no drop notice reached the warn func; want a WARN line with the coalesced delta")
	}
	if dropLines > 1 {
		t.Fatalf("drop notices must be rate-limited to one line per minute: got %d line(s); last: %q", dropLines, joined)
	}
	if !strings.Contains(joined, "total=5") {
		t.Fatalf("the single drop line must carry the coalesced total (5); got %q", joined)
	}
}

// RED 3: RESEARCH_SNAPSHOT env gate — unset or 0 means the recorder is not
// started and the boot line says OFF.
func TestVolumeResearchSnapshotEnvGate(t *testing.T) {
	t.Run("explicit zero is OFF", func(t *testing.T) {
		t.Setenv("RESEARCH_SNAPSHOT", "0")
		log := &lineCapture{}
		closeFn := Start(t.TempDir()+"/r.db", log.add, log.add)
		defer closeFn()
		if Active() != nil {
			t.Fatal("recorder must NOT start when RESEARCH_SNAPSHOT=0; Active() is non-nil")
		}
		if !strings.Contains(CurrentBootLine(), "research snapshot: OFF (RESEARCH_SNAPSHOT=0)") {
			t.Fatalf("boot line must say OFF (RESEARCH_SNAPSHOT=0) when the gate is closed; got %q", CurrentBootLine())
		}
	})
	t.Run("explicit false is OFF", func(t *testing.T) {
		t.Setenv("RESEARCH_SNAPSHOT", "false")
		log := &lineCapture{}
		closeFn := Start(t.TempDir()+"/r.db", log.add, log.add)
		defer closeFn()
		if Active() != nil {
			t.Fatal("recorder must NOT start when RESEARCH_SNAPSHOT=false")
		}
	})
	t.Run("unset stays ON (opt-out, not opt-in)", func(t *testing.T) {
		os.Unsetenv("RESEARCH_SNAPSHOT")
		log := &lineCapture{}
		closeFn := Start(t.TempDir()+"/r.db", log.add, log.add)
		defer closeFn()
		if Active() == nil {
			t.Fatal("recorder must START when RESEARCH_SNAPSHOT is unset (opt-out); Active() is nil")
		}
		lines := log.snapshot()
		found := false
		for _, l := range lines {
			if strings.Contains(l, "research snapshot: ON (default)") {
				found = true
			}
		}
		if !found {
			t.Fatalf("boot line must read ON (default) when the env is unset; got %v", lines)
		}
	})
}

// RED 5: RESEARCH_RETAIN_DAYS (default 7) prunes research_facts by captured_ms
// at boot and daily; VACUUM is never run automatically.
func TestVolumeRetentionPrunesAtStart(t *testing.T) {
	t.Setenv("RESEARCH_SNAPSHOT", "1")
	t.Setenv("RESEARCH_RETAIN_DAYS", "7")
	path := t.TempDir() + "/r.db"
	a, err := Open(path, "")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	// One row 8 days old, one row now — inserted directly so captured_ms is
	// under test control (Save stamps now).
	old := time.Now().Add(-8 * 24 * time.Hour).UnixMilli()
	if _, err := a.db.Exec(`INSERT INTO research_facts (writer_revision, schema_version, object, snapshot_id, event, captured_ms, fields_json, missing_json, null_fields) VALUES ('', 1, 'market', NULL, 'test', ?, '{}', '{}', 0), ('', 1, 'market', NULL, 'test', ?, '{}', '{}', 0)`, old, time.Now().UnixMilli()); err != nil {
		t.Fatalf("seed: %v", err)
	}
	_ = a.Close()

	log := &lineCapture{}
	closeFn := Start(path, log.add, log.add)
	defer closeFn()

	count := func() int {
		a2, err := Open(path, "")
		if err != nil {
			t.Fatalf("reopen: %v", err)
		}
		defer a2.Close()
		var n int
		if err := a2.db.QueryRow(`SELECT COUNT(*) FROM research_facts WHERE captured_ms < ?`, time.Now().Add(-7*24*time.Hour).UnixMilli()).Scan(&n); err != nil {
			t.Fatalf("count: %v", err)
		}
		return n
	}
	deadline := time.Now().Add(5 * time.Second)
	for count() != 0 && time.Now().Before(deadline) {
		time.Sleep(50 * time.Millisecond)
	}
	if got := count(); got != 0 {
		t.Fatalf("rows older than RESEARCH_RETAIN_DAYS must be pruned after boot; %d remain", got)
	}
}

// B2: RESEARCH_RETAIN_DAYS unset means NO prune, ever.
func TestVolumeRetentionUnsetPrunesNothing(t *testing.T) {
	t.Setenv("RESEARCH_SNAPSHOT", "1")
	os.Unsetenv("RESEARCH_RETAIN_DAYS")
	path := t.TempDir() + "/r.db"
	a, err := Open(path, "")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	old := time.Now().Add(-8 * 24 * time.Hour).UnixMilli()
	if _, err := a.db.Exec(`INSERT INTO research_facts (writer_revision, schema_version, object, snapshot_id, event, captured_ms, fields_json, missing_json, null_fields) VALUES ('', 1, 'market', NULL, 'test', ?, '{}', '{}', 0)`, old); err != nil {
		t.Fatalf("seed: %v", err)
	}
	_ = a.Close()

	log := &lineCapture{}
	closeFn := Start(path, log.add, log.add)
	defer closeFn()
	time.Sleep(300 * time.Millisecond)

	a2, err := Open(path, "")
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	defer a2.Close()
	var n int
	if err := a2.db.QueryRow(`SELECT COUNT(*) FROM research_facts WHERE captured_ms < ?`, time.Now().Add(-7*24*time.Hour).UnixMilli()).Scan(&n); err != nil {
		t.Fatalf("count: %v", err)
	}
	if n != 1 {
		t.Fatalf("retention unset must prune NOTHING; %d rows changed", 1-n)
	}
}

// B3: the periodic rollup and drop-warn must fire WITHOUT any Flush call —
// the ticker is the production path.
func TestVolumeRollupTickerFiresWithoutFlush(t *testing.T) {
	t.Setenv("RESEARCH_LOG_EVERY_S", "1")
	sink := &captureSink{}
	log := &lineCapture{}
	r := NewRecorder(sink, 128, log.add)
	defer r.Close()
	now := time.Date(2026, 9, 16, 12, 0, 0, 0, time.UTC)
	clock := func() time.Time { return now }
	r.offerWithClock(clock, "market", func() []Fact { return []Fact{NewFact("market", "tick", nil, Clocks{})} })
	r.drop("queue full: market")

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		lines := log.snapshot()
		hasRollup := false
		hasWarn := false
		for _, l := range lines {
			if strings.Contains(l, "research snapshot rollup") {
				hasRollup = true
			}
			if strings.Contains(l, "WARN research snapshot dropped") {
				hasWarn = true
			}
		}
		if hasRollup && hasWarn {
			return // GREEN
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("ticker path never emitted without Flush; got %v", log.snapshot())
}

func firstContaining(lines []string, sub string) string {
	for _, l := range lines {
		if strings.Contains(l, sub) {
			return l
		}
	}
	return "<none>"
}

// Re-review pin: the rollup ticker must be STOPPED on shutdown, and no rollup
// line may fire after Close returns. RED on d2f0a8cf (the ticker leaked — no
// r.rollupT.Stop() existed anywhere in the package).
func TestVolumeRollupStopsOnClose(t *testing.T) {
	t.Setenv("RESEARCH_LOG_EVERY_S", "1")
	sink := &captureSink{}
	log := &lineCapture{}
	r := NewRecorder(sink, 128, log.add)

	now := time.Date(2026, 9, 16, 12, 0, 0, 0, time.UTC)
	clock := func() time.Time { return now }
	r.offerWithClock(clock, "market", func() []Fact { return []Fact{NewFact("market", "tick", nil, Clocks{})} })
	time.Sleep(1200 * time.Millisecond) // let one 1s rollup fire
	before := len(log.snapshot())
	if before == 0 {
		t.Fatal("expected at least one rollup line before Close")
	}

	r.Close()
	after := len(log.snapshot())        // Close emits its final rollup — count from here
	time.Sleep(1500 * time.Millisecond) // a leaked ticker would fire here
	if n := len(log.snapshot()); n != after {
		t.Fatalf("rollup fired after Close returned: %d -> %d lines", after, n)
	}
	if !r.rollupStopped.Load() {
		t.Fatalf("rollup ticker not stopped on shutdown (leak): rollupStopped=false after Close")
	}
}
