package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"nofx/kernel"
	"nofx/store"
)

// ── W-EXEC-TRUTH W5 (builder A) — the API's two folds and its one door ──────
//
// handlePlanToday and resolvePlanFinal fold through kernel.ResolvePlanFinal
// (byte-identical with no machine overlay), the card says what composed the
// doc, and applyPlanOverlay — the ONE mutation door for the owner and the
// Ask-Planner Apply — refuses any edit that adds, alters or removes a
// machine (Picture) scenario.

const w5aPlanDoc = `{
  "reasoning": "Balance below PDH; fade edges, long the reclaim.",
  "bias": {"direction": "long", "conviction": "medium", "flip_condition": "2x5m < 15480"},
  "levels": [
    {"price": 15480, "label": "PWL", "grade": "A", "instruction": "fade"},
    {"price": 15620, "label": "PDH", "grade": "A", "instruction": "fade"}
  ],
  "scenarios": [{"id": "S1", "trigger": "sweep 15480 reclaim", "condition": "sweep_reclaim", "direction": "long", "target_chain": [15550, 15620], "invalid": "2x5m<15470", "quality": "A", "confirm":{"rule":"touch","ref_price":15480,"side":"below"}}],
  "no_trade": ["first 5m"],
  "death_condition": "acceptance above 15620",
  "day_type": "balance"
}`

const w5aRef = "t-picture-gate|sim101|mnq 12-26|long|resistance|1789920000000|1790002800000"

// w5aMachineScenario is a Picture scenario exactly as the plan source writes it.
func w5aMachineScenario(id, ref string) kernel.PlanScenario {
	return kernel.PlanScenario{
		ID: id, Trigger: "H1 close 21531.00 above the 4H body 21520.00 (h1_close_break v1, H1 closed 09:59 CT)",
		Condition: "acceptance", Direction: "long", TargetChain: []float64{21590},
		Invalid: "back inside the 4H body 21500.00–21520.00, or the eligibility window closes (10:00:10 CT)", Quality: "B",
		Economics: &kernel.ScenarioEconomics{Version: 1, EntryZone: []float64{21528.5, 21530}},
		Arm:       &kernel.PlanArmSpec{Enabled: true, Entry: 21530, Stop: 21510, Target: 21590, Policy: kernel.EntryPolicyMarketInZone},
		Source:    kernel.ScenarioSourcePicture,
		Machine: &kernel.PlanMachineSource{Rule: kernel.MachineRulePictureH1CloseBreak, RuleVer: 1, Ref: ref,
			EligibleFromMs: 1790002800000, EligibleUntilMs: 1790002810000, RunEpoch: 7, Evidence: json.RawMessage(`{"opp_key":"` + ref + `"}`)},
	}
}

// w5aNYDate is the chain date handlePlanToday reads for ?session=NY right now.
func w5aNYDate(t *testing.T, now time.Time) string {
	t.Helper()
	ny, ok := kernel.DefaultSessionRegistry().SessionByName(kernel.SessionNY)
	if !ok {
		t.Fatal("no NY session")
	}
	if d, ok := kernel.PlanChainTradeDate(ny, now); ok {
		return d
	}
	return now.In(planChicago()).Format("2006-01-02")
}

func w5aSeedPlan(t *testing.T, st *store.Store, date, doc, trigger string) *store.PlanDB {
	t.Helper()
	row := &store.PlanDB{PlanID: store.MakePlanIDForTrader(ppgTrader, date, "NY"), StrategyID: ppgTrader, TradeDate: date, Session: "NY",
		TriggerReason: trigger, Lifecycle: "active", Doc: doc}
	if _, err := st.Plan().AppendPlan(row); err != nil {
		t.Fatal(err)
	}
	return row
}

func w5aOverlay(t *testing.T, st *store.Store, row *store.PlanDB, origin, id, patch string) {
	t.Helper()
	if _, err := st.Plan().AppendOverlay(&store.PlanOverlayDB{PlanID: row.PlanID, PlanVersion: row.Version, Origin: origin, OverlayID: id, Patch: patch}); err != nil {
		t.Fatal(err)
	}
}

func w5aMachineOverlay(t *testing.T, st *store.Store, row *store.PlanDB, sc kernel.PlanScenario) {
	t.Helper()
	patch, err := kernel.MachineOverlayPatch(sc)
	if err != nil {
		t.Fatal(err)
	}
	w5aOverlay(t, st, row, kernel.MachineOverlayOriginPicture, "picture:"+sc.Machine.Ref, patch)
}

type w5aToday struct {
	Found         bool            `json:"found"`
	Doc           json.RawMessage `json:"doc"`
	OverlayErrors []string        `json:"overlay_errors"`
	OverlayCount  int             `json:"overlay_count"`
	MachinePlan   *bool           `json:"machine_plan"`
	ComposedOf    *struct {
		UserOverlays []int `json:"user_overlays"`
		Machine      []struct {
			OverlayVersion int    `json:"overlay_version"`
			ScenarioID     string `json:"scenario_id"`
			Ref            string `json:"ref"`
		} `json:"machine"`
	} `json:"composed_of"`
}

func w5aGetToday(t *testing.T, s *Server, tok string) w5aToday {
	t.Helper()
	rec, _ := olDo(t, s, tok, http.MethodGet, "/api/plan/today?trader_id="+ppgTrader+"&session=NY", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("plan/today: %d %s", rec.Code, rec.Body.String())
	}
	var out w5aToday
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !out.Found {
		t.Fatalf("plan/today found=false: %s", rec.Body.String())
	}
	return out
}

// legacyPlanTodayFold is handlePlanToday's fold as it stood before W5
// (api/handler_plan.go at eb7294c9), kept verbatim as the identity oracle.
func legacyPlanTodayFold(row *store.PlanDB, overlays []*store.PlanOverlayDB) (kernel.PlanDoc, []string) {
	var doc kernel.PlanDoc
	_ = json.Unmarshal([]byte(row.Doc), &doc)
	var overlayErrStrings []string
	if len(overlays) > 0 {
		patches := make([]string, 0, len(overlays))
		for _, ov := range overlays {
			patches = append(patches, ov.Patch)
		}
		if base, mErr := json.Marshal(doc); mErr == nil {
			final, overlayErrs := kernel.ApplyOverlayPatches(base, patches)
			for _, oe := range overlayErrs {
				if oe != nil {
					overlayErrStrings = append(overlayErrStrings, oe.Error())
				}
			}
			var merged kernel.PlanDoc
			if json.Unmarshal(final, &merged) == nil && kernel.ValidatePlanDocWithCaps(&merged, kernel.PlanHardMaxLevels, kernel.PlanHardMaxScenarios) == nil {
				doc = merged
			}
		}
	}
	return doc, overlayErrStrings
}

// legacyResolvePlanFinal is Server.resolvePlanFinal before W5, verbatim.
func legacyResolvePlanFinal(row *store.PlanDB, overlays []*store.PlanOverlayDB) kernel.PlanDoc {
	var doc kernel.PlanDoc
	_ = json.Unmarshal([]byte(row.Doc), &doc)
	if len(overlays) == 0 {
		return doc
	}
	patches := make([]string, 0, len(overlays))
	for _, ov := range overlays {
		patches = append(patches, ov.Patch)
	}
	base, err := json.Marshal(doc)
	if err != nil {
		return doc
	}
	final, _ := kernel.ApplyOverlayPatches(base, patches)
	var merged kernel.PlanDoc
	if json.Unmarshal(final, &merged) == nil && kernel.ValidatePlanDocWithCaps(&merged, kernel.PlanHardMaxLevels, kernel.PlanHardMaxScenarios) == nil {
		return merged
	}
	return doc
}

func w5aFoldFixtures() map[string][]string {
	sc := func(i int) string {
		return fmt.Sprintf(`{"id":"S%d","trigger":"t","condition":"reclaim","direction":"long","target_chain":[15600],"invalid":"i","quality":"B"}`, i)
	}
	stack := `[`
	for i := 2; i <= 6; i++ {
		if i > 2 {
			stack += ","
		}
		stack += `{"op":"add","path":"/scenarios/-","value":` + sc(i) + `}`
	}
	stack += `]`
	return map[string][]string{
		"no overlays":      nil,
		"good owner level": {`[{"op":"add","path":"/levels/-","value":{"price":15600,"label":"OWN","grade":"A","instruction":"fade"}}]`},
		"bad patch skipped": {
			`[{"op":"replace","path":"/levels/99/price","value":1}]`,
			`[{"op":"replace","path":"/bias/conviction","value":"high"}]`,
		},
		"fold fails → base": {stack},
	}
}

// BYTE IDENTITY at both API fold sites with no machine overlay.
func TestAPIFoldsAreByteIdenticalWithoutMachineOverlays(t *testing.T) {
	for name, patches := range w5aFoldFixtures() {
		t.Run(name, func(t *testing.T) {
			s, tok := newPicturePlanGateServer(t, `{"day_plan":{"plan_enabled":true}}`)
			row := w5aSeedPlan(t, s.store, w5aNYDate(t, time.Now()), w5aPlanDoc, "NY_scheduled_read")
			for _, p := range patches {
				w5aOverlay(t, s.store, row, "owner", "", p)
			}
			ovs, _ := s.store.Plan().ListOverlays(row.PlanID, row.Version)
			wantDoc, wantErrs := legacyPlanTodayFold(row, ovs)
			wb, _ := json.Marshal(wantDoc)
			got := w5aGetToday(t, s, tok)
			if string(got.Doc) != string(wb) {
				t.Fatalf("handlePlanToday's doc moved with no machine overlay:\n got %s\nwant %s", got.Doc, wb)
			}
			if strings.Join(got.OverlayErrors, "|") != strings.Join(wantErrs, "|") {
				t.Fatalf("overlay_errors moved: got %q want %q", got.OverlayErrors, wantErrs)
			}
			if got.ComposedOf == nil || got.ComposedOf.UserOverlays == nil || got.ComposedOf.Machine == nil || len(got.ComposedOf.Machine) != 0 {
				t.Fatalf("composed_of must always be present with [] lists, never null: %+v", got.ComposedOf)
			}
			if got.MachinePlan == nil || *got.MachinePlan {
				t.Fatalf("an AI plan reads machine_plan=false: %v", got.MachinePlan)
			}
			rf, ok := s.resolvePlanFinal(row)
			rb, _ := json.Marshal(rf)
			lb, _ := json.Marshal(legacyResolvePlanFinal(row, ovs))
			if !ok || string(rb) != string(lb) {
				t.Fatalf("resolvePlanFinal moved with no machine overlay:\n got %s\nwant %s", rb, lb)
			}
		})
	}
}

// F2 PIN at the card's fold: a FULL plan (5 scenarios) with an owner overlay
// and a machine scenario keeps the owner's overlay active, appends P1 last,
// and composed_of names both.
func TestPlanTodayKeepsOwnerOverlaysBesideAMachineScenario(t *testing.T) {
	s, tok := newPicturePlanGateServer(t, `{"day_plan":{"plan_enabled":true}}`)
	var doc kernel.PlanDoc
	_ = json.Unmarshal([]byte(w5aPlanDoc), &doc)
	for i := 2; len(doc.Scenarios) < kernel.PlanHardMaxScenarios; i++ {
		sc := doc.Scenarios[0]
		sc.ID = fmt.Sprintf("S%d", i)
		doc.Scenarios = append(doc.Scenarios, sc)
	}
	blob, _ := json.Marshal(doc)
	row := w5aSeedPlan(t, s.store, w5aNYDate(t, time.Now()), string(blob), "NY_scheduled_read")
	w5aOverlay(t, s.store, row, "owner", "", `[{"op":"add","path":"/levels/-","value":{"price":15600,"label":"OWN","grade":"A","instruction":"fade"}}]`)
	w5aMachineOverlay(t, s.store, row, w5aMachineScenario("P1", w5aRef))
	got := w5aGetToday(t, s, tok)
	var final kernel.PlanDoc
	if err := json.Unmarshal(got.Doc, &final); err != nil {
		t.Fatal(err)
	}
	hasOwner := false
	for _, l := range final.Levels {
		hasOwner = hasOwner || l.Label == "OWN"
	}
	if !hasOwner {
		t.Fatal("the owner overlay must stay active beside a machine scenario (F2)")
	}
	if len(final.Scenarios) != kernel.PlanHardMaxScenarios+1 || final.Scenarios[len(final.Scenarios)-1].ID != "P1" {
		t.Fatalf("P1 appended last, never counted against the cap: %d scenarios", len(final.Scenarios))
	}
	c := got.ComposedOf
	if c == nil || len(c.UserOverlays) != 1 || c.UserOverlays[0] != 1 || len(c.Machine) != 1 ||
		c.Machine[0].OverlayVersion != 2 || c.Machine[0].ScenarioID != "P1" || c.Machine[0].Ref != w5aRef {
		t.Fatalf("composed_of = %+v", c)
	}
	if len(got.OverlayErrors) != 0 || got.OverlayCount != 2 {
		t.Fatalf("overlay_errors %v overlay_count %d", got.OverlayErrors, got.OverlayCount)
	}
}

func TestPlanTodaySaysMachinePlan(t *testing.T) {
	s, tok := newPicturePlanGateServer(t, `{"day_plan":{"plan_enabled":true}}`)
	mp := kernel.PlanDoc{Reasoning: "MACHINE-AUTHORED (Picture HTF rule v1) — no AI plan existed for NY at 10:00 CT; superseded by the first AI plan",
		Bias: kernel.PlanBias{Direction: "neutral"}, Levels: []kernel.PlanLevel{}, NoTrade: []string{},
		Scenarios: []kernel.PlanScenario{w5aMachineScenario("P1", w5aRef)}, DeathCondition: "machine plan: ends when an AI plan is written"}
	blob, _ := json.Marshal(mp)
	w5aSeedPlan(t, s.store, w5aNYDate(t, time.Now()), string(blob), kernel.MachinePlanTriggerPicture)
	got := w5aGetToday(t, s, tok)
	if got.MachinePlan == nil || !*got.MachinePlan {
		t.Fatalf("a machine plan row reads machine_plan=true, got %v", got.MachinePlan)
	}
	if got.ComposedOf == nil || len(got.ComposedOf.UserOverlays) != 0 || len(got.ComposedOf.Machine) != 0 {
		t.Fatalf("no overlays → composed_of lists are empty: %+v", got.ComposedOf)
	}
}

// ── D18 — the one door refuses an edit that touches a machine scenario ─────

// w5aDoorNow is Monday 2026-09-14 10:00 CT (NY live) — applyPlanOverlay's clock.
var w5aDoorNow = time.Date(2026, 9, 14, 10, 0, 0, 0, time.FixedZone("CDT", -5*3600))

func w5aPatch(ops ...string) string { return "[" + strings.Join(ops, ",") + "]" }

func TestApplyPlanOverlayRefusesEditsThatTouchAMachineScenario(t *testing.T) {
	p2, _ := json.Marshal(w5aMachineScenario("P2", w5aRef+"-2"))
	s1, _ := json.Marshal(kernel.PlanScenario{ID: "S1", Trigger: "sweep 15480 reclaim", Condition: "reclaim", Direction: "long",
		TargetChain: []float64{15620}, Invalid: "2x5m<15470", Quality: "B"})
	t.Run("machine overlay on an AI plan", func(t *testing.T) {
		s, _ := newPicturePlanGateServer(t, `{"day_plan":{"plan_enabled":true}}`)
		row := w5aSeedPlan(t, s.store, "2026-09-14", w5aPlanDoc, "NY_scheduled_read")
		w5aMachineOverlay(t, s.store, row, w5aMachineScenario("P1", w5aRef))
		for _, origin := range []string{"owner", "planner-revised"} {
			_, _, code, msg := s.applyPlanOverlay(ppgTrader, "MNQ", w5aPatch(`{"op":"add","path":"/scenarios/-","value":`+string(p2)+`}`), origin, w5aDoorNow)
			if code != 409 || !strings.Contains(msg, "may not add a machine scenario") {
				t.Fatalf("%s adding a machine scenario must be refused 409, got %d %q", origin, code, msg)
			}
		}
		// The owner's index can never reach P1: it is not in the doc a user
		// patch applies to (index 1 does not exist there).
		if _, _, code, _ := s.applyPlanOverlay(ppgTrader, "MNQ", w5aPatch(`{"op":"replace","path":"/scenarios/1/quality","value":"A"}`), "owner", w5aDoorNow); code != 409 {
			t.Fatalf("an index past the user doc must be refused, got %d", code)
		}
		// A whole-array replace by the owner is legal — and P1 stays in plan_final.
		if _, _, code, msg := s.applyPlanOverlay(ppgTrader, "MNQ", w5aPatch(`{"op":"replace","path":"/scenarios","value":[`+string(s1)+`]}`), "owner", w5aDoorNow); code != 0 {
			t.Fatalf("an owner replace of /scenarios is legal: %d %q", code, msg)
		}
		final, _ := s.resolvePlanFinal(row)
		if sc, ok := kernel.MachineScenarioByRef(final, w5aRef); !ok || sc.ID != "P1" || final.Scenarios[0].Condition != "reclaim" {
			t.Fatalf("after the owner's replace, plan_final = owner S1 + P1: %+v", final.Scenarios)
		}
		ovs, _ := s.store.Plan().ListOverlays(row.PlanID, row.Version)
		if len(ovs) != 2 {
			t.Fatalf("exactly the accepted owner overlay was appended beside the machine one: %d rows", len(ovs))
		}
	})
	t.Run("machine plan", func(t *testing.T) {
		s, _ := newPicturePlanGateServer(t, `{"day_plan":{"plan_enabled":true}}`)
		mp := kernel.PlanDoc{Reasoning: "MACHINE-AUTHORED", Bias: kernel.PlanBias{Direction: "neutral"}, Levels: []kernel.PlanLevel{}, NoTrade: []string{},
			Scenarios: []kernel.PlanScenario{w5aMachineScenario("P1", w5aRef)}, DeathCondition: "machine plan: ends when an AI plan is written"}
		blob, _ := json.Marshal(mp)
		w5aSeedPlan(t, s.store, "2026-09-14", string(blob), kernel.MachinePlanTriggerPicture)
		for name, c := range map[string]struct{ patch, origin, want string }{
			"owner alters P1":           {w5aPatch(`{"op":"replace","path":"/scenarios/0/quality","value":"A"}`), "owner", "may not alter machine scenario P1"},
			"planner-revised alters P1": {w5aPatch(`{"op":"replace","path":"/scenarios/0/machine/eligible_until_ms","value":1790099999999}`), "planner-revised", "may not alter machine scenario P1"},
			"owner replaces /scenarios": {w5aPatch(`{"op":"replace","path":"/scenarios","value":[` + string(s1) + `]}`), "owner", "may not remove machine scenario P1"},
			"planner-revised adds a P2": {w5aPatch(`{"op":"add","path":"/scenarios/-","value":` + string(p2) + `}`), "planner-revised", "may not add a machine scenario"},
		} {
			_, _, code, msg := s.applyPlanOverlay(ppgTrader, "MNQ", c.patch, c.origin, w5aDoorNow)
			if code != 409 || !strings.Contains(msg, c.want) {
				t.Fatalf("%s: want 409 %q, got %d %q", name, c.want, code, msg)
			}
		}
		// An edit that leaves P1 alone passes.
		if _, _, code, msg := s.applyPlanOverlay(ppgTrader, "MNQ", w5aPatch(`{"op":"add","path":"/levels/-","value":{"price":21520,"label":"OWN","grade":"A","instruction":"fade"}}`), "owner", w5aDoorNow); code != 0 {
			t.Fatalf("an edit that leaves the machine scenario alone is legal: %d %q", code, msg)
		}
	})
}

// ── F15 — every trade names its plan and its source ────────────────────────

func TestPlanTradesNamesPlanSessionAndSource(t *testing.T) {
	s, tok := newPicturePlanGateServer(t, `{"day_plan":{"plan_enabled":true}}`)
	st := s.store
	pid := store.MakePlanIDForTrader(ppgTrader, "2026-09-14", "NY")
	arm := func(sig, source, ref string) {
		a := &store.ArmedOrderDB{TraderID: ppgTrader, PlanID: pid, Version: 1, Session: "NY", Scenario: "S" + sig, Side: "LONG", Kind: "limit",
			EntryPx: 21530, StopPx: 21510, TargetPx: 21590, State: store.StateArmed, Source: source, SourceRef: ref}
		if err := st.ArmedOrders().UpsertArm(a); err != nil {
			t.Fatal(err)
		}
		if err := st.ArmedOrders().BeginPlacement(a.ID, sig); err != nil {
			t.Fatal(err)
		}
	}
	arm("sig-pic", store.ArmSourcePicture, w5aRef)
	arm("sig-arm", "", "")
	pos := func(entryOrder string, exit int64) {
		p := &store.TraderPosition{TraderID: ppgTrader, Symbol: "MNQ", Side: "LONG", Quantity: 1, EntryPrice: 21530, EntryOrderID: entryOrder, EntryTime: exit - 60_000}
		if err := st.Position().Create(p); err != nil {
			t.Fatal(err)
		}
		if err := st.Position().SetPlanLinkFull(p.ID, 1, "P1", true, "", pid, "2026-09-14", "NY"); err != nil {
			t.Fatal(err)
		}
		if err := st.Position().SetAdherence(p.ID, "A"); err != nil {
			t.Fatal(err)
		}
		if _, err := st.Position().ClosePosition(p.ID, 21590, "x-"+entryOrder, 120, 0, "tp"); err != nil {
			t.Fatal(err)
		}
	}
	pos("sig-pic", 1790003000000)
	pos("sig-arm", 1790003100000)
	pos("sig-none", 1790003200000)
	rec, body := olDo(t, s, tok, http.MethodGet, "/api/plan/trades?trader_id="+ppgTrader, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("plan/trades: %d %s", rec.Code, rec.Body.String())
	}
	trades, _ := body["trades"].([]any)
	if len(trades) != 3 {
		t.Fatalf("3 graded trades expected: %s", rec.Body.String())
	}
	bySource := map[string]map[string]any{}
	for _, x := range trades {
		m := x.(map[string]any)
		if m["plan_id"] != pid || m["plan_session"] != "NY" {
			t.Fatalf("every trade names its plan and session: %+v", m)
		}
		bySource[fmt.Sprint(m["source"])] = m
	}
	if p := bySource["picture"]; p == nil || p["source_ref"] != w5aRef {
		t.Fatalf("the Picture trade is joined to its armed row: %+v", bySource)
	}
	if a := bySource["armed_entry"]; a == nil || a["source_ref"] != nil {
		t.Fatalf("a planner arm reads armed_entry with no source_ref: %+v", bySource)
	}
	if bySource["ai"] == nil {
		t.Fatalf("a position with no armed row (the decision path) reads ai: %+v", bySource)
	}
}
