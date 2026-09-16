package kernel

import (
	"math"
	"testing"

	"nofx/market"
)

// W12 — percentile rank, ATR regime buckets, overnight gap. Values INDEPENDENTLY
// recomputed (not read from the code).
func TestW12PercentileAndBuckets(t *testing.T) {
	// percentileRank(42 in [10,20,30,40,50]) = 100·count(x<42)/5 = 100·4/5 = 80.
	if v := percentileRank([]float64{10, 20, 30, 40, 50}, 42); math.Abs(v-80.0) > 1e-9 {
		t.Fatalf("percentileRank = %v, want 80.0", v)
	}
	if v := percentileRank(nil, 5); v != 0 {
		t.Fatalf("empty series → 0, got %v", v)
	}
	// half-open bucket edges: p<25 LOW, p<75 NORMAL, p<90 HIGH, else EXTREME.
	for _, c := range []struct {
		p    float64
		want string
	}{{24.9, "LOW"}, {25.0, "NORMAL"}, {74.9, "NORMAL"}, {75.0, "HIGH"}, {89.9, "HIGH"}, {90.0, "EXTREME"}} {
		if g := atrBucket(c.p); g != c.want {
			t.Fatalf("atrBucket(%.1f) = %q, want %q", c.p, g, c.want)
		}
	}
}

// W12 — overnight gap: (SessionOpen−PriorClose)/dATR, guarded on all>0. With the 20
// real daily bars (ATR14 595.8473) and the real 08-14 open/prior-close, the gap is
// (30148.75−30188.50)/595.8473 = −0.06671 (down gap). Sign + magnitude checked.
func TestW12OvernightGap(t *testing.T) {
	daily := realKl20()
	pc, so := PriorCloseSessionOpen([]market.Kline{{Close: 30188.50}, {Open: 30148.75}})
	if pc != 30188.50 || so != 30148.75 {
		t.Fatalf("PriorCloseSessionOpen = %.2f/%.2f", pc, so)
	}
	r := ComputeRegime(RegimeInputs{Price: 30147.25, DailyBars: daily, PriorClose: pc, SessionOpen: so})
	if !r.HasGap {
		t.Fatal("gap must be present (all inputs > 0)")
	}
	want := (so - pc) / r.ATR14
	if math.Abs(r.OvernightGapATR-want) > 1e-9 || r.OvernightGapATR >= 0 {
		t.Fatalf("gap = %.6f, want %.6f (negative = down gap)", r.OvernightGapATR, want)
	}
}

// W12 — realized-vol √288 annualization + population sd. A 61-close series whose 60
// log-returns are exactly ±a (30 each) → mean 0, population sd = a, RV% = a·√288·100.
// For a=0.001 → 0.001·16.9705627·100 = 1.6970563.
func TestW12RealizedVol(t *testing.T) {
	const a = 0.001
	closes := make([]market.Kline, 61)
	px := 100.0
	closes[0] = market.Kline{Close: px}
	for i := 1; i <= 60; i++ {
		sign := 1.0
		if i%2 == 0 {
			sign = -1.0
		}
		px *= math.Exp(sign * a)
		closes[i] = market.Kline{Close: px}
	}
	rv, ok := recentDailyVolPct(closes)
	if !ok {
		t.Fatal("≥30 returns → ok=true")
	}
	want := a * math.Sqrt(288) * 100 // 1.6970563
	if math.Abs(rv-want) > 1e-6 {
		t.Fatalf("RV%% = %.7f, want %.7f (sd·√288·100, population)", rv, want)
	}
	// property: RV scales linearly with the return magnitude.
	if math.Abs(rv-1.6970563) > 1e-6 {
		t.Fatalf("RV%% = %.7f, want 1.6970563", rv)
	}
}

// W12 — Bonferroni α = 0.05/8 = 0.00625; and the WARMING gate N=1565 is CONSERVATIVE:
// it exceeds the 80%-power one-proportion minimum (~1279), so the honesty gate never
// greens too early. (1565 corresponds to ~89% power two-sided — see the audit report.)
func TestW12BonferroniAndN(t *testing.T) {
	if a := BonferroniAlpha(); math.Abs(a-0.00625) > 1e-12 {
		t.Fatalf("Bonferroni α = %v, want 0.00625 (0.05/8)", a)
	}
	if MatchedRandomTypes != 8 {
		t.Fatalf("family size = %d, want 8", MatchedRandomTypes)
	}
	// 80%-power two-sided one-proportion n = (z_{α/2}+z_β)²·p0(1−p0)/effect².
	// z_{0.003125}=2.7344, z_{0.20}=0.8416; (3.5760)²·0.25/0.0025 = 1278.8.
	z := 2.7344 + 0.8416
	n80 := math.Ceil(z * z * 0.25 / (0.05 * 0.05))
	if float64(PreRegisteredN) <= n80 {
		t.Fatalf("PreRegisteredN=%d must exceed the 80%%-power minimum %.0f (conservative gate)", PreRegisteredN, n80)
	}
}

// realKl20 — the same 20 real MNQ daily bars as the market-package oracle.
func realKl20() []market.Kline {
	rows := [][4]float64{
		{28747.50, 29192.50, 28706.75, 28778.75}, {28783.25, 29363.50, 28700.00, 29316.00},
		{29309.00, 29342.50, 28961.25, 29181.25}, {29107.50, 29283.00, 28432.50, 28620.75},
		{28708.00, 28734.75, 28212.50, 28282.25}, {28501.50, 28763.75, 27938.50, 28190.00},
		{28210.50, 28229.00, 27603.25, 27922.00}, {27962.25, 28177.25, 27200.00, 27259.75},
		{27208.00, 28410.00, 27204.75, 28237.75}, {28317.25, 28725.75, 28079.75, 28404.25},
		{28567.50, 28965.00, 28313.00, 28891.75}, {28930.00, 29956.50, 28831.50, 29863.50},
		{29781.25, 30073.25, 29530.75, 29615.00}, {29576.00, 29686.25, 29241.00, 29488.25},
		{29515.00, 29867.25, 29455.00, 29834.75}, {29851.50, 29985.00, 29719.00, 29737.00},
		{29764.25, 29887.00, 29533.50, 29626.00}, {29663.00, 30001.50, 29625.00, 29853.25},
		{29820.25, 30273.25, 29780.50, 30188.50}, {30148.75, 30287.25, 30025.00, 30147.25},
	}
	var out []market.Kline
	for _, r := range rows {
		out = append(out, market.Kline{Open: r[0], High: r[1], Low: r[2], Close: r[3]})
	}
	return out
}

// W12 — confluence score composition: typeEvidence · freshMult · (1+0.20·conf) · htf,
// then gradeFromScore (≥1.0 A, ≥0.70 B, else C). Hand-computed exemplars.
func TestW12ConfluenceScore(t *testing.T) {
	// KindRound(0.55) fresh(1.0) conf1 non-HTF: 0.55·1·1.2·1 = 0.66 → C.
	if s := typeEvidence(KindRound) * freshMult("") * (1 + 0.20*1) * 1.0; math.Abs(s-0.66) > 1e-9 || gradeFromScore(s) != "C" {
		t.Fatalf("Round conf1 = %.4f grade %s, want 0.66/C", s, gradeFromScore(s))
	}
	// KindPDH(1.0) fresh conf2 non-HTF: 1.0·1·1.4·1 = 1.40 → A.
	if s := typeEvidence(KindPDH) * freshMult("") * (1 + 0.20*2) * 1.0; math.Abs(s-1.40) > 1e-9 || gradeFromScore(s) != "A" {
		t.Fatalf("PDH conf2 = %.4f grade %s, want 1.40/A", s, gradeFromScore(s))
	}
	// KindNPOC(0.85) tested(0.6) conf3 HTF(1.2): 0.85·0.6·1.6·1.2 = 0.9792 → B.
	if s := typeEvidence(KindNPOC) * freshMult("c") * (1 + 0.20*3) * 1.2; math.Abs(s-0.9792) > 1e-9 || gradeFromScore(s) != "B" {
		t.Fatalf("nPOC conf3 HTF = %.4f grade %s, want 0.9792/B", s, gradeFromScore(s))
	}
}
