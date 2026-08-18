package kernel

import (
	"time"
)

// P1 — THE MAP: shared level types.
//
// Every detector (multi-day extractor, round numbers, gaps, OR/IB, naked-POC,
// EQH/EQL, S/D zones, FVG/OB) emits DetectedLevel so the confluence scorer and
// the KEY LEVELS prompt block consume one uniform list. Detectors are pure and
// deterministic (facts=Go); nothing here calls an LLM.

// LevelKind identifies a reference level's type (drives grading + display).
type LevelKind string

const (
	KindPDH    LevelKind = "PDH"    // prior calendar-day high
	KindPDL    LevelKind = "PDL"    // prior calendar-day low
	KindPDC    LevelKind = "PDC"    // prior calendar-day close
	KindRTHH   LevelKind = "RTH-H"  // prior regular-hours (NY) high
	KindRTHL   LevelKind = "RTH-L"  // prior regular-hours (NY) low
	KindASH    LevelKind = "AS-H"   // overnight Asia high
	KindASL    LevelKind = "AS-L"   // overnight Asia low
	KindLDNH   LevelKind = "LDN-H"  // overnight London high
	KindLDNL   LevelKind = "LDN-L"  // overnight London low
	KindONH    LevelKind = "ONH"    // overnight (Asia+London) composite high
	KindONL    LevelKind = "ONL"    // overnight composite low
	KindPWH    LevelKind = "PWH"    // prior week high
	KindPWL    LevelKind = "PWL"    // prior week low
	KindPMH    LevelKind = "PMH"    // prior month high
	KindPML    LevelKind = "PML"    // prior month low
	KindRound  LevelKind = "RN"     // round number
	KindGap    LevelKind = "GAP"    // unfilled gap edge
	KindORH    LevelKind = "OR-H"   // opening-range high
	KindORL    LevelKind = "OR-L"   // opening-range low
	KindIBH    LevelKind = "IB-H"   // initial-balance high
	KindIBL    LevelKind = "IB-L"   // initial-balance low
	KindNPOC   LevelKind = "nPOC"   // naked point of control
	KindEQH    LevelKind = "EQH"    // equal highs (liquidity)
	KindEQL    LevelKind = "EQL"    // equal lows (liquidity)
	KindSupply LevelKind = "SUPPLY" // supply zone
	KindDemand LevelKind = "DEMAND" // demand zone
	KindFVG    LevelKind = "FVG"    // fair-value gap
	KindOB     LevelKind = "OB"     // order block
	KindOwner  LevelKind = "OWNER"  // sticky owner-set level (P3.6-C)
)

// DetectedLevel is the uniform output of every detector (a line or a zone). For
// a line, Lo == Hi == Price. HTF marks a higher-timeframe origin (grading input).
type DetectedLevel struct {
	Kind       LevelKind `json:"kind"`
	Price      float64   `json:"price"`          // line price (zones: midpoint)
	Lo         float64   `json:"lo"`             // zone bottom (== Price for a line)
	Hi         float64   `json:"hi"`             // zone top (== Price for a line)
	Label      string    `json:"label"`          // display label, e.g. "PDH", "RN 15500", "nPOC·Tue"
	OriginDate string    `json:"origin_date"`    // YYYY-MM-DD of formation
	HTF        bool      `json:"htf"`            // higher-timeframe origin
	Info       string    `json:"info,omitempty"` // extra (gap size, fill state, ...)
}

// lineLevel builds a single-price DetectedLevel (Lo==Hi==price).
func lineLevel(kind LevelKind, price float64, label, origin string, htf bool) DetectedLevel {
	return DetectedLevel{Kind: kind, Price: price, Lo: price, Hi: price, Label: label, OriginDate: origin, HTF: htf}
}

// zoneLevel builds a banded DetectedLevel (Price = midpoint of [lo,hi]).
func zoneLevel(kind LevelKind, lo, hi float64, label, origin string) DetectedLevel {
	if hi < lo {
		lo, hi = hi, lo
	}
	return DetectedLevel{Kind: kind, Price: (lo + hi) / 2, Lo: lo, Hi: hi, Label: label, OriginDate: origin}
}

// chicago is the legacy name for the canonical CT location — it delegates
// to CTLocation() (kernel/tz.go), the SINGLE timezone source every renderer
// must go through (owner rule 2026-08-19: CT is canonical everywhere).
func chicago() *time.Location { return CTLocation() }
