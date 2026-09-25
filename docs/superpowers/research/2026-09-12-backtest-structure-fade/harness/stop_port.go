package main

// stop_port.go — VERBATIM PORT of trader/arm_stop_anchor.go's pure
// composeArmStop. It is UNEXPORTED in package trader, so a harness in another
// package cannot call it (Go has no cross-package access to unexported symbols
// without linkname tricks). Precedent: the accepted vet-08 replay shipped the
// same verbatim copy ("Verbatim pure source from trader/arm_stop_anchor.go
// (pinned base)", docs/superpowers/reports/2026-09-05-vet-08-stretch-data/
// complete-0905/replay.go).
//
// Divergence guard: verifyPort() at startup reads the SOURCE file from this
// checkout and asserts the function text below is byte-identical to what is in
// the tree, so this copy cannot drift silently (class 97).
import (
	"fmt"
	"math"
	"os"
	"strings"

	"nofx/kernel"
)

type StopComposition struct {
	Stop         float64 // the chosen stop
	Authored     float64 // what the planner wrote
	AnchorPrice  float64 // the seated level the stop sits beyond (0 = none)
	AnchorLabel  string  // that level's provenance chip
	AnchorStop   float64 // anchor ± clearance (0 = none)
	ATRFloorStop float64 // entry ∓ mult×ATR5m (0 = no ATR)
	Bound        string  // anchor | atr_floor | authored — which one won
	Unanchored   bool    // no seated level within the dead-zone bound
}

// composeArmStop is the pure stop composition (fixture-tested).
//
//	side      long|short
//	entry     the arm's entry price
//	authored  the planner's stop (never tightened)
//	atr5m     ATR(14) on 5m; ≤0 → the ATR leg is skipped (fail-open)
//	tick      instrument tick size
//	levels    the plan's seated levels
//	mult      MIN_SL_ATR_MULT (resolved)
//	clearTicks the level-clearance leg (MinSLTickClearance)
//	maxAnchorATR the dead-zone bound in ATR units; ≤0 disables anchoring
func composeArmStop(side string, entry, authored, atr5m, tick float64, levels []kernel.PlanLevel, mult float64, clearTicks int, maxAnchorATR float64) StopComposition {
	c := StopComposition{Stop: authored, Authored: authored, Bound: "authored"}
	long := strings.EqualFold(strings.TrimSpace(side), "long")
	if entry <= 0 || authored <= 0 {
		return c // no usable geometry — leave the authored stop untouched
	}
	if tick <= 0 {
		tick = 0.25
	}
	clearance := float64(clearTicks) * tick

	// ATR floor leg.
	if atr5m > 0 && mult > 0 {
		if long {
			c.ATRFloorStop = entry - mult*atr5m
		} else {
			c.ATRFloorStop = entry + mult*atr5m
		}
	}

	// Anchor leg: the NEAREST seated level on the RISK side (below entry for a
	// long, above for a short), within the dead-zone bound.
	if maxAnchorATR > 0 {
		bound := math.MaxFloat64
		if atr5m > 0 {
			bound = maxAnchorATR * atr5m
		}
		best, bestDist, found := 0.0, math.MaxFloat64, false
		for _, l := range levels {
			if l.Price <= 0 {
				continue
			}
			var dist float64
			if long {
				if l.Price >= entry {
					continue // not on the risk side
				}
				dist = entry - l.Price
			} else {
				if l.Price <= entry {
					continue
				}
				dist = l.Price - entry
			}
			if dist > bound {
				continue // dead zone
			}
			if dist < bestDist {
				best, bestDist, found = l.Price, dist, true
				c.AnchorLabel = l.Label
			}
		}
		if found {
			c.AnchorPrice = best
			if long {
				c.AnchorStop = best - clearance
			} else {
				c.AnchorStop = best + clearance
			}
		} else {
			c.Unanchored = true
			c.AnchorLabel = ""
		}
	} else {
		c.Unanchored = true
	}

	// WIDEST WINS. For a long a wider stop is LOWER; for a short, HIGHER.
	pick := func(cand float64, name string) {
		if cand <= 0 {
			return
		}
		if (long && cand < c.Stop) || (!long && cand > c.Stop) {
			c.Stop, c.Bound = cand, name
		}
	}
	pick(c.AnchorStop, "anchor")
	pick(c.ATRFloorStop, "atr_floor")
	return c
}

// portSourceText is the byte-identical function text embedded above (the
// signature through its closing brace). verifyPort compares it against the
// checkout's trader/arm_stop_anchor.go so the port cannot drift silently.
func verifyPort() error {
	src, err := os.ReadFile("trader/arm_stop_anchor.go")
	if err != nil {
		return fmt.Errorf("cannot read trader/arm_stop_anchor.go: %w", err)
	}
	body := string(src)
	start := strings.Index(body, "func composeArmStop(")
	if start < 0 {
		return fmt.Errorf("func composeArmStop not found in trader/arm_stop_anchor.go")
	}
	// The function ends at the first line that is exactly "}" at column 0.
	lines := strings.Split(body[start:], "\n")
	fn := []string{}
	for _, ln := range lines {
		fn = append(fn, ln)
		if ln == "}" {
			break
		}
	}
	fromSrc := strings.Join(fn, "\n")
	want := strings.TrimSpace(embeddedPort)
	got := strings.TrimSpace(fromSrc)
	if want != got {
		return fmt.Errorf("PORT DRIFT: harness composeArmStop no longer byte-identical to trader/arm_stop_anchor.go — abort (class 97). src=%d bytes embedded=%d bytes", len(got), len(want))
	}
	return nil
}

// embeddedPort mirrors composeArmStop exactly (signature through closing brace).
const embeddedPort = `func composeArmStop(side string, entry, authored, atr5m, tick float64, levels []kernel.PlanLevel, mult float64, clearTicks int, maxAnchorATR float64) StopComposition {
	c := StopComposition{Stop: authored, Authored: authored, Bound: "authored"}
	long := strings.EqualFold(strings.TrimSpace(side), "long")
	if entry <= 0 || authored <= 0 {
		return c // no usable geometry — leave the authored stop untouched
	}
	if tick <= 0 {
		tick = 0.25
	}
	clearance := float64(clearTicks) * tick

	// ATR floor leg.
	if atr5m > 0 && mult > 0 {
		if long {
			c.ATRFloorStop = entry - mult*atr5m
		} else {
			c.ATRFloorStop = entry + mult*atr5m
		}
	}

	// Anchor leg: the NEAREST seated level on the RISK side (below entry for a
	// long, above for a short), within the dead-zone bound.
	if maxAnchorATR > 0 {
		bound := math.MaxFloat64
		if atr5m > 0 {
			bound = maxAnchorATR * atr5m
		}
		best, bestDist, found := 0.0, math.MaxFloat64, false
		for _, l := range levels {
			if l.Price <= 0 {
				continue
			}
			var dist float64
			if long {
				if l.Price >= entry {
					continue // not on the risk side
				}
				dist = entry - l.Price
			} else {
				if l.Price <= entry {
					continue
				}
				dist = l.Price - entry
			}
			if dist > bound {
				continue // dead zone
			}
			if dist < bestDist {
				best, bestDist, found = l.Price, dist, true
				c.AnchorLabel = l.Label
			}
		}
		if found {
			c.AnchorPrice = best
			if long {
				c.AnchorStop = best - clearance
			} else {
				c.AnchorStop = best + clearance
			}
		} else {
			c.Unanchored = true
			c.AnchorLabel = ""
		}
	} else {
		c.Unanchored = true
	}

	// WIDEST WINS. For a long a wider stop is LOWER; for a short, HIGHER.
	pick := func(cand float64, name string) {
		if cand <= 0 {
			return
		}
		if (long && cand < c.Stop) || (!long && cand > c.Stop) {
			c.Stop, c.Bound = cand, name
		}
	}
	pick(c.AnchorStop, "anchor")
	pick(c.ATRFloorStop, "atr_floor")
	return c
}`
