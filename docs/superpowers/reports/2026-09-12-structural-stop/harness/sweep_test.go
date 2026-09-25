package main

import (
	"nofx/market"
	"testing"
)

func TestCorrectedFills(t *testing.T) {
	e := event{Side: "long", Bars: []market.Kline{{Open: 105, High: 106, Low: 101, Close: 103, OpenTime: 1}}}
	a := sweepRow{Entry: 100, Stop: 99, Target: 104, Fill: "A_touch"}
	simulateCorrected(e, &a)
	if a.Filled {
		t.Fatal("band intersection cannot manufacture anchor fill")
	}
	e.Bars = []market.Kline{{Open: 101, High: 105, Low: 98, Close: 103, OpenTime: 1}}
	a = sweepRow{Entry: 100, Stop: 99, Target: 104, Fill: "A_touch"}
	simulateCorrected(e, &a)
	if !a.Filled || a.Net != -3 || a.NetUpper != 2 || a.Exit != "stop" {
		t.Fatalf("fill-bar stop and ambiguity bounds must count: %+v", a)
	}
	c := sweepRow{Entry: 100, Stop: 99, Target: 104, Fill: "C_adverse_tick"}
	simulateCorrected(e, &c)
	if c.Net != a.Net-.25 {
		t.Fatalf("adverse entry must cost .25, got %+v", c)
	}
	e.Bars = []market.Kline{{Open: 101, High: 102, Low: 100, Close: 101, OpenTime: 1}}
	b := sweepRow{Entry: 100, Stop: 99, Target: 104, Fill: "B_through"}
	simulateCorrected(e, &b)
	if b.Filled {
		t.Fatal("touch cannot satisfy through-tick model")
	}
	e.Side = "short"
	e.Bars = []market.Kline{{Open: 99, High: 102, Low: 95, Close: 98, OpenTime: 1}}
	a = sweepRow{Entry: 100, Stop: 101, Target: 96, Fill: "A_touch"}
	simulateCorrected(e, &a)
	c = sweepRow{Entry: 100, Stop: 101, Target: 96, Fill: "C_adverse_tick"}
	simulateCorrected(e, &c)
	if a.Net != -3 || c.Net != a.Net-.25 {
		t.Fatalf("short mirror: A=%+v C=%+v", a, c)
	}
}
