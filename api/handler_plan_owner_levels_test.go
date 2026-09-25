// W-OWNER-LEVELS-API (2026-09-17) — GET /api/plan/owner-levels at the
// PRODUCTION call site: the real router (NewServer → setupRoutes), the real
// JWT + ownership middleware, a real sqlite store, and a trader loaded through
// TraderManager.LoadUserTradersFromStore. Nothing is stubbed; the POST and
// DELETE legs go through their own routes so the list reflects what those
// handlers actually wrote.

package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"nofx/auth"
	"nofx/kernel"
	"nofx/manager"
	"nofx/store"
)

const (
	olTestUser   = "u-owner-levels"
	olTestTrader = "t-owner-levels"
)

// newOwnerLevelsServer builds a full Server over a temp store with ONE loaded
// trader owned by olTestUser, and returns a Bearer token for that user.
func newOwnerLevelsServer(t *testing.T) (*Server, string) {
	t.Helper()
	st, err := store.New(filepath.Join(t.TempDir(), "ol.db"))
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	t.Cleanup(func() { st.Plan().Close(); _ = st.Close() })

	if err := st.AIModel().Create(olTestUser, "m-ol", "m", "deepseek", true, "sk-test-not-a-real-key", ""); err != nil {
		t.Fatalf("ai model: %v", err)
	}
	// A binance CEX row: its trader constructs without any network or NT8
	// TCP side effect, which is all the manager needs to seat the trader.
	exID, err := st.Exchange().Create(olTestUser, "binance", "Default", true,
		"test-key", "test-secret", "", false, "", true, "", "", "", "", "", "", 0, "", "", 0)
	if err != nil {
		t.Fatalf("exchange: %v", err)
	}
	if err := st.Strategy().Create(&store.Strategy{ID: "s-ol", UserID: olTestUser, Name: "s", Config: "{}"}); err != nil {
		t.Fatalf("strategy: %v", err)
	}
	if err := st.Trader().Create(&store.Trader{ID: olTestTrader, UserID: olTestUser, Name: "t", AIModelID: "m-ol", ExchangeID: exID, StrategyID: "s-ol", InitialBalance: 1000}); err != nil {
		t.Fatalf("trader: %v", err)
	}
	tm := manager.NewTraderManager()
	if err := tm.LoadUserTradersFromStore(st, olTestUser); err != nil {
		t.Fatalf("load traders: %v", err)
	}
	if _, err := tm.GetTrader(olTestTrader); err != nil {
		t.Fatalf("trader not loaded into the manager: %v (load error: %v)", err, tm.GetLoadError(olTestTrader))
	}

	auth.SetJWTSecret("owner-levels-test-secret")
	seedTokenOwner(t, st, olTestUser, "ol@test")    // M3 H2: a token needs its account row
	seedTokenOwner(t, st, "someone-else", "x@test") // the foreign-trader probe's user
	tok, err := auth.GenerateJWT(olTestUser, "ol@test")
	if err != nil {
		t.Fatalf("jwt: %v", err)
	}
	return NewServer(tm, st, nil, "127.0.0.1", 0), tok
}

func olDo(t *testing.T, s *Server, tok, method, url, body string) (*httptest.ResponseRecorder, map[string]any) {
	t.Helper()
	var rdr *strings.Reader
	if body != "" {
		rdr = strings.NewReader(body)
	} else {
		rdr = strings.NewReader("")
	}
	req := httptest.NewRequest(method, url, rdr)
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if tok != "" {
		req.Header.Set("Authorization", "Bearer "+tok)
	}
	rec := httptest.NewRecorder()
	s.router.ServeHTTP(rec, req)
	out := map[string]any{}
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	return rec, out
}

func TestOwnerLevelsListRequiresJWT(t *testing.T) {
	s, _ := newOwnerLevelsServer(t)
	rec, _ := olDo(t, s, "", http.MethodGet, "/api/plan/owner-levels?trader_id="+olTestTrader, "")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("no JWT must be 401, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestOwnerLevelsListRequiresTraderID(t *testing.T) {
	s, tok := newOwnerLevelsServer(t)
	rec, _ := olDo(t, s, tok, http.MethodGet, "/api/plan/owner-levels", "")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("missing trader_id must be 400, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestOwnerLevelsListForeignTraderIs404(t *testing.T) {
	s, _ := newOwnerLevelsServer(t)
	other, err := auth.GenerateJWT("someone-else", "x@test")
	if err != nil {
		t.Fatal(err)
	}
	rec, _ := olDo(t, s, other, http.MethodGet, "/api/plan/owner-levels?trader_id="+olTestTrader, "")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("another user's trader must be 404 (never 403), got %d body=%s", rec.Code, rec.Body.String())
	}
}

// The core contract: empty → [] + count 0 (never null); one POSTed level →
// listed as pending with the id the POST returned; after the DELETE route →
// empty again. Every leg rides its production route.
func TestOwnerLevelsListLifecycle(t *testing.T) {
	s, tok := newOwnerLevelsServer(t)
	listURL := "/api/plan/owner-levels?trader_id=" + olTestTrader

	// 1. Empty list is `[]`, never null, count 0.
	rec, _ := olDo(t, s, tok, http.MethodGet, listURL, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("empty list: want 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	raw := rec.Body.String()
	if !strings.Contains(raw, `"levels":[]`) {
		t.Fatalf("empty list must serialize as \"levels\":[] (never null): %s", raw)
	}
	if !strings.Contains(raw, `"count":0`) {
		t.Fatalf("empty list must carry count:0: %s", raw)
	}
	if !strings.Contains(raw, `"judged_against":null`) {
		t.Fatalf("no plan row → judged_against must be null, not fabricated: %s", raw)
	}

	// 2. POST one level through the existing route.
	rec, posted := olDo(t, s, tok, http.MethodPost, "/api/plan/owner-level",
		`{"trader_id":"`+olTestTrader+`","symbol":"MNQ","price":30150.25,"label":"👤","note":"watch","scenario_tag":"S1"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("POST owner-level: want 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	id, _ := posted["id"].(float64)
	if id <= 0 {
		t.Fatalf("POST must return a positive id: %v", posted)
	}

	// 3. Listed as pending, carrying the POST's id and fields.
	rec, out := olDo(t, s, tok, http.MethodGet, listURL, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("list after POST: want 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	levels, _ := out["levels"].([]any)
	if len(levels) != 1 || out["count"].(float64) != 1 {
		t.Fatalf("want exactly one level, got %v", out)
	}
	row := levels[0].(map[string]any)
	if row["id"].(float64) != id {
		t.Fatalf("listed id %v must equal the POSTed id %v", row["id"], id)
	}
	if row["status"] != "pending" {
		t.Fatalf("a level no plan carries must be pending, got %v", row["status"])
	}
	if row["price"].(float64) != 30150.25 || row["label"] != "👤" || row["note"] != "watch" || row["scenario_tag"] != "S1" || row["symbol"] != "MNQ" {
		t.Fatalf("row fields must round-trip the POST: %v", row)
	}
	if row["consumed"] != false {
		t.Fatalf("an active row is consumed=false, got %v", row["consumed"])
	}
	if ca, _ := row["created_at"].(float64); ca <= 0 {
		t.Fatalf("created_at must be the POST's unix stamp, got %v", row["created_at"])
	}

	// 4. A different symbol lists nothing — the symbol filter is real.
	rec, out = olDo(t, s, tok, http.MethodGet, listURL+"&symbol=ES", "")
	if rec.Code != http.StatusOK || out["count"].(float64) != 0 {
		t.Fatalf("symbol=ES must list nothing, got %d %v", rec.Code, out)
	}

	// 5. DELETE through the existing route → empty again.
	rec, _ = olDo(t, s, tok, http.MethodPost, "/api/plan/owner-level/delete",
		`{"trader_id":"`+olTestTrader+`","id":`+strconv.FormatInt(int64(id), 10)+`}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("DELETE owner-level: want 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	rec, out = olDo(t, s, tok, http.MethodGet, listURL, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("list after delete: want 200, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"levels":[]`) || out["count"].(float64) != 0 {
		t.Fatalf("after delete the list must be [] / count 0, got %s", rec.Body.String())
	}
}

// status is judged against the card's plan for the requested session: a
// level the plan_final carries at the same tick reads "applied", one it does
// not reads "pending", and judged_against names the plan the verdict came
// from. Pinned to ?session=NY so the wall clock cannot move the referent.
func TestOwnerLevelsStatusAppliedAgainstSessionPlan(t *testing.T) {
	s, tok := newOwnerLevelsServer(t)
	reg := kernel.DefaultSessionRegistry()
	ny, ok := reg.SessionByName("NY")
	if !ok {
		t.Fatal("default registry has no NY session")
	}
	now := time.Now()
	tradeDate := now.In(planChicago()).Format("2006-01-02")
	if d, okD := kernel.PlanChainTradeDate(ny, now); okD {
		tradeDate = d
	}
	planID := store.MakePlanIDForTrader(olTestTrader, tradeDate, "NY")
	ver, err := s.store.Plan().AppendPlan(&store.PlanDB{
		PlanID: planID, StrategyID: olTestTrader, TradeDate: tradeDate, Session: "NY",
		Doc: `{"levels":[{"price":30150.25,"label":"PDH","grade":"A","instruction":"monitor"}]}`,
	})
	if err != nil || ver != 1 {
		t.Fatalf("seed plan: v=%d err=%v", ver, err)
	}

	for _, body := range []string{
		`{"trader_id":"` + olTestTrader + `","symbol":"MNQ","price":30150.25,"label":"in-plan"}`,
		`{"trader_id":"` + olTestTrader + `","symbol":"MNQ","price":30200,"label":"not-yet"}`,
	} {
		if rec, _ := olDo(t, s, tok, http.MethodPost, "/api/plan/owner-level", body); rec.Code != http.StatusOK {
			t.Fatalf("POST %s: %d %s", body, rec.Code, rec.Body.String())
		}
	}

	rec, out := olDo(t, s, tok, http.MethodGet, "/api/plan/owner-levels?trader_id="+olTestTrader+"&session=NY", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("list: %d %s", rec.Code, rec.Body.String())
	}
	judged, _ := out["judged_against"].(map[string]any)
	if judged == nil || judged["plan_id"] != planID || judged["version"].(float64) != 1 || judged["session"] != "NY" || judged["trade_date"] != tradeDate {
		t.Fatalf("judged_against must name the seeded plan, got %v", out["judged_against"])
	}
	levels, _ := out["levels"].([]any)
	if len(levels) != 2 {
		t.Fatalf("want 2 levels, got %v", out)
	}
	got := map[string]string{}
	for _, l := range levels {
		row := l.(map[string]any)
		got[row["label"].(string)] = row["status"].(string)
	}
	if got["in-plan"] != "applied" {
		t.Fatalf("a level the session plan carries must be applied, got %v", got)
	}
	if got["not-yet"] != "pending" {
		t.Fatalf("a level no plan carries must stay pending, got %v", got)
	}
	// The status is per-referent: a session with NO plan row judges nothing.
	rec, out = olDo(t, s, tok, http.MethodGet, "/api/plan/owner-levels?trader_id="+olTestTrader+"&session=ASIA", "")
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"judged_against":null`) {
		t.Fatalf("no ASIA plan → judged_against null, got %d %s", rec.Code, rec.Body.String())
	}
	for _, l := range out["levels"].([]any) {
		if l.(map[string]any)["status"] != "pending" {
			t.Fatalf("with no referent every row is pending, got %v", out["levels"])
		}
	}
}
