package mentor

import (
	"testing"

	"vl/market"
)

// TestISBIdentifiedByBodyOnly — RULES-FIX-v3 R1 [D1.4 p1 @ 06:13–08:47]: the
// ISB candle's BODY must sit inside the PREVIOUS candle's FULL range, wicks
// included ("Inside có nghĩa là cái BODY của cây nến đó phải nằm BÊN TRONG cây
// nến trend"); the ISB candle's wicks and colour are free.
func TestISBIdentifiedByBodyOnly(t *testing.T) {
	prev := market.Kline{Open: 100, Close: 105, High: 107, Low: 98}
	// body inside the full range, wicks far outside → ISB (wicks ignored)
	cur := market.Kline{Open: 101, Close: 104, High: 110, Low: 90}
	if !IsISB(prev, cur) {
		t.Fatal("body inside the full range is an ISB regardless of its own wicks")
	}
	// body pokes ABOVE the previous HIGH → not an ISB
	cur2 := market.Kline{Open: 103, Close: 108, High: 108.5, Low: 99}
	if IsISB(prev, cur2) {
		t.Fatal("body beyond the previous FULL range is NOT an ISB")
	}
	// body below the previous LOW → not an ISB
	cur3 := market.Kline{Open: 99, Close: 97, High: 99.5, Low: 96}
	if IsISB(prev, cur3) {
		t.Fatal("body below the previous low is NOT an ISB")
	}
	// equal range → ISB (resting in place)
	cur4 := market.Kline{Open: 100, Close: 105, High: 105.5, Low: 99.5}
	if !IsISB(prev, cur4) {
		t.Fatal("equal body is an ISB")
	}
}

// TestISBOrdersWickInclusiveWithBuffer — R1 [D1.4 p1 @ 10:33–11:25, 22:22–22:30,
// 24:41–24:55]: ENTRY at the ISB candle's OWN extreme, stop at the opposite
// extreme, 1.5-pt buffer BOTH sides outward, order type STOP-LIMIT with
// limit = the stop (trigger) price.
func TestISBOrdersWickInclusiveWithBuffer(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Enabled = true
	c := market.Kline{High: 100.5, Low: 98.25}
	long, short := ISBOrders(c, cfg)
	if long.Price != 102.0 || long.Stop != 96.75 || long.Limit != 102.0 {
		t.Fatalf("long = %+v, want price 102.0 stop 96.75 limit 102.0", long)
	}
	if short.Price != 96.75 || short.Stop != 102.0 || short.Limit != 96.75 {
		t.Fatalf("short = %+v, want price 96.75 stop 102.0 limit 96.75", short)
	}
	if long.Action != PlaceStopLimitEntry || short.Action != PlaceStopLimitEntry {
		t.Fatalf("actions = %q/%q, want stop-limit both sides", long.Action, short.Action)
	}
}

// TestISBDirectionAndStopLimitOrder — R1 [D1.4 p1 @ 09:20–10:20]: direction is
// the COLOUR OF CANDLE 1 (green → long, red → short); the emitted order is the
// single stop-limit on that side, and a twenties stop is skipped.
func TestISBDirectionAndStopLimitOrder(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Enabled = true
	green := market.Kline{Open: 100, Close: 106, High: 106.5, Low: 99.5}
	red := market.Kline{Open: 106, Close: 100, High: 106.5, Low: 99.5}
	cur := market.Kline{Open: 102, Close: 104, High: 106.0, Low: 100.0}
	if ISBDirection(green) != SideLong || ISBDirection(red) != SideShort {
		t.Fatalf("direction = %q/%q, want long/short", ISBDirection(green), ISBDirection(red))
	}
	dir, in, ok, _ := ISBStopLimitOrder(green, cur, cfg)
	if !ok || dir != SideLong || in.Side != SideLong || in.Action != PlaceStopLimitEntry {
		t.Fatalf("order = %+v ok=%v dir=%q, want a long stop-limit", in, ok, dir)
	}
	dir, _, ok, _ = ISBStopLimitOrder(red, cur, cfg)
	if !ok || dir != SideShort {
		t.Fatalf("short side order: ok=%v dir=%q", ok, dir)
	}
	// a candle whose stop lands in the twenties is skipped, citing D4.1
	wide := market.Kline{Open: 102, Close: 104, High: 112.0, Low: 93.0}
	if _, _, ok, reason := ISBStopLimitOrder(green, wide, cfg); ok || reason == "" {
		t.Fatalf("twenties stop: ok=%v reason=%q, want skip with a cited reason", ok, reason)
	}
}

// TestISBStopVerdict — RULES-FIX-v3 R1 [D4.1 p1 @ 05:41]: NO fixed 5–6-pt stop
// and NO ceiling for the ISB; the ONLY skip is a stop in the twenties.
func TestISBStopVerdict(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Enabled = true
	cases := []struct {
		name   string
		span   float64
		wantOK bool
	}{
		{"small 4pt stop", 1.0, true},        // 1 + 3 buffer = 4 — taken (no minimum)
		{"5pt stop", 2.0, true},              // 2 + 3 = 5
		{"6pt stop", 3.0, true},              // 3 + 3 = 6
		{"twenties", 17.0, false},            // 17 + 3 = 20 — the only skip
		{"thirties not skipped", 28.0, true}, // 31 — only the twenties skip (no ceiling)
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
