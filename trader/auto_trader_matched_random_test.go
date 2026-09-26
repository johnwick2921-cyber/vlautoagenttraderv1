package trader

import (
	"testing"
	"time"

	"nofx/kernel"
	"nofx/store"
)

// TestRecordMatchedRandomFoldsOverlay (WAVE 1a-plan P2) — the level-type
// attribution for a closed trade reads the plan through the ONE fold: an owner
// overlay adding a level AT the entry price must be the attributed level.
// RED on the base-only reader: the tally lands on the base PDL type. GREEN:
// the overlay level's type.
func TestRecordMatchedRandomFoldsOverlay(t *testing.T) {
	at, st := resetTrader(t, store.StrategyConfig{DayPlan: &store.DayPlanConfig{PlanEnabled: true}})
	entryT := time.Date(2026, 8, 25, 10, 0, 0, 0, kernel.CTLocation())
	sess, ok := at.sessionRegistry(time.Now()).ActiveSession(entryT)
	if !ok {
		t.Fatalf("proving line: 10:00 CT must be inside an active session (NY)")
	}
	tradeDate := sessionChainDate(sess, entryT)
	pid := tradeDate + ":" + sess.Name
	base := `{"reasoning":"wave-1a-p2-mr","bias":{"direction":"neutral"},"death_condition":"flat","levels":[{"price":29100,"label":"PDL","grade":"A"}],"scenarios":[{"id":"S1","condition":"reject","direction":"long","quality":"A"}]}`
	if _, err := st.Plan().AppendPlan(&store.PlanDB{PlanID: pid, StrategyID: at.id, TradeDate: tradeDate, Session: sess.Name, Lifecycle: "active", Doc: base}); err != nil {
		t.Fatal(err)
	}
	// The owner adds an OB level at the entry price through an overlay.
	if _, err := st.Plan().AppendOverlay(&store.PlanOverlayDB{PlanID: pid, PlanVersion: 1, OverlayID: "owner-add-ob", Origin: "owner",
		Patch: `[{"op":"add","path":"/levels/-","value":{"price":29101,"label":"OB(bull)·1h","grade":"B"}}]`}); err != nil {
		t.Fatal(err)
	}

	p := &store.TraderPosition{TraderID: at.id, Symbol: "MNQ", Side: "LONG", EntryPrice: 29101, EntryTime: entryT.UnixMilli()}
	at.recordMatchedRandomForClose(p, kernel.Excursion{MFE: 5, MAE: 1})
	tally, err := st.MatchedRandom().CountsByType()
	if err != nil {
		t.Fatal(err)
	}
	obType := kernel.LevelTypeFromLabel("OB(bull)·1h")
	if obType == kernel.LevelTypeFromLabel("PDL") {
		t.Fatalf("proving line: the two labels must map to different types")
	}
	if n := tally[obType].Touches; n != 1 {
		t.Fatalf("the folded overlay level must own the attribution (%s touches=%d, want 1); tally=%+v", obType, n, tally)
	}
}
