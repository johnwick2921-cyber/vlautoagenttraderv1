package kernel

import (
	"fmt"
	"os"
	"regexp"
	"sort"
	"strconv"
	"testing"
)

// TestChecklistNoUnnumberedClassHeadings pins the checklist's own numbering
// law: every class heading carries a number, and the header's "Highest
// occupied class" equals the largest heading number. A heading that still
// reads "(assigned at merge)" or "(pending)" fails the pin
// (docs/checklist-numbers, PASS 1, DS-102 2026-09-24).
func TestChecklistNoUnnumberedClassHeadings(t *testing.T) {
	mb, err := os.ReadFile("../docs/superpowers/AUDIT-CHECKLIST.md")
	if err != nil {
		t.Fatalf("cannot read AUDIT-CHECKLIST.md: %v", err)
	}
	if off := checklistNumberingOffences(string(mb)); len(off) > 0 {
		t.Fatal(off[0])
	}
}

// checklistNumberingOffences returns why src breaks the numbering law ("" =
// none): a non-canonical class heading, a duplicate class number, an
// unnumbered class heading, or a header that does not equal the largest
// heading number.
//
// Scope (CTO 1790306266164): every heading that NAMES a class is seen, in any
// spelling — `## Class N —`, `### CLASS N —`, `##  CLASS N —`, `##\tCLASS N —`
// — the non-canonical ones are refused with the canonical form, and a duplicate
// number fails. Two headings are legal and NOT counted: the un-numbered
// placeholder `## CLASS NN (assigned at merge) — …` and a `### Class N
// follow-up — …` note (the real file's :2966), which names a class but is not
// one.
func checklistNumberingOffences(src string) []string {
	classHeading := regexp.MustCompile(`(?im)^#{2,3}[ \t]*CLASS\b.*$`)
	canonical := regexp.MustCompile(`^## CLASS (\d+) — `)
	placeholder := regexp.MustCompile(`^## CLASS NN \(assigned at merge\) — `)
	followUp := regexp.MustCompile(`(?i)^#{2,3}[ \t]*class\s+\d+\s+follow[- ]up\b`)
	var refused []string
	counts := map[int]int{}
	for _, line := range classHeading.FindAllString(src, -1) {
		switch {
		case canonical.MatchString(line):
			n, err := strconv.Atoi(canonical.FindStringSubmatch(line)[1])
			if err != nil {
				return []string{fmt.Sprintf("bad class number in %q: %v", line, err)}
			}
			counts[n]++
		case placeholder.MatchString(line):
			// legal un-numbered placeholder; not counted (CTO 1790306266164).
		case followUp.MatchString(line):
			// a follow-up note, not a class; not counted.
		default:
			refused = append(refused, line)
		}
	}
	refused = append(refused, regexp.MustCompile(`(?m)^### \(pending\) `).FindAllString(src, -1)...)
	if len(refused) > 0 {
		shown := refused
		if len(shown) > 5 {
			shown = shown[:5]
		}
		return []string{fmt.Sprintf("%d non-canonical class heading(s) — the only legal forms are `## CLASS <number> — `, `## CLASS NN (assigned at merge) — ` and `### Class <n> follow-up —`: %v", len(refused), shown)}
	}
	var dupes []int
	for n, c := range counts {
		if c > 1 {
			dupes = append(dupes, n)
		}
	}
	sort.Ints(dupes)
	if len(dupes) > 0 {
		return []string{fmt.Sprintf("duplicate class number(s): %v (numbers are assigned AT MERGE and never reused — AUDIT-CHECKLIST.md:13-14)", dupes)}
	}
	max := 0
	for n := range counts {
		if n > max {
			max = n
		}
	}
	header := regexp.MustCompile(`Highest occupied class: \*\*(\d+)\*\*`).FindStringSubmatch(src)
	if header == nil {
		return []string{"no 'Highest occupied class' header found"}
	}
	hn, err := strconv.Atoi(header[1])
	if err != nil {
		return []string{fmt.Sprintf("bad header number %q: %v", header[1], err)}
	}
	if hn != max {
		return []string{fmt.Sprintf("header says highest occupied class is %d but the max heading number is %d", hn, max)}
	}
	return nil
}

// The unnumbered-heading rule is an ALLOW pattern, not a list of the
// placeholders someone has thought of: a deny-list that recognises only
// "NN (assigned at merge)" passed a stray "## CLASS M3-07 —" silently (M3
// class drafts, CTO 1790245281578 — the class-168 shape inside the guard).
func TestChecklistNumberingRefusesEveryNonNumberedClassHeading(t *testing.T) {
	const header = "*Highest occupied class: **7** (2026-09-24).\n\n## CLASS 7 — fixture\n\n"
	for _, h := range []string{
		"## CLASS M3-07 — x",
		"## CLASS NN — x",
		"## CLASS 12a — x",
		"## CLASS  8 — x",
		"## CLASS ?? — x",
	} {
		if off := checklistNumberingOffences(header + h + "\n"); len(off) == 0 {
			t.Errorf("heading %q is not numbered and must be refused", h)
		}
	}
	// CTO 1790306266164: "## CLASS NN (assigned at merge) — …" is the legal
	// un-numbered placeholder (not counted) — it must NOT be refused.
	if off := checklistNumberingOffences(header + "## CLASS NN (assigned at merge) — x\n"); len(off) != 0 {
		t.Errorf("the assigned-at-merge placeholder must stay legal: %v", off)
	}
	if off := checklistNumberingOffences(header); len(off) != 0 {
		t.Errorf("control: a numbered checklist must pass: %v", off)
	}
	if off := checklistNumberingOffences(header + "### (pending) x\n"); len(off) == 0 {
		t.Errorf("a '### (pending)' heading must still be refused")
	}
}

// TestChecklistNumberingCountsEveryClassSpelling pins the CTO-1790306266164
// scope: every heading that names a class in any spelling — "## Class N —",
// "### CLASS N —", "##  CLASS N —", "##\tCLASS N —" — is seen; the
// non-canonical spellings are refused, and a DUPLICATE class number fails.
// RED: each evasion below passed offences=[] under the old `(?m)^## CLASS .*$`
// selector and the max-only count (no uniqueness).
func TestChecklistNumberingCountsEveryClassSpelling(t *testing.T) {
	const header = "*Highest occupied class: **8** (2026-09-24).\n\n## CLASS 8 — base fixture\n\n"
	for _, h := range []string{
		"## Class 9 — mixed-case h2",
		"### CLASS 9 — h3",
		"##  CLASS 9 — double-space h2",
		"##\tCLASS 9 — tab h2",
	} {
		if off := checklistNumberingOffences(header + h + "\n"); len(off) == 0 {
			t.Errorf("non-canonical heading %q must be refused", h)
		}
	}
	// Duplicate number: the same number twice is the A27 collision shape.
	if off := checklistNumberingOffences(header + "## CLASS 8 — colliding duplicate\n"); len(off) == 0 {
		t.Error("a duplicate class number must fail the pin")
	} else {
		t.Logf("duplicate refused: %v", off)
	}
	// The follow-up heading that exists in the real file (:2966) is not a
	// class and must stay legal.
	if off := checklistNumberingOffences(header + "### Class 8 follow-up — context note\n"); len(off) != 0 {
		t.Errorf("a '### Class N follow-up' heading is not a class and must pass: %v", off)
	}
	if off := checklistNumberingOffences(header); len(off) != 0 {
		t.Errorf("control: header 8 == max 8 must pass: %v", off)
	}
}
