package mentor

// Box trade evaluation — CONFLUENCE (R2), CTO ruling 2026-10-03 verified in
// the sources [A]. The box ENTRY path itself is the box-edge level path in
// eval.go (one implementation — DS-103); this file only carries the R2
// confluence test that DS-103 splices onto box-edge entry intents for
// DS-102's exit-C / size-10 branch.

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
