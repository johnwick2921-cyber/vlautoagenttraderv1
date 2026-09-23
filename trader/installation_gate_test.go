package trader

import (
	"strings"
	"testing"
	"time"

	ntwire "nofx/provider/ninjatrader"
	"nofx/store"
	ntTrader "nofx/trader/ninjatrader"
)

// ── W-ONE-BUTTON M2 — the installation-wide gate ───────────────────────────
//
// One verdict for the whole installation: every leg must pass, a leg that
// cannot be evaluated FAILS, every leg quotes its source, and the function
// never panics. The fixture is a gate that is READY; each case breaks exactly
// one thing and names the leg that must fail.

type gateFixture struct {
	dir    string
	st     *store.Store
	loaded map[string]*AutoTrader
	wire   installationWire
}

func goodCensusAck(job string) *ntwire.MaintenanceAckPayload {
	return &ntwire.MaintenanceAckPayload{Held: true, JobID: job, BuildID: "2026-09-22-m2",
		Connections: []ntwire.CensusConnection{{Sim: true, Connected: true}, {Sim: false, Connected: false}},
		Accounts:    []ntwire.CensusAccount{{Sim: true, Positions: 0, Working: 0}}}
}

func newGateFixture(t *testing.T) *gateFixture {
	t.Helper()
	dir := withMaintenanceDir(t)
	at, st := resetTrader(t, store.StrategyConfig{})
	at.id = "gate-t1"
	at.trader = ntTrader.NewTCPTrader(ntwire.NewTCPServer(nil), "MNQ", "Sim101")
	setHold(t, dir, "job-g")
	f := &gateFixture{dir: dir, st: st, loaded: map[string]*AutoTrader{at.id: at},
		wire: installationWire{Connected: true, Rec: ntwire.ConnectionRecord{AcceptSeq: 3, RemotePort: 50123, Ack: goodCensusAck("job-g")}, HasAck: true, AckAge: time.Second}}
	prevW, prevC := installationWireView, installationTraderCutover
	installationWireView = func([]*AutoTrader) (installationWire, bool) { return f.wire, true }
	installationTraderCutover = func(*AutoTrader) []CutoverLeg {
		return []CutoverLeg{{N: 1, Name: "db_open_positions", Pass: true}, {N: 2, Name: "api_positions", Pass: true}, {N: 4, Name: "working_orders", Pass: true}}
	}
	t.Cleanup(func() { installationWireView, installationTraderCutover = prevW, prevC })
	return f
}

func (f *gateFixture) run() InstallationGate { return InstallationGateStatus(f.loaded, f.st) }

func legOf(g InstallationGate, name string) (InstallationGateLeg, bool) {
	for _, l := range g.Legs {
		if l.Name == name {
			return l, true
		}
	}
	return InstallationGateLeg{}, false
}

func mustFail(t *testing.T, g InstallationGate, name, want string) {
	t.Helper()
	if g.Ready {
		t.Fatalf("gate READY although leg %s must fail: %+v", name, g.Legs)
	}
	l, ok := legOf(g, name)
	if !ok {
		t.Fatalf("leg %s missing: %+v", name, g.Legs)
	}
	if l.Pass {
		t.Fatalf("leg %s passed; want fail (%s)", name, l.Detail)
	}
	if want != "" && !strings.Contains(l.Detail, want) {
		t.Fatalf("leg %s detail %q does not say %q", name, l.Detail, want)
	}
}

func TestInstallationGateReadyWhenEveryLegPasses(t *testing.T) {
	g := newGateFixture(t).run()
	if !g.Ready {
		t.Fatalf("the fixture must be READY: %+v", g.Legs)
	}
	for _, l := range g.Legs {
		if l.Source == "" {
			t.Errorf("leg %s quotes no source", l.Name)
		}
	}
	if g.JobID != "job-g" {
		t.Fatalf("job_id: %q", g.JobID)
	}
	for _, name := range []string{"hold", "go_drained", "in_flight_sends", "queued_signals", "planner_in_flight", "traders_nt8", "addon_ack", "addon_census", "ledger_exposure", "trader_cutover:gate-t1"} {
		if _, ok := legOf(g, name); !ok {
			t.Errorf("leg %s missing", name)
		}
	}
}

func TestInstallationGateFailsWithoutAHold(t *testing.T) {
	f := newGateFixture(t)
	if err := store.ClearMaintenanceHold(f.dir, "job-g"); err != nil {
		t.Fatal(err)
	}
	mustFail(t, f.run(), "hold", "no hold")
}

func TestInstallationGateFailsWhileAnEntrySendHoldsAPermit(t *testing.T) {
	f := newGateFixture(t)
	if err := store.ClearMaintenanceHold(f.dir, "job-g"); err != nil {
		t.Fatal(err)
	}
	release, ok := MaintenanceEntryPermit() // a send in flight when the hold lands
	if !ok {
		t.Fatal("fixture: permit refused before the hold")
	}
	defer release()
	setHold(t, f.dir, "job-g")
	g := f.run()
	mustFail(t, g, "in_flight_sends", "1")
	mustFail(t, g, "go_drained", "")
}

func TestInstallationGateAddOnAckCases(t *testing.T) {
	cases := map[string]struct {
		mut  func(w *installationWire)
		want string
	}{
		"old AddOn never acks": {func(w *installationWire) { w.Rec.Ack, w.HasAck = nil, false }, "n/a"},
		"ack for another job":  {func(w *installationWire) { w.Rec.Ack.JobID = "job-old" }, "job-old"},
		"ack says not held":    {func(w *installationWire) { w.Rec.Ack.Held = false }, "not held"},
		"stale ack":            {func(w *installationWire) { w.AckAge = ntwire.MaintenanceAckMaxAge() + time.Second }, "old"},
		"AddOn disconnected":   {func(w *installationWire) { w.Connected = false }, "not connected"},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			f := newGateFixture(t)
			c.mut(&f.wire)
			mustFail(t, f.run(), "addon_ack", c.want)
		})
	}
}

func TestInstallationGateCensusCases(t *testing.T) {
	cases := map[string]struct {
		mut  func(a *ntwire.MaintenanceAckPayload)
		want string
	}{
		"census error": {func(a *ntwire.MaintenanceAckPayload) {
			a.CensusError, a.Connections, a.Accounts = "census failed: X", nil, nil
		}, "census failed"},
		"connections not taken": {func(a *ntwire.MaintenanceAckPayload) { a.Connections = nil }, "connections"},
		"accounts not taken":    {func(a *ntwire.MaintenanceAckPayload) { a.Accounts = nil }, "accounts"},
		"connected non-SIM":     {func(a *ntwire.MaintenanceAckPayload) { a.Connections[1].Connected = true }, "non-SIM"},
		"position on any account": {func(a *ntwire.MaintenanceAckPayload) {
			a.Accounts = append(a.Accounts, ntwire.CensusAccount{Sim: false, Positions: 1})
		}, "position"},
		"working order anywhere": {func(a *ntwire.MaintenanceAckPayload) { a.Accounts[0].Working = 2 }, "working"},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			f := newGateFixture(t)
			c.mut(f.wire.Rec.Ack)
			mustFail(t, f.run(), "addon_census", c.want)
		})
	}
}

func TestInstallationGateQueuedSignalsFail(t *testing.T) {
	f := newGateFixture(t)
	f.wire.Queued = 1
	mustFail(t, f.run(), "queued_signals", "1")
}

func TestInstallationGateNoWireFailsTheWireLegs(t *testing.T) {
	f := newGateFixture(t)
	installationWireView = func([]*AutoTrader) (installationWire, bool) { return installationWire{}, false }
	g := f.run()
	for _, leg := range []string{"addon_ack", "addon_census", "queued_signals"} {
		mustFail(t, g, leg, "no NT8")
	}
}

// Leg (a): the UNION of every planner-class claim, any trader.
func TestInstallationGatePlannerUnion(t *testing.T) {
	maps := map[string]func(key string) func(){
		"plannerReadInFlight": func(k string) func() {
			plannerReadInFlight.Store(k, struct{}{})
			return func() { plannerReadInFlight.Delete(k) }
		},
		"weeklyReadClaim": func(k string) func() {
			weeklyReadClaim.Store(k, struct{}{})
			return func() { weeklyReadClaim.Delete(k) }
		},
		"flipRereadInFlight": func(k string) func() {
			flipRereadInFlight.Store(k, time.Now())
			return func() { flipRereadInFlight.Delete(k) }
		},
		"deathRereadInFlight": func(k string) func() {
			deathRereadInFlight.Store(k, time.Now())
			return func() { deathRereadInFlight.Delete(k) }
		},
	}
	for name, claim := range maps {
		t.Run(name, func(t *testing.T) {
			f := newGateFixture(t)
			undo := claim("some-OTHER-trader:" + name)
			defer undo()
			mustFail(t, f.run(), "planner_in_flight", name)
		})
	}
}

// Leg (b): a loaded trader that is not an NT8 TCP trader is unknown coverage.
func TestInstallationGateNonNT8TraderFails(t *testing.T) {
	f := newGateFixture(t)
	other, _ := resetTrader(t, store.StrategyConfig{})
	other.id = "csv-or-crypto"
	f.loaded[other.id] = other
	mustFail(t, f.run(), "traders_nt8", "csv-or-crypto")
}

// U3 + M2.1 (CTO ruling on review item b): the Picture HTF registry never
// unregisters, so it keeps every trader that ever Run() — crypto and removed
// ones too. A REGISTRY-ONLY trader that is not an NT8 TCP trader cannot reach
// NT8 (its evaluator is nil unless the exchange is ninjatrader, and the send
// needs a *TCPTrader), so it is listed as covered and informational — it must
// not keep the gate failing until a restart the update itself needs. A RUNNING
// (manager-loaded) non-NT8 trader still fails the leg.
func TestInstallationGateRegistryOnlyCryptoTraderIsInformational(t *testing.T) {
	f := newGateFixture(t)
	ghost, _ := resetTrader(t, store.StrategyConfig{})
	ghost.id = "picture-ghost-crypto"
	ghost.exchange = "binance"
	pictureHtfTraders.Store(ghost.id, ghost)
	defer pictureHtfTraders.Delete(ghost.id)
	g := f.run()
	l, _ := legOf(g, "traders_nt8")
	if !l.Pass || !strings.Contains(l.Detail, "picture-ghost-crypto") {
		t.Fatalf("a registry-only crypto trader is informational (listed, not failing): %+v", l)
	}
	if !g.Ready {
		t.Fatalf("a registry-only crypto trader must not keep the gate closed: %+v", g.Legs)
	}
	found := false
	for _, id := range g.Traders {
		found = found || id == "picture-ghost-crypto"
	}
	if !found {
		t.Fatalf("the registry-only trader must still be listed as covered: %v", g.Traders)
	}
	// ...while a RUNNING non-NT8 trader in the manager still fails the leg.
	running, _ := resetTrader(t, store.StrategyConfig{})
	running.id = "running-crypto"
	running.exchange = "binance"
	f.loaded[running.id] = running
	mustFail(t, f.run(), "traders_nt8", "running-crypto")
}

// Leg (e): the ledger for EVERY trader id, loaded or not; an authorized but
// unplaced arm is informational.
func TestInstallationGateLedgerExposureAllTraders(t *testing.T) {
	f := newGateFixture(t)
	led := f.st.ArmedOrders()
	arm := func(trader, plan string) int64 {
		r := store.ArmedOrderDB{TraderID: trader, PlanID: plan, Version: 1, Session: "NY", Scenario: "S1", Side: "LONG", EntryPx: 100, StopPx: 95, TargetPx: 110, State: store.StateArmed}
		if err := f.st.ArmedOrders().UpsertArm(&r); err != nil {
			t.Fatal(err)
		}
		return r.ID
	}
	arm("never-loaded", "2026-09-22:NY:never-loaded")
	if g := f.run(); !g.Ready {
		leg, _ := legOf(g, "ledger_exposure")
		t.Fatalf("an unplaced arm is informational, not exposure: %+v", leg)
	}
	id := arm("never-loaded-2", "2026-09-22:NY:never-loaded-2")
	if err := led.BeginPlacement(id, "sig-orphan"); err != nil {
		t.Fatal(err)
	}
	mustFail(t, f.run(), "ledger_exposure", "never-loaded-2")
}

func TestInstallationGatePicturePendingFails(t *testing.T) {
	f := newGateFixture(t)
	if _, _, err := f.st.PictureHtfClaim(&store.PictureHtfOpportunityDB{OppKey: "opp-1", TraderID: "someone", Stage: "confirmed"}); err != nil {
		t.Fatal(err)
	}
	if won, err := f.st.PictureHtfClaimSubmission("opp-1", "picture-htf-1"); err != nil || !won {
		t.Fatalf("fixture: %v %v", won, err)
	}
	mustFail(t, f.run(), "ledger_exposure", "picture")
}

// The production per-trader seam is CutoverGateStatus (canon 53): a trader
// whose own cutover leg fails fails the installation gate, quoting that leg.
// (Leg 4 with no broker snapshot falls back to the ledger by design — F12's
// transition case — so the failing leg here is leg 1, a real open DB row.)
func TestInstallationGateUsesEachTradersCutoverGate(t *testing.T) {
	f := newGateFixture(t)
	installationTraderCutover = func(at *AutoTrader) []CutoverLeg { return at.CutoverGateStatus().Legs }
	if g := f.run(); !g.Ready {
		t.Fatalf("fixture: with the real seam and a flat trader the gate must be ready: %+v", g.Legs)
	}
	if err := f.st.Position().CreateOpenPosition(&store.TraderPosition{
		TraderID: "gate-t1", ExchangeType: "ninjatrader", ExchangePositionID: "gate-pos-1",
		Symbol: "MNQ", Side: "LONG", Quantity: 1, EntryQuantity: 1, EntryPrice: 29000,
		EntryOrderID: "sig-gate", Leverage: 1, Status: "OPEN", Source: "test", Account: "Sim101",
	}); err != nil {
		t.Fatal(err)
	}
	mustFail(t, f.run(), "trader_cutover:gate-t1", "db_open_positions")
}

// (g) never panics: a leg that panics fails ITSELF, the rest still compute.
func TestInstallationGateNeverPanics(t *testing.T) {
	f := newGateFixture(t)
	installationTraderCutover = func(*AutoTrader) []CutoverLeg { panic("boom") }
	g := f.run()
	mustFail(t, g, "trader_cutover:gate-t1", "panicked")
	if _, ok := legOf(g, "addon_census"); !ok {
		t.Fatal("the other legs must still be computed")
	}
}

// A hold file the process cannot read HOLDS (entries stay refused) but the
// update must NOT proceed on it: no job can be bound to it (the AddOn acks job
// "" and a job-scoped clear refuses it). Found by the M2 adversarial review:
// the gate returned ready=true job_id=n/a with every leg passing.
func TestInstallationGateFailsOnAnUnreadableHold(t *testing.T) {
	f := newGateFixture(t)
	if err := writeRaw(f.dir, "{broken"); err != nil {
		t.Fatal(err)
	}
	f.wire.Rec.Ack.JobID = "" // what the AddOn acks for a corrupt hold
	g := f.run()
	mustFail(t, g, "hold", "unreadable")
	mustFail(t, g, "addon_ack", "")
}

// The AddOn reports its command-queue depth; anything above zero is work in
// flight at the AddOn (PROTOCOL.md: an async dispatcher must report it).
func TestInstallationGateFailsOnAddOnQueuedCommands(t *testing.T) {
	f := newGateFixture(t)
	f.wire.Rec.Ack.QueuedCommands = 2
	mustFail(t, f.run(), "addon_ack", "queued_commands=2")
}

// A panic outside any single leg (the wire read, the registry scan) still
// yields a failed verdict, never a crash.
func TestInstallationGateSurvivesAPanicOutsideALeg(t *testing.T) {
	f := newGateFixture(t)
	installationWireView = func([]*AutoTrader) (installationWire, bool) { panic("wire boom") }
	g := f.run()
	if g.Ready {
		t.Fatalf("a panicking gate must not be ready: %+v", g)
	}
	found := false
	for _, l := range g.Legs {
		found = found || (!l.Pass && strings.Contains(l.Detail, "panicked"))
	}
	if !found {
		t.Fatalf("the panic must be reported as a failed leg: %+v", g.Legs)
	}
}
