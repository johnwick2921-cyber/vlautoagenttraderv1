package mentor

// PHL / PLH [§2.2] — the return trade at a level whose old extreme lies at
// least PHLMinCandlesFromExtreme candles away [D4.1 p1 @ 09:40 written:
// "Phải đợi đi xa đỉnh/đáy cũ trước đó (ít nhất là 3 cây nến backtest)"].
// Target: "gần đỉnh cũ" — NEAR the old extreme, not exactly at it
// [D4.1 p1 @ 07:27], PHLTargetShyPts short (Q2 knob; worked example 5–15
// [D2.2 p1 @ 06:50]).
//
// A PHL is the higher low: reject-touch at a support level → buy stop beyond
// the reference candle, target near the old HIGH. A PLH is the lower high:
// reject-touch at a resistance level → sell stop, target near the old LOW.

// PHLPLH builds the setup intent from a reject touch (§3) at a level, or
// reports why it is not a trade. barIdx and extremeIdx are indexes into the
// 1m series the evaluator ticks on.
func PHLPLH(t Touch, oldExtreme Level, extremeIdx, barIdx int, cfg Config) (Intent, bool, string) {
	if !cfg.Enabled {
		return Intent{}, false, "mentor mode off"
	}
	if t.Outcome != TouchReject {
		return Intent{}, false, "not a reject touch — no PHL/PLH [D5.2 p1 @ 19:51]"
	}
	side, price, ok := RejectEntry(t, cfg)
	if !ok {
		return Intent{}, false, "no reject entry"
	}
	// The old extreme must be on the TARGET side: long → old high above;
	// short → old low below. The dispatch's "higher low / lower high at the
	// level" names exactly this pairing.
	onTargetSide := side == SideLong && oldExtreme.Price > price ||
		side == SideShort && oldExtreme.Price < price
	if !onTargetSide {
		return Intent{}, false, "old extreme not on the target side"
	}
	if barIdx-extremeIdx < cfg.PHLMinCandlesFromExtreme {
		return Intent{}, false, "too close to the old extreme — need at least " + itoa(cfg.PHLMinCandlesFromExtreme) + " candles [D4.1 p1 @ 09:40 written]"
	}
	stop := t.RefBar.Low - cfg.ISBBufferPts
	target := oldExtreme.Price - cfg.PHLTargetShyPts
	if side == SideShort {
		stop = t.RefBar.High + cfg.ISBBufferPts
		target = oldExtreme.Price + cfg.PHLTargetShyPts
	}
	risk := price - stop
	reward := target - price
	if side == SideShort {
		risk = stop - price
		reward = price - target
	}
	if risk <= 0 || reward <= 0 {
		return Intent{}, false, "degenerate stop/target geometry"
	}
	if risk > cfg.StopCeilingPts {
		return Intent{}, false, "stop over the 25-pt ceiling — not worth trading [D3.3 p1 @ 02:04]"
	}
	if reward < cfg.RoomMultiple*risk {
		return Intent{}, false, "room rule: reward < " + fnum(cfg.RoomMultiple) + "x risk — not enough room [D5.3 p1 @ 09:16]"
	}
	return Intent{
		Action: PlaceStopEntry,
		Side:   side,
		Price:  price,
		Stop:   stop,
		Target: target,
		Reason: "PHL/PLH: stop order beyond the rejecting candle, target near the old extreme [D4.1 p1 @ 07:27, 09:40 written]",
	}, true, ""
}
