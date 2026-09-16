package researchsnapshot

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

type testSink struct {
	mu    sync.Mutex
	facts []Fact
	write func([]Fact) error
}

func (s *testSink) Save(_ context.Context, facts []Fact) error {
	if s.write != nil {
		return s.write(facts)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.facts = append(s.facts, facts...)
	return nil
}

func TestResearchNullIsNotZero(t *testing.T) {
	f := NewFact("candidate", "scorer", nil, Clocks{})
	f.Set("raw_score_components", map[string]any{"confluence": 0})
	b, err := json.Marshal(f)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(b, []byte(`"final_score":null`)) || !bytes.Contains(b, []byte(`"confluence":0`)) {
		t.Fatalf("NULL versus computed zero lost: %s", b)
	}
}

func TestResearchExportFourClocksAndEmptyRange(t *testing.T) {
	a, err := Open(filepath.Join(t.TempDir(), "research.db"), "test-revision")
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	// Late backfill: event 10:00, receipt 10:30, unrelated publication/permission.
	observed, received := int64(1788879600000), int64(1788881400000)
	f := NewFact("market", "late_backfill", Value("snapshot-a"), Clocks{
		ObservationMS: &observed, ReceiptMS: &received,
		PublicationMS: Value(received + 60000), PermissionMS: Value(received + 120000)})
	f.Set("close", 0.0)
	if err := a.Save(context.Background(), []Fact{f}); err != nil {
		t.Fatal(err)
	}
	x, err := a.Export(context.Background(), received, received+1)
	if err != nil {
		t.Fatal(err)
	}
	y, err := a.Export(context.Background(), received, received+1)
	if err != nil || !bytes.Equal(x, y) {
		t.Fatalf("export not reproducible: %v", err)
	}
	if err := VerifyBundle(x); err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(x, []byte(`"observation_ms":1788879600000`)) || !bytes.Contains(x, []byte(`"receipt_ms":1788881400000`)) {
		t.Fatalf("distinct clocks lost: %s", x)
	}
	early, err := a.Export(context.Background(), observed, received)
	if err != nil {
		t.Fatal(err)
	}
	if err := VerifyBundle(early); err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(early, []byte(`"market":0`)) {
		t.Fatalf("backfill credited before receipt: %s", early)
	}
	if bytes.Equal(x, early) {
		t.Fatal("empty range has the populated bundle")
	}
}

func TestResearchRecorderContainsFaults(t *testing.T) {
	for _, fault := range []string{"panic", "db_error"} {
		t.Run(fault, func(t *testing.T) {
			s := &testSink{write: func([]Fact) error {
				if fault == "panic" {
					panic("injected recorder panic")
				}
				return errors.New("injected database failure")
			}}
			var warnings []string
			r := NewRecorder(s, 2, func(message string) { warnings = append(warnings, message) })
			defer r.Close()
			continued := false
			r.Offer("fixture", func() []Fact { return []Fact{NewFact("plan", "fixture", nil, Clocks{})} })
			continued = true
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			if err := r.Flush(ctx); err != nil {
				t.Fatal(err)
			}
			warned := false
			for _, message := range warnings {
				if strings.Contains(message, "WARN research snapshot dropped:") {
					warned = true
				}
			}
			if !warned {
				t.Fatal("recorder failure did not emit a WARN")
			}
			if !continued || r.Dropped() != 1 {
				t.Fatalf("loop continuation=%v dropped=%d", continued, r.Dropped())
			}
		})
	}
}

func TestResearchOfferNeverWaitsForWriter(t *testing.T) {
	entered, release := make(chan struct{}), make(chan struct{})
	s := &testSink{write: func([]Fact) error { close(entered); <-release; return nil }}
	r := NewRecorder(s, 1, func(string) {})
	defer r.Close()
	defer func() {
		select {
		case <-release:
		default:
			close(release)
		}
	}()
	build := func() []Fact { return []Fact{NewFact("plan", "fixture", nil, Clocks{})} }
	if !r.Offer("first", build) {
		t.Fatal("first offer refused")
	}
	<-entered
	if !r.Offer("queued", func() []Fact { return nil }) {
		t.Fatal("queue slot not available")
	}
	result := make(chan bool, 1)
	start := time.Now()
	go func() { result <- r.Offer("full", build) }()
	select {
	case accepted := <-result:
		if accepted {
			t.Fatal("full queue accepted a record")
		}
		if took := time.Since(start); took > OfferBudget {
			t.Fatalf("offer blocked for %s, budget %s", took, OfferBudget)
		}
	case <-time.After(OfferBudget):
		t.Fatal("offer blocked behind the writer beyond admission budget")
	}

	close(release)
	if r.Dropped() != 1 {
		t.Fatalf("drop not counted: %d", r.Dropped())
	}
}

func TestResearchOfferExceededCaptureBudgetDrops(t *testing.T) {
	r := NewRecorder(&testSink{}, 1, func(string) {})
	defer r.Close()
	start := time.Date(2026, 9, 8, 20, 0, 0, 0, time.UTC)
	call := 0
	clock := func() time.Time {
		call++
		if call == 1 {
			return start
		}
		return start.Add(OfferBudget + time.Millisecond)
	}
	if r.offerWithClock(clock, "late", func() []Fact { return nil }) {
		t.Fatal("over-budget capture was enqueued")
	}
	if r.Dropped() != 1 {
		t.Fatalf("over-budget drop not counted: %d", r.Dropped())
	}
}

func TestResearchExportDetectsTampering(t *testing.T) {
	a, err := Open(filepath.Join(t.TempDir(), "research.db"), "revision")
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	x, err := a.Export(context.Background(), 0, 1)
	if err != nil {
		t.Fatal(err)
	}
	broken := strings.Replace(string(x), `"market":0`, `"market":1`, 1)
	if err := VerifyBundle([]byte(broken)); err == nil {
		t.Fatal("altered count passed checksum/count verification")
	}
}

func TestResearchSQLNullAndComputedZero(t *testing.T) {
	a, err := Open(filepath.Join(t.TempDir(), "research.db"), "null-fixture")
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	f := NewFact("candidate", "fixture", nil, Clocks{})
	f.Set("confluence_raw", 0)
	if err = a.Save(context.Background(), []Fact{f}); err != nil {
		t.Fatal(err)
	}
	var missing any
	var zero int
	if err = a.db.QueryRow(`SELECT json_extract(fields_json,'$.final_score'),json_extract(fields_json,'$.confluence_raw') FROM research_facts`).Scan(&missing, &zero); err != nil {
		t.Fatal(err)
	}
	if missing != nil || zero != 0 {
		t.Fatalf("SQL null/zero contract broken: missing=%v zero=%d", missing, zero)
	}
}
