package kernel

import (
	"reflect"
	"testing"
	"time"
)

func ctTime(t *testing.T, h, m int) time.Time {
	t.Helper()
	loc, err := time.LoadLocation("America/Chicago")
	if err != nil {
		t.Fatalf("load chicago: %v", err)
	}
	// 2026-08-14 is a Friday (CDT); the window math is minutes-of-day only, so
	// the calendar date does not matter.
	return time.Date(2026, 8, 14, h, m, 0, 0, loc)
}

func TestDefaultRegistryRows(t *testing.T) {
	r := DefaultSessionRegistry()
	if len(r.Sessions) != 3 {
		t.Fatalf("want 3 sessions, got %d", len(r.Sessions))
	}
	ny, ok := r.SessionByName("ny")
	if !ok {
		t.Fatalf("NY session missing")
	}
	if ny.ReadCT != "08:00" || ny.WindowStartCT != "08:30" || ny.FlatCT != "14:45" {
		t.Fatalf("NY row wrong: %+v", ny)
	}
	if !ny.Enabled {
		t.Fatalf("NY must be enabled by default")
	}
	asia, _ := r.SessionByName("ASIA")
	london, _ := r.SessionByName("LONDON")
	if asia.Enabled || london.Enabled {
		t.Fatalf("ASIA/LONDON must be disabled by default")
	}
	if asia.ReadCT != "16:30" || london.ReadCT != "01:30" {
		t.Fatalf("read times wrong: asia=%s london=%s", asia.ReadCT, london.ReadCT)
	}
	if got := r.EnabledSessions(); !reflect.DeepEqual(got, []string{"NY"}) {
		t.Fatalf("EnabledSessions = %v want [NY]", got)
	}
}

func TestActiveSession(t *testing.T) {
	r := DefaultSessionRegistry()
	cases := []struct {
		h, m int
		want string // "" = none
	}{
		{9, 0, "NY"},    // RTH
		{14, 30, "NY"},  // late RTH
		{15, 30, ""},    // post-close gap
		{16, 30, ""},    // pre-Asia gap / daily break
		{18, 0, "ASIA"}, // Asia evening
		{23, 30, "ASIA"},
		{1, 0, "ASIA"},    // Asia after midnight (wrap)
		{2, 30, "LONDON"}, // London
		{7, 0, "LONDON"},
		{8, 15, "LONDON"}, // 08:15 < 08:30 NY open → still London
	}
	for _, c := range cases {
		got := ""
		if s, ok := r.ActiveSession(ctTime(t, c.h, c.m)); ok {
			got = s.Name
		}
		if got != c.want {
			t.Fatalf("ActiveSession(%02d:%02d) = %q want %q", c.h, c.m, got, c.want)
		}
	}
}

func TestActiveSessionBoundariesExclusive(t *testing.T) {
	r := DefaultSessionRegistry()
	// Window is [start, end): NY starts 08:30, so 08:30 is NY and 15:00 is NOT NY.
	if s, ok := r.ActiveSession(ctTime(t, 8, 30)); !ok || s.Name != "NY" {
		t.Fatalf("08:30 should be NY open")
	}
	if s, ok := r.ActiveSession(ctTime(t, 15, 0)); ok {
		t.Fatalf("15:00 should be outside all windows, got %s", s.Name)
	}
	// ASIA start 17:00 inclusive; 02:00 end exclusive → 02:00 is not ASIA.
	if s, ok := r.ActiveSession(ctTime(t, 17, 0)); !ok || s.Name != "ASIA" {
		t.Fatalf("17:00 should be ASIA open")
	}
	if s, ok := r.ActiveSession(ctTime(t, 2, 0)); !ok || s.Name != "LONDON" {
		t.Fatalf("02:00 should roll to LONDON open, got ok=%v", ok)
	}
}

func TestIsReadTime(t *testing.T) {
	r := DefaultSessionRegistry()
	ny, _ := r.SessionByName("NY")
	if !ny.IsReadTime(ctTime(t, 8, 0)) {
		t.Fatalf("08:00 should be NY read time")
	}
	if ny.IsReadTime(ctTime(t, 7, 59)) || ny.IsReadTime(ctTime(t, 8, 1)) {
		t.Fatalf("only 08:00 exact should be NY read time")
	}
	asia, _ := r.SessionByName("ASIA")
	if !asia.IsReadTime(ctTime(t, 16, 30)) {
		t.Fatalf("16:30 should be ASIA read time")
	}
}

func TestInKillzone(t *testing.T) {
	r := DefaultSessionRegistry()
	ny, _ := r.SessionByName("NY")
	if !ny.InKillzone(ctTime(t, 9, 0)) {
		t.Fatalf("09:00 should be in ny_am killzone")
	}
	if ny.InKillzone(ctTime(t, 12, 0)) {
		t.Fatalf("12:00 (lunch) should not be in any NY killzone")
	}
	if !ny.InKillzone(ctTime(t, 13, 30)) {
		t.Fatalf("13:30 should be in ny_pm killzone")
	}
}

func TestHalfDayFlatOverride(t *testing.T) {
	r := DefaultSessionRegistry()
	// Default day: NY flat = 14:45.
	if flat, ok := r.EffectiveFlatCT("NY", "2026-08-14"); !ok || flat != "14:45" {
		t.Fatalf("default NY flat = %q want 14:45", flat)
	}
	// FOLD (owner ruling 2026-09-07): the override is no longer injected into the
	// registry — it comes from the session calendar, the single owner. And the
	// value is the SOURCED 12:15, not the 12:00 convention guess this test used
	// to hand-feed: CME's archived 2026 calendar gives the day after Thanksgiving
	// an equity FINAL close of 12:15 CT (settlement 12:00).
	if flat, ok := r.EffectiveFlatCT("NY", "2026-11-27"); !ok || flat != "12:15" {
		t.Fatalf("half-day NY flat = %q want 12:15 (sourced, not the 12:00 guess)", flat)
	}
	// A non-half-day still returns the default flat.
	if flat, _ := r.EffectiveFlatCT("NY", "2026-08-14"); flat != "14:45" {
		t.Fatalf("non-half-day flat = %q want 14:45", flat)
	}
}

func TestRegistryRoundTrip(t *testing.T) {
	r := DefaultSessionRegistry()
	raw, err := r.Marshal()
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	back, err := LoadSessionRegistry(raw)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if !reflect.DeepEqual(back, r) {
		t.Fatalf("round-trip mismatch:\n got: %+v\nwant: %+v", back, r)
	}
	// Empty input → default (never empty).
	def, err := LoadSessionRegistry("")
	if err != nil || len(def.Sessions) != 3 {
		t.Fatalf("empty should load default: %+v err=%v", def, err)
	}
	// Malformed input → default + error (fail-safe, never empty).
	bad, err := LoadSessionRegistry("{not json")
	if err == nil {
		t.Fatalf("malformed registry should return an error")
	}
	if len(bad.Sessions) != 3 {
		t.Fatalf("malformed should fall back to default registry")
	}
}

// THE FOLD PIN (owner ruling 2026-09-07): ONE FACT, ONE OWNER.
//
// Before the fold this answer lived in three places — half_days.json at the repo
// root, SessionRegistry.HalfDays in the store, and the session calendar — with
// two key conventions and, on three dates, two different times. The gate stopped
// trading at 12:00 on days the sourced file said 12:15.
//
// This asserts the three surfaces now agree, by value, on the two dates that
// disagreed. It fails the moment a second copy of this fact reappears.
func TestFoldOneFactOneOwner(t *testing.T) {
	cases := []struct{ day, want string }{
		{"2026-11-27", "12:15"}, // day after Thanksgiving — sourced final close
		{"2026-12-24", "12:15"}, // Christmas Eve — sourced final close
	}
	r := DefaultSessionRegistry()
	for _, c := range cases {
		// 1. THE CALENDAR (what the gate reads).
		got, ok := SessionEarlyCloseCTForKey(c.day)
		if !ok || got != c.want {
			t.Errorf("%s calendar early close = %q ok=%v, want %q", c.day, got, ok, c.want)
		}
		// 2. THE REGISTRY / EOD FLAT (what the flatten reads).
		flat, ok2 := r.EffectiveFlatCT("NY", c.day)
		if !ok2 || flat != c.want {
			t.Errorf("%s EffectiveFlatCT = %q ok=%v, want %q", c.day, flat, ok2, c.want)
		}
		// 3. THEY ARE THE SAME VALUE, not merely both correct.
		if got != flat {
			t.Errorf("%s: gate says %q but the EOD flat says %q — the fact has two owners again", c.day, got, flat)
		}
	}
	// Thanksgiving itself is a FULL CLOSURE (owner ruling): no early close at all,
	// and the superseded 12:00 row is recorded in the calendar, not applied.
	if got, ok := SessionEarlyCloseCTForKey("2026-11-26"); ok {
		t.Errorf("2026-11-26 is a full closure — it must expose no early close, got %q", got)
	}
	// A normal day is untouched.
	if flat, _ := r.EffectiveFlatCT("NY", "2026-08-14"); flat != "14:45" {
		t.Errorf("a normal day must keep its configured flat, got %q", flat)
	}
}
