package trader

import (
	"testing"
	"time"

	"nofx/kernel"
)

// P3 (ledger-close 2026-08-19) — contract-roll gate for continuous "MNQ".

// Third-Friday table, verified against CME's ProductCalendar API for product
// 146 (NQU26 lastTrade = 18 Sep 2026; archived original cited in
// contract_roll.go). DEC26/MAR27/JUN27 follow the same quarterly rule.
func TestThirdFridayTable(t *testing.T) {
	cases := []struct {
		y, m int
		want string
	}{
		{2026, 9, "2026-09-18"},  // SEP26 — CME ProductCalendar NQU26
		{2026, 12, "2026-12-18"}, // DEC26
		{2027, 3, "2027-03-19"},  // MAR27
		{2027, 6, "2027-06-18"},  // JUN27
	}
	for _, c := range cases {
		if got := thirdFriday(c.y, c.m).Format("2006-01-02"); got != c.want {
			t.Errorf("thirdFriday(%d,%d) = %s, want %s", c.y, c.m, got, c.want)
		}
	}
}

func TestParseResolvedContract(t *testing.T) {
	if root, m, y, ok := parseResolvedContract("MNQ 09-26"); !ok || root != "MNQ" || m != 9 || y != 2026 {
		t.Fatalf(`parse "MNQ 09-26" = %s %d %d %v`, root, m, y, ok)
	}
	if root, m, y, ok := parseResolvedContract("ES 12-26"); !ok || root != "ES" || m != 12 || y != 2026 {
		t.Fatalf(`parse "ES 12-26" = %s %d %d %v`, root, m, y, ok)
	}
	for _, bad := range []string{"", "pending", "MNQ", "MNQU6", "MNQ 13-26", "MNQ 09_26", "MNQ 09-26 x"} {
		if _, _, _, ok := parseResolvedContract(bad); ok {
			t.Errorf("parse %q must fail", bad)
		}
	}
}

func ctDate(t *testing.T, iso string) time.Time {
	t.Helper()
	tm, err := time.ParseInLocation("2006-01-02 15:04", iso, kernel.CTLocation())
	if err != nil {
		t.Fatal(err)
	}
	return tm
}

func TestRollVerdictWindow(t *testing.T) {
	const window = 3 // the ROLL_BLOCK_DAYS_BEFORE_EXPIRY default

	// Sep 15–18 (daysLeft 3..0) → blocked; Sep 14 (daysLeft 4) → open.
	// The dispatch's Sep 16–17 dying-liquidity window is fully covered.
	for date, wantBlocked := range map[string]bool{
		"2026-09-14 10:00": false,
		"2026-09-15 10:00": true,
		"2026-09-16 10:00": true,
		"2026-09-17 10:00": true,
		"2026-09-18 08:00": true,
		"2026-09-21 10:00": false, // past expiry: daysLeft negative → never blocked
	} {
		display, expiry, _, blocked, resolved := rollVerdict("MNQ 09-26", ctDate(t, date), window)
		if !resolved {
			t.Fatalf("SEP26 must resolve at %s", date)
		}
		if blocked != wantBlocked {
			t.Errorf("%s: blocked=%v want %v", date, blocked, wantBlocked)
		}
		if display != "MNQ SEP26" || expiry.Format("2006-01-02") != "2026-09-18" {
			t.Errorf("display/expiry wrong: %s %s", display, expiry)
		}
	}

	// Sep 21 with the ROLLED front (DEC26) → passes with ~88 days left.
	_, _, daysLeft, blocked, resolved := rollVerdict("MNQ 12-26", ctDate(t, "2026-09-21 10:00"), window)
	if !resolved || blocked || daysLeft < 80 {
		t.Fatalf("DEC26 on Sep 21 must pass (daysLeft=%d blocked=%v resolved=%v)", daysLeft, blocked, resolved)
	}

	// Resolution missing/unparseable → resolved=false (the gate fail-opens).
	if _, _, _, _, resolved := rollVerdict("pending", ctDate(t, "2026-09-16 10:00"), window); resolved {
		t.Fatal("unresolved contract must report resolved=false (fail-open)")
	}
}

// The gate refusal carries the dispatch's exact message shape.
func TestRollGateMessage(t *testing.T) {
	at := &AutoTrader{id: "tr", exchange: "ninjatrader"}
	at.config.Exchange = "ninjatrader"
	// No TCPTrader bound → unresolved → fail-open pass (and only WARNs once).
	if reason, blocked := at.entryBlockedByRoll(ctDate(t, "2026-09-16 10:00")); blocked {
		t.Fatalf("unresolved must pass, got %q", reason)
	}

	display, expiry, _, blocked, _ := rollVerdict("MNQ 09-26", ctDate(t, "2026-09-16 10:00"), rollBlockDays())
	if !blocked {
		t.Fatal("Sep 16 must be inside the default window")
	}
	got := display + " expires " + expiry.Format("2006-01-02")
	if got != "MNQ SEP26 expires 2026-09-18" {
		t.Fatalf("refusal core = %q", got)
	}
}

func TestRollWindowEnvOverride(t *testing.T) {
	t.Setenv("ROLL_BLOCK_DAYS_BEFORE_EXPIRY", "5")
	if rollBlockDays() != 5 {
		t.Fatal("env override not honored")
	}
	// A 5-day window pulls Sep 14 (daysLeft 4) INTO the block.
	if _, _, _, blocked, _ := rollVerdict("MNQ 09-26", ctDate(t, "2026-09-14 10:00"), rollBlockDays()); !blocked {
		t.Fatal("window=5 must block Sep 14")
	}
	t.Setenv("ROLL_BLOCK_DAYS_BEFORE_EXPIRY", "junk")
	if rollBlockDays() != 3 {
		t.Fatal("malformed env must fall back to 3")
	}
}
