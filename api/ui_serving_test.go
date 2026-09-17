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
	// binary rev unknown → not judged; the timestamp is still READ
	got := UIServingBootLine(dist, stamp.Add(-time.Hour), "")
	if !strings.Contains(got, "served-by=go-static") {
		t.Errorf("boot line must name the server: %q", got)
	}
	if !strings.Contains(got, "build=2026-09-03T18:30:00Z") {
		t.Errorf("boot line must READ the bundle timestamp: %q", got)
	}
	if strings.Contains(got, "STALE") {
		t.Errorf("an unjudgeable bundle is not called stale: %q", got)
	}
}

// THE DEFECT THAT STARTED THIS: the 08-31 dist under a 09-03 binary. Judged
// by rev now — a bundle carrying another rev MUST say STALE, and the older
// mtime rides along as the secondary note so the age is still visible.
func TestUIBootLineShoutsWhenTheBundleIsForAnotherRev(t *testing.T) {
	dist := revDistFixture(t, revA)
	built := time.Date(2026, 8, 31, 15, 34, 0, 0, time.UTC)
	if err := os.Chtimes(filepath.Join(dist, "index.html"), built, built); err != nil {
		t.Fatalf("chtimes: %v", err)
	}
	got := UIServingBootLine(dist, time.Date(2026, 9, 3, 20, 26, 0, 0, time.UTC), revB)
	if !strings.Contains(got, "STALE") {
		t.Fatalf("a bundle built for another rev MUST say STALE: %q", got)
	}
	if !strings.Contains(got, "build=2026-08-31T15:34:00Z") || !strings.Contains(got, "mtime predates the binary") {
		t.Errorf("the stale line must still name the timestamp and the age: %q", got)
	}
}

func TestUIBootLineSaysNoneWhenThereIsNoBundle(t *testing.T) {
	got := UIServingBootLine(filepath.Join(t.TempDir(), "absent"), time.Now(), revB)
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

// ── THE 🖥 LINE COMPARES REVS, NOT TIMESTAMPS (2026-09-16, boot of c6579347) ──
//
// The timestamp rule was RIGHT that boot by luck: the served bundle was the
// bcd70c0d dist (GUIDE_BUILT_REV=9e200002) under a c6579347 binary, and it
// also happened to be older. Had the correct f53f4e94 dist been installed —
// built 17 minutes BEFORE the binary — the same rule would have called the
// right bundle STALE. The bundle carries GUIDE_BUILT_REV (web/src/guide/
// types.ts) as a 40-hex literal; the truth is whether that rev is the binary's.
func revDistFixture(t *testing.T, bundleRev string) string {
	t.Helper()
	dir := t.TempDir()
	index := `<!doctype html><html><head><script type="module" crossorigin src="/assets/index-AbC123.js"></script></head><body></body></html>`
	if err := os.WriteFile(filepath.Join(dir, "index.html"), []byte(index), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "assets"), 0o755); err != nil {
		t.Fatal(err)
	}
	js := `var x=1;const G="` + bundleRev + `";export{G};`
	if err := os.WriteFile(filepath.Join(dir, "assets", "index-AbC123.js"), []byte(js), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

const (
	revA = "9e200002d6a2864ef9d3af041431763213641e1b"
	revB = "c6579347580a1720c343f6bba90717eacf3f810c"
)

func TestUIBootLineIsStaleByRevEvenWhenTheBundleIsNewer(t *testing.T) {
	dist := revDistFixture(t, revA)
	newer := time.Date(2026, 9, 16, 20, 0, 0, 0, time.UTC)
	if err := os.Chtimes(filepath.Join(dist, "index.html"), newer, newer); err != nil {
		t.Fatal(err)
	}
	got := UIServingBootLine(dist, newer.Add(-time.Hour), revB)
	if !strings.Contains(got, "STALE") || !strings.Contains(got, "bundle-rev=9e200002") || !strings.Contains(got, "binary c6579347") {
		t.Fatalf("a bundle built for another rev is STALE whatever its mtime: %q", got)
	}
}

func TestUIBootLineIsFreshByRevEvenWhenTheBundleIsOlder(t *testing.T) {
	dist := revDistFixture(t, revB)
	older := time.Date(2026, 9, 16, 19, 40, 15, 0, time.UTC) // the f53f4e94 dist, 17 min before the binary
	if err := os.Chtimes(filepath.Join(dist, "index.html"), older, older); err != nil {
		t.Fatal(err)
	}
	got := UIServingBootLine(dist, older.Add(17*time.Minute), revB)
	if strings.Contains(got, "STALE") || !strings.Contains(got, "bundle-rev=c6579347") || !strings.Contains(got, "matches the binary") {
		t.Fatalf("a bundle carrying the binary's rev is not stale, whatever its mtime: %q", got)
	}
	if !strings.Contains(got, "build=2026-09-16T19:40:15Z") {
		t.Fatalf("the timestamp stays on the line as a secondary field: %q", got)
	}
}

func TestUIBootLineSaysUnknownRevWhenTheBundleCarriesNone(t *testing.T) {
	dist := distFixture(t, `<script src="/assets/app.js"></script>`)
	got := UIServingBootLine(dist, time.Now(), revB)
	if !strings.Contains(got, "bundle-rev=UNKNOWN") {
		t.Fatalf("no 40-hex rev in the served bundle → UNKNOWN, never a guess: %q", got)
	}
}
