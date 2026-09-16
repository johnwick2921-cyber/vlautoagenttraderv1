// W5 / E1 — the GUIDE may not type a clock the machine resolves.
//
// TestNoTradeWindowsHaveNoSurfaceLiterals already forbids a hardcoded lunch
// window in Go source. It scans Go files only, so the Guide — the surface the
// OWNER actually reads — was the one place the window could drift unwatched,
// and it did: web/src/guide/content/plays.ts told the operator
//
//	lunch 11:30–13:30 ET → no entries
//
// while kernel.LunchWindowCT() resolves 12:00–13:30 CT. That is an hour off, in
// the wrong clock, describing a window where entries are genuinely refused.
//
// The same line typed "10:30 ET" twice, where the prompt renders the identical
// cue through ETtoCT(). One class, one fixture: the Guide states a machine
// window by NAMING ITS RESOLVER, never by typing digits.

package kernel

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// guideContentFiles are read relative to the kernel package (../web/...).
func guideContentFiles(t *testing.T) map[string]string {
	t.Helper()
	dir := filepath.Join("..", "web", "src", "guide", "content")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read guide content dir: %v", err)
	}
	out := map[string]string{}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".ts") || strings.HasSuffix(e.Name(), ".test.ts") {
			continue
		}
		b, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			t.Fatalf("read %s: %v", e.Name(), err)
		}
		out[e.Name()] = string(b)
	}
	if len(out) == 0 {
		t.Fatal("no guide content files found — the scan would pass vacuously")
	}
	return out
}

// ctPlusOneHour renders a CT "HH:MM" in Eastern. Test-local by design.
func ctPlusOneHour(hhmm string) string {
	var h, m int
	if _, err := fmt.Sscanf(hhmm, "%d:%d", &h, &m); err != nil {
		return hhmm
	}
	return fmt.Sprintf("%02d:%02d", (h+1)%24, m)
}

// Any clock range stated NEAR the word "lunch" must EQUAL the resolved window.
//
// THIS TEST WAS WRONG, AND E5 IS WHAT FOUND IT. The first version built its
// forbidden-literal list FROM LunchWindowCT() and asked whether the Guide
// contained those literals. Mutating the resolver to 12:15–13:45 left the
// Guide's stale "12:00–13:30" matching none of the new literals, so the loop
// fell through and the test reported ok. A guard that derives its expectation
// from the thing it guards cannot see that thing drift — it can only confirm
// today's agreement (checklist class 97: one source, both readers; never two
// readers that happen to agree).
//
// Inverted: find every HH:MM–HH:MM range written near "lunch" and assert it
// EQUALS the resolved pair. Now moving the resolver fails loudly, which is the
// only behaviour that protects the Guide tomorrow.
var clockRange = regexp.MustCompile(`(\d{1,2}:\d{2})\s*[–-]\s*(\d{1,2}:\d{2})`)

func TestGuideLunchWindowEqualsTheResolvedWindow(t *testing.T) {
	ls, le := LunchWindowCT()
	want := ls + "-" + le

	for name, src := range guideContentFiles(t) {
		lower := strings.ToLower(src)
		checked := 0
		for idx := 0; ; {
			at := strings.Index(lower[idx:], "lunch")
			if at < 0 {
				break
			}
			at += idx
			idx = at + 5
			// A TIGHT window around the word. Calibrated against the real text:
			// the range sits adjacent to "lunch" in every case that matters
			// ("lunch 12:00–13:30 CT"; time: '12:00–13:30' … label: 'Lunch
			// gate'). A wider window dragged in tradingDay.ts's unrelated NY
			// session range 08:30–14:45 and failed on a sentence that was right.
			lo, hi := at-80, at+80
			if lo < 0 {
				lo = 0
			}
			if hi > len(src) {
				hi = len(src)
			}
			for _, m := range clockRange.FindAllStringSubmatch(src[lo:hi], -1) {
				checked++
				got := m[1] + "-" + m[2]
				if got != want {
					t.Errorf("%s states the lunch window as %q but kernel.LunchWindowCT() resolves %q — the Guide has drifted from its resolver", name, m[0], ls+"–"+le)
				}
			}
		}
		if checked == 0 && strings.Contains(lower, "lunch") && strings.Contains(src, "LunchWindowCT") {
			// A file that names the resolver and states no digits is the ideal
			// shape; nothing to compare. Recorded so the pin cannot pass
			// vacuously everywhere at once.
			t.Logf("%s: names the resolver and types no window — nothing to compare", name)
		}
	}
}

// No raw Eastern clock may appear in Guide content. The machine speaks CT; an
// ET time in the Guide is either a second clock the reader must convert or a
// window that has silently drifted from its resolver.
var guideETClock = regexp.MustCompile(`\d{1,2}:\d{2}\s*ET\b`)

func TestGuideStatesNoRawEasternClock(t *testing.T) {
	for name, src := range guideContentFiles(t) {
		if hits := guideETClock.FindAllString(src, -1); len(hits) > 0 {
			t.Errorf("%s states %d raw Eastern clock(s) %v — the prompt renders the same cues through ETtoCT(); the Guide must state CT or name the resolver", name, len(hits), hits)
		}
	}
}
