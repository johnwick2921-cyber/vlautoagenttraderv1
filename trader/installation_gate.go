package trader

import (
	"fmt"
	"sort"
	"strings"
	"time"

	ntwire "nofx/provider/ninjatrader"
	"nofx/store"
	ntTrader "nofx/trader/ninjatrader"
)

// ── W-ONE-BUTTON M2 — THE INSTALLATION-WIDE GATE ───────────────────────────
//
// One read-only verdict for the whole installation: may an update proceed past
// its drain step? The class-33 cutover gate answers for ONE trader; an update
// replaces the binary every trader shares and the AddOn every account shares,
// so this gate answers for all of them at once:
//
//	hold              the hold file is present and held (a corrupt file holds)
//	go_drained        the entry barrier is engaged and no entry send holds a permit
//	in_flight_sends   the permit count, quoted on its own
//	queued_signals    nothing waits in the wire's reconnect queue
//	planner_in_flight NO planner-class read is claimed, any trader: the UNION of
//	                  plannerReadInFlight ∪ weeklyReadClaim ∪ flipRereadInFlight
//	                  ∪ deathRereadInFlight
//	traders_nt8       every loaded trader (TraderManager ∪ the Picture HTF
//	                  registry, which never unregisters — gap U3) is an NT8 TCP
//	                  trader; anything else is coverage we cannot see (CSV
//	                  transport, other brokers). Replaces cutover leg 3's
//	                  vacuous "not an NT8 trader — not applicable" pass.
//	addon_ack         the CURRENT connection acked THIS job, held, recently
//	addon_census      the AddOn's census (all accounts, CTO ruling Q1): taken,
//	                  no connected non-SIM connection, no position and no
//	                  working order of any action on any account
//	ledger_exposure   no placed/unconfirmed armed row and no unresolved Picture
//	                  send for ANY trader id, loaded or not (canonical arm-state
//	                  predicates); authorized-but-unplaced arms are informational
//	trader_cutover:*  each NT8 trader's own cutover legs 1, 2 and 4
//
// Every leg quotes its source; a leg that cannot be evaluated FAILS; a panic in
// one leg fails that leg and the rest are still computed.

// InstallationGateLeg is one leg's verdict.
type InstallationGateLeg struct {
	Name   string `json:"name"`
	Pass   bool   `json:"pass"`
	Detail string `json:"detail"`
	Source string `json:"source"`
}

// InstallationGate is the whole verdict. Ready only when every leg passes.
type InstallationGate struct {
	Ready   bool                  `json:"ready"`
	JobID   string                `json:"job_id"` // "n/a" when no well-formed hold names one
	Legs    []InstallationGateLeg `json:"legs"`
	Traders []string              `json:"traders"` // every trader id the gate covered
	Note    string                `json:"note"`
}

// installationWire is what the gate reads from the NT8 wire.
type installationWire struct {
	Rec       ntwire.ConnectionRecord
	Connected bool
	Queued    int
	HasAck    bool
	AckAge    time.Duration
}

// installationWireView reads the wire through the first NT8 TCP trader (all of
// them share one server). A seam so the leg logic is testable without NT8.
var installationWireView = func(nts []*AutoTrader) (installationWire, bool) {
	for _, at := range nts {
		nt, ok := at.trader.(*ntTrader.TCPTrader)
		if !ok {
			continue
		}
		rec, connected, queued, ok := nt.MaintenanceView()
		if !ok {
			continue
		}
		age, has := rec.AckAge()
		return installationWire{Rec: rec, Connected: connected, Queued: queued, HasAck: has, AckAge: age}, true
	}
	return installationWire{}, false
}

// installationTraderCutover is each NT8 trader's own cutover verdict (canon
// 53: the production seam is CutoverGateStatus, pinned by a test).
var installationTraderCutover = func(at *AutoTrader) []CutoverLeg { return at.CutoverGateStatus().Legs }

// InstallationGateStatus computes the verdict. loaded is the TraderManager's
// set; st is the store (ledger reads). Read-only; never panics.
func InstallationGateStatus(loaded map[string]*AutoTrader, st *store.Store) (g InstallationGate) {
	g = InstallationGate{JobID: "n/a"}
	// A panic OUTSIDE any single leg (the wire read, the registry scan) still
	// yields a failed verdict — never a crash, never a pass.
	defer func() {
		if r := recover(); r != nil {
			g.Ready = false
			g.Legs = append(g.Legs, InstallationGateLeg{Name: "gate", Pass: false, Source: "InstallationGateStatus",
				Detail: fmt.Sprintf("gate panicked outside a leg: %v — fail-closed", r)})
		}
	}()
	leg := func(name, source string, fn func() (bool, string)) {
		l := InstallationGateLeg{Name: name, Source: source}
		func() {
			defer func() {
				if r := recover(); r != nil {
					l.Pass, l.Detail = false, fmt.Sprintf("leg panicked: %v — fail-closed", r)
				}
			}()
			l.Pass, l.Detail = fn()
		}()
		g.Legs = append(g.Legs, l)
	}

	// The covered set: TraderManager ∪ Picture HTF registry (U3), by id.
	all := map[string]*AutoTrader{}
	registryOnly := []string{}
	for id, at := range loaded {
		if at != nil {
			all[id] = at
		}
	}
	pictureHtfTraders.Range(func(k, v any) bool {
		id, _ := k.(string)
		at, _ := v.(*AutoTrader)
		if at != nil && all[id] == nil {
			all[id] = at
			registryOnly = append(registryOnly, id)
		}
		return true
	})
	ids := make([]string, 0, len(all))
	for id := range all {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	g.Traders = ids
	var nts []*AutoTrader
	for _, id := range ids {
		if _, ok := all[id].trader.(*ntTrader.TCPTrader); ok {
			nts = append(nts, all[id])
		}
	}

	// hold
	st0, configured := maintenanceState()
	holdJob := ""
	leg("hold", "data/updater/hold.json (store.ReadMaintenanceHold)", func() (bool, string) {
		switch {
		case !configured:
			return false, "maintenance data dir not configured — the hold cannot be read"
		case !st0.Held:
			return false, "no hold present (an update must write the hold before it drains)"
		case st0.Corrupt:
			// It HOLDS (every entry stays refused) but no job can be bound to
			// it: the AddOn acks job "" and a job-scoped clear refuses it. An
			// update may not proceed on a hold it cannot identify.
			return false, maintenanceReason(st0) + "; no job can be bound to it — repair it or force-clear it with the operator CLI before an update proceeds"
		}
		holdJob = st0.Hold.JobID
		g.JobID = holdJob
		return true, maintenanceReason(st0)
	})

	// go_drained + in_flight_sends
	inFlight := MaintenanceInFlight()
	leg("go_drained", "EntryBarrier (trader/maintenance_gate.go)", func() (bool, string) {
		if !maintenanceBarrier.Held() {
			return false, "entry barrier not engaged"
		}
		if inFlight != 0 {
			return false, fmt.Sprintf("%d entry send(s) still hold a permit", inFlight)
		}
		return true, "barrier engaged, 0 permits held"
	})
	leg("in_flight_sends", "EntryBarrier in-flight counter", func() (bool, string) {
		return inFlight == 0, fmt.Sprintf("in_flight_sends=%d", inFlight)
	})

	// wire-read legs
	wire, haveWire := installationWireView(nts)
	noWire := "no NT8 TCP trader loaded — the AddOn cannot be asked (fail-closed)"
	leg("queued_signals", "TCPServer reconnect queue", func() (bool, string) {
		if !haveWire {
			return false, noWire
		}
		return wire.Queued == 0, fmt.Sprintf("queued_signals=%d", wire.Queued)
	})

	// planner_in_flight — the union
	leg("planner_in_flight", "plannerReadInFlight ∪ weeklyReadClaim ∪ flipRereadInFlight ∪ deathRereadInFlight", func() (bool, string) {
		var held []string
		for _, m := range []struct {
			name string
			keys func() []string
		}{
			{"plannerReadInFlight", func() []string { return syncMapKeys(&plannerReadInFlight) }},
			{"weeklyReadClaim", func() []string { return syncMapKeys(&weeklyReadClaim) }},
			{"flipRereadInFlight", func() []string { return syncMapKeys(&flipRereadInFlight) }},
			{"deathRereadInFlight", func() []string { return syncMapKeys(&deathRereadInFlight) }},
		} {
			for _, k := range m.keys() {
				held = append(held, m.name+"["+k+"]")
			}
		}
		if len(held) > 0 {
			return false, "IN FLIGHT: " + strings.Join(held, ", ")
		}
		return true, "no planner-class read claimed, any trader"
	})

	// traders_nt8
	leg("traders_nt8", "TraderManager ∪ pictureHtfTraders", func() (bool, string) {
		registry := map[string]bool{}
		for _, id := range registryOnly {
			registry[id] = true
		}
		var bad, inert []string
		for _, id := range ids {
			if _, ok := all[id].trader.(*ntTrader.TCPTrader); ok {
				continue
			}
			if registry[id] {
				// M2.1 (CTO ruling on review item b): registry-only and not NT8
				// — it cannot reach NT8 (no evaluator off ninjatrader, and the
				// send needs a *TCPTrader). Listed, never failing: the registry
				// never unregisters, so failing here would hold the gate shut
				// until a restart the update itself needs.
				inert = append(inert, id)
				continue
			}
			bad = append(bad, fmt.Sprintf("%s (%T)", id, all[id].trader))
		}
		extra := ""
		if len(registryOnly) > 0 {
			sort.Strings(registryOnly)
			extra = "; registry-only (not in the manager): " + strings.Join(registryOnly, ", ")
		}
		if len(inert) > 0 {
			sort.Strings(inert)
			extra += "; of which not NT8 (cannot reach NT8, informational): " + strings.Join(inert, ", ")
		}
		if len(bad) > 0 {
			return false, "not an NT8 TCP trader — coverage unknown: " + strings.Join(bad, ", ") + extra
		}
		return true, fmt.Sprintf("%d trader(s); every running trader is NT8 TCP%s", len(ids), extra)
	})

	// addon_ack
	leg("addon_ack", "maintenance_ack on the current connection's record", func() (bool, string) {
		if !haveWire {
			return false, noWire
		}
		r := wire.Rec
		id := fmt.Sprintf("accept_seq=%d remote_port=%d", r.AcceptSeq, r.RemotePort)
		switch {
		case !wire.Connected:
			return false, "AddOn not connected (" + id + ")"
		case !wire.HasAck || r.Ack == nil:
			return false, "addon_ack=n/a — no maintenance_ack on this connection (an AddOn older than 2026-09-22-m2 never acks) (" + id + ")"
		case !r.Ack.Held:
			return false, "the AddOn's ack says not held (" + id + ")"
		case holdJob == "":
			return false, "no hold job to match the ack against"
		case r.Ack.JobID != holdJob:
			return false, fmt.Sprintf("ack names job %q, the hold is %q (%s)", r.Ack.JobID, holdJob, id)
		case wire.AckAge > ntwire.MaintenanceAckMaxAge():
			return false, fmt.Sprintf("ack is %s old (max %s) (%s)", wire.AckAge.Round(time.Second), ntwire.MaintenanceAckMaxAge(), id)
		case r.Ack.QueuedCommands != 0:
			return false, fmt.Sprintf("the AddOn reports queued_commands=%d — work still in flight at the AddOn (%s)", r.Ack.QueuedCommands, id)
		}
		return true, fmt.Sprintf("held job=%s build=%s age=%s queued_commands=%d (%s)", r.Ack.JobID, r.Ack.BuildID, wire.AckAge.Round(time.Millisecond), r.Ack.QueuedCommands, id)
	})

	// addon_census
	leg("addon_census", "maintenance_ack census (AddOn; every connection and account, no names)", func() (bool, string) {
		if !haveWire {
			return false, noWire
		}
		a := wire.Rec.Ack
		switch {
		case a == nil:
			return false, "addon_ack=n/a — no census"
		case a.CensusError != "":
			return false, a.CensusError
		case a.Connections == nil:
			return false, "connections were not enumerated (absent is not empty)"
		case a.Accounts == nil:
			return false, "accounts were not enumerated (absent is not empty)"
		}
		nonSim, unsettled, positions, working := 0, 0, 0, 0
		for _, c := range a.Connections {
			if c.Connected && !c.Sim {
				nonSim++
			}
			if !c.Settled {
				unsettled++ // M2.1: Connecting / ConnectionLost — its accounts cannot be vouched for
			}
		}
		for _, ac := range a.Accounts {
			positions += ac.Positions
			working += ac.Working
		}
		detail := fmt.Sprintf("connections=%d connected_non_SIM=%d accounts=%d positions=%d working=%d",
			len(a.Connections), nonSim, len(a.Accounts), positions, working)
		var why []string
		if nonSim > 0 {
			why = append(why, fmt.Sprintf("%d connected non-SIM connection(s)", nonSim))
		}
		if unsettled > 0 {
			why = append(why, fmt.Sprintf("%d connection(s) in a transitional state (neither Connected nor Disconnected)", unsettled))
		}
		if positions > 0 {
			why = append(why, fmt.Sprintf("%d open position(s)", positions))
		}
		if working > 0 {
			why = append(why, fmt.Sprintf("%d working order(s) of any action", working))
		}
		if len(why) > 0 {
			return false, strings.Join(why, "; ") + " — " + detail
		}
		return true, detail
	})

	// ledger_exposure — every trader id
	leg("ledger_exposure", "armed_orders + picture_htf_opportunities, all trader ids (canonical arm-state predicates)", func() (bool, string) {
		if st == nil || st.ArmedOrders() == nil {
			return false, "store unavailable — ledger cannot be read"
		}
		rows, err := st.ArmedOrders().ListNonTerminalAllTraders()
		if err != nil {
			return false, "armed ledger read failed: " + err.Error()
		}
		unplaced := 0
		var exposed []string
		for _, r := range rows {
			if store.IsUnplacedArm(r.State, r.SignalID) {
				unplaced++
				continue
			}
			exposed = append(exposed, fmt.Sprintf("armed#%d %s %s", r.ID, r.TraderID, r.State))
		}
		pics, perr := st.PictureHtfRecoverableAll()
		if perr != nil {
			return false, "picture ledger read failed: " + perr.Error()
		}
		for _, p := range pics {
			exposed = append(exposed, fmt.Sprintf("picture %s %s %s", p.OppKey, p.TraderID, p.Stage))
		}
		info := fmt.Sprintf("%d authorized-but-unplaced arm(s) (informational)", unplaced)
		if len(exposed) > 0 {
			return false, "placed/unresolved: " + strings.Join(exposed, ", ") + "; " + info
		}
		return true, "no placed or unresolved row; " + info
	})

	// trader_cutover:<id> — each NT8 trader's legs 1, 2, 4
	for _, at := range nts {
		at := at
		leg("trader_cutover:"+at.id, "CutoverGateStatus legs 1 (db_open_positions), 2 (api_positions), 4 (working_orders)", func() (bool, string) {
			var failed, passed []string
			for _, l := range installationTraderCutover(at) {
				if l.N == 3 || l.N == 5 {
					continue // 3 → traders_nt8 + addon_census; 5 → planner_in_flight
				}
				if l.Pass {
					passed = append(passed, l.Name)
				} else {
					failed = append(failed, l.Name+": "+l.Detail)
				}
			}
			if len(failed) > 0 {
				return false, strings.Join(failed, " | ")
			}
			if len(passed) == 0 {
				return false, "no cutover leg evaluated (fail-closed)"
			}
			return true, "passed: " + strings.Join(passed, ", ")
		})
	}

	g.Ready = len(g.Legs) > 0
	for _, l := range g.Legs {
		if !l.Pass {
			g.Ready = false
		}
	}
	g.Note = "READ-ONLY. An update may proceed past drain only when ready=true. Every leg that cannot be evaluated fails."
	return g
}

func syncMapKeys(m interface{ Range(func(k, v any) bool) }) []string {
	var out []string
	m.Range(func(k, _ any) bool {
		out = append(out, fmt.Sprint(k))
		return true
	})
	sort.Strings(out)
	return out
}
