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

// PHLPLHGated is the call site the evaluator uses for every PHL/PLH: the
// §2.2 rules in PHLPLH, then the two direction gates on top —
//
//		§5.4 (fold item 3): entries only WITH the 4h trigger direction; the 1h
//		agreeing or silent. A 1h opposite the 4h refuses the setup until it
//		flips [D4.4 p1 @ 16:00].
//
//	  §7 (fold item 4): a SPENT day with a 4h/1h conflict shuts the day off
//	  (latched at the 08:30 read — a later 1h flip does not reopen it); a
//	  spent day that agrees caps the target at TargetCapPts
//	  ("15 điểm bán, 10 điểm bán") [D5.1 p1 @ 15:57]; an unmeasured run
//	  fails closed ("any trade you are vague about — don't" [§12]). On a
//	  NORMAL day a conflict is §5.4 case 3 — the HTF gate above sits out
//	  per tick until the 1h flips, not a day off.
//
// A touch-and-reject at a level IS a PHL/PLH and carries these rules
// unchanged (fold item 2) — the location itself is gated upstream by the
// evaluator's key-level gate (DS-103's item 1).
func PHLPLHGated(t Touch, oldExtreme Level, extremeIdx, barIdx int, cfg Config, htf HTF, day DayVerdict, dg DayGate) (Intent, bool, string) {
	in, ok, reason := PHLPLH(t, oldExtreme, extremeIdx, barIdx, cfg)
	if !ok {
		return in, false, reason
	}
	if htfOK, side, htfReason := HTFVerdict(htf); !htfOK {
		return in, false, "HTF direction gate: no trade — " + htfReason
	} else if side != in.Side {
		return in, false, "HTF direction gate: entry side " + string(in.Side) + " against the " + string(side) + " trigger — entries only with the 4h direction [D4.4 p1 @ 16:00]"
	}
	if day == DayOff {
		return in, false, "day gate: spent + 4h/1h conflict at the pre-open read — 'TẮT MÁY NGHỈ LUÔN CHO EM', no trades today [D5.1 p1 @ 19:22]"
	}
	if day == DayNotMeasured {
		return in, false, "day gate: day run not measured — no mentor entries ('any trade you are vague about — don't' [§12])"
	}
	return CapTargetForDay(in, day, dg), true, ""
}
