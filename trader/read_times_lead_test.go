package trader

import (
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	"nofx/kernel"
	"nofx/store"
)

// Lead-time wave (owner ruling 2026-08-31): session reads move to open−30 —
// ASIA 16:30 · LONDON 01:30 · NY 08:00 CT; Sunday sequencing weekly→ASIA.

func ct(h, m int, wd time.Weekday) time.Time {
	loc := kernel.CTLocation()
	// 2026-08-31 is a Monday; walk to the requested weekday.
	base := time.Date(2026, 8, 31, h, m, 0, 0, loc)
	for base.Weekday() != wd {
		base = base.AddDate(0, 0, 1)
	}
	return base
}

func TestDefaultRegistryReadTimesOpenMinus30(t *testing.T) {
	reg := kernel.DefaultSessionRegistry()
	want := map[string]string{
		kernel.SessionAsia:   "16:30",
		kernel.SessionLondon: "01:30",
		kernel.SessionNY:     "08:00",
	}
	got := map[string]string{}
	for _, s := range reg.Sessions {
		got[s.Name] = s.ReadCT
	}
	for name, w := range want {
		if got[name] != w {
			t.Fatalf("%s read = %q, want %q", name, got[name], w)
		}
	}
	// The once-a-day trigger primitive fires at the new minutes.
	for _, s := range reg.Sessions {
		if !s.IsReadTime(ct(16, 30, time.Sunday).Add(0)) && s.Name == kernel.SessionAsia {
			t.Fatalf("%s IsReadTime must fire at its ReadCT", s.Name)
		}
	}
	if !reg.Sessions[0].IsReadTime(ct(16, 30, time.Sunday)) {
		t.Fatalf("ASIA IsReadTime(16:30) = false")
	}
	if !reg.Sessions[1].IsReadTime(ct(1, 30, time.Monday)) {
		t.Fatalf("LONDON IsReadTime(01:30) = false")
	}
	if !reg.Sessions[2].IsReadTime(ct(8, 0, time.Monday)) {
		t.Fatalf("NY IsReadTime(08:00) = false")
	}
}

func TestSundayAsiaDeferred(t *testing.T) {
	asia := &kernel.SessionDef{Name: kernel.SessionAsia}
	ny := &kernel.SessionDef{Name: kernel.SessionNY}
	doc := &kernel.WeeklyDoc{WeeklyLevels: []kernel.WeeklyLevel{{Name: "PWH", Px: 1}, {Name: "PWL", Px: 1}}, Narrative: "facts"}
	// Sunday 16:30, no weekly doc → deferred.
	if !sundayAsiaDeferred(asia, ct(16, 30, time.Sunday), nil) {
		t.Fatal("Sunday ASIA with no weekly doc must defer")
	}
	// Sunday 16:30, weekly doc landed → fires.
	if sundayAsiaDeferred(asia, ct(16, 30, time.Sunday), doc) {
		t.Fatal("Sunday ASIA with a landed weekly doc must fire")
	}
	// Weekday ASIA → never deferred.
	if sundayAsiaDeferred(asia, ct(16, 30, time.Monday), nil) {
		t.Fatal("weekday ASIA must not defer")
	}
	// Non-ASIA → never deferred.
	if sundayAsiaDeferred(ny, ct(16, 30, time.Sunday), nil) {
		t.Fatal("NY must never defer")
	}
	if sundayAsiaDeferred(nil, ct(16, 30, time.Sunday), nil) {
		t.Fatal("nil session must never defer")
	}
}

// TestWeeklyDocCachedNegativeNotSticky — the F1 miss class: a cycle that ran
// before the Sunday weekly read landed used to pin "WEEKLY: none" for the whole
// week. Negative results are no longer cached.
func TestWeeklyDocCachedNegativeNotSticky(t *testing.T) {
	st, err := store.New(filepath.Join(t.TempDir(), "w.db"))
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	defer st.Close()
	at := &AutoTrader{id: "t1", exchange: "ninjatrader", store: st}
	now := ct(16, 30, time.Sunday)

	if at.weeklyDocCached(now) != nil {
		t.Fatal("no doc yet — must return nil")
	}
	monday := kernel.WeekGoverningMonday(now).Format("2006-01-02")
	doc := kernel.WeeklyDoc{WeeklyLevels: []kernel.WeeklyLevel{{Name: "PWH", Px: 1}, {Name: "PWL", Px: 1}}, Narrative: "facts"}
	b, _ := json.Marshal(doc)
	if _, err := st.Plan().AppendPlan(&store.PlanDB{
		PlanID:        st.Plan().ResolvePlanID(monday, "WEEKLY", at.id),
		StrategyID:    at.id,
		TradeDate:     monday,
		Session:       "WEEKLY",
		TriggerReason: "test_weekly",
		Lifecycle:     "active",
		Doc:           string(b),
	}); err != nil {
		t.Fatalf("append weekly: %v", err)
	}
	got := at.weeklyDocCached(now)
	if got == nil {
		t.Fatal("negative result must NOT be cached — the landed doc must be returned on the next call")
	}
	if len(got.WeeklyLevels) != 2 || got.WeeklyLevels[0].Name != "PWH" || got.WeeklyLevels[1].Name != "PWL" {
		t.Fatalf("doc round-trip wrong (class 50 refs-only): %+v", got)
	}
}
