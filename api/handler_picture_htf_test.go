package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"nofx/store"

	"github.com/gin-gonic/gin"
)

// W-EXEC-TRUTH W5 (display) — GET /api/picture-htf/opportunities links a
// Picture opportunity to the Day Plan scenario that carries it: the
// armed_orders row whose source_ref is the opp_key. Driven through the real
// route (handlePictureHtfOpportunities), not the join helper.

const (
	picKeyPlanned  = "t1|acct|MNQ|long|resistance|1790186400000|1790190000000"
	picKeyNoArm    = "t1|acct|MNQ|short|support|1790186400000|1790193600000"
	picKeyLegacy   = "t1|acct|MNQ|long|resistance|1790100000000|1790103600000"
	picKeyTerminal = "t1|acct|MNQ|short|resistance|1790186400000|1790197200000"
)

func pictureHtfServe(t *testing.T, st *store.Store) map[string]json.RawMessage {
	t.Helper()
	router := gin.New()
	router.GET("/api/picture-htf/opportunities", (&Server{store: st}).handlePictureHtfOpportunities)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/picture-htf/opportunities?trader_id=t1", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}
	var body struct {
		Rows []map[string]json.RawMessage `json:"rows"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	out := map[string]json.RawMessage{}
	for _, r := range body.Rows {
		var key string
		if err := json.Unmarshal(r["opp_key"], &key); err != nil {
			t.Fatal(err)
		}
		out[key] = r["plan_link"]
	}
	if strings.Contains(rec.Body.String(), "plan_links_unread") {
		t.Fatalf("the arm ledger read was reported unread: %s", rec.Body.String())
	}
	return out
}

func TestPictureHtfOpportunitiesPlanLink(t *testing.T) {
	st, err := store.New(filepath.Join(t.TempDir(), "picture-link.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	opps := []store.PictureHtfOpportunityDB{
		{OppKey: picKeyPlanned, TraderID: "t1", Stage: "planned", StageReason: "Day Plan scenario P1", Direction: "long"},
		{OppKey: picKeyNoArm, TraderID: "t1", Stage: "planned", Direction: "short"},
		{OppKey: picKeyLegacy, TraderID: "t1", Stage: "filled", Direction: "long", SignalID: "picture-htf-1"},
		{OppKey: picKeyTerminal, TraderID: "t1", Stage: "planned", Direction: "short"},
	}
	if err := st.GormDB().Create(&opps).Error; err != nil {
		t.Fatal(err)
	}
	arms := []store.ArmedOrderDB{
		// picKeyPlanned: an OLDER live row and a NEWER terminal one — the live
		// row is the link (a live row keeps the opportunity's slot).
		{ID: 51, TraderID: "t1", PlanID: "t1:2026-09-23:NY", Scenario: "P1", Version: 3, PlacementSeq: 1, Kind: "limit", State: "working", SignalID: "arm-sig-51", Source: "picture", SourceRef: picKeyPlanned},
		{ID: 52, TraderID: "t1", PlanID: "t1:2026-09-23:NY", Scenario: "P1", Version: 2, PlacementSeq: 2, Kind: "limit", State: "cancelled", Source: "picture", SourceRef: picKeyPlanned},
		// Another trader's row on the same key never links.
		{ID: 53, TraderID: "t2", PlanID: "t2:2026-09-23:NY", Scenario: "P1", Version: 9, Kind: "limit", State: "armed", Source: "picture", SourceRef: picKeyNoArm},
		// A planner row carrying a stray ref is not a Picture row.
		{ID: 54, TraderID: "t1", PlanID: "t1:2026-09-23:NY", Scenario: "S1", Version: 3, Kind: "limit", State: "armed", SourceRef: picKeyNoArm},
		// picKeyTerminal: only terminal rows — the newest is the link.
		{ID: 55, TraderID: "t1", PlanID: "t1:2026-09-23:NY", Scenario: "P2", Version: 2, PlacementSeq: 1, Kind: "limit", State: "filled", SignalID: "arm-sig-55", Source: "picture", SourceRef: picKeyTerminal},
		{ID: 56, TraderID: "t1", PlanID: "t1:2026-09-23:NY", Scenario: "P2", Version: 3, PlacementSeq: 2, Kind: "limit", State: "expired", Source: "picture", SourceRef: picKeyTerminal},
	}
	if err := st.GormDB().Create(&arms).Error; err != nil {
		t.Fatal(err)
	}

	links := pictureHtfServe(t, st)
	if len(links) != len(opps) {
		t.Fatalf("rows: got %d want %d", len(links), len(opps))
	}

	var planned pictureHtfPlanLinkDTO
	if raw := links[picKeyPlanned]; raw == nil {
		t.Fatal("the planned opportunity with an armed row has no plan_link")
	} else if err := json.Unmarshal(raw, &planned); err != nil {
		t.Fatal(err)
	}
	if planned != (pictureHtfPlanLinkDTO{PlanID: "t1:2026-09-23:NY", ScenarioID: "P1", PlanVersion: 3, ArmRowID: 51, ArmState: "working", ArmSignalID: "arm-sig-51"}) {
		t.Fatalf("planned link: %+v", planned)
	}
	if !strings.Contains(string(links[picKeyPlanned]), `"arm_signal_id":"arm-sig-51"`) {
		t.Fatalf("link JSON: %s", links[picKeyPlanned])
	}

	var terminal pictureHtfPlanLinkDTO
	if err := json.Unmarshal(links[picKeyTerminal], &terminal); err != nil {
		t.Fatal(err)
	}
	if terminal.ArmRowID != 56 || terminal.ArmState != "expired" || terminal.PlanVersion != 3 {
		t.Fatalf("all-terminal link must be the newest row: %+v", terminal)
	}
	if strings.Contains(string(links[picKeyTerminal]), "arm_signal_id") {
		t.Fatalf("an unsent row fabricated a signal: %s", links[picKeyTerminal])
	}

	// Absent — not null, not {} — when no Picture row of this trader carries it.
	for _, k := range []string{picKeyNoArm, picKeyLegacy} {
		if raw := links[k]; raw != nil {
			t.Errorf("%s: plan_link must be absent, got %s", k, raw)
		}
	}
}

// A legacy ledger (no Picture row ever became a scenario) serves the rows
// exactly as before: no plan_link key anywhere, no plan_links_unread.
func TestPictureHtfOpportunitiesNoLinkIsAbsent(t *testing.T) {
	st, err := store.New(filepath.Join(t.TempDir(), "picture-legacy.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	if err := st.GormDB().Create(&store.PictureHtfOpportunityDB{OppKey: picKeyLegacy, TraderID: "t1", Stage: "filled"}).Error; err != nil {
		t.Fatal(err)
	}
	router := gin.New()
	router.GET("/x", (&Server{store: st}).handlePictureHtfOpportunities)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/x?trader_id=t1", nil))
	if rec.Code != http.StatusOK || strings.Contains(rec.Body.String(), "plan_link") {
		t.Fatalf("legacy response changed: %d %s", rec.Code, rec.Body.String())
	}
}

// A link read that FAILS is not "no link": the response says so in
// plan_links_unread, and the rows are still served (without plan_link).
func TestPictureHtfOpportunitiesLinkReadFailureIsSaid(t *testing.T) {
	st, err := store.New(filepath.Join(t.TempDir(), "picture-unread.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	if err := st.GormDB().Create(&store.PictureHtfOpportunityDB{OppKey: picKeyPlanned, TraderID: "t1", Stage: "planned"}).Error; err != nil {
		t.Fatal(err)
	}
	if err := st.GormDB().Exec("DROP TABLE armed_orders").Error; err != nil {
		t.Fatal(err)
	}
	router := gin.New()
	router.GET("/x", (&Server{store: st}).handlePictureHtfOpportunities)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/x?trader_id=t1", nil))
	var body struct {
		Rows            []map[string]json.RawMessage `json:"rows"`
		PlanLinksUnread string                       `json:"plan_links_unread"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if rec.Code != http.StatusOK || len(body.Rows) != 1 || !strings.HasPrefix(body.PlanLinksUnread, "arm ledger unavailable") {
		t.Fatalf("an unread link ledger must be said: %d %s", rec.Code, rec.Body.String())
	}
	if _, ok := body.Rows[0]["plan_link"]; ok {
		t.Fatalf("an unread ledger fabricated a link: %s", rec.Body.String())
	}
}
