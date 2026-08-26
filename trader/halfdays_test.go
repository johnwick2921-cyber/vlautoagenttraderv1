package trader

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"nofx/kernel"
	"nofx/store"
)

// P4 (ledger-close 2026-08-19) — HalfDays producer + the cutoff pull-forward.

func writeHalfDays(t *testing.T, content string) {
	t.Helper()
	p := filepath.Join(t.TempDir(), "half_days.json")
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("NOFX_HALF_DAYS", p)
}

func TestLoadHalfDaysFileSeedTable(t *testing.T) {
	// The REAL repo seed must load: 4 entries, Labor Day first.
	t.Setenv("NOFX_HALF_DAYS", filepath.Join("..", "half_days.json"))
	entries, err := LoadHalfDaysFile()
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 4 {
		t.Fatalf("seed table must carry 4 official 2026 half-days, got %d", len(entries))
	}
	next, ok := NextUpcomingHalfDay(entries, ctDate(t, "2026-08-19 10:00"))
	if !ok || next.Date != "2026-09-07" || next.EarlyCloseCT != "12:00" {
		t.Fatalf("next half-day from Aug 19 must be Labor Day 2026-09-07 12:00, got %+v", next)
	}
	if !strings.Contains(next.Label, "Labor Day") {
		t.Fatalf("label must name the holiday, got %q", next.Label)
	}
}

func TestLoadHalfDaysMalformedFailsOpen(t *testing.T) {
	writeHalfDays(t, `{"not":"an array"`)
	if _, err := LoadHalfDaysFile(); err == nil {
		t.Fatal("malformed JSON must return an error for the CRITICAL log")
	}
	// A bad ENTRY is dropped; valid siblings survive.
	writeHalfDays(t, `[
	  {"date":"garbage","early_close_ct":"12:00","label":"bad"},
	  {"date":"2026-09-07","early_close_ct":"25:99","label":"bad time"},
	  {"date":"2026-09-07","early_close_ct":"12:00","label":"Labor Day"}]`)
	entries, err := LoadHalfDaysFile()
	if err != nil || len(entries) != 1 || entries[0].Date != "2026-09-07" {
		t.Fatalf("invalid entries must drop, valid survive: %v %+v", err, entries)
	}
	// Missing file → nil, nil (trade normally).
	t.Setenv("NOFX_HALF_DAYS", filepath.Join(t.TempDir(), "nope.json"))
	if entries, err := LoadHalfDaysFile(); err != nil || entries != nil {
		t.Fatalf("missing file must fail open, got %v %+v", err, entries)
	}
}

func TestSeedHalfDaysMergeIsIdempotentAndAdditive(t *testing.T) {
	st, err := store.New(filepath.Join(t.TempDir(), "hd.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })

	// Pre-existing DB-only half-day must survive the merge.
	reg := kernel.DefaultSessionRegistry()
	reg.HalfDays = map[string]string{"2026-10-01": "11:00"}
	raw, _ := jsonMarshalForTest(reg)
	_ = st.SetSystemConfig(kernel.SessionRegistryConfigKey, raw)

	entries := []HalfDayEntry{{Date: "2026-09-07", EarlyCloseCT: "12:00", Label: "Labor Day"}}
	changed, err := SeedHalfDaysIntoRegistry(st, entries)
	if err != nil || changed != 1 {
		t.Fatalf("first merge: changed=%d err=%v", changed, err)
	}
	changed, err = SeedHalfDaysIntoRegistry(st, entries)
	if err != nil || changed != 0 {
		t.Fatalf("second merge must be a no-op, changed=%d err=%v", changed, err)
	}

	stored, _ := st.GetSystemConfig(kernel.SessionRegistryConfigKey)
	got, err := kernel.LoadSessionRegistry(stored)
	if err != nil {
		t.Fatal(err)
	}
	// E7-v2 HIGH fix: the file's CALENDAR date 2026-09-07 lands under the
	// session-day KEY 2026-09-06 (the 17:00 Sun start of Monday's session).
	if got.HalfDays["2026-09-06"] != "12:00" || got.HalfDays["2026-10-01"] != "11:00" {
		t.Fatalf("merge must land calendar dates under session-day keys and keep DB-only keys: %+v", got.HalfDays)
	}
}

// Sep 7 sim: with HalfDays["2026-09-07"]="12:00", the NY session's LAST-ENTRY
// and EOD-FLAT cutoffs resolve against the early close (12:00 − offset), while
// a normal day is unchanged — exact values asserted.
func TestHalfDayPullsNYCutoffsForward(t *testing.T) {
	reg := kernel.DefaultSessionRegistry()
	reg.HalfDays = map[string]string{"2026-09-07": "12:00"}

	// last-entry offset 30 → normal cutoff 14:15, half-day cutoff 11:30.
	adj, hhmm, ok := halfDayCutoffMin(reg, "2026-09-07", 30)
	if !ok || adj != 11*60+30 || hhmm != "11:30" {
		t.Fatalf("early 12:00 − 30min must be 11:30, got %q ok=%v", hhmm, ok)
	}
	// eod-flat offset 0 → flat pulled to exactly 12:00.
	adj, hhmm, ok = halfDayCutoffMin(reg, "2026-09-07", 0)
	if !ok || adj != 12*60 || hhmm != "12:00" {
		t.Fatalf("early close with offset 0 must be 12:00, got %q", hhmm)
	}
	// Normal day: no half-day entry → ok=false → cutoffs untouched.
	if _, _, ok := halfDayCutoffMin(reg, "2026-09-08", 30); ok {
		t.Fatal("a normal day must not resolve a half-day cutoff")
	}
	// ASIA is unaffected by min-semantics: its cutoff (01:45 = 105) is already
	// earlier than 12:00−offset, so callers keep the smaller session value.
	if adj, _, ok := halfDayCutoffMin(reg, "2026-09-07", 15); !ok || adj <= 105 {
		t.Fatalf("the 11:45 adj (%d) must be LARGER than ASIA's 105 — callers min() it away", adj)
	}
	// Garbage half-day value fail-safes (never invents a cutoff).
	reg.HalfDays["2026-09-07"] = "junk"
	if _, _, ok := halfDayCutoffMin(reg, "2026-09-07", 0); ok {
		t.Fatal("unparseable half-day must fail safe")
	}
}

// Registry validation now rejects garbage HalfDays through the API door.
func TestValidateSessionRegistryChecksHalfDays(t *testing.T) {
	reg := kernel.DefaultSessionRegistry()
	reg.HalfDays = map[string]string{"2026-09-07": "12:00"}
	if err := kernel.ValidateSessionRegistry(reg); err != nil {
		t.Fatalf("valid half-days must pass: %v", err)
	}
	reg.HalfDays = map[string]string{"garbage": "12:00"}
	if err := kernel.ValidateSessionRegistry(reg); err == nil {
		t.Fatal("a non-date key must be rejected")
	}
	reg.HalfDays = map[string]string{"2026-09-07": "25:99"}
	if err := kernel.ValidateSessionRegistry(reg); err == nil {
		t.Fatal("a non-HH:MM value must be rejected")
	}
}

func jsonMarshalForTest(reg kernel.SessionRegistry) (string, error) {
	b, err := json.Marshal(reg)
	return string(b), err
}


// E7-v2 HIGH regression — the calendar-vs-session-day-key skew: seeding the
// CME calendar date must protect the REAL half-day (Monday daytime) and never
// the next full session (Tuesday).
func TestHalfDaySeedUsesSessionDayKeys(t *testing.T) {
	key, ok := sessionDayKeyForCalendarDate("2026-09-07")
	if !ok || key != "2026-09-06" {
		t.Fatalf("calendar 2026-09-07 must convert to session-day key 2026-09-06, got %q ok=%v", key, ok)
	}

	reg := kernel.DefaultSessionRegistry()
	reg.HalfDays = map[string]string{key: "12:00"}

	monday := ctDate(t, "2026-09-07 11:00") // the actual Labor Day daytime
	if got := kernel.CMESessionDayKey(monday); got != key {
		t.Fatalf("Monday daytime must key the Sunday-start session, got %q", got)
	}
	if _, _, ok := halfDayCutoffMin(reg, kernel.CMESessionDayKey(monday), 0); !ok {
		t.Fatal("the pull-in must HIT on the real half-day")
	}
	tuesday := ctDate(t, "2026-09-08 11:00") // full session — must be untouched
	if _, _, ok := halfDayCutoffMin(reg, kernel.CMESessionDayKey(tuesday), 0); ok {
		t.Fatal("Tuesday (a normal full session) must MISS — the wrong-day 12:00 flatten class")
	}
}

// E7-v2 fix — the pull-in compares in SESSION-RELATIVE space: an early close
// outside the session's window truncates nothing (no pull-in, original cutoff
// KEPT), and wrapped sessions order correctly.
func TestHalfDayPullsInSessionRelative(t *testing.T) {
	ny := &kernel.SessionDef{Name: "NY", WindowStartCT: "08:30", WindowEndCT: "14:45"}
	// 12:00 early close, NY cutoff 14:15 → pulls in.
	if !halfDayPullsIn(ny, 12*60, 14*60+15) {
		t.Fatal("an in-window earlier close must pull the NY cutoff in")
	}
	// Custom evening session entirely AFTER the 12:00 early close: no pull-in —
	// previously the raw compare replaced 22:30 with 12:00 and pastSessionCutoff
	// could never fire (protection silently REMOVED).
	evening := &kernel.SessionDef{Name: "EVE", WindowStartCT: "18:00", WindowEndCT: "23:00"}
	if halfDayPullsIn(evening, 12*60, 22*60+30) {
		t.Fatal("an out-of-window early close must NOT touch the session's cutoff")
	}
	// Wrapped ASIA (17:00→02:00): a 01:30 early close vs 01:45 cutoff → pulls in
	// (session-relative beats the raw-minute inversion), and 12:00 does not.
	asia := &kernel.SessionDef{Name: "ASIA", WindowStartCT: "17:00", WindowEndCT: "02:00"}
	if !halfDayPullsIn(asia, 1*60+30, 1*60+45) {
		t.Fatal("wrapped session: 01:30 must pull in a 01:45 cutoff")
	}
	if halfDayPullsIn(asia, 12*60, 1*60+45) {
		t.Fatal("wrapped session: an out-of-window 12:00 close must not touch ASIA")
	}
}

// E7-v2 medium fix — deleting a row from half_days.json prunes the producer-
// owned registry key on the next seed; DB-only (admin) keys survive.
func TestHalfDayDeletionHonored(t *testing.T) {
	st, err := store.New(filepath.Join(t.TempDir(), "hdp.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })

	// Admin-written key, never owned by the producer.
	reg := kernel.DefaultSessionRegistry()
	reg.HalfDays = map[string]string{"2026-10-01": "11:00"}
	raw, _ := jsonMarshalForTest(reg)
	_ = st.SetSystemConfig(kernel.SessionRegistryConfigKey, raw)

	both := []HalfDayEntry{
		{Date: "2026-09-07", EarlyCloseCT: "12:00", Label: "Labor Day"},
		{Date: "2026-11-26", EarlyCloseCT: "12:00", Label: "Thanksgiving"},
	}
	if _, err := SeedHalfDaysIntoRegistry(st, both); err != nil {
		t.Fatal(err)
	}
	// The owner deletes Thanksgiving from the file → next seed prunes its key.
	if changed, err := SeedHalfDaysIntoRegistry(st, both[:1]); err != nil || changed != 1 {
		t.Fatalf("prune must count as a change: changed=%d err=%v", changed, err)
	}
	stored, _ := st.GetSystemConfig(kernel.SessionRegistryConfigKey)
	got, _ := kernel.LoadSessionRegistry(stored)
	if _, alive := got.HalfDays["2026-11-25"]; alive { // 11-26's session-day key
		t.Fatal("deleted file row must leave the registry")
	}
	if got.HalfDays["2026-09-06"] != "12:00" || got.HalfDays["2026-10-01"] != "11:00" {
		t.Fatalf("kept file key + admin key must survive: %+v", got.HalfDays)
	}
}
