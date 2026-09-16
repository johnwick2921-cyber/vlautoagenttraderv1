// UI SERVING PATH (owner ruling 2026-09-03).
//
// Found during the 1D cutover: web/dist was stale since 2026-08-31, the Go
// server registered NO static route, and :8080/ returned 404 — the entire UI
// was served by a vite DEV server on :3000 started by hand. Nothing brought it
// back after a reboot, and a dev server is not a production server (no
// minification, HMR websocket, unbounded rebuild memory).
//
// These pin the production path: the bot's own process serves the UI, a stale
// or missing bundle is LOUD rather than silent, and /api keeps its exact
// previous behaviour.

package api

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

// distFixture writes a minimal built bundle and returns its directory.
func distFixture(t *testing.T, indexBody string) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "index.html"), []byte(indexBody), 0o644); err != nil {
		t.Fatalf("write index: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "assets"), 0o755); err != nil {
		t.Fatalf("mkdir assets: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "assets", "app.js"), []byte("console.log(1)"), 0o644); err != nil {
		t.Fatalf("write asset: %v", err)
	}
	return dir
}

func routerWithUI(t *testing.T, dist string) *gin.Engine {
	t.Helper()
	r := gin.New()
	// The /api group must behave EXACTLY as before: an unknown /api path is a
	// 404 from the API, never the SPA shell.
	r.GET("/api/health", func(c *gin.Context) { c.JSON(200, gin.H{"status": "ok"}) })
	MountUI(r, dist)
	return r
}

func TestUIServesTheBundleAtRoot(t *testing.T) {
	dist := distFixture(t, "<!doctype html><title>VL</title>")
	r := routerWithUI(t, dist)

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET / = %d, want 200 (the UI must answer from the bot's own process)", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "<title>VL</title>") {
		t.Errorf("GET / did not return index.html, got %q", rec.Body.String())
	}
}

func TestUIServesHashedAssets(t *testing.T) {
	dist := distFixture(t, "x")
	r := routerWithUI(t, dist)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/assets/app.js", nil))
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "console.log") {
		t.Fatalf("asset = %d %q, want 200 with the file", rec.Code, rec.Body.String())
	}
}

// A deep link is a client route, not a file. It must return the shell, or the
// owner's bookmark to /studio 404s after a reload.
func TestUIDeepLinkFallsBackToTheShell(t *testing.T) {
	dist := distFixture(t, "<!doctype html><title>VL</title>")
	r := routerWithUI(t, dist)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/studio/expectancy", nil))
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "VL") {
		t.Fatalf("deep link = %d, want the SPA shell", rec.Code)
	}
}

// THE REGRESSION THAT WOULD MATTER MOST: the SPA fallback must never swallow an
// unknown API path. A 404 that returns HTML turns a broken endpoint into a
// silent blank screen.
func TestUIFallbackNeverHijacksAPI(t *testing.T) {
	dist := distFixture(t, "<!doctype html><title>VL</title>")
	r := routerWithUI(t, dist)

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/no-such-endpoint", nil))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("unknown /api path = %d, want 404", rec.Code)
	}
	if strings.Contains(rec.Body.String(), "<title>") {
		t.Errorf("unknown /api path returned the SPA shell: %q", rec.Body.String())
	}

	// and a real API route still works
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/health", nil))
	if rec.Code != http.StatusOK {
		t.Errorf("GET /api/health = %d, want 200", rec.Code)
	}
}

// The bundle sits next to .env and data/data.db. A traversal out of dist would
// serve either.
func TestUIRefusesPathTraversal(t *testing.T) {
	dist := distFixture(t, "x")
	secret := filepath.Join(filepath.Dir(dist), "secret.txt")
	if err := os.WriteFile(secret, []byte("JWT_SECRET=nope"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	r := routerWithUI(t, dist)
	for _, p := range []string{"/../secret.txt", "/assets/../../secret.txt", "/%2e%2e/secret.txt"} {
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, p, nil))
		if strings.Contains(rec.Body.String(), "JWT_SECRET") {
			t.Errorf("traversal %q leaked a file outside dist", p)
		}
	}
}

// MISSING BUNDLE: the server must still boot and still serve the API. The UI
// being absent is a loud degradation, never a crash and never a silent 404 that
// looks like a routing bug.
func TestUIMissingBundleLeavesTheAPIAlive(t *testing.T) {
	r := routerWithUI(t, filepath.Join(t.TempDir(), "does-not-exist"))
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/health", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("API must survive a missing bundle, got %d", rec.Code)
	}
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code == http.StatusOK {
		t.Errorf("a missing bundle must not answer 200 at /")
	}
}

// ─────────────────────────────────────────────────────────────────────
// The boot line. READ from disk, never a literal (A24), and a field the
// process cannot know prints n/a rather than a plausible value.
// ─────────────────────────────────────────────────────────────────────

func TestUIBootLineReadsTheBundleTimestamp(t *testing.T) {
	dist := distFixture(t, "x")
	stamp := time.Date(2026, 9, 3, 18, 30, 0, 0, time.UTC)
	if err := os.Chtimes(filepath.Join(dist, "index.html"), stamp, stamp); err != nil {
		t.Fatalf("chtimes: %v", err)
	}
	// binaryAt older than the bundle → fresh
	got := UIServingBootLineAt(dist, stamp.Add(-time.Hour))
	if !strings.Contains(got, "served-by=go-static") {
		t.Errorf("boot line must name the server: %q", got)
	}
	if !strings.Contains(got, "build=2026-09-03T18:30:00Z") {
		t.Errorf("boot line must READ the bundle timestamp: %q", got)
	}
	if strings.Contains(got, "STALE") {
		t.Errorf("a bundle newer than the binary is not stale: %q", got)
	}
}

// THE DEFECT THAT STARTED THIS: a bundle older than the binary is exactly the
// 08-31 dist under a 09-03 binary. It must be impossible to miss.
func TestUIBootLineShoutsWhenTheBundleIsOlderThanTheBinary(t *testing.T) {
	dist := distFixture(t, "x")
	built := time.Date(2026, 8, 31, 15, 34, 0, 0, time.UTC)
	if err := os.Chtimes(filepath.Join(dist, "index.html"), built, built); err != nil {
		t.Fatalf("chtimes: %v", err)
	}
	got := UIServingBootLineAt(dist, time.Date(2026, 9, 3, 20, 26, 0, 0, time.UTC))
	if !strings.Contains(got, "STALE") {
		t.Fatalf("a bundle older than the binary MUST say STALE: %q", got)
	}
	if !strings.Contains(got, "build=2026-08-31T15:34:00Z") {
		t.Errorf("the stale line must still name the timestamp: %q", got)
	}
}

func TestUIBootLineSaysNoneWhenThereIsNoBundle(t *testing.T) {
	got := UIServingBootLineAt(filepath.Join(t.TempDir(), "absent"), time.Now())
	if !strings.Contains(got, "served-by=none") {
		t.Errorf("no bundle → served-by=none, got %q", got)
	}
	// n/a, never a zero time — a field the process cannot know says so.
	if !strings.Contains(got, "build=n/a") {
		t.Errorf("unknown build must print n/a, got %q", got)
	}
	if strings.Contains(got, "0001-01-01") {
		t.Errorf("a zero time leaked into the boot line: %q", got)
	}
}
