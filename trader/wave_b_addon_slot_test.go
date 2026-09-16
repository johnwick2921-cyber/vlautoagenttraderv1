// WAVE B (2026-09-05) — the AddOn's CreateOrder slot, pinned from Go.
//
// THERE IS NO C# TEST HARNESS IN THIS REPO (no .csproj, no .sln, no *Test*.cs —
// checked 2026-09-05). NinjaScript compiles only inside NT8, so the only way to
// assert the C# argument slot from a suite that actually runs is to read the
// source. This is a SOURCE pin, not a behavioural one, and it is stated as such
// in the report: it proves the shipped .cs text, and the far-side BUILD GATE
// (MinAddonBuildStopSlot) proves the DLL NT8 actually loaded.

package trader

import (
	"os"
	"regexp"
	"strings"
	"testing"

	ntwire "nofx/provider/ninjatrader"
)

const addonSourcePath = "../ninjascript/VLTraderTCPClient.cs"

func addonSource(t *testing.T) string {
	t.Helper()
	b, err := os.ReadFile(addonSourcePath)
	if err != nil {
		t.Fatalf("cannot read the AddOn source at %s: %v", addonSourcePath, err)
	}
	return string(b)
}

// TestAddonEntryOrderPassesTheTriggerInTheStopSlot — E2. NinjaTrader's
// Account.CreateOrder is positional: after `quantity` come (limitPrice,
// stopPrice, oco, name, gtd, customOrder). A StopMarket entry's TRIGGER belongs
// in stopPrice. Until 2026-09-05 the entry call passed ONE price into the
// limitPrice slot for both kinds and a literal 0 into stopPrice, so every stop
// entry reached NT8 as `Limit price=<trigger> Stop price=0` — a zero trigger,
// inert forever (22 of 22 lifetime submissions, 0 fills).
func TestAddonEntryOrderPassesTheTriggerInTheStopSlot(t *testing.T) {
	src := addonSource(t)

	// The two price arguments of the ENTRY CreateOrder call, as shipped.
	re := regexp.MustCompile(`(?s)submitAccount\.CreateOrder\(\s*` +
		`instrument,\s*entryAction,\s*orderT,\s*OrderEntry\.Manual,\s*` +
		`TimeInForce\.Day,\s*qty,\s*([A-Za-z0-9_]+),\s*([A-Za-z0-9_]+),`)
	m := re.FindStringSubmatch(src)
	if m == nil {
		t.Fatal("could not find the entry CreateOrder call — the pin cannot read the slot it exists to check")
	}
	limitArg, stopArg := m[1], m[2]
	if limitArg == stopArg {
		t.Fatalf("entry CreateOrder passes the same expression into both price slots (%q) — a StopMarket and a Limit cannot share one price argument", limitArg)
	}
	if stopArg == "0" {
		t.Fatalf("entry CreateOrder passes a literal 0 into the stopPrice slot (limit=%q stop=%q) — a StopMarket built this way has a ZERO trigger and rests inert forever", limitArg, stopArg)
	}
	// The stop slot must be fed by the stop_entry branch and the limit slot by
	// the limit branch — not merely "two different names".
	limitDecl := regexp.MustCompile(`double\s+` + regexp.QuoteMeta(limitArg) + `\s*=\s*isLimit\s*\?`)
	stopDecl := regexp.MustCompile(`double\s+` + regexp.QuoteMeta(stopArg) + `\s*=\s*isStopEntry\s*\?`)
	if !limitDecl.MatchString(src) {
		t.Errorf("the limitPrice argument %q is not selected by isLimit", limitArg)
	}
	if !stopDecl.MatchString(src) {
		t.Errorf("the stopPrice argument %q is not selected by isStopEntry", stopArg)
	}

	// The bracket stop-loss at the other end of the same file is the in-file
	// control: it has always built a StopMarket correctly (limit 0 / stop Sl)
	// and is proven correct by a live fill (2026-09-03, filled at 29355).
	//
	// 2026-09-07: the call's TIF became Gtc (D6) and its quantity now comes from
	// the fill event (D2). Neither touches what this control proves — the SLOT
	// ORDER, a literal 0 in limitPrice and b.Sl in stopPrice — which is asserted
	// below on the two arguments themselves rather than on the whole line, so
	// the next unrelated edit does not read as the slot order being lost.
	if !strings.Contains(src, "0, b.Sl, exitOco, signalId + \"-sl\",") {
		t.Error("the bracket stop-loss control no longer passes (limitPrice=0, stopPrice=b.Sl) — the in-file proof of the slot order is gone")
	}
}

// TestAddonSubmissionLogNamesTheFourValuesItSent — A9. The old log printed the
// PARSED variable ("stop@29590.5") while submitting Stop price=0, which is why a
// slot bug survived in a file that logs every placement. The line must name the
// arguments actually handed to CreateOrder.
func TestAddonSubmissionLogNamesTheFourValuesItSent(t *testing.T) {
	src := addonSource(t)
	i := strings.Index(src, "VLTraderTCPClient: submitted entry signal_id=")
	if i < 0 {
		t.Fatal("the entry submission log line is gone")
	}
	line := src[i:]
	if j := strings.Index(line, ");"); j > 0 {
		line = line[:j]
	}
	for _, want := range []string{"action=", "type=", "limitPrice=", "stopPrice="} {
		if !strings.Contains(line, want) {
			t.Errorf("the submission log does not name %s — the NT8 log alone must prove the shape it sent: %q", want, line)
		}
	}
}

// TestAddonEntryActionFoldsCase — class 77, the C# half. The Go ledger
// canonicalizes side to UPPERCASE at the write and the AddOn decided the ORDER
// DIRECTION with an ordinal ternary: "LONG" is not "long", so an uppercase side
// fell to the else branch and a LONG entry was submitted as OrderAction.SellShort.
// Go now folds before it sends; the AddOn must fold on arrival too, because a
// ternary whose unknown branch opens a position in the OPPOSITE direction must
// not be the last line of defence. This is a SOURCE pin (see the file header).
func TestAddonEntryActionFoldsCase(t *testing.T) {
	src := addonSource(t)
	if regexp.MustCompile(`var\s+entryAction\s*=\s*side\s*==\s*"long"\s*\?`).MatchString(src) ||
		regexp.MustCompile(`var\s+exitAction\s*=\s*side\s*==\s*"long"\s*\?`).MatchString(src) {
		t.Fatal("the AddOn still decides the order DIRECTION with an ordinal `side == \"long\"` — an uppercase side submits a live SellShort")
	}
	if !strings.Contains(src, `string.Equals(side, "long", StringComparison.OrdinalIgnoreCase)`) {
		t.Error("the entry action is not case-folded — the store writes LONG/SHORT")
	}
}

// TestAddonBuildIDMovesInLockstep — VL_BUILD_ID (C#), ExpectedAddonBuild (Go)
// and MinAddonBuildStopSlot (the gate) are three copies of one value in two
// languages. A half-bump makes every boot print match=NO, or lets the gate pass
// a build that never landed. The pin READS the .cs rather than restating it, so
// it cannot go green on a typo (A24 — a fixture must not hold its own copy).
func TestAddonBuildIDMovesInLockstep(t *testing.T) {
	src := addonSource(t)
	m := regexp.MustCompile(`VL_BUILD_ID\s*=\s*"([^"]+)"`).FindStringSubmatch(src)
	if m == nil {
		t.Fatal("VL_BUILD_ID not found in the AddOn source")
	}
	got := m[1]
	if got != ntwire.ExpectedAddonBuild {
		t.Errorf("VL_BUILD_ID=%q but ExpectedAddonBuild=%q — every boot would print match=NO", got, ntwire.ExpectedAddonBuild)
	}
	// The floor is a MINIMUM, not a twin. This read `!=` while its own message
	// said ">=", which was invisible only because the two constants happened to
	// be equal from 2026-09-05 until the build id next moved. MinAddonBuildStopSlot
	// is a historical floor and must stay put; VL_BUILD_ID advances past it.
	if !ntwire.FarSideProven(got, ntwire.MinAddonBuildStopSlot) {
		t.Errorf("VL_BUILD_ID=%q but the stop-slot gate needs >= %q — the fixed AddOn would refuse itself", got, ntwire.MinAddonBuildStopSlot)
	}
	// The gate must actually refuse the build that shipped the defect.
	if ntwire.FarSideProven("2026-09-03-f12", ntwire.MinAddonBuildStopSlot) {
		t.Errorf("the stop-slot gate accepts 2026-09-03-f12 — the build whose CreateOrder put the trigger in the limit slot")
	}
	if !ntwire.FarSideProven(ntwire.MinAddonBuildStopSlot, ntwire.MinAddonBuildStopSlot) {
		t.Errorf("the gate refuses its own minimum build")
	}
	// FarSideProven compares strings bytewise and build suffixes are NOT
	// zero-padded ("...-f9" > "...-f12"), so only a strictly NEWER DATE prefix
	// is a safe minimum. Pin the date, not the suffix.
	if len(ntwire.MinAddonBuildStopSlot) < 10 || ntwire.MinAddonBuildStopSlot[:10] <= "2026-09-03" {
		t.Errorf("MinAddonBuildStopSlot=%q must carry a DATE strictly newer than 2026-09-03: the suffix does not sort numerically", ntwire.MinAddonBuildStopSlot)
	}
}
