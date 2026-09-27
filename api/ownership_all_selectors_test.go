package api

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"nofx/auth"
	"nofx/manager"
	"nofx/store"
)

// F16 (WAVE 117 PR-D, ports #117 576bd75b) — the C1 ownership middleware must
// check EVERY trader selector an authenticated request can carry, not only the
// query trader_id on /api/plan/ and /api/risk/. Today it reads ONLY
// ?trader_id (query) and only under those two prefixes: a second selector
// (a body trader_id on PATCH/DELETE, a conflicting body beside an owned query,
// or the /api/traders/:id path id) — and a trader_id on any other protected
// route (desk, accounts, ai-costs, audit) — names another owner's trader and
// sails through.
//
// The RED runs the PRODUCTION router (setupRoutes + authMiddleware + the
// ownership middleware), not a copy of it: each row below must end in the
// ownership refusal ("Trader not found", 404 — never 403, never the data).
func TestOwnershipMiddlewareChecksEverySelectorAtTheProductionRouter(t *testing.T) {
	st, err := store.New(filepath.Join(t.TempDir(), "f16.db"))
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}
	for _, row := range []store.Trader{
		{ID: "own", UserID: "reader", Name: "Own", AIModelID: "m", ExchangeID: "e"},
		{ID: "foreign", UserID: "owner", Name: "Foreign", AIModelID: "m", ExchangeID: "e"},
	} {
		if err := st.Trader().Create(&row); err != nil {
			t.Fatalf("create trader %s: %v", row.ID, err)
		}
	}
	if err := st.Equity().Save(&store.EquitySnapshot{TraderID: "foreign", TotalEquity: 123}); err != nil {
		t.Fatalf("seed foreign equity: %v", err)
	}
	seedTokenOwner(t, st, "reader", "reader@example.invalid")
	seedTokenOwner(t, st, "owner", "owner@example.invalid")
	previousSecret := append([]byte(nil), auth.JWTSecret...)
	auth.SetJWTSecret("offline-f16-ownership-test")
	t.Cleanup(func() { auth.JWTSecret = previousSecret })
	tok, err := auth.GenerateJWT("reader", "reader@example.invalid")
	if err != nil {
		t.Fatalf("jwt: %v", err)
	}
	live := NewServer(manager.NewTraderManager(), st, nil, "127.0.0.1", 0)

	for _, tc := range []struct{ method, url, body string }{
		// trader_id on protected routes the middleware does not gate today
		{"GET", "/api/config/resolved?trader_id=foreign", ""},
		{"GET", "/api/desk?trader_id=foreign", ""},
		{"GET", "/api/accounts?trader_id=foreign", ""},
		{"GET", "/api/ai-costs?trader_id=foreign", ""},
		{"GET", "/api/audit/decisions?trader_id=foreign", ""},
		// the path :id selector
		{"GET", "/api/traders/foreign/grid-risk", ""},
		{"DELETE", "/api/traders/foreign", ""},
		// a query selector the user owns beside a body selector they do not
		{"POST", "/api/account/select?trader_id=foreign", `{"account":"Sim101"}`},
		{"POST", "/api/plan/overlay?trader_id=own", `{"trader_id":"foreign"}`},
	} {
		t.Run(tc.method+" "+tc.url, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.url, strings.NewReader(tc.body))
			req.Header.Set("Authorization", "Bearer "+tok)
			if tc.body != "" {
				req.Header.Set("Content-Type", "application/json")
			}
			rec := httptest.NewRecorder()
			live.router.ServeHTTP(rec, req)
			if rec.Code != http.StatusNotFound {
				t.Fatalf("expected ownership 404, got %d: %s", rec.Code, rec.Body.String())
			}
			if !strings.Contains(rec.Body.String(), "Trader not found") {
				t.Fatalf("expected the ownership refusal, got %d: %s", rec.Code, rec.Body.String())
			}
		})
	}

	// Positive control: the user's OWN trader stays readable.
	req := httptest.NewRequest("GET", "/api/config/resolved?trader_id=own", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	rec := httptest.NewRecorder()
	live.router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("own trader read must remain available: %d: %s", rec.Code, rec.Body.String())
	}

	// The refused requests must have changed NOTHING about the foreign trader.
	var count int64
	if err := st.GormDB().Model(&store.EquitySnapshot{}).Where("trader_id = ?", "foreign").Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("foreign equity modified: count=%d", count)
	}
	if row, err := st.Trader().GetByID("foreign"); err != nil || row == nil {
		t.Fatalf("foreign trader modified: row=%v err=%v", row, err)
	}
}
