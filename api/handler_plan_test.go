// P4.1 — /api/plan/* handler tests. These lock the trader_id-required contract
// (mirror of /api/risk/*) and the alert-ack argument validation, which all
// return before touching the store/traderManager (nil-safe with &Server{}).

package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

// TestHandlePlanTodayRequiresTraderID: GET /plan/today with no trader_id → 400.
func TestHandlePlanTodayRequiresTraderID(t *testing.T) {
	s := &Server{}
	router := gin.New()
	router.GET("/api/plan/today", s.handlePlanToday)

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/plan/today", nil))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 missing trader_id, got %d body=%s", rec.Code, rec.Body.String())
	}
}

// TestHandlePlanHistoryRequiresTraderID: GET /plan/history with no trader_id → 400.
func TestHandlePlanHistoryRequiresTraderID(t *testing.T) {
	s := &Server{}
	router := gin.New()
	router.GET("/api/plan/history", s.handlePlanHistory)

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/plan/history", nil))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 missing trader_id, got %d body=%s", rec.Code, rec.Body.String())
	}
}

// TestHandlePlanAlertsRequiresTraderID: GET /plan/alerts with no trader_id → 400.
func TestHandlePlanAlertsRequiresTraderID(t *testing.T) {
	s := &Server{}
	router := gin.New()
	router.GET("/api/plan/alerts", s.handlePlanAlerts)

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/plan/alerts", nil))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 missing trader_id, got %d body=%s", rec.Code, rec.Body.String())
	}
}

// TestHandlePlanAlertAckRequiresTraderID: POST /plan/alert-ack with no trader_id → 400.
func TestHandlePlanAlertAckRequiresTraderID(t *testing.T) {
	s := &Server{}
	router := gin.New()
	router.POST("/api/plan/alert-ack", s.handlePlanAlertAck)

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/plan/alert-ack", nil))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 missing trader_id, got %d body=%s", rec.Code, rec.Body.String())
	}
}

// TestHandlePlanAlertAckRequiresAlertID: trader_id present but alert_id ≤ 0 → 400.
func TestHandlePlanAlertAckRequiresAlertID(t *testing.T) {
	s := &Server{}
	router := gin.New()
	router.POST("/api/plan/alert-ack", s.handlePlanAlertAck)

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/plan/alert-ack?trader_id=t1", nil))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 missing alert_id, got %d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "alert_id") {
		t.Fatalf("400 body should mention alert_id, got %s", rec.Body.String())
	}
}
