package mentor

import "vl/market"

// Box trade evaluation — BOX REUSE (R1) and CONFLUENCE (R2), CTO rulings
// 2026-10-03 verified in the sources [A].

// BoxReturn is one post-formation return visit [D3.2 p2 @ 06:25: "minh se
// dung box nay hoai ne" — we use this box again and again]. A visit =
// price was outside the box on the approach side, then a candle touches an
// edge; RefBar is that first touching candle — the visit's reference
// candle. EVERY return gets its own reference candle, the same reject rule
// (close outside on the approach side) and the same filters. The FIRST
// return is the third touch overall, counting the two extremes that built
// the box [D3.2 p1 @ 04:29].
type BoxReturn struct {
	N      int // 1-based return number (1 = the third touch)
	RefBar int // bar index of the visit's reference candle
}

// BoxReturnBars lists every post-formation return visit, one entry per
// visit (not per candle). A candle that closes outside the box on the
// approach side opens a visit; the next candle whose wick touches an edge
// is the visit's reference candle. Consecutive touching candles are the
// same visit.
func BoxReturnBars(bars []market.Kline, b Box, formedAt int, cfg BoxCfg) []BoxReturn {
	var out []BoxReturn
	outside := false
	for i := formedAt + 1; i < len(bars); i++ {
		c := bars[i]
		if c.CloseTime == 0 {
			continue
		}
		switch b.Kind {
		case FTGH:
			if c.Close < b.Bottom {
				outside = true
				continue
			}
		case FTGL:
			if c.Close > b.Top {
				outside = true
				continue
			}
		}
		if outside && touchesEdge(b, c, cfg) {
			out = append(out, BoxReturn{N: len(out) + 1, RefBar: i})
			outside = false
			continue
		}
		if !touchesEdge(b, c, cfg) {
			outside = false
		}
	}
	return out
}

// ConfluenceWithinPts is the R2 "at" tolerance: the key level must lie
// INSIDE the box or within 2 pts of its edge [00-METHOD Risk-reward;
// D3.4 p3 @ 07:38].
const ConfluenceWithinPts = 2.0

// ConfluenceFlag is the R2 output for DS-102's exit-C / size-10 branch.
// On=false means normal sizing; the size-20 escalation (4h AND 1h agree AND
// room >= 30 pts) is DS-102's tier rule, not computed here.
type ConfluenceFlag struct {
	On   bool
	Side Side // the confluence side ("" when off)
}

// ConfluenceVerdict evaluates the R2 confluence test [00-METHOD Risk-reward,
// D3.4 p3 @ 07:38] for one box trade setup:
//
//	LONG  = an FTGL box (support) AND a key level inside the box or within
//	        2 pts of its edge AND the 5m BUY trigger agrees.
//	SHORT = FTGH + key level + 5m SELL trigger.
//
// side is the trade side; trig is the 5m trigger line. Fail-closed: no
// trigger line (empty direction) can never agree, so confluence stays off.
func ConfluenceVerdict(b Box, side Side, keyLevels []Level, trig TriggerLine) ConfluenceFlag {
	if side != SideLong && side != SideShort {
		return ConfluenceFlag{}
	}
	entry := b.Top
	if side == SideShort {
		entry = b.Bottom
	}
	if ok, trigSide, _ := TriggerVerdict(trig, entry); !ok || trigSide == "" || trigSide != side {
		return ConfluenceFlag{}
	}
	switch side {
	case SideLong:
		if b.Kind != FTGL {
			return ConfluenceFlag{}
		}
	case SideShort:
		if b.Kind != FTGH {
			return ConfluenceFlag{}
		}
	}
	for _, l := range keyLevels {
		if l.Kind != KindKeyLevel {
			continue
		}
		if l.Price >= b.Bottom-ConfluenceWithinPts && l.Price <= b.Top+ConfluenceWithinPts {
			return ConfluenceFlag{On: true, Side: side}
		}
	}
	return ConfluenceFlag{}
}
