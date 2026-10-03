package mentor

import (
	"testing"

	"vl/market"
)

// TestISBIdentifiedByBodyOnly — §2.1 [D2.1 p1 @ 04:18]: "cây nến ISB em không
// tính râu, em chỉ tính phần thân". Wicks may poke outside; the BODY must sit
// inside the previous body.
func TestISBIdentifiedByBodyOnly(t *testing.T) {
	prev := market.Kline{Open: 100, Close: 105, High: 107, Low: 98}
	// body inside, wicks far outside → ISB
	cur := market.Kline{Open: 101, Close: 104, High: 110, Low: 90}
	if !IsISB(prev, cur) {
		t.Fatal("body inside body is an ISB regardless of wicks")
	}
	// body pokes above → not an ISB even though the whole range is inside
	cur2 := market.Kline{Open: 101, Close: 106, High: 106.5, Low: 99}
	if IsISB(prev, cur2) {
		t.Fatal("body beyond the previous body is NOT an ISB")
	}
	// equal bodies (exact range) → ISB (resting in place)
	cur3 := market.Kline{Open: 100, Close: 105, High: 105.5, Low: 99.5}
	if !IsISB(prev, cur3) {
		t.Fatal("equal body is an ISB")
	}
}

// TestISBOrdersWickInclusiveWithBuffer — §2.1 [D2.1 p1 @ 04:39, 05:36;
// D3.4 p1 @ 23:04]: orders at the WICK-INCLUSIVE extremes ± 1.5 buffer.
func TestISBOrdersWickInclusiveWithBuffer(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Enabled = true
	c := market.Kline{High: 100.5, Low: 98.25}
	long, short := ISBOrders(c, cfg)
	if long.Price != 102.0 || long.Stop != 96.75 {
		t.Fatalf("long = %+v, want price 102.0 stop 96.75", long)
	}
	if short.Price != 96.75 || short.Stop != 102.0 {
		t.Fatalf("short = %+v, want price 96.75 stop 102.0", short)
	}
	if long.Action != PlaceStopEntry || short.Action != PlaceStopEntry {
		t.Fatalf("actions = %q/%q", long.Action, short.Action)
	}
}

// TestISBStopVerdict — §6 [D3.2 p1 @ 14:18; D4.1 p1 @ 05:41; D3.3 p1 @ 02:04]:
// 5–6 pts normal; below 5 too small; in the twenties never; over 25 never.
func TestISBStopVerdict(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Enabled = true
	cases := []struct {
		name   string
		span   float64
		wantOK bool
	}{
		{"normal 5pt stop", 2.0, true}, // 2 + 3 buffer = 5
		{"normal 6pt stop", 3.0, true}, // 3 + 3 = 6
		{"too small", 1.0, false},      // 1 + 3 = 4 < 5
		{"twenties", 17.0, false},      // 17 + 3 = 20
		{"over ceiling", 23.0, false},  // 23 + 3 = 26 > 25
	}
	for _, c := range cases {
		stop, ok, _ := ISBStopVerdict(market.Kline{High: 100 + c.span/2, Low: 100 - c.span/2}, cfg)
		if ok != c.wantOK {
			t.Fatalf("%s: ok=%v (stop %.2f), want %v", c.name, ok, stop, c.wantOK)
		}
	}
}

// TestISBStackingArithmetic — §2.1 rule 1 + stacking [D4.2 p2 @ 08:49–16:08]:
// 1 → cancel, 2–3 → hold, 4 → cancel (becomes a 5m ISB).
func TestISBStackingArithmetic(t *testing.T) {
	if got := ISBStackAdvice(1); got != "cancel" {
		t.Fatalf("1 inside = %q, want cancel", got)
	}
	if got := ISBStackAdvice(2); got != "hold" {
		t.Fatalf("2 inside = %q, want hold", got)
	}
	if got := ISBStackAdvice(3); got != "hold" {
		t.Fatalf("3 inside = %q, want hold", got)
	}
	if got := ISBStackAdvice(4); got != "cancel" {
		t.Fatalf("4 inside = %q, want cancel", got)
	}
}

// TestISBStackTickEmitsCancelAtFour — the stacking counter: 1 inside candle
// → CANCEL (rule 1 [D4.1 p1 @ 08:05]); 2–3 → hold; 4 → CANCEL [D4.2 p2
// @ 08:49–16:08]; a body escaping the first ISB ends the arm.
func TestISBStackTickEmitsCancelAtFour(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Enabled = true
	first := market.Kline{Open: 100, Close: 105, High: 106, Low: 99}
	inside := market.Kline{Open: 101, Close: 104, High: 105, Low: 100}

	// 1 → cancel (the ISB order that did not fill after the next candle)
	arm := &ISBArm{FirstBar: first}
	got := ISBStackTick(arm, inside, cfg)
	if len(got) != 1 || got[0].Action != CancelArm {
		t.Fatalf("1st inside = %+v, want one cancel (rule 1)", got)
	}

	// 2–3 → hold, 4 → cancel
	arm = &ISBArm{FirstBar: first}
	if got := ISBStackTick(arm, inside, cfg); len(got) != 1 {
		t.Fatalf("count 1 must cancel: %+v", got)
	}
	// restart the counter the way the evaluator does: a re-placed arm counts again
	arm.Inside = 0
	for n := 1; n <= 4; n++ {
		got := ISBStackTick(arm, inside, cfg)
		switch n {
		case 1:
			if len(got) != 1 || got[0].Action != CancelArm {
				t.Fatalf("count 1 = %+v, want cancel", got)
			}
		case 2, 3:
			if len(got) != 0 {
				t.Fatalf("count %d = %+v, want hold (no intents)", n, got)
			}
		case 4:
			if len(got) != 1 || got[0].Action != CancelArm {
				t.Fatalf("count 4 = %+v, want cancel", got)
			}
		}
	}
	// a candle whose body escapes the first ISB ends the arm
	arm2 := &ISBArm{FirstBar: first}
	ISBStackTick(arm2, inside, cfg)
	escape := market.Kline{Open: 101, Close: 107, High: 107, Low: 100}
	if got := ISBStackTick(arm2, escape, cfg); len(got) != 0 || arm2.Inside != -1 {
		t.Fatalf("escape = %+v inside=%d, want no intents and arm over", got, arm2.Inside)
	}
}

// TestISBOnRecordedTape — canon 53: every body-inside-bar on the recorded
// 2026-09-15 RTH day produces orders whose stop verdict matches the knobs,
// and the tape contains a sane number of ISBs.
func TestISBOnRecordedTape(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Enabled = true
	bars := loadFixture(t, "mnq_1m_2026-09-15_rth", "1m")
	isbs := 0
	for i := 1; i < len(bars); i++ {
		if !IsISB(bars[i-1], bars[i]) {
			continue
		}
		isbs++
		long, short := ISBOrders(bars[i], cfg)
		if long.Stop >= long.Price || short.Stop <= short.Price {
			t.Fatalf("bar %d: inverted ISB orders %+v %+v", i, long, short)
		}
	}
	if isbs < 20 {
		t.Fatalf("recorded day has %d ISBs — tape too thin", isbs)
	}
	t.Logf("recorded day: %d body-ISBs on %d bars", isbs, len(bars))
}
