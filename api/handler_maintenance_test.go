package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"testing"
	"time"

	"nofx/store"
	"nofx/trader"

	"github.com/gin-gonic/gin"
)

// W-ONE-BUTTON M2 — the two maintenance surfaces are GETs that serve the
// trader package's views; nothing in the API can write the hold.

func TestMaintenanceRoutesAreReadOnlyGets(t *testing.T) {
	b, err := os.ReadFile("server.go")
	if err != nil {
		t.Fatal(err)
	}
	src := string(b)
	for _, path := range []string{"/maintenance", "/installation-gate"} {
		if !regexp.MustCompile(`"GET", "` + regexp.QuoteMeta(path) + `"`).MatchString(src) {
			t.Errorf("GET %s is not registered", path)
		}
		if regexp.MustCompile(`"(POST|PUT|PATCH|DELETE)", "` + regexp.QuoteMeta(path) + `[/"]`).MatchString(src) {
			t.Errorf("%s must have no write route", path)
		}
	}
}

func getJSON(t *testing.T, h gin.HandlerFunc) map[string]any {
	t.Helper()
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	h(c)
	if w.Code != http.StatusOK {
		t.Fatalf("status %d: %s", w.Code, w.Body.String())
	}
	var m map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &m); err != nil {
		t.Fatal(err)
	}
	return m
}

func TestMaintenanceHandlersServeTheTraderViews(t *testing.T) {
	dir := t.TempDir()
	prev := trader.MaintenanceDataDir()
	trader.SetMaintenanceDataDir(dir)
	t.Cleanup(func() { trader.SetMaintenanceDataDir(prev) })
	st, err := store.New(filepath.Join(t.TempDir(), "m.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	if err := store.WriteMaintenanceHold(dir, store.MaintenanceHold{Held: true, JobID: "job-api", Since: time.Now().UTC().Format(time.RFC3339), Owner: "cli"}); err != nil {
		t.Fatal(err)
	}
	s := &Server{store: st} // no trader manager: no loaded traders

	m := getJSON(t, s.handleMaintenanceStatus)
	if m["held"] != true || m["job_id"] != "job-api" || m["addon_ack"] != nil {
		t.Fatalf("GET /api/maintenance: %v", m)
	}
	g := getJSON(t, s.handleInstallationGate)
	if g["ready"] != false || g["job_id"] != "job-api" {
		t.Fatalf("GET /api/installation-gate with no AddOn must not be ready: %v", g)
	}
	if legs, ok := g["legs"].([]any); !ok || len(legs) == 0 {
		t.Fatalf("legs missing: %v", g)
	}
}
