package kernel

import (
	"testing"
	"time"
)

// P0-B — ONE SESSION INSTANCE = ONE PLAN CHAIN. The chain identity must not
// roll at midnight for a window that wraps midnight, and the pre-open read gap
// (ASIA 16:55, inside the CME maintenance break) belongs to the instance about
// to open.

func ct(t *testing.T, y int, m time.Month, d, hh, mm int) time.Time {
	t.Helper()
	loc, err := time.LoadLocation("America/Chicago")
	if err != nil {
		t.Fatal(err)
	}
	return time.Date(y, m, d, hh, mm, 0, 0, loc)
}

func asiaSess() *SessionDef {
	return &SessionDef{Name: SessionAsia, WindowStartCT: "17:00", WindowEndCT: "02:00", ReadCT: "16:55", FlatCT: "02:00", Enabled: true}
}

func TestPlanChainTradeDateAsiaWrap(t *testing.T) {
	// Tuesday 2026-08-18 is a plain trading day for these fixtures.
	cases := []struct {
		name string
		now  time.Time
		want string
	}{
		{"pre-open read 16:55 belongs to the instance about to open", ct(t, 2026, 8, 18, 16, 55), "2026-08-18"},
		{"at the open 17:00", ct(t, 2026, 8, 18, 17, 0), "2026-08-18"},
		{"mid-session 22:00", ct(t, 2026, 8, 18, 22, 0), "2026-08-18"},
		{"after midnight 00:30 — still the 17:00 instance", ct(t, 2026, 8, 19, 0, 30), "2026-08-18"},
		{"tail end 01:59", ct(t, 2026, 8, 19, 1, 59), "2026-08-18"},
		{"post-close 02:00 rolls to the next instance day", ct(t, 2026, 8, 19, 2, 0), "2026-08-19"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, ok := PlanChainTradeDate(asiaSess(), c.now)
			if !ok || got != c.want {
				t.Fatalf("PlanChainTradeDate(%v) = %q,%v — want %q", c.now.Format("15:04"), got, ok, c.want)
			}
		})
	}
}

func TestPlanChainTradeDateSameDaySessions(t *testing.T) {
	ny := &SessionDef{Name: SessionNY, WindowStartCT: "08:30", WindowEndCT: "14:45", ReadCT: "08:25"}
	cases := []struct {
		name string
		now  time.Time
		want string
	}{
		{"NY pre-open read 08:25", ct(t, 2026, 8, 18, 8, 25), "2026-08-18"},
		{"NY mid-session", ct(t, 2026, 8, 18, 12, 0), "2026-08-18"},
		{"NY after close", ct(t, 2026, 8, 18, 15, 0), "2026-08-18"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, ok := PlanChainTradeDate(ny, c.now)
			if !ok || got != c.want {
				t.Fatalf("PlanChainTradeDate(%v) = %q,%v — want %q", c.now.Format("15:04"), got, ok, c.want)
			}
		})
	}
}

func TestSessionInstanceStartPreOpenVsTail(t *testing.T) {
	// 16:55 → today 17:00 (upcoming instance), 00:30 → yesterday 17:00 (tail).
	up, ok := SessionInstanceStart(asiaSess(), ct(t, 2026, 8, 18, 16, 55))
	if !ok || up.Format("2006-01-02 15:04") != "2026-08-18 17:00" {
		t.Fatalf("pre-open instance start = %v, want 2026-08-18 17:00", up)
	}
	tail, ok := SessionInstanceStart(asiaSess(), ct(t, 2026, 8, 19, 0, 30))
	if !ok || tail.Format("2006-01-02 15:04") != "2026-08-18 17:00" {
		t.Fatalf("tail instance start = %v, want 2026-08-18 17:00", tail)
	}
}
