package expectancy

import (
	"strings"

	"nofx/kernel"
)

// levelKinds is the canonicalizer's vocabulary. It holds the kernel constants
// THEMSELVES, not copies of their spellings — adding a kind to kernel/levels.go
// and forgetting it here yields an unresolved label (counted), never a silently
// wrong one.
//
// Order is irrelevant: the match is longest-first, computed below.
var levelKinds = []kernel.LevelKind{
	kernel.KindPDH, kernel.KindPDL, kernel.KindPDC,
	kernel.KindRTHH, kernel.KindRTHL,
	kernel.KindASH, kernel.KindASL, kernel.KindLDNH, kernel.KindLDNL,
	kernel.KindONH, kernel.KindONL,
	kernel.KindPWH, kernel.KindPWL, kernel.KindPMH, kernel.KindPML,
	kernel.KindRound, kernel.KindGap,
	kernel.KindORH, kernel.KindORL, kernel.KindIBH, kernel.KindIBL,
	kernel.KindNPOC, kernel.KindVWAP, kernel.KindEVWAP, kernel.KindPOC,
	kernel.KindVAH, kernel.KindVAL, kernel.KindPDVWAP, kernel.KindSETT,
	kernel.KindMIDO, kernel.KindEQH, kernel.KindEQL,
	kernel.KindSWGH, kernel.KindSWGL, kernel.KindVWAP2S,
	kernel.KindSupply, kernel.KindDemand,
	kernel.KindFVG, kernel.KindIFVG, kernel.KindOB, kernel.KindOwner,
}

// LevelKindFromLabel is THE canonicalizer from a plan level's provenance chip to
// a level kind — one canonicalizer, called where the value enters (canonical-
// casing law, checklist 28).
//
// A plan doc stores only Label ("ONL", "SWG-L·5m", "OB(bull)·1h (HTF)",
// "nPOC·Tue"); the typed kernel.LevelKind lives on the detector side and is not
// carried into the doc. The label is built FROM the kind with decorations
// appended, so the kind is recovered by stripping the decorations and matching
// the enum:
//
//	"·<suffix>"  timeframe / origin-date suffix  → dropped
//	"(...)"      polarity or provenance note     → dropped
//	" (HTF)"     higher-timeframe marker         → dropped by the above
//
// An unrecognised label returns "" — the level kind is UNKNOWN and the caller
// counts it. It never guesses the nearest kind, and it never invents one.
func LevelKindFromLabel(label string) string {
	s := strings.TrimSpace(label)
	if s == "" {
		return ""
	}
	// Drop the "·suffix" first: "SWG-L·5m" → "SWG-L", "nPOC·Tue" → "nPOC".
	if i := strings.Index(s, "·"); i >= 0 {
		s = s[:i]
	}
	// Drop any parenthetical: "OB(bull)" → "OB", "Demand (HTF)" → "Demand".
	if i := strings.Index(s, "("); i >= 0 {
		s = s[:i]
	}
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	// Longest match wins so that PDH is never mistaken for PD-anything and
	// VWAP±2σ is preferred over VWAP when the label actually carries the band.
	best := ""
	for _, k := range levelKinds {
		ks := string(k)
		if len(ks) <= len(best) {
			continue
		}
		if strings.EqualFold(s, ks) || strings.HasPrefix(strings.ToUpper(s), strings.ToUpper(ks)) {
			best = ks
		}
	}
	return best
}
