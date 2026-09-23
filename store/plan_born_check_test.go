package store

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// W-EXEC-TRUTH W2 A1 — the grammar refusal is its own counted kind, separate
// from the tape-UNKNOWN authored_unknown count; an unlisted kind stays refused.
func TestLivenessGrammarRefusalCountedSeparately(t *testing.T) {
	st := newPlanTestStore(t)
	now, _ := time.Parse(time.RFC3339, "2026-09-23T01:51:47-05:00")
	for _, id := range []string{"t:2026-09-23:LONDON:S1:1", "t:2026-09-23:LONDON:S2:1"} {
		if _, err := st.RecordPlanLivenessEvent(LivenessAuthoredGrammarRefusal, id, now, "refused"); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := st.RecordPlanLivenessEvent(LivenessAuthoredUnknown, "t:tape", now, "missing minute"); err != nil {
		t.Fatal(err)
	}
	if _, err := st.RecordPlanLivenessEvent("authored_grammar_refusals", "x", now, "typo kind"); err == nil {
		t.Fatal("an unlisted kind must still be refused")
	}
	c, err := st.PlanLivenessCounts()
	if err != nil || c.GrammarRefusals != 2 || c.AuthoredUnknown != 1 {
		t.Fatalf("counts: %+v %v", c, err)
	}
}

// W2 A2 — a refused attempt carries the born-check record in its event
// detail; with no record the stored JSON is exactly the legacy shape.
func TestLivenessEventCarriesBornCheck(t *testing.T) {
	st := newPlanTestStore(t)
	now, _ := time.Parse(time.RFC3339, "2026-09-23T01:51:47-05:00")
	check := `{"policy":"enforced (grammar)","read_clock_ms":1790145027000,"publish_clock_ms":1790146307000,"groups":[1790145000000,1790145300000,1790145600000,1790145900000],"verdicts":[]}`
	if _, err := st.RecordPlanLivenessEventWithCheck(LivenessBornDeadRefusal, "with", now, "refused", check); err != nil {
		t.Fatal(err)
	}
	if _, err := st.RecordPlanLivenessEvent(LivenessBornDeadRefusal, "legacy", now, "refused"); err != nil {
		t.Fatal(err)
	}
	if _, err := st.RecordPlanLivenessEventWithCheck(LivenessBornDeadRefusal, "bad", now, "refused", "{not json"); err == nil {
		t.Fatal("a non-JSON record must not be stored as one")
	}
	read := func(prefix string) string {
		var v string
		st.gdb.Raw("SELECT value FROM system_config WHERE substr(key,1,?)=?", len(prefix), prefix).Scan(&v)
		return v
	}
	with := read(LivenessEventPrefix + LivenessBornDeadRefusal + ":with:")
	var got struct {
		BornCheck struct {
			ReadClockMs *int64  `json:"read_clock_ms"`
			Groups      []int64 `json:"groups"`
		} `json:"born_check"`
	}
	if json.Unmarshal([]byte(with), &got) != nil || got.BornCheck.ReadClockMs == nil || *got.BornCheck.ReadClockMs != 1790145027000 || len(got.BornCheck.Groups) != 4 {
		t.Fatalf("event detail lost the born check: %s", with)
	}
	legacyWant, _ := json.Marshal(struct {
		At     time.Time `json:"at"`
		Detail string    `json:"detail"`
	}{now, "refused"})
	if legacy := read(LivenessEventPrefix + LivenessBornDeadRefusal + ":legacy:"); legacy != string(legacyWant) {
		t.Fatalf("legacy event JSON changed:\n got %s\nwant %s", legacy, legacyWant)
	}
}

// W2 A2 — the plan row stores read/publish clocks + the born check; a row
// written without them (pre-W2 / fail-closed) reads NULL, never zero.
func TestPlanRowBornCheckColumnsNullable(t *testing.T) {
	st := newPlanTestStore(t)
	ps := st.Plan()
	read, pub, check := int64(1790145027000), int64(1790146307000), `{"policy":"enforced (grammar)","groups":[1790145000000]}`
	p := samplePlan("2026-09-23:LONDON")
	p.ReadClockMs, p.PublishClockMs, p.BornCheck = &read, &pub, &check
	if _, err := ps.AppendPlan(p); err != nil {
		t.Fatal(err)
	}
	if _, err := ps.AppendPlan(samplePlan("2026-09-23:LONDON")); err != nil {
		t.Fatal(err)
	}
	v1, err := ps.GetPlan("2026-09-23:LONDON", 1)
	if err != nil || v1.ReadClockMs == nil || *v1.ReadClockMs != read || *v1.PublishClockMs != pub || *v1.BornCheck != check {
		t.Fatalf("born-check columns did not round-trip: %+v %v", v1, err)
	}
	v2, _ := ps.GetPlan("2026-09-23:LONDON", 2)
	if v2 == nil || v2.ReadClockMs != nil || v2.PublishClockMs != nil || v2.BornCheck != nil {
		t.Fatalf("a row without a born check must read NULL: %+v", v2)
	}
}

// An existing DB whose plans table predates W2 gains the columns at open
// (ALTER TABLE ADD COLUMN; the duplicate-column error on re-open is swallowed).
func TestPlanBornCheckColumnsMigrateExistingTable(t *testing.T) {
	path := filepath.Join(t.TempDir(), "old.db")
	st, err := New(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range []string{"read_clock_ms", "publish_clock_ms", "born_check"} {
		if err := st.gdb.Exec("ALTER TABLE plans DROP COLUMN " + c).Error; err != nil {
			t.Fatalf("simulate pre-W2 table: %v", err)
		}
	}
	st.Plan().Close()
	_ = st.Close()
	st, err = New(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Plan().Close(); _ = st.Close() })
	var cols []struct{ Name string }
	st.gdb.Raw("PRAGMA table_info(plans)").Scan(&cols)
	have := map[string]bool{}
	for _, c := range cols {
		have[c.Name] = true
	}
	for _, c := range []string{"read_clock_ms", "publish_clock_ms", "born_check"} {
		if !have[c] {
			t.Fatalf("column %s not added to a pre-W2 plans table (have %v)", c, cols)
		}
	}
	if _, err := st.Plan().AppendPlan(samplePlan("2026-09-23:NY")); err != nil && !strings.Contains(err.Error(), "UNIQUE") {
		t.Fatalf("append after migration: %v", err)
	}
}
