package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"nofx/kernel"
	"nofx/store"
)

// TestHealthReportsRealState pins P2-6 at the production call site
// (handleHealth): time must be a real RFC3339 timestamp (never null), db must
// be probed, and a dead database must answer 503 — not 200.
func TestHealthReportsRealState(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// No store: degraded (the DB leg is unavailable) but still 200 with a real
	// time and the revision cutover scripts read.
	srv := &Server{}
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/health", nil)
	srv.handleHealth(c)

	if w.Code != http.StatusOK {
		t.Fatalf("no-store health = %d, want 200 (degraded)", w.Code)
	}
	var body map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("health body not JSON: %v", err)
	}
	if ts, ok := body["time"].(string); !ok || ts == "" {
		t.Fatalf("health time must be a real timestamp, got %v", body["time"])
	} else if _, err := time.Parse(time.RFC3339, ts); err != nil {
		t.Fatalf("health time not RFC3339: %q", ts)
	}
	if body["revision"] != kernel.RunningRevision() {
		t.Fatalf("revision must stay %q for cutover scripts, got %v", kernel.RunningRevision(), body["revision"])
	}
	if body["db"] != "unavailable" || body["status"] != "degraded" {
		t.Fatalf("no-store health must read db=unavailable status=degraded, got %+v", body)
	}
	if tr, ok := body["traders_running"].(float64); !ok || tr != 0 {
		t.Fatalf("traders_running must be 0, got %v", body["traders_running"])
	}

	// A CLOSED database must flip the endpoint to 503 — the old handler
	// answered 200 to a dead DB.
	st, err := store.New(filepath.Join(t.TempDir(), "dead.db"))
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}
	st.Close()
	srv2 := &Server{store: st}
	w2 := httptest.NewRecorder()
	c2, _ := gin.CreateTestContext(w2)
	c2.Request = httptest.NewRequest(http.MethodGet, "/api/health", nil)
	srv2.handleHealth(c2)
	if w2.Code != http.StatusServiceUnavailable {
		t.Fatalf("dead-DB health = %d, want 503", w2.Code)
	}
}
