package trader

// CLASS 60 LINT — adding a time-dependent rule without a clock seam fails here.
//
// The list of rules lives in ONE place, ../clock-seams.list, and this test
// reads it. It does not carry its own copy of the rule names (A24: a fixture
// with its own copy of a constant is not a check, it is a second thing to keep
// in sync).
//
// For each row it asserts the two halves of the seam:
//   1. the …At variant exists in that file
//   2. the entry point is a ONE-LINE DELEGATE to it — nothing else in the body
//
// (2) is the half that matters. An entry point that reads the clock AND does
// work is exactly class 60: the test pins a fixture, the code reads the wall,
// and the suite is green until the hour changes.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type seamRule struct{ file, entry, at string }

func loadSeamRules(t *testing.T) []seamRule {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("..", "clock-seams.list"))
	if err != nil {
		t.Fatalf("read clock-seams.list: %v", err)
	}
	var out []seamRule
	for _, ln := range strings.Split(string(raw), "\n") {
		ln = strings.TrimSpace(ln)
		if ln == "" || strings.HasPrefix(ln, "#") {
			continue
		}
		p := strings.Split(ln, ":")
		if len(p) != 3 {
			t.Fatalf("malformed row %q — want <file>:<entry>:<At>", ln)
		}
		out = append(out, seamRule{p[0], p[1], p[2]})
	}
	if len(out) == 0 {
		t.Fatal("clock-seams.list has no rules — an empty list passes vacuously")
	}
	return out
}

// bodyOf returns the lines between the opening and closing brace of the first
// top-level func whose declaration contains "<name>(".
func bodyOf(src, name string) ([]string, bool) {
	lines := strings.Split(src, "\n")
	for i, l := range lines {
		if !strings.HasPrefix(l, "func ") || !strings.Contains(l, name+"(") {
			continue
		}
		var body []string
		for j := i + 1; j < len(lines); j++ {
			if lines[j] == "}" {
				return body, true
			}
			body = append(body, lines[j])
		}
	}
	return nil, false
}

// statements strips blank lines and comments — a delegate may be commented.
func statements(body []string) []string {
	var out []string
	for _, l := range body {
		s := strings.TrimSpace(l)
		if s == "" || strings.HasPrefix(s, "//") {
			continue
		}
		out = append(out, s)
	}
	return out
}

func TestEveryTimeDependentRuleHasAClockSeam(t *testing.T) {
	for _, r := range loadSeamRules(t) {
		raw, err := os.ReadFile(filepath.Join("..", r.file))
		if err != nil {
			t.Errorf("%s: %v", r.file, err)
			continue
		}
		src := string(raw)

		if _, ok := bodyOf(src, r.at); !ok {
			t.Errorf("%s: rule %q has no %q variant — the clock cannot be stated by a test",
				r.file, r.entry, r.at)
			continue
		}
		body, ok := bodyOf(src, r.entry)
		if !ok {
			t.Errorf("%s: entry point %q not found (renamed? update clock-seams.list)", r.file, r.entry)
			continue
		}
		st := statements(body)
		if len(st) != 1 {
			t.Errorf("%s: %q must be a ONE-LINE delegate to %q, has %d statements: %v",
				r.file, r.entry, r.at, len(st), st)
			continue
		}
		if !strings.Contains(st[0], r.at+"(") {
			t.Errorf("%s: %q does not delegate to %q — body is %q", r.file, r.entry, r.at, st[0])
		}
	}
}

// E2 — the detector must FAIL on an unseamed rule. Pinned against synthetic
// source rather than by breaking a real file, so it keeps working forever.
func TestSeamLintRejectsAnUnseamedRule(t *testing.T) {
	unseamed := `package x

func (at *AutoTrader) somethingTimed() bool {
	now := time.Now()
	return now.Hour() > 12
}
`
	if _, ok := bodyOf(unseamed, "somethingTimedAt"); ok {
		t.Fatal("detector found an At variant that does not exist")
	}
	body, ok := bodyOf(unseamed, "somethingTimed")
	if !ok {
		t.Fatal("detector failed to find the entry point at all")
	}
	if n := len(statements(body)); n == 1 {
		t.Fatal("detector called a 2-statement wall-clock body a one-line delegate")
	}

	seamed := `package x

func (at *AutoTrader) somethingTimed() bool {
	// the clock lives here and nowhere below
	return at.somethingTimedAt(time.Now())
}

func (at *AutoTrader) somethingTimedAt(now time.Time) bool {
	return now.Hour() > 12
}
`
	if _, ok := bodyOf(seamed, "somethingTimedAt"); !ok {
		t.Fatal("detector missed a real At variant")
	}
	body, _ = bodyOf(seamed, "somethingTimed")
	st := statements(body)
	if len(st) != 1 || !strings.Contains(st[0], "somethingTimedAt(") {
		t.Fatalf("detector rejected a correct seam: %v", st)
	}
}

// THE THIRD HALF OF THE SEAM: THE TESTS MUST USE IT.
//
// Added 2026-09-10 after dev's Go suite went red for 90 minutes a day with no
// commit involved. The two assertions above were both GREEN throughout: the …At
// variant existed and the entry point was a clean one-line delegate. A28 was
// perfectly honoured — by production. Eight tests then called the WALL-CLOCK
// entry point and handed a correctly-seamed rule the real hour of the day, and
// the suite failed inside the lunch no-trade band and passed outside it.
//
// A seam only the production path honours is half a seam.
//
// SCOPED, DELIBERATELY, to the entry point that caused the outage. The general
// form — "no test calls any seamed entry point" — was written first and finds 35
// sites, of which most are FALSE POSITIVES: clock-seams.list contains entries
// named `Save` and `observe`, and a textual `.Save(` cannot tell `at.Save(` from
// `db.Save(` without resolving the receiver's type. A lint that cries wolf 20
// times gets deleted, and then the 15 real ones go unwatched too.
//
// So this asserts the one rule with a KNOWN, LIVE consequence, and the general
// case is recorded as owed rather than shipped noisy:
//
//	OWED: a receiver-aware version of this check (go/ast, not strings.Contains)
//	covering every entry in clock-seams.list. Until it exists, a time-banded
//	rule added to any OTHER seamed entry point can reintroduce exactly this
//	outage and nothing will fail.
func TestArmEntryPointIsNotCalledFromTests(t *testing.T) {
	files, err := filepath.Glob("*_test.go")
	if err != nil || len(files) == 0 {
		t.Fatalf("no _test.go files found to scan: %v", err)
	}
	found := 0
	for _, f := range files {
		// THIS FILE CARRIES THE PATTERN AS A STRING LITERAL and would match
		// itself — the same self-match that makes `pkill -f` kill its own shell.
		if f == "clock_seam_lint_test.go" {
			continue
		}
		b, err := os.ReadFile(f)
		if err != nil {
			t.Fatalf("read %s: %v", f, err)
		}
		for i, ln := range strings.Split(string(b), "\n") {
			code := ln
			if k := strings.Index(code, "//"); k >= 0 {
				code = code[:k] // a mention in a comment is not a call
			}
			if !strings.Contains(code, ".maybeManageArmedOrders(") {
				continue
			}
			found++
			t.Errorf(`%s:%d calls the wall-clock entry point maybeManageArmedOrders().

  Use maybeManageArmedOrdersAt(snap, now) with a clock the test controls.

  maybeManageArmedOrders() reads time.Now() and delegates. Calling it from a test
  hands a correctly-seamed rule the real hour of the day, so the test passes or
  fails on WHEN it ran. On 2026-09-10 that made dev RED from 12:00 to 13:30 CT
  daily and green either side, with no commit involved — the arm path refuses
  inside the lunch no-trade band.

  armTestClock(t, at) in arm_test_clock_test.go returns a moment inside an
  enabled session and outside every no-trade window.

    %s`, f, i+1, strings.TrimSpace(ln))
		}
	}
	_ = found
}
