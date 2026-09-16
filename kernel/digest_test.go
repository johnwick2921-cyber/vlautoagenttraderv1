package kernel

import (
	"strings"
	"testing"
)

func TestFormatSessionDigest(t *testing.T) {
	d := FormatSessionDigest("NY", "2026-08-14", "trend", 2, 120.5)
	if lines := strings.Split(d, "\n"); len(lines) != 3 {
		t.Fatalf("session digest must be 3 lines, got %d: %q", len(lines), d)
	}
	if !strings.Contains(d, "NY 2026-08-14 — trend") || !strings.Contains(d, "2 entries") || !strings.Contains(d, "green") {
		t.Fatalf("session digest content wrong: %q", d)
	}
	if !strings.Contains(FormatSessionDigest("NY", "d", "", 0, -5), "red") {
		t.Fatalf("negative pnl → red")
	}
}

func TestBuildDigestChainTapered(t *testing.T) {
	// Synthetic week: 2 current-date session digests + 7 dailies (newest first).
	sessions := []string{"NY 08-14 — trend\nl2\nl3", "LDN 08-14 — balance\nl2\nl3"}
	dailies := make([]string, 7)
	for i := 0; i < 7; i++ {
		dailies[i] = FormatDailyDigest("day", "t", 1, 1, float64(i))
	}
	chain := BuildDigestChain(sessions, dailies)

	// 2 sessions + 3 FULL dailies + 4 one-liner dailies = 9 entries.
	if len(chain) != 9 {
		t.Fatalf("chain len = %d want 9", len(chain))
	}
	// The current-date session digests come first, verbatim.
	if chain[0] != sessions[0] || chain[1] != sessions[1] {
		t.Fatalf("current session digests must lead the chain")
	}
	// dailies[0:3] full (3 lines each).
	for i := 2; i < 5; i++ {
		if strings.Count(chain[i], "\n") != 2 {
			t.Fatalf("chain[%d] should be a full 3-line daily: %q", i, chain[i])
		}
	}
	// dailies[3:7] one-liners (no newline).
	for i := 5; i < 9; i++ {
		if strings.Contains(chain[i], "\n") {
			t.Fatalf("chain[%d] should be a one-liner: %q", i, chain[i])
		}
	}
}

func TestBuildDigestChainShort(t *testing.T) {
	// Fewer than 3 dailies → all full, no one-liners.
	chain := BuildDigestChain(nil, []string{"a\nb\nc", "x\ny\nz"})
	if len(chain) != 2 {
		t.Fatalf("short chain len = %d want 2", len(chain))
	}
}

// P0-cleanup (2026-08-19) — a closed trade's MAE/MFE + adherence grade must be
// visible in the digest line and linkable to its plan version.
func TestLearningLine(t *testing.T) {
	// E4 (wave 1A): a trade now says whether its excursion was MEASURED. The
	// old test `MAE > 0 || MFE > 0` treated an uncomputed row and a genuine
	// zero identically, which is the ambiguity this wave exists to remove.
	trades := []LearningTrade{
		{MAE: 12, MFE: 90, Measured: true, Grade: "A", PlanVersion: 2},
		{MAE: 30, MFE: 45, Measured: true, Grade: "C", PlanVersion: 2},
	}
	line := LearningLine(trades)
	if line == "" {
		t.Fatalf("learning line must render for graded trades")
	}
	for _, want := range []string{"avg MAE 21.0", "avg MFE 67.5", "(n=2 of 2)", "adherence map[A:1 C:1]", "plan v[2 2]"} {
		if !strings.Contains(line, want) {
			t.Fatalf("learning line %q missing %q", line, want)
		}
	}
	if LearningLine(nil) != "" {
		t.Fatalf("empty trades → empty line")
	}

	// An UNMEASURED trade must not be averaged in as a zero, and the n must
	// say how many of the trades the averages actually rest on.
	mixed := LearningLine([]LearningTrade{
		{MAE: 12, MFE: 90, Measured: true, Grade: "A", PlanVersion: 2},
		{Grade: "B", PlanVersion: 2}, // never computed
	})
	for _, want := range []string{"avg MAE 12.0", "avg MFE 90.0", "(n=1 of 2)"} {
		if !strings.Contains(mixed, want) {
			t.Fatalf("mixed line %q missing %q — an unmeasured trade must not read as a zero", mixed, want)
		}
	}

	// A genuine measured zero still counts.
	if z := LearningLine([]LearningTrade{{MAE: 0, MFE: 0, Measured: true, Grade: "A"}}); !strings.Contains(z, "(n=1 of 1)") {
		t.Fatalf("a measured zero must count: %q", z)
	}
}
