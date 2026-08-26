package kernel

import (
	"testing"
	"time"

	"nofx/market"
)

// htfKlines builds `n` 1h bars (OpenTime step = 1h), all closed before `now`.
func htfKlines(n int, o, h, l, c []float64, now time.Time) []market.Kline {
	out := make([]market.Kline, n)
	start := now.Add(-time.Duration(n) * time.Hour).Truncate(time.Hour)
	for i := 0; i < n; i++ {
		out[i] = market.Kline{
			OpenTime:  start.Add(time.Duration(i) * time.Hour).UnixMilli(),
			Open:      o[i], High: h[i], Low: l[i], Close: c[i],
			CloseTime: start.Add(time.Duration(i)*time.Hour + time.Hour - 1).UnixMilli(),
		}
	}
	return out
}

// TestDetectHTFLevelsDemandZone covers G2/G3: a 1h base+departure becomes a
// "Demand·1h" zone with HTF=true, and intraday/daily TFs are skipped.
func TestDetectHTFLevelsDemandZone(t *testing.T) {
	now := time.Date(2026, 8, 24, 14, 0, 0, 0, time.UTC)
	n := 21
	o, h, l, c := make([]float64, n), make([]float64, n), make([]float64, n), make([]float64, n)
	for i := 0; i < n; i++ {
		o[i], c[i] = 100, 100.1 // small bodies (0.1)
		h[i], l[i] = 101, 99
	}
	// departure candle: big up move
	o[n-1], c[n-1] = 100, 110
	h[n-1], l[n-1] = 110.5, 99.5

	fetch := func(tf string, count int) []market.Kline {
		if tf != "1h" {
			return nil
		}
		return htfKlines(n, o, h, l, c, now)
	}
	got := DetectHTFLevels(fetch, []string{"1m", "5m", "1h", "D"}, "MNQ", now)
	var demand *DetectedLevel
	for i := range got {
		if got[i].Kind == KindDemand {
			demand = &got[i]
		}
	}
	if demand == nil {
		t.Fatalf("expected a Demand zone from the 1h base+departure, got %d levels", len(got))
	}
	if !demand.HTF {
		t.Fatalf("1h demand zone must carry HTF=true: %+v", demand)
	}
	if demand.Label != "Demand·1h" {
		t.Fatalf("label = %q, want \"Demand·1h\"", demand.Label)
	}
}

// TestDetectHTFLevelsEQH covers the 1h swing side: two strict pivot highs
// within tolerance become "EQH·1h" with HTF=true.
func TestDetectHTFLevelsEQH(t *testing.T) {
	now := time.Date(2026, 8, 24, 14, 0, 0, 0, time.UTC)
	n := 30
	o, h, l, c := make([]float64, n), make([]float64, n), make([]float64, n), make([]float64, n)
	for i := 0; i < n; i++ {
		o[i], c[i] = 100, 100.5
		h[i], l[i] = 103, 97 // range 6 → ATR14 ≈ 6 → tol ≈ 0.9
	}
	// two strict pivot highs (isolated by 2 bars each side) within tolerance
	h[6], h[6+0] = 110.0, 110.0
	h[16] = 110.6
	for i := 4; i <= 8; i++ {
		if i != 6 {
			h[i] = 104
		}
	}
	for i := 14; i <= 18; i++ {
		if i != 16 {
			h[i] = 104
		}
	}

	fetch := func(tf string, count int) []market.Kline {
		if tf != "1h" {
			return nil
		}
		return htfKlines(n, o, h, l, c, now)
	}
	got := DetectHTFLevels(fetch, []string{"1h"}, "MNQ", now)
	var eqh *DetectedLevel
	for i := range got {
		if got[i].Kind == KindEQH {
			eqh = &got[i]
		}
	}
	if eqh == nil {
		t.Fatalf("expected EQH·1h from two 1h pivot highs, got %d levels", len(got))
	}
	if !eqh.HTF || eqh.Label != "EQH·1h" {
		t.Fatalf("bad HTF tag/label: %+v", eqh)
	}
}

// TestHTFZoneGradesB covers the v3 floors: 15m zones floor B cap B, 1m zones
// stay C, 1h zones may reach A with confluence (1h wave, 2026-08-25), 4h zones
// may reach A with confluence.
func TestHTFZoneGradesB(t *testing.T) {
	dATR := 100.0
	price := 1000.0
	mkZone := func(tf string) []DetectedLevel {
		lv := []DetectedLevel{{Kind: KindDemand, Price: 1000, Lo: 995, Hi: 1005, Label: "Demand·" + tf, HTF: true, TF: tf}}
		for i := 0; i < 5; i++ { // 5 neighbors inside confBand (0.10×dATR=10)
			lv = append(lv, DetectedLevel{Kind: KindRound, Price: 1001 + float64(i), Lo: 1001 + float64(i), Hi: 1001 + float64(i), Label: "RN"})
		}
		return lv
	}
	gradeOf := func(s []ScoredLevel) string {
		for _, l := range s {
			if l.Kind == KindDemand {
				return l.Grade
			}
		}
		return ""
	}
	if g := gradeOf(ScoreLevels(mkZone("1h"), price, dATR, nil, 8, 1.5)); g != "A" {
		t.Fatalf("1h zone with 5 confluence = %q, want A (1h wave: 1h cap A)", g)
	}
	if g := gradeOf(ScoreLevels(mkZone("1m"), price, dATR, nil, 8, 1.5)); g != "C" {
		t.Fatalf("1m zone with 5 confluence = %q, want C (never above)", g)
	}
	if g := gradeOf(ScoreLevels(mkZone("15m"), price, dATR, nil, 8, 1.5)); g != "B" {
		t.Fatalf("15m zone with 5 confluence = %q, want B (15m stays capped below A)", g)
	}
	if g := gradeOf(ScoreLevels(mkZone("4h"), price, dATR, nil, 8, 1.5)); g != "A" {
		t.Fatalf("4h zone with 5 confluence = %q, want A (v3 allows A on 4h)", g)
	}
}

// TestZoneReversalBonus covers v3 pattern classification: RBD/DBR (reversal)
// outgrades RBR/DBD (continuation) at the same TF.
func TestZoneReversalBonus(t *testing.T) {
	rev := DetectedLevel{Kind: KindSupply, Price: 1000, Lo: 995, Hi: 1005, Label: "Supply·1h", HTF: true, TF: "1h", ZonePattern: "reversal"}
	cont := DetectedLevel{Kind: KindSupply, Price: 1000, Lo: 995, Hi: 1005, Label: "Supply·1h", HTF: true, TF: "1h", ZonePattern: "continuation"}
	if eR, eC := zoneEvidence(rev), zoneEvidence(cont); eR <= eC {
		t.Fatalf("reversal zone evidence %.3f must exceed continuation %.3f", eR, eC)
	}
}

// TestSeatHTFPromotesSwingLevels covers G6: up to 2 HTF swing/zone levels win
// seats over weaker non-priority entries so the model's top-N table sees them.
func TestSeatHTFPromotesSwingLevels(t *testing.T) {
	scored := make([]ScoredLevel, 0, 10)
	for i := 0; i < 8; i++ { // weak round numbers fill the head
		scored = append(scored, ScoredLevel{
			DetectedLevel: DetectedLevel{Kind: KindRound, Price: 990 + float64(i), Label: "RN"},
			Score: 0.4, Grade: "C", Fresh: "fresh", Distance: 990 + float64(i) - 1000,
		})
	}
	for i := 0; i < 2; i++ { // HTF swings lost the cut
		scored = append(scored, ScoredLevel{
			DetectedLevel: DetectedLevel{Kind: KindEQH, Price: 1100 + float64(i), Label: "EQH·1h", HTF: true},
			Score: 0.84, Grade: "B", Fresh: "fresh", Distance: 100 + float64(i),
		})
	}
	out := seatHTF(scored, 8)
	head, tail := out[:8], out[8:]
	htfInHead := 0
	for _, l := range head {
		if isHTFSwingZone(l) {
			htfInHead++
		}
	}
	if htfInHead != 2 {
		t.Fatalf("seatHTF promoted %d HTF levels (want 2); head: %+v", htfInHead, head)
	}
	if len(tail) != 2 || isHTFSwingZone(tail[0]) || isHTFSwingZone(tail[1]) {
		t.Fatalf("demoted entries must be the weak non-HTF ones: %+v", tail)
	}
	// today-priority entries must never be demoted
	pri := []ScoredLevel{
		{DetectedLevel: DetectedLevel{Kind: KindPDH, Price: 1200, Label: "PDH", HTF: true}, Score: 1.4, Grade: "A", Fresh: "fresh", Distance: 200},
	}
	all := append(pri, scored...)
	out2 := seatHTF(all, 8)
	priSeated := false
	for _, l := range out2[:8] {
		if l.Kind == KindPDH {
			priSeated = true
		}
	}
	if !priSeated {
		t.Fatal("seatHTF demoted a today-priority level")
	}
}

// TestSeat1HZonePromotesInBandSD covers the 1h wave (2026-08-25): when the
// head holds no 1h S/D zone but the tail has one, the strongest tail candidate
// wins a seat by demoting the weakest non-priority, non-HTF head entry.
func TestSeat1HZonePromotesInBandSD(t *testing.T) {
	mk := func(kind LevelKind, tf string, price, score float64, grade string) ScoredLevel {
		return ScoredLevel{
			DetectedLevel: DetectedLevel{Kind: kind, Price: price, Label: "L·" + tf, TF: tf},
			Score: score, Grade: grade, Fresh: "fresh", Distance: price - 1000,
		}
	}
	scored := []ScoredLevel{
		mk(KindRound, "1m", 990, 0.5, "C"),
		mk(KindRound, "1m", 991, 0.5, "C"),
		mk(KindRound, "1m", 992, 0.5, "C"),
		mk(KindEQH, "4h", 1200, 0.84, "B"),
		mk(KindEQH, "4h", 1201, 0.84, "B"),
		mk(KindSupply, "4h", 1210, 0.7, "B"),
		mk(KindRound, "1m", 993, 0.5, "C"),
		mk(KindRound, "1m", 994, 0.5, "C"),
		// tail candidate: a 1h S/D zone that seatHTF left out
		mk(KindDemand, "1h", 950, 0.9, "B"),
	}
	out := Seat1HZone(scored, 8)
	head, tail := out[:8], out[8:]
	seated1h := false
	for _, l := range head {
		if is1HSDZone(l) {
			seated1h = true
		}
	}
	if !seated1h {
		t.Fatalf("Seat1HZone did not promote the 1h S/D zone; head: %+v", head)
	}
	if len(tail) != 1 || isHTFSwingZone(tail[0]) || is1HSDZone(tail[0]) {
		t.Fatalf("demoted entry must be a weak non-HTF level: %+v", tail)
	}
	// No-op when nothing was cut.
	if out2 := Seat1HZone(scored[:6], 8); len(out2) != 6 {
		t.Fatalf("no-cut case must be a no-op, got %d", len(out2))
	}
	// No-op when a 1h S/D zone is already seated.
	seated := append([]ScoredLevel{mk(KindSupply, "1h", 940, 0.9, "B")}, scored...)
	out3 := Seat1HZone(seated, 8)
	if len(out3) != len(seated) {
		t.Fatalf("already-seated case must be a no-op")
	}
}
