// P5.4 — Ask-Planner handler contract tests. trader_id/question/qa_id validation
// runs before the LLM call / store, so &Server{} covers the guard paths. The
// anti-sycophancy enforcement + verdict KPI are proven in kernel/ + store/.

package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestHandlePlanAskRequiresTraderID(t *testing.T) {
	s := &Server{}
	router := gin.New()
	router.POST("/api/plan/ask", s.handlePlanAsk)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/plan/ask", nil))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 missing trader_id, got %d", rec.Code)
	}
}

func TestHandlePlanThreadRequiresTraderID(t *testing.T) {
	s := &Server{}
	router := gin.New()
	router.GET("/api/plan/ask", s.handlePlanThread)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/plan/ask", nil))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 missing trader_id, got %d", rec.Code)
	}
}

func TestHandlePlanAskApplyRequiresTraderID(t *testing.T) {
	s := &Server{}
	router := gin.New()
	router.POST("/api/plan/ask/apply", s.handlePlanAskApply)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/plan/ask/apply", nil))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 missing trader_id, got %d", rec.Code)
	}
}

func TestHandlePlanTradesRequiresTraderID(t *testing.T) {
	s := &Server{}
	router := gin.New()
	router.GET("/api/plan/trades", s.handlePlanTrades)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/plan/trades", nil))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 missing trader_id, got %d", rec.Code)
	}
}

func TestHandlePlanStatsRequiresTraderID(t *testing.T) {
	s := &Server{}
	router := gin.New()
	router.GET("/api/plan/stats", s.handlePlanStats)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/plan/stats", nil))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 missing trader_id, got %d", rec.Code)
	}
}
