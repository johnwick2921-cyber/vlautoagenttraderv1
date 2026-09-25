# Dispatch 105 — formation evidence and bounded seam feasibility

Read-only source audit at pinned revision `770e2297`. Evidence **[A]** means
the exact pinned source was read; **[B]** means a consequence inferred from that
source. No detector, fixture, database, or production code was changed and no
build was run for this audit. This artifact accompanies the correction report;
it is not authorization to implement a different detector or formation meaning.

## Corrections required before implementation

### 1. The existing field records the completing candle's OPEN

**[A]** `kernel/levels.go:86–90` explicitly defines:

```go
// FormedAtMs is the formation birth instant (bar open time of the candle
// that completed the pattern), in unix ms. 0 = unknown/older detections.
// The W6 wake loop (2026-08-25) diffs this against the plan row's birth
// time to find events the plan never saw.
FormedAtMs int64 `json:"formed_at_ms,omitempty"`
```

**[A]** `kernel/levels_zones.go:147,152,260,265,313,322` writes `OpenTime`
for S/D, FVG/iFVG, and OB. Representative actual lines:

```go
zl.FormedAtMs = d.OpenTime // W6: birth = departure bar
lvl.FormedAtMs = c.OpenTime
zl.FormedAtMs = cb[i].OpenTime // W6: birth = displacement bar
```

These are not candle-close timestamps or first-availability timestamps.
The inverted FVG output retains the original gap's completing candle timestamp,
not the later inversion candle's timestamp.

**[A]** `trader/auto_trader_wake_levels.go:102,112,127,143,159` compares this
field directly with plan birth, and includes it in event keys. For example:

```go
if z.ZonePattern != "reversal" || z.FormedAtMs <= birthMs {
	continue
}
```

**[B]** Changing those existing timestamps from open to close can change which
events qualify for a wake and change their deduplication keys. A31 forbids that
cadence change. Preserve the existing field's semantics or obtain a ruling for a
separate recording-only close/availability field before implementing D1's
different definition. Simply calling the existing value a close would misstate
the evidence.

**[A]** The episode recorder also consumes this value. At
`trader/detector_record.go:67–69`:

```go
if lv.FormedAtMs > 0 {
	bars = barsSince(bars, lv.FormedAtMs)
	preFormation += len(scope.Bars) - len(bars)
}
```

### 2. PDH/PDL/PDC are prior CALENDAR-day levels

**[A]** `kernel/levels_multiday.go:38–41` says:

```go
// ExtractMultiDayLevels returns the structural map levels relative to `now`:
// PDH/PDL/PDC (prior calendar day), RTH-H/L (prior day's NY session), AS/LDN/ON
// (the current overnight of now's futures session-day), and PW/PM (prior week /
// month). Closed bars only.
```

At lines 68–69 the grouping is explicitly CT calendar date:

```go
bt := time.UnixMilli(b.OpenTime).In(loc)
calKey := bt.Format("2006-01-02")
```

At lines 83–85 it retains the latest bar's open timestamp alongside the close
price:

```go
if b.OpenTime >= a.closeOpenTime {
	a.closePx = b.Close
	a.closeOpenTime = b.OpenTime
}
```

**[B]** D1's “prior session close” cannot safely mean the CME 16:00 close for
these levels. After midnight, the prior calendar bucket can include the preceding
evening, which belongs to the current CME session. An output seam can retain the
actual bucket-closing bar timestamp without changing the price-producing bars,
but the terminology and intended timestamp need correction. Relabelling the
existing bucket as a CME session would be false; changing the bucket would alter
the detector.

### 3. W-TF's dedupe does NOT include formation

**[A]** The exact condition at `kernel/levels_assemble.go:272` is:

```go
if o.Kind == l.Kind && o.TF == l.TF && math.Abs(o.Price-l.Price) <= dedupeTick {
```

`dedupeTick` at line 258 is `0.25`. Its source comment names the key as
`(kind, tf, price±tick)`. Thus distinct formations of the same kind/TF at one
price can still be deduped. The dispatch's `(kind, tf, price, formed_at)`
description overstates W-TF's dedupe implementation.

**[B]** A canonical ID hash can distinguish raw references without changing this
selection rule. The report must not claim that the selected map consequently
retains both references, and this wave must not change the dedupe or seat rule to
make that claim true.

## Formation-capture matrix

“Derivable” below identifies source evidence available to an output seam; it does
not certify complete live bar coverage or authorize stamping a scheduled time
without its required evidence. All citations refer to `770e2297`.

| Family | Current output | Actual source / bounded recording seam |
|---|---|---|
| PDH/PDL/PDC | No formation, TF, or lookback | Calendar aggregate has `closeOpenTime`; retain the associated actual `CloseTime` and bucket count. Do not replace calendar grouping with CME-session grouping. `kernel/levels_multiday.go:22–28,68–85,156–160`. |
| RTH-H/L | No formation, TF, or lookback | Existing `reg.ActiveSession(bt)` NY membership determines bars. Track the actual final bar in that bucket. Default NY end is **14:45 CT**, not 15:00. `kernel/levels_multiday.go:88–103,161–165`; `kernel/session_registry.go:108–117`. |
| AS-H/L, LDN-H/L, ONH/L | No formation, TF, or lookback | Track final actual bar in the existing registry-defined windows. Default AS closes 02:00 and London/overnight closes 08:30. They already emit while developing; a completed-window timestamp must remain NULL before completion or absent terminal evidence. `kernel/levels_multiday.go:105–128,169–190`; `kernel/session_registry.go:89–105`. |
| OR-H/L | No formation, TF, or lookback | `orEnd = rthOpen + 5m`, normally 08:35. Retain actual completing-bar evidence. Existing output emits as soon as any closed OR bar exists. `kernel/levels_intraday.go:132–170`. |
| IB-H/L and extensions | No formation, TF, or lookback | `ibEnd = rthOpen + 60m`, normally 09:30. Same developing-window caveat; no detector delay may be introduced. `kernel/levels_intraday.go:143–180`. |
| Session VWAP, ±1σ, ±2σ | No formation, TF, or lookback | Existing anchor is `CMESessionDayStart(now)` at 17:00; output exists after at least two closed session bars. Dispatch explicitly chooses this anchor, which differs from the bar at which the value is first computable. Preserve that distinction. `kernel/levels_volume.go:40–60`; `kernel/cme_calendar.go:116–130`. |
| eVWAP | No formation, TF, or lookback | Existing 15:00 cash-close anchor, not 17:00. `kernel/levels_volume.go:103–118`. |
| POC/VAH/VAL, pdVWAP | No formation, TF, or lookback | Existing prior CME-session bar slice is available; profile cache currently stores output levels. Retain actual final-bar evidence alongside cached rows. `kernel/levels_volume.go:132–157,323–336`. |
| nPOC | No formation in output | **Already computes** `birth: dayBars[len(dayBars)-1].CloseTime`, then drops it at `lineLevel`. Narrow output seam can copy it. `kernel/levels_volume.go:285–308`. |
| SETT | No formation | Existing scan picks the final prior-session close price; retain that same bar's timestamp. `kernel/levels_volume.go:343–360`. |
| MID-O | No formation | Develops overnight and normally uses 08:30 cutoff. Current predicate includes a bar whose **open equals 08:30**, so blindly stamping 08:30 can predate contributing data. Preserve the predicate and report this caveat. `kernel/levels_volume.go:371–389`. |
| Gap | No formation | Gap's completing bar `c` is available in output construction; its close can be captured without changing gap detection. `kernel/levels_intraday.go:58–120`. |
| EQH/EQL, all admitted TFs | No formation | Pivot timestamps are discarded into `[]float64` before clustering; output `OriginDate` comes from the newest read bar. Formation cannot be truthfully reconstructed from the final level alone. Carry pivot/confirmation metadata through unchanged clustering or leave NULL. `kernel/levels_zones.go:34–53,75–97`. |
| S/D, FVG/iFVG, OB | Formation **open** captured | Existing `OpenTime` semantics above. Inverted FVG keeps original gap formation, not inversion instant. Close capture needs separate semantics to preserve wakes. `kernel/levels_zones.go:147,152,260,265,313,322`. |
| SWG-H/L 5m/15m | TF captured; formation/lookback dropped | Internal `pt.timeMs` is pivot bar close (`closed[i].OpenTime + iv`), but output omits it. Detection also needs right-side confirmation bars, so pivot close differs from first availability. `kernel/levels_swing.go:107,129–164`. |
| Round numbers | No formation | NULL by nature; no timestamp or hash should be invented. `kernel/levels_intraday.go:16–50`. |

## Supporting exact output and window quotes

**[A]** `lineLevel` has no bars or clock and does not stamp formation. At
`kernel/levels.go:100–103`:

```go
// lineLevel builds a single-price DetectedLevel (Lo==Hi==price).
func lineLevel(kind LevelKind, price float64, label, origin string, htf bool) DetectedLevel {
	return DetectedLevel{Kind: kind, Price: price, Lo: price, Hi: price, Label: label, OriginDate: origin, HTF: htf}
}
```

**[A]** OR and IB windows derive from the configured NY start, not independent
hardcoded formation constants. At `kernel/levels_intraday.go:132–144`:

```go
// RTH open = 08:30 CT of now's calendar day (from the NY registry row).
openMin := 8*60 + 30
if ny, ok := reg.SessionByName(SessionNY); ok {
	if m, ok := parseHHMM(ny.WindowStartCT); ok {
		openMin = m
	}
}
rthOpen := time.Date(nowCT.Year(), nowCT.Month(), nowCT.Day(), openMin/60, openMin%60, 0, 0, loc)
if nowCT.Before(rthOpen) {
	return nil // pre-open: no OR/IB yet
}
orEnd := rthOpen.Add(5 * time.Minute)
ibEnd := rthOpen.Add(60 * time.Minute)
```

At lines 154–160, any qualifying closed bar sets `hasOR`/`hasIB`; output later
depends on those booleans, not whether the whole window completed:

```go
if !bt.Before(rthOpen) && bt.Before(orEnd) {
	hasOR = true
	orH, orL = math.Max(orH, b.High), math.Min(orL, b.Low)
}
if !bt.Before(rthOpen) && bt.Before(ibEnd) {
	hasIB = true
	ibH, ibL = math.Max(ibH, b.High), math.Min(ibL, b.Low)
}
```

**[A]** MID-O's predicate at `kernel/levels_volume.go:379–383`:

```go
for _, b := range cb {
	if b.OpenTime >= sessStart.UnixMilli() && b.OpenTime <= cutover.UnixMilli() {
		hi = math.Max(hi, b.High)
		lo = math.Min(lo, b.Low)
	}
}
```

**[A]** EQH/EQL discards timestamps at `kernel/levels_zones.go:39–49`:

```go
const k = 2
var hi, lo []float64
for i := k; i < len(cb)-k; i++ {
	if isStrictPivotHigh(cb, i, k) {
		hi = append(hi, cb[i].High)
	}
	if isStrictPivotLow(cb, i, k) {
		lo = append(lo, cb[i].Low)
	}
}
origin := time.UnixMilli(cb[len(cb)-1].OpenTime).In(chicago()).Format("2006-01-02")
```

**[A]** Swing pivot close is computed at `kernel/levels_swing.go:107`:

```go
t := closed[i].OpenTime + iv // bar CLOSE instant (repo convention)
```

The output at lines 156–164 retains only:

```go
out = append(out, DetectedLevel{
	Kind:       kind,
	Price:      s.price,
	Lo:         s.price,
	Hi:         s.price,
	Label:      label + tfName(tfMin),
	OriginDate: day,
	TF:         tfName(tfMin),
})
```

## What W-TF actually captures

**[A]** W-TF did not fix formation capture. Its `tagHTFLevel` output decorator
sets only `HTF`, `TF`, `LookbackBars`, and label. The per-TF loop passes the
actual searched closed-bar count at `kernel/levels_assemble.go:391–395`:

```go
lookback := len(cb)
before := len(out)
for _, d := range htfDetectors {
	for _, l := range d.Run(cb, atr, tol, now) {
		out = append(out, tagHTFLevel(l, tf, lookback))
	}
}
```

Formation remains populated only by the existing zone-family assignments.
No separate stored age field was added to `DetectedLevel`; age is a read-time
interpretation of formation and observation. The W-TF report at this pinned
revision is `docs/superpowers/reports/2026-09-10-every-detector-every-timeframe.md`;
its later live record counts belong in the parent report's independent census,
not in this source-only artifact.

**[A]** Existing `DetectedLevel.TF` is a grading input as well as a dedupe input.
At `kernel/levels.go:79–81`:

```go
// TF is the DETECTION timeframe ("1m"…"4h"; "" = the 1m slice). Drives the
// v3 zone evidence tiers (owner-approved 2026-08-24).
TF string `json:"tf,omitempty"`
```

**[B]** Filling it for formerly blank line levels is not automatically
recording-only. A new record-side metadata field or production parity proof is
necessary to ensure scorer, dedupe, seats, and existing prompt columns remain
unchanged. An output-only source edit can still alter trading behavior when an
existing consumer reads the field.

## Backfill feasibility and bounded footprint

**[A]** `store/touch_outcomes.go` retains `LevelPrice`, `LevelKind`, and
`FormedAtMs`, but does not retain all seven canonical hash inputs: TF, bounds,
and origin date are missing. The complete struct was read at the pinned
revision. The hash at `trader/research_snapshot.go:228–232` is:

```go
func researchCandidateID(symbol string, l kernel.DetectedLevel) string {
	identity := fmt.Sprintf("%s|%s|%g|%g|%s|%s|%d", symbol, l.Kind, l.Lo, l.Hi, l.OriginDate, l.TF, l.FormedAtMs)
	h := sha256.Sum256([]byte(identity))
	return hex.EncodeToString(h[:])
}
```

At `trader/research_snapshot.go:31–39`, the candidate record stores the hash,
raw origin, and explicit caveat:

```go
f.Set("stable_id", researchCandidateID(symbol, l))
f.Set("identity_basis", "symbol/kind/bounds/origin date/timeframe/formation; unknown formation may alias episodes")
f.Set("root_symbol", symbol)
f.Set("raw_origin", l)
f.Set("price", l.Price)
f.Set("lo", l.Lo)
f.Set("hi", l.Hi)
if l.FormedAtMs > 0 {
	f.Set("formation_ms", l.FormedAtMs)
}
```

**[B]** A nonzero formation alone therefore does not establish recomputability
from `touch_outcomes`. An exact join to a recorded research `raw_origin` may
supply missing inputs; if no such exact record exists, a further honest
unrecomputable reason is needed. Guessing zone bounds from midpoint or timeframe
from a name would violate A24.

The bounded footprint, subject to the corrections above, is additive metadata at
each emitter's output and its record-side consumers. `lineLevel` itself has
neither bars nor a clock and cannot infer formation truthfully. EQH/EQL and swing
confirmation need metadata carried from existing detection indices; duplicating
their algorithms in a second formation detector would create a second source of
truth. No modification to scoring, dedupe, seats, detector definitions, or wake
cadence is justified by this evidence.

## Source freshness ledger

Each line below is the actual result of
`git log 770e2297 -1 --format='%H %cI %s' -- <file>`.
The tracked `docs/superpowers/CLAUDE-canon.md` was read; repo-root `CLAUDE.md`
does not exist in the pinned Git tree.

```text
docs/superpowers/CLAUDE-canon.md
290044296c482afdee04acd740d189b89bfd040d 2026-09-10T11:55:22-05:00 docs(canon): two different rc 3s sat on adjacent lines

docs/superpowers/reports/2026-09-10-every-detector-every-timeframe.md
050cd5c1549b656ed06f6eaae8954b55a723ede4 2026-09-10T13:18:24-05:00 docs(report): A15 — nofx/trader is RED on dev, and it is not the lunch band

kernel/levels.go
57d0ff5802c3c633b570a125016ac4f0e5a054ef 2026-09-10T12:51:56-05:00 W-TF D1/D2/C6: the daily family reaches the map, and timeframe becomes identity

kernel/levels_assemble.go
32c1cfb5c198cbd2fed90d9de155e1b71a4cfcd1 2026-09-10T13:03:38-05:00 W-TF D6: the boot line, the detector table, and the docs in the same commit

kernel/levels_multiday.go
7b19e7533975225964613ba4bc976f7f7f969e56 2026-09-03T23:00:08-05:00 style(kernel): gofmt the 10 files that were unformatted on dev — formatting ONLY

kernel/levels_intraday.go
b01d164d1506c406f85d97c6281df9642cd82ec1 2026-08-17T02:34:09-05:00 fix(h1+h2): proximity_filter_atr governs level generation AND seating, not just activation

kernel/levels_volume.go
7b19e7533975225964613ba4bc976f7f7f969e56 2026-09-03T23:00:08-05:00 style(kernel): gofmt the 10 files that were unformatted on dev — formatting ONLY

kernel/levels_zones.go
41b1428848a62c1fa00dfcb7b036902faaecddcd 2026-08-26T16:37:07-05:00 S-FIX WAVE: mega-research triage (4×S + 4 cheap A) (#80)

kernel/levels_swing.go
422680c6a4a329a4c5998176b091f07cc420034b 2026-08-27T14:11:57-05:00 level-truth T3: SWG-H/SWG-L swing-point detector (5m+15m structure swings, react_zone, anchor-class evidence 0.85) + role/label/family registration

kernel/session_registry.go
3f23d9bb5404696c7eeabc43bb133c118d60a6f7 2026-09-07T19:30:13-05:00 feat(session-calendar): the fold — one calendar, one owner, and the sourced close times win

kernel/cme_calendar.go
5457ac5accd97c3519bf6d16ead147a0db2ab0d0 2026-09-07T19:39:02-05:00 fix(session-calendar): five bare "15:04" layouts the tz guard caught — and the guard is worktree-blind

trader/auto_trader_wake_levels.go
fa86029e95fdd4ff82e093a7b721e1fb41c12372 2026-09-03T14:47:08-05:00 fix(wake): a clock seam — the enforcing cutoff made a fixed-fixture test time-of-day dependent

trader/detector_record.go
c6f75756f3e54a56646ca3cfa9c86f116b5d4541 2026-09-10T11:31:42-05:00 feat(W1 4/n): wire the link — the recorder stamps it, the gate keeps it wired

trader/research_snapshot.go
0babd0902d1bb74e5f44fce1fc4350e42a8cbdeb 2026-09-08T16:22:32-05:00 feat(research): link attempts, permissions, broker receipts and corrected outcomes

store/touch_outcomes.go
c35dfecb2e11cbc763174516983919885cf0b4cc 2026-09-10T11:31:42-05:00 feat(W1 2/n): the touch → scenario link as a HEURISTIC, and the boot line that names its resolver
```
