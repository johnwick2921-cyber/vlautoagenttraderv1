package trader

// DEMO VERIFY — proves what the dashboard card will actually show, without a
// JWT (minting one is blocked by policy in this sandbox). It replays the exact
// data path GET /api/plan/today runs — ActiveSession → GetLatestPlanForSession
// → overlay fold → ValidatePlanDoc armor — against the LIVE db, read-only, at a
// simulated Sunday 10:00 CT (inside the NY window).
//
// Guarded by NOFX_DEMO_VERIFY=1. Reads only; writes nothing.

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"nofx/kernel"
	"nofx/store"
)

func TestDemoVerify(t *testing.T) {
	if os.Getenv("NOFX_DEMO_VERIFY") != "1" {
		t.Skip("set NOFX_DEMO_VERIFY=1")
	}
	dbPath := os.Getenv("NOFX_DEMO_DB")
	if dbPath == "" {
		dbPath = filepath.Join("..", "data", "data.db")
	}
	st, err := store.New(dbPath)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { st.Plan().Close(); _ = st.Close() })

	ct, _ := time.LoadLocation("America/Chicago")
	when := time.Date(2026, 8, 16, 10, 0, 0, 0, ct) // Sunday, inside NY 08:30–15:00

	// 1. the gate the card hits first
	reg := kernel.DefaultSessionRegistry()
	sess, ok := reg.ActiveSession(when)
	if !ok || !sess.Enabled {
		t.Fatalf("card would render found:false at %s (active=%v)", when.Format(time.Kitchen), ok)
	}
	t.Logf("✓ active session at Sun 10:00 CT = %s (enabled=%v) → card renders", sess.Name, sess.Enabled)
	t.Logf("  night mode at this hour: %v", reg.IsNightMode(when))

	// 2. the row the card fetches
	tradeDate := when.Format("2006-01-02")
	row, err := st.Plan().GetLatestPlanForSession(tradeDate, sess.Name)
	if err != nil || row == nil {
		t.Fatalf("no plan at (%s,%s): %v", tradeDate, sess.Name, err)
	}
	t.Logf("✓ plan row v%d lifecycle=%s trigger=%s", row.Version, row.Lifecycle, row.TriggerReason)

	// 3. overlay fold + armor — exactly what the handler does
	var doc kernel.PlanDoc
	if err := json.Unmarshal([]byte(row.Doc), &doc); err != nil {
		t.Fatalf("base doc unparseable: %v", err)
	}
	overlays, _ := st.Plan().ListOverlays(row.PlanID, row.Version)
	if len(overlays) > 0 {
		patches := make([]string, 0, len(overlays))
		for _, ov := range overlays {
			patches = append(patches, ov.Patch)
		}
		base, _ := json.Marshal(doc)
		final, _ := kernel.ApplyOverlayPatches(base, patches)
		var merged kernel.PlanDoc
		if json.Unmarshal(final, &merged) == nil && kernel.ValidatePlanDoc(&merged) == nil {
			doc = merged
			t.Logf("✓ %d overlay(s) folded into plan_final and re-armored", len(overlays))
		} else {
			t.Errorf("overlay fold failed armor — card would show the BASE doc")
		}
	}

	// 4. what the owner will see
	t.Logf("BIAS      %s (%s) · flip: %s", doc.Bias.Direction, doc.Bias.Conviction, doc.Bias.FlipCondition)
	t.Logf("DAY TYPE  %s", doc.DayType)
	for _, l := range doc.Levels {
		t.Logf("LEVEL     %-9.2f [%s] %-18s %s", l.Price, l.Grade, l.Label, l.Instruction)
	}
	for _, s := range doc.Scenarios {
		t.Logf("SCENARIO  %s [%s] %-16s %-5s → %v", s.ID, s.Quality, s.Condition, s.Direction, s.TargetChain)
	}
	for _, nt := range doc.NoTrade {
		t.Logf("NO-TRADE  %s", nt)
	}
	t.Logf("DIES IF   %s", doc.DeathCondition)

	// 5. ZoneTable rows come from level_facts, which the SERVER computes from the
	// live BarCache. This test binary has no provider bound, so the check that
	// matters is done against the running bot's own /api/klines (see the report).

	// 6. Monday must still be untouched
	if monday, _ := st.Plan().GetLatestPlanForSession("2026-08-17", "NY"); monday != nil {
		t.Errorf("✗ Monday 2026-08-17:NY EXISTS (v%d) — must be empty", monday.Version)
	} else {
		t.Log("✓ Monday 2026-08-17:NY is EMPTY — the real path is untouched")
	}
}
