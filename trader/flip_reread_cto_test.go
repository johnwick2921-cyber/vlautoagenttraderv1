package trader

import (
	"encoding/json"
	"strings"
	"sync"
	"testing"
	"time"

	"nofx/kernel"
	"nofx/market"
	"nofx/mcp"
	"nofx/store"
)

// W-FLIP-REREAD — CTO takeover (2026-09-17). The first draft's suite
// substituted flipRereadRun with a recorder, so the real read path (claimed
// read → planner core → write site → store) was never exercised and three
// BLOCKERs hid behind green tests. These tests keep flipRereadRun REAL and
// script only the AI client (the same seam TestRunPlannerReadCore* uses).

// scriptedPlannerClient is an mcp.AIClient whose CallWithMessages answers from
// a script; it counts calls so a test can prove "no read happened".
type scriptedPlannerClient struct {
	mu      sync.Mutex
	n       int
	respond func(n int, userPrompt string) (string, error)
}

func (c *scriptedPlannerClient) SetAPIKey(string, string, string) {}
func (c *scriptedPlannerClient) SetTimeout(time.Duration)         {}
func (c *scriptedPlannerClient) ResolvedModel() string            { return "test-planner-model" }
func (c *scriptedPlannerClient) CallWithMessages(_, user string) (string, error) {
	c.mu.Lock()
	c.n++
	n := c.n
	c.mu.Unlock()
	return c.respond(n, user)
}
func (c *scriptedPlannerClient) CallWithRequest(*mcp.Request) (string, error) { return "", nil }
func (c *scriptedPlannerClient) CallWithRequestStream(*mcp.Request, func(string)) (string, error) {
	return "", nil
}
func (c *scriptedPlannerClient) CallWithRequestFull(*mcp.Request) (*mcp.LLMResponse, error) {
	return nil, nil
}
func (c *scriptedPlannerClient) calls() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.n
}

// validShortPlanJSON is validTraderPlanJSON's mirror: bias SHORT, flip ABOVE
// 15620 → long (CLASS 140 relation), death BELOW 15430, levels on both sides
// of a ~15470 tape (P0.1 both-side rule) and targets inside the 45-pt
// proximity band, so the REAL write site accepts it.
const validShortPlanJSON = `{
  "reasoning": "Flip fired below PWL; fade rallies into the broken level, short the reject.",
  "bias": {"direction": "short", "conviction": "medium", "flip_condition": "2x5m > 15620"},
  "levels": [
    {"price": 15430, "label": "RN 15430", "grade": "B", "instruction": "fade"},
    {"price": 15450, "label": "RN 15450", "grade": "B", "instruction": "fade"},
    {"price": 15480, "label": "PWL", "grade": "A", "instruction": "fade"},
    {"price": 15520, "label": "RN 15525", "grade": "B", "instruction": "fade"},
    {"price": 15575, "label": "RN 15575", "grade": "B", "instruction": "fade"},
    {"price": 15620, "label": "PDH", "grade": "A", "instruction": "fade"}
  ],
  "scenarios": [{"id": "S1", "trigger": "reject 15480 from below", "condition": "reject", "direction": "short", "target_chain": [15450, 15430], "invalid": "2x5m>15490", "quality": "A", "confirm":{"rule":"touch","ref_price":15480,"side":"above"},"economics":{"entry_zone":[15480,15480],"geometry":{"entry":15480,"stop":15490,"target":15430},"first_obstacle":{"price":15450,"level":"fixture reference","family":"reference","response":"pass_through"},"r_to_obstacle":3.0,"r_to_arm_target":5.0}}],
  "no_trade": ["first 5m"],
  "death_condition": "acceptance below 15430",
  "death": {"price": 15430, "side": "below", "rule": "2x5m"},
  "flip": {"price": 15620, "side": "above", "rule": "2x5m", "flip_to": "long"},
  "day_type": "balance"
}`

// flipFixtureDoc is the seeded LONG plan whose flip (2x5m below 15480) fires
// → short on the seedFlipBars(15500, 15470, …) tape.
func flipFixtureDoc() kernel.PlanDoc {
	return kernel.PlanDoc{
		Bias:           kernel.PlanBias{Direction: "long", FlipCondition: "flips short on 2x5m below 15480"},
		FlipStructured: &kernel.PlanCondition{Price: 15480, Side: "below", Rule: "2x5m", FlipTo: "short"},
	}
}

// realPathTrader is resetTrader + the scripted client + the clock-hold seam
// neutralised (the fixture tape is dated, and F6 would otherwise defer the
// read on "drift"). flipRereadRun is NOT substituted.
func realPathTrader(t *testing.T, flipReread bool, respond func(n int, user string) (string, error)) (*AutoTrader, *store.Store, *scriptedPlannerClient) {
	t.Helper()
	t.Setenv("FLIP_ATR_BUFFER", "0")
	t.Setenv("DORMANT_MIN_HOLD_MIN", "0")
	// Level-event wakes OFF: maybeWakePlannerOnLevelEventsAt throttles on the
	// WALL clock, so on this dated fixture the dormant row's "keeps eyes" wake
	// would fire every cycle and its calls would be indistinguishable from
	// the flip read's. This suite isolates the structure_flip path.
	off := false
	cfg := store.StrategyConfig{DayPlan: &store.DayPlanConfig{
		PlanEnabled: true, ReplanCap: 4, SessionsEnabled: []string{"NY"}, FlipReread: flipReread,
		WakeOn15mZone: &off, WakeOnHTFZone: &off, WakeOnHTFOB: false, WakeOnSeatedInvalidation: &off, WakeOnIFVG: &off,
	}}
	at, st := resetTrader(t, cfg)
	client := &scriptedPlannerClient{respond: respond}
	at.mcpClient = client
	origDrift := clockHoldDriftFn
	clockHoldDriftFn = func(string) (int64, bool) { return 0, false }
	t.Cleanup(func() { clockHoldDriftFn = origDrift })
	t.Cleanup(func() { market.FuturesBarsProvider = nil; traderTestBarsInstalled = false })
	return at, st, client
}

func flipRereadTestNow(t *testing.T, now time.Time) {
	t.Helper()
	testNow = func() time.Time { return now }
	t.Cleanup(func() { testNow = nil })
}

// (i) BLOCKER 1 at the PRODUCTION WRITE SITE: the structure_flip prior line
// for long→short must make the write site demand SHORT. Before the fix
// kernel.FlipToDirection read the old-bias echo ("PRIOR PLAN v3 bias long")
// and demanded LONG — a correct short plan was rejected three times and a
// wrong long plan accepted. The read core is the real one; only the AI call
// is scripted (the TestRunPlannerReadCoreRetryThenSuccess seam).
func TestFlipRereadWriteSiteRequiresFlippedBias(t *testing.T) {
	at := plannerTestTrader(t)
	killer := "flip-condition: 2x5m close below 15480.00 (2× 5m closes) → bias short"
	prior := flipRereadPriorLine(3, "long", "short", killer)
	requiredBias := kernel.FlipToDirection(prior) // the exact expression at the write site (requiredBias := kernel.FlipToDirection(priorKiller))
	if requiredBias != "short" {
		t.Fatalf("FlipToDirection(prior) = %q, want short; prior:\n%s", requiredBias, prior)
	}
	logBuf := captureTraderLog(t)

	// Attempt 1: the model keeps the OLD bias (long) → MUST be rejected.
	// Attempt 2: the flipped bias (short) → written active.
	n := 0
	ver, lc, err := at.runPlannerReadCoreWithFactsGradesClock(time.Now, "NY", "2026-08-14", "structure_flip", "m", "hashF", "", "", requiredBias, "prompt", kernel.PlanFacts{}, nil, nil, nil, false, func(string) (string, error) {
		n++
		if n == 1 {
			return validTraderPlanJSON, nil // long — the stale bias
		}
		return validShortPlanJSON, nil
	})
	if err != nil || lc != "active" || ver != 1 {
		t.Fatalf("flipped-bias plan must be written active: ver=%d lc=%q err=%v (calls=%d)", ver, lc, err, n)
	}
	if n != 2 {
		t.Fatalf("expected the long plan rejected once then the short accepted (2 calls), got %d", n)
	}
	if !strings.Contains(logBuf.String(), `bias short is MANDATORY, got "long"`) {
		t.Fatalf("the write site must reject the stale bias by name; log:\n%s", logBuf.String())
	}
	row, _ := at.store.Plan().GetLatestPlanForTraderSession("2026-08-14", "NY", "t1")
	if row == nil || row.TriggerReason != "structure_flip" || row.Lifecycle != "active" {
		t.Fatalf("stored row: %+v", row)
	}
	var doc kernel.PlanDoc
	if json.Unmarshal([]byte(row.Doc), &doc) != nil || doc.Bias.Direction != "short" {
		t.Fatalf("stored bias must be the flipped one: %+v", doc.Bias)
	}

	// A model that never flips exhausts the 3 attempts on this wake-class
	// (failClosed=false) read: NO row, "kept_active" — the dormant plan stands.
	at2 := plannerTestTrader(t)
	m := 0
	ver2, lc2, err2 := at2.runPlannerReadCoreWithFactsGradesClock(time.Now, "NY", "2026-08-14", "structure_flip", "m", "hashG", "", "", requiredBias, "prompt", kernel.PlanFacts{}, nil, nil, nil, false, func(string) (string, error) {
		m++
		return validTraderPlanJSON, nil
	})
	if err2 != nil || ver2 != 0 || lc2 != "kept_active" || m != 3 {
		t.Fatalf("stale-bias-only model: ver=%d lc=%q err=%v calls=%d — want (0, kept_active, nil, 3)", ver2, lc2, err2, m)
	}
	if row2, _ := at2.store.Plan().GetLatestPlanForTraderSession("2026-08-14", "NY", "t1"); row2 != nil {
		t.Fatalf("no row may be written when the flipped bias is never authored: %+v", row2)
	}
}

// (ii) BLOCKER 2 on the REAL path: flipRereadRun returns true after the
// planner core exhausted its 3 attempts with NO row (kept_active). Success
// must be decided by the store: the once-key is cleared, the dormant plan
// stands, and the NEXT cycle retries through the dormant branch.
func TestFlipRereadRealPathExhaustedRetriesClearsOnceKeyAndRetries(t *testing.T) {
	at, st, client := realPathTrader(t, true, func(int, string) (string, error) { return "not json", nil })
	now := time.Date(2026, 8, 18, 14, 0, 0, 0, time.UTC)
	flipRereadTestNow(t, now)
	td := "2026-08-18"
	row := seedActivePlan(t, at, td, "NY", now.Add(-40*time.Minute), flipFixtureDoc())
	seedFlipBars(15500, 15470, 6*time.Minute, now)
	logBuf := captureTraderLog(t)

	at.maybeRunSessionReadsAt(now)

	if got := versionLifecycle(t, st, td, "NY", at.id, 1); got != "dormant" {
		t.Fatalf("flip must park dormant first, got %q", got)
	}
	if !waitFor(t, 10*time.Second, func() bool {
		return strings.Contains(logBuf.String(), "wrote NO new version — the dormant plan stands; the once-key is cleared for a retry next cycle")
	}) {
		t.Fatalf("no 'wrote NO new version' line; log:\n%s", logBuf.String())
	}
	if client.calls() != 3 {
		t.Fatalf("the real planner core makes 3 attempts, got %d", client.calls())
	}
	if v := sysCfgVal(t, st, flipRereadDoneKey(row)); v != "" && v != "0" {
		t.Fatalf("once-key must be clear after a read that wrote nothing, got %q", v)
	}
	latest, _ := st.Plan().GetLatestPlanForTraderSession(td, "NY", at.id)
	if latest == nil || latest.Version != 1 || latest.Lifecycle != "dormant" {
		t.Fatalf("dormant v1 must stand alone: %+v", latest)
	}
	// Wait for the in-flight guard to clear (goroutine exit), then the NEXT
	// cycle: the row is still dormant, the key is clear → the dormant branch
	// retries the read (wake_min_interval elapsed, fresh bars).
	if !waitFor(t, 5*time.Second, func() bool {
		_, running := flipRereadInFlight.Load(flipRereadInFlightKey(at, row))
		return !running
	}) {
		t.Fatal("in-flight guard never cleared")
	}
	next := now.Add(time.Duration(store.DefaultWakeMinIntervalMin+1) * time.Minute)
	flipRereadTestNow(t, next)
	seedFlipBars(15500, 15470, 6*time.Minute, next)
	before := client.calls()
	at.maybeRunSessionReadsAt(next)
	if !waitFor(t, 10*time.Second, func() bool {
		return strings.Count(logBuf.String(), "wrote NO new version") >= 2
	}) {
		t.Fatalf("the dormant branch must RETRY the read next cycle and report it: calls before=%d after=%d; log:\n%s", before, client.calls(), logBuf.String())
	}
	if strings.Count(logBuf.String(), "waking the planner (W-FLIP-REREAD)") != 2 {
		t.Fatalf("exactly two structure_flip launches (fire + retry); log:\n%s", logBuf.String())
	}
	if client.calls() != 6 {
		t.Fatalf("two real reads × 3 attempts = 6 calls, got %d", client.calls())
	}
	if v := sysCfgVal(t, st, flipRereadDoneKey(row)); v != "" && v != "0" {
		t.Fatalf("once-key must still be clear after the failed retry, got %q", v)
	}
}

// (iii) BLOCKER 3 on the REAL path: the re-arm path moves v1 dormant → active
// while the planner call is open; the read then lands v2 (flipped bias). The
// supersede is a compare-and-set FROM dormant → refused; both rows stand; the
// newest version governs at read time.
func TestFlipRereadRealPathRearmedMeanwhileSupersedeRefused(t *testing.T) {
	var at *AutoTrader
	var st *store.Store
	at, st, client := realPathTrader(t, true, func(n int, _ string) (string, error) {
		// "Meanwhile": the dormant re-arm path (close-back predicate) restores v1.
		pid := store.MakePlanIDForTrader(at.id, "2026-08-18", "NY")
		if err := st.Plan().UpdatePlanLifecycle(pid, 1, "active", "rearmed:2x5m close back above 15480"); err != nil {
			return "", err
		}
		return validShortPlanJSON, nil
	})
	now := time.Date(2026, 8, 18, 14, 0, 0, 0, time.UTC)
	flipRereadTestNow(t, now)
	td := "2026-08-18"
	row := seedActivePlan(t, at, td, "NY", now.Add(-40*time.Minute), flipFixtureDoc())
	seedFlipBars(15500, 15470, 6*time.Minute, now)
	logBuf := captureTraderLog(t)

	at.maybeRunSessionReadsAt(now)

	if !waitFor(t, 10*time.Second, func() bool {
		return strings.Contains(logBuf.String(), "supersede REFUSED")
	}) {
		t.Fatalf("no 'supersede REFUSED' line; calls=%d log:\n%s", client.calls(), logBuf.String())
	}
	if client.calls() != 1 {
		t.Fatalf("one AI call, got %d", client.calls())
	}
	v1, _ := st.Plan().GetPlan(row.PlanID, 1)
	v2, _ := st.Plan().GetPlan(row.PlanID, 2)
	if v1 == nil || v1.Lifecycle != "active" {
		t.Fatalf("re-armed v1 must keep its active lifecycle, got %+v", v1)
	}
	if v2 == nil || v2.Lifecycle != "active" || v2.TriggerReason != "structure_flip" {
		t.Fatalf("the flip read's v2 must stand as written, got %+v", v2)
	}
	var doc kernel.PlanDoc
	if json.Unmarshal([]byte(v2.Doc), &doc) != nil || doc.Bias.Direction != "short" {
		t.Fatalf("v2 must carry the flipped bias: %+v", doc.Bias)
	}
	latest, _ := st.Plan().GetLatestPlanForTraderSession(td, "NY", at.id)
	if latest == nil || latest.Version != 2 {
		t.Fatalf("the newest version governs at read time, got %+v", latest)
	}
	// A newer active version landed → the read SUCCEEDED → key consumed.
	if v := sysCfgVal(t, st, flipRereadDoneKey(row)); v == "" || v == "0" {
		t.Fatalf("a successful read must consume the once-key, got %q", v)
	}
	// The refusal must name both rows and the winner.
	if !strings.Contains(logBuf.String(), "the re-armed v1 and the new v2 both stand as written, and the newest version (v2) governs at read time") {
		t.Fatalf("refusal line must name both rows; log:\n%s", logBuf.String())
	}
}

// (iii-b) BLOCKER 3(b): the goroutine re-reads the row before the read; a row
// that is no longer dormant is skipped — no AI call, no key.
func TestFlipRereadPreReadRecheckSkipsRearmedRow(t *testing.T) {
	at, st, client := realPathTrader(t, true, func(int, string) (string, error) { return validShortPlanJSON, nil })
	now := time.Date(2026, 8, 18, 14, 0, 0, 0, time.UTC)
	flipRereadTestNow(t, now)
	td := "2026-08-18"
	row := seedActivePlan(t, at, td, "NY", now.Add(-40*time.Minute), flipFixtureDoc()) // ACTIVE, never dormant
	seedFlipBars(15500, 15470, 6*time.Minute, now)
	logBuf := captureTraderLog(t)

	at.maybeRereadAfterFlip(now, "NY", td, row, "flip-condition: 2x5m close below 15480.00 → bias short")

	if !waitFor(t, 10*time.Second, func() bool {
		return strings.Contains(logBuf.String(), `SKIPPED before the read: the row is "active", no longer dormant`)
	}) {
		t.Fatalf("no pre-read skip line; log:\n%s", logBuf.String())
	}
	if client.calls() != 0 {
		t.Fatalf("no AI call may happen for a non-dormant row, got %d", client.calls())
	}
	if v := sysCfgVal(t, st, flipRereadDoneKey(row)); v != "" && v != "0" {
		t.Fatalf("once-key must stay clear, got %q", v)
	}
	if latest, _ := st.Plan().GetLatestPlanForTraderSession(td, "NY", at.id); latest == nil || latest.Version != 1 {
		t.Fatalf("nothing may be authored: %+v", latest)
	}
}

// (iv) OFF path on the REAL seam: no read, no key, and the dormant line is
// byte-identical to the pre-wave text — on the fire cycle AND on the next
// (dormant-branch) cycle.
func TestFlipRereadOffPathRealSeamByteIdentical(t *testing.T) {
	at, st, client := realPathTrader(t, false, func(int, string) (string, error) { return validShortPlanJSON, nil })
	now := time.Date(2026, 8, 18, 14, 0, 0, 0, time.UTC)
	flipRereadTestNow(t, now)
	td := "2026-08-18"
	row := seedActivePlan(t, at, td, "NY", now.Add(-40*time.Minute), flipFixtureDoc())
	seedFlipBars(15500, 15470, 6*time.Minute, now)
	logBuf := captureTraderLog(t)

	at.maybeRunSessionReadsAt(now)

	wantLine := "😴 plan 2026-08-18 NY v1 DORMANT — flip-condition: 2x5m close below 15480.00 (buffer 0.0×ATR14, 3× 5m closes) → bias short (entries blocked; auto re-arms when price closes back; replan budget untouched)"
	if !strings.Contains(logBuf.String(), wantLine) {
		t.Fatalf("dormant line must be byte-identical to the pre-wave text; log:\n%s", logBuf.String())
	}
	if strings.Contains(logBuf.String(), "structure_flip") {
		t.Fatalf("knob OFF must not mention structure_flip; log:\n%s", logBuf.String())
	}
	// Next cycle through the dormant branch: still nothing.
	next := now.Add(time.Duration(store.DefaultWakeMinIntervalMin+1) * time.Minute)
	flipRereadTestNow(t, next)
	seedFlipBars(15500, 15470, 6*time.Minute, next)
	at.maybeRunSessionReadsAt(next)
	time.Sleep(200 * time.Millisecond) // any launched goroutine would have called by now
	if client.calls() != 0 {
		t.Fatalf("knob OFF must never call the planner, got %d calls", client.calls())
	}
	if v := sysCfgVal(t, st, flipRereadDoneKey(row)); v != "" {
		t.Fatalf("knob OFF must never touch the once-key, got %q", v)
	}
	if latest, _ := st.Plan().GetLatestPlanForTraderSession(td, "NY", at.id); latest == nil || latest.Version != 1 || latest.Lifecycle != "dormant" {
		t.Fatalf("OFF = dormant v1 and nothing else: %+v", latest)
	}
}

// BLOCKER 2 — the once-key is no longer written at launch, so an in-memory
// guard must stop a second cycle from launching a second read while the
// first planner call is still open.
func TestFlipRereadInFlightGuardBlocksSecondLaunch(t *testing.T) {
	release := make(chan struct{})
	at, st, client := realPathTrader(t, true, func(int, string) (string, error) {
		<-release
		return validShortPlanJSON, nil
	})
	now := time.Date(2026, 8, 18, 14, 0, 0, 0, time.UTC)
	flipRereadTestNow(t, now)
	td := "2026-08-18"
	row := seedActivePlan(t, at, td, "NY", now.Add(-40*time.Minute), flipFixtureDoc())
	seedFlipBars(15500, 15470, 6*time.Minute, now)
	logBuf := captureTraderLog(t)

	at.maybeRunSessionReadsAt(now)
	if !waitFor(t, 10*time.Second, func() bool { return client.calls() == 1 }) {
		t.Fatalf("first read never reached the client; log:\n%s", logBuf.String())
	}
	// The key is NOT set while the call is open (a restart here loses nothing).
	if v := sysCfgVal(t, st, flipRereadDoneKey(row)); v != "" && v != "0" {
		t.Fatalf("once-key must not be set at launch, got %q", v)
	}
	// Next cycle, well past wake_min_interval: dormant branch → guard refuses.
	next := now.Add(time.Duration(store.DefaultWakeMinIntervalMin+1) * time.Minute)
	flipRereadTestNow(t, next)
	seedFlipBars(15500, 15470, 6*time.Minute, next)
	at.maybeRunSessionReadsAt(next)
	time.Sleep(200 * time.Millisecond)
	if client.calls() != 1 {
		t.Fatalf("a second read must not launch while the first is open, got %d calls", client.calls())
	}
	close(release)
	if !waitFor(t, 10*time.Second, func() bool {
		return versionLifecycle(t, st, td, "NY", at.id, 1) == "superseded:flip"
	}) {
		t.Fatalf("v1 not superseded after the open call returned; log:\n%s", logBuf.String())
	}
	if v := sysCfgVal(t, st, flipRereadDoneKey(row)); v == "" || v == "0" {
		t.Fatalf("once-key must be set after the store-decided success, got %q", v)
	}
	if client.calls() != 1 {
		t.Fatalf("exactly one AI call end to end, got %d", client.calls())
	}
}
