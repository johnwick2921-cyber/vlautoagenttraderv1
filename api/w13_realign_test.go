package api

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"nofx/kernel"
	"nofx/store"

	"github.com/gin-gonic/gin"
)

// W13 — the auto-trigger gate. Debounce is what makes a BULK-ADD batch one call;
// the cap hands the owner the manual button; manual bypasses both.
func TestW13RealignGate(t *testing.T) {
	now := int64(1_800_000_000)
	cases := []struct {
		name   string
		manual bool
		used   int
		capN   int
		lastAt int64
		want   string
	}{
		{"first auto call proceeds", false, 0, 5, -1, ""},
		{"bulk-add: 2nd row inside the window is debounced", false, 1, 5, now - 2, "debounced"},
		{"rapid re-save at 19s still debounced", false, 1, 5, now - 19, "debounced"},
		{"after the window it proceeds", false, 1, 5, now - 21, ""},
		{"cap reached → capped (manual fallback)", false, 5, 5, -1, "capped"},
		{"over cap → capped", false, 9, 5, -1, "capped"},
		{"MANUAL bypasses the debounce", true, 1, 5, now - 1, ""},
		{"MANUAL bypasses the cap", true, 99, 5, -1, ""},
		{"cap 0 = unlimited", false, 99, 0, -1, ""},
	}
	for _, c := range cases {
		if got := realignGateDecision(c.manual, c.used, c.capN, c.lastAt, now); got != c.want {
			t.Fatalf("%s: got %q want %q", c.name, got, c.want)
		}
	}
}

// W13 — trader_id is required (mirrors the rest of /api/plan/*).
func TestW13RealignRequiresTraderID(t *testing.T) {
	s := &Server{}
	r := gin.New()
	r.POST("/api/plan/realign", s.handlePlanRealign)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/plan/realign", strings.NewReader(`{}`)))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", rec.Code, rec.Body.String())
	}
}

// W13 — the KPI row: an auto re-align lands in the SAME plan_qa table as
// Ask-Planner, carrying trigger/challenge_type/cost/latency, and the cap + debounce
// queries read it back. This is what keeps the sycophancy stats one series.
func TestW13KPIRowAndGuardrailQueries(t *testing.T) {
	st, err := store.New(filepath.Join(t.TempDir(), "w13.db"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })

	const trader, plan = "t1", "2026-08-16:NY"
	now := time.Now().Unix()

	// an Ask-Planner reply (no trigger) must NOT count against the auto cap
	_, _ = st.PlanQA().Append(&store.PlanQADB{
		TraderID: trader, PlanID: plan, Role: "planner", Verdict: "DEFEND",
		PointClass: "BARE-DISAGREEMENT", CreatedAt: now - 100,
	})
	// two auto re-aligns
	for i := 0; i < 2; i++ {
		if _, err := st.PlanQA().Append(&store.PlanQADB{
			TraderID: trader, PlanID: plan, Role: "planner",
			Verdict: "PROPOSE-MERGE", PointClass: "NEW-INFO",
			Patch:   `[{"op":"replace","path":"/bias/direction","value":"short"}]`,
			Trigger: kernel.TriggerOwnerEdit, ChallengeType: "add-level",
			CostUSD: 0.0021, LatencyMs: 4200, CreatedAt: now - int64(10-i),
		}); err != nil {
			t.Fatalf("append: %v", err)
		}
	}

	used, err := st.PlanQA().CountPlannerByTrigger(trader, plan, kernel.TriggerOwnerEdit)
	if err != nil || used != 2 {
		t.Fatalf("auto count = %d (err %v), want 2 — Ask-Planner rows must not count", used, err)
	}
	last, err := st.PlanQA().LastPlannerByTrigger(trader, plan, kernel.TriggerOwnerEdit)
	if err != nil || last == nil {
		t.Fatalf("last auto row missing: %v", err)
	}
	if last.ChallengeType != "add-level" || last.CostUSD <= 0 || last.LatencyMs != 4200 {
		t.Fatalf("cost/latency/challenge not persisted: %+v", last)
	}
	if last.Trigger != kernel.TriggerOwnerEdit {
		t.Fatalf("trigger not persisted: %q", last.Trigger)
	}
	// unknown trigger → no rows, no error (a manual re-align hasn't happened yet)
	if n, err := st.PlanQA().CountPlannerByTrigger(trader, plan, kernel.TriggerManual); err != nil || n != 0 {
		t.Fatalf("manual count = %d (err %v), want 0", n, err)
	}

	// KPI series stays comparable: all three planner rows aggregate together
	stats, err := st.PlanQA().VerdictStats(trader)
	if err != nil || stats.Total != 3 || stats.ProposeMerge != 2 || stats.DefendOnBare != 1 {
		t.Fatalf("VerdictStats = %+v (err %v); want total 3, propose 2, defend-on-bare 1", stats, err)
	}
}

// W13 — fail-closed writes an alert row and never touches the plan.
func TestW13FailClosedWritesAlert(t *testing.T) {
	st, err := store.New(filepath.Join(t.TempDir(), "w13f.db"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	s := &Server{store: st}

	row := &store.PlanDB{PlanID: "2026-08-16:NY", Version: 1, Doc: "{}"}
	s.realignFailClosed("t1", row, "NY", "2026-08-16", "planner reply malformed")

	alerts, _ := st.Alert().List("t1", 10)
	if len(alerts) != 1 {
		t.Fatalf("expected 1 alert row, got %d", len(alerts))
	}
	if alerts[0].Kind != "realign-failed" || alerts[0].Level != "P1" {
		t.Fatalf("wrong alert: %+v", alerts[0])
	}
	// no plan/overlay was created by the failure path
	if ovs, _ := st.Plan().ListOverlays(row.PlanID, row.Version); len(ovs) != 0 {
		t.Fatalf("fail-closed must never write an overlay, got %d", len(ovs))
	}
}

// W13 — cost estimate is positive, scales with size, and stays small per call.
func TestW13CostEstimate(t *testing.T) {
	small := estimateRealignCostUSD(strings.Repeat("x", 4000), strings.Repeat("y", 400))
	big := estimateRealignCostUSD(strings.Repeat("x", 40000), strings.Repeat("y", 4000))
	if small <= 0 || big <= small {
		t.Fatalf("cost must be positive and scale: small=%v big=%v", small, big)
	}
	if big > 0.05 {
		t.Fatalf("a single re-align should cost cents, got $%.4f", big)
	}
}
