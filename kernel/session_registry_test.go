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
	if ny.ReadCT != "08:25" || ny.WindowStartCT != "08:30" || ny.FlatCT != "14:45" {
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
	if asia.ReadCT != "16:55" || london.ReadCT != "01:55" {
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
	if !ny.IsReadTime(ctTime(t, 8, 25)) {
		t.Fatalf("08:25 should be NY read time")
	}
	if ny.IsReadTime(ctTime(t, 8, 24)) || ny.IsReadTime(ctTime(t, 8, 26)) {
		t.Fatalf("only 08:25 exact should be NY read time")
	}
	asia, _ := r.SessionByName("ASIA")
	if !asia.IsReadTime(ctTime(t, 16, 55)) {
		t.Fatalf("16:55 should be ASIA read time")
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
	// Half-day override (e.g. day after Thanksgiving early close 12:00 CT).
	r.HalfDays = map[string]string{"2026-11-27": "12:00"}
	if flat, ok := r.EffectiveFlatCT("NY", "2026-11-27"); !ok || flat != "12:00" {
		t.Fatalf("half-day NY flat = %q want 12:00", flat)
	}
	// A non-half-day still returns the default flat.
	if flat, _ := r.EffectiveFlatCT("NY", "2026-08-14"); flat != "14:45" {
		t.Fatalf("non-half-day flat = %q want 14:45", flat)
	}
}

func TestRegistryRoundTrip(t *testing.T) {
	r := DefaultSessionRegistry()
	r.HalfDays = map[string]string{"2026-12-24": "12:00"}
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
