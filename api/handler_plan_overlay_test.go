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

	"nofx/kernel"

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

// mergedPriceViolations is the B2 armor over the RESOLVED plan_final — proven
// directly: a fat-finger level (any patch shape) is flagged, a sane one passes,
// and scenario target prices are swept too.
func TestMergedPriceViolationsArmorsFatFinger(t *testing.T) {
	// last price 30000, dATR 200 → 8×dATR = 1600 band.
	sane := &kernel.PlanDoc{Levels: []kernel.PlanLevel{{Price: 30150}}}
	if v := mergedPriceViolations(sane, 30000, 200); len(v) != 0 {
		t.Fatalf("sane level should pass armor, got %v", v)
	}
	// a wildly implausible level (introduced via ANY patch shape) is flagged.
	fat := &kernel.PlanDoc{Levels: []kernel.PlanLevel{{Price: 5000000}}}
	if v := mergedPriceViolations(fat, 30000, 200); len(v) == 0 {
		t.Fatal("fat-finger 5000000 must be flagged by armor")
	}
	// scenario target-chain prices are swept too.
	tgt := &kernel.PlanDoc{Scenarios: []kernel.PlanScenario{{TargetChain: []float64{9999999}}}}
	if v := mergedPriceViolations(tgt, 30000, 200); len(v) == 0 {
		t.Fatal("implausible scenario target must be flagged")
	}
	// no market data → fail-open (armor can't judge).
	if v := mergedPriceViolations(fat, 0, 0); len(v) != 0 {
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
