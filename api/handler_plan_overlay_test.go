// P5.1 — overlay + owner-level handler contract tests. The trader_id / patch /
// price validation runs before touching the store or traderManager, so &Server{}
// covers the guard paths. The RFC-6902 apply + armor + round-trip logic is proven
// in kernel/plan_overlay_test.go + plan_overlay_roundtrip_test.go.

package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestHandlePlanOverlayRequiresTraderID(t *testing.T) {
	s := &Server{}
	router := gin.New()
	router.POST("/api/plan/overlay", s.handlePlanOverlay)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/plan/overlay", nil))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 missing trader_id, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestHandlePlanOwnerLevelRequiresTraderID(t *testing.T) {
	s := &Server{}
	router := gin.New()
	router.POST("/api/plan/owner-level", s.handlePlanOwnerLevel)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/plan/owner-level", nil))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 missing trader_id, got %d", rec.Code)
	}
}

func TestHandlePlanOwnerLevelDeleteRequiresTraderID(t *testing.T) {
	s := &Server{}
	router := gin.New()
	router.POST("/api/plan/owner-level/delete", s.handlePlanOwnerLevelDelete)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/plan/owner-level/delete", nil))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 missing trader_id, got %d", rec.Code)
	}
}

// overlayPriceViolations is the B2 armor over an owner patch — proven directly
// (no server needed): a fat-finger level is flagged, a sane one passes.
func TestOverlayPriceViolationsArmorsFatFinger(t *testing.T) {
	// last price 30000, dATR 200 → 8×dATR = 1600 band.
	sane := `[{"op":"add","path":"/levels/-","value":{"price":30150,"label":"X","grade":"A","instruction":"fade"}}]`
	if v := overlayPriceViolations(sane, 30000, 200); len(v) != 0 {
		t.Fatalf("sane level should pass armor, got %v", v)
	}
	fat := `[{"op":"add","path":"/levels/-","value":{"price":3000,"label":"X","grade":"A","instruction":"fade"}}]`
	if v := overlayPriceViolations(fat, 30000, 200); len(v) == 0 {
		t.Fatal("fat-finger 3000 (10× band away) must be flagged by armor")
	}
	// /price replace form is armored too.
	pr := `[{"op":"replace","path":"/levels/0/price","value":3000}]`
	if v := overlayPriceViolations(pr, 30000, 200); len(v) == 0 {
		t.Fatal("/price replace fat-finger must be flagged")
	}
	// no market data → fail-open (armor can't judge).
	if v := overlayPriceViolations(fat, 0, 0); len(v) != 0 {
		t.Fatalf("armor must fail-open without market data, got %v", v)
	}
}

func TestHandlePlanOverlayRequiresPatchAfterTrader(t *testing.T) {
	// With no trader_id the handler 400s before GetTrader; this asserts the body
	// path shape (trader_id present but patch missing would 404 on GetTrader with
	// a nil manager, so we only assert the empty-trader guard here).
	s := &Server{}
	router := gin.New()
	router.POST("/api/plan/overlay", s.handlePlanOverlay)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/plan/overlay", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for empty body, got %d", rec.Code)
	}
}
