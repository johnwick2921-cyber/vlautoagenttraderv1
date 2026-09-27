package api

// CTO fold 1790280466263 (skeptic pass on live 4c05158b, finding [5]):
// updatesForbid WARNed on EVERY refusal, and the header badge polls
// GET /api/updates every 60 s — the un-enrolled live box wrote one 🔒 WARN
// (and one log_events row) per minute per tab, forever. Now: ONE WARN per
// (route, category) per process, then DEBUG; every refusal, first or
// repeat, is COUNTED in nofx_updates_refused_total{route,category} — never
// silent, and a pair that never refused has no series (no fabricated 0 —
// pinned in a fresh process: TestUpdatesRefusalSeriesIsAbsentUntilTheFirstRefusal).
// "category" comes from a CLOSED mapping of the refusal reason, never the
// free text. Driven at the production call site: the gin routes through
// updatesGate (and the install handler's own refusals).

import (
	"go/ast"
	"go/parser"
	"go/token"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"nofx/config"
	"nofx/logger"

	"github.com/sirupsen/logrus"
)

// scrapeRefused reads nofx_updates_refused_total off the PRODUCTION /metrics
// route (the promhttp handler NewServer mounts) — never through
// WithLabelValues, which would itself create the series it is asked about.
// It returns every series as "route\x00category" → value.
func scrapeRefused(t *testing.T, e *updEnv) map[string]float64 {
	t.Helper()
	r := httptest.NewRequest("GET", "/metrics", nil)
	r.RemoteAddr, r.Host = "127.0.0.1:52000", "127.0.0.1:8080"
	w := httptest.NewRecorder()
	e.s.router.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("GET /metrics = %d", w.Code)
	}
	line := regexp.MustCompile(`^nofx_updates_refused_total\{category="([^"]*)",route="([^"]*)"\} (\S+)$`)
	out := map[string]float64{}
	for _, l := range strings.Split(w.Body.String(), "\n") {
		if !strings.HasPrefix(l, "nofx_updates_refused_total") {
			continue
		}
		m := line.FindStringSubmatch(l)
		if m == nil {
			t.Fatalf("/metrics series in an unexpected shape: %q", l)
		}
		v, err := strconv.ParseFloat(m[3], 64)
		if err != nil {
			t.Fatalf("/metrics value %q: %v", m[3], err)
		}
		out[m[2]+"\x00"+m[1]] = v
	}
	return out
}

// refusedCount is one series' value from the scrape; an ABSENT series reads
// as 0 here (a delta baseline) — whether it is absent is asserted on its own.
func refusedCount(t *testing.T, e *updEnv, route, category string) float64 {
	t.Helper()
	return scrapeRefused(t, e)[route+"\x00"+category]
}

func warnLines(s, needle string) int {
	n := 0
	for _, l := range strings.Split(s, "\n") {
		if strings.Contains(l, "WARN") && strings.Contains(l, needle) {
			n++
		}
	}
	return n
}

func TestUpdatesRefusalWarnsOncePerRouteAndCategoryThenCounts(t *testing.T) {
	logs := captureLogs(t)
	e := newUpdEnv(t)
	noHeader := func(r *http.Request) { r.Header.Del(UpdateHeader) }
	const headerWhy = ": update header missing or wrong"

	before := refusedCount(t, e, "/api/updates", "update_header")
	mark := len(logs())
	for i := 0; i < 2; i++ { // the badge's poll, twice
		if w := e.do("GET", "/api/updates", "", noHeader); w.Code != http.StatusForbidden || w.Body.String() != forbiddenBody {
			t.Fatalf("poll %d = %d %s, want the uniform 403", i, w.Code, w.Body.String())
		}
	}
	tail := logs()[mark:]
	if n := warnLines(tail, headerWhy); n != 1 {
		t.Fatalf("two refusals of one (route, category) wrote %d WARN lines, want exactly 1:\n%s", n, tail)
	}
	if d := refusedCount(t, e, "/api/updates", "update_header") - before; d != 2 {
		t.Fatalf("nofx_updates_refused_total{route=/api/updates,category=update_header} rose by %v, want 2 (every refusal counted)", d)
	}

	// a repeat is DEBUG, never silent: visible at debug level, with its category
	prev := logger.Log.GetLevel()
	logger.Log.SetLevel(logrus.DebugLevel)
	t.Cleanup(func() { logger.Log.SetLevel(prev) })
	mark = len(logs())
	e.do("GET", "/api/updates", "", noHeader)
	tail = logs()[mark:]
	if warnLines(tail, headerWhy) != 0 || !strings.Contains(tail, "DEBU") || !strings.Contains(tail, headerWhy) {
		t.Fatalf("the third refusal: want one DEBUG line naming the category and no WARN, got:\n%s", tail)
	}
	logger.Log.SetLevel(prev)

	// a DIFFERENT category on the same route ⇒ a second WARN
	other := mintJWT(t, updOtherID, updOtherEmail, time.Now().Add(-5*time.Second), time.Now().Add(time.Hour), updSecret)
	mark = len(logs())
	e.do("GET", "/api/updates", "", withToken(other))
	e.do("GET", "/api/updates", "", withToken(other))
	if n := warnLines(logs()[mark:], ": not the enrolled admin"); n != 1 {
		t.Fatalf("a new category on the same route: %d WARN lines, want 1:\n%s", n, logs()[mark:])
	}
	// the SAME category on another route ⇒ its own first WARN
	mark = len(logs())
	e.do("POST", "/api/updates/check", "{}", noHeader)
	if n := warnLines(logs()[mark:], headerWhy); n != 1 {
		t.Fatalf("the header category on /check: %d WARN lines, want 1 (per route):\n%s", n, logs()[mark:])
	}
	// the install handler's own refusal goes through the same door
	g := e.grant(updRelease)
	bad := g
	bad.HMAC = strings.Repeat("b", 64)
	before = refusedCount(t, e, "/api/updates/install", "install_mac")
	mark = len(logs())
	e.do("POST", "/api/updates/install", grantBody(bad))
	bad.JobID = strings.Repeat("c", 32)
	e.do("POST", "/api/updates/install", grantBody(bad))
	if n := warnLines(logs()[mark:], ": install: MAC mismatch"); n != 1 {
		t.Fatalf("two MAC refusals: %d WARN lines, want 1:\n%s", n, logs()[mark:])
	}
	if d := refusedCount(t, e, "/api/updates/install", "install_mac") - before; d != 2 {
		t.Fatalf("install_mac rose by %v, want 2", d)
	}

	// the job route's label is the route PATTERN, never the client's id
	mark = len(logs())
	e.do("GET", "/api/updates/jobs/aaaaaaaaaaaaaaaa", "", noHeader)
	e.do("GET", "/api/updates/jobs/bbbbbbbbbbbbbbbb", "", noHeader)
	if n := warnLines(logs()[mark:], headerWhy); n != 1 {
		t.Fatalf("two ids, one route pattern: %d WARN lines, want 1:\n%s", n, logs()[mark:])
	}

	// exposed where the counters live: the production /metrics route
	r := httptest.NewRequest("GET", "/metrics", nil)
	r.RemoteAddr, r.Host = "127.0.0.1:52000", "127.0.0.1:8080"
	w := httptest.NewRecorder()
	e.s.router.ServeHTTP(w, r)
	body := w.Body.String()
	for _, series := range []string{
		`nofx_updates_refused_total{category="update_header",route="/api/updates"}`,
		`nofx_updates_refused_total{category="update_header",route="/api/updates/jobs/:id"}`,
		`nofx_updates_refused_total{category="install_mac",route="/api/updates/install"}`,
	} {
		if !strings.Contains(body, series) {
			t.Fatalf("/metrics lacks %s", series)
		}
	}
	if regexp.MustCompile(`nofx_updates_refused_total\{[^}]*(aaaaaaaa|bbbbbbbb)`).MatchString(body) {
		t.Fatal("/metrics carries a client-supplied id in a label")
	}
}

// "A pair that never refused has no series" — pinned in a FRESH process.
// The counter is package-global and the test binary shares it, so an earlier
// test's refusals would already have created series; the pin therefore
// re-runs itself as a child of the test binary (only this test, nothing
// before it), where no refusal has ever happened. There it scrapes the
// production /metrics: NO nofx_updates_refused_total series at all before
// the first refusal (a series pre-created at registration — a fabricated 0 —
// fails here), then one refusal ⇒ exactly that one pair, at 1, and every
// other pair still absent.
func TestUpdatesRefusalSeriesIsAbsentUntilTheFirstRefusal(t *testing.T) {
	const childEnv = "NOFX_TEST_REFUSAL_SERIES_CHILD"
	if os.Getenv(childEnv) != "1" {
		cmd := exec.Command(os.Args[0], "-test.run=^TestUpdatesRefusalSeriesIsAbsentUntilTheFirstRefusal$", "-test.count=1", "-test.v")
		cmd.Env = append(os.Environ(), childEnv+"=1")
		out, err := cmd.CombinedOutput()
		if err != nil || !strings.Contains(string(out), "--- PASS: TestUpdatesRefusalSeriesIsAbsentUntilTheFirstRefusal") {
			t.Fatalf("the fresh-process pin failed (%v):\n%s", err, out)
		}
		return
	}
	e := newUpdEnv(t)
	if got := scrapeRefused(t, e); len(got) != 0 {
		t.Fatalf("before any refusal /metrics already carries %d nofx_updates_refused_total series (%v) — a pair that never refused must have NO series, never a fabricated 0", len(got), got)
	}
	noHeader := func(r *http.Request) { r.Header.Del(UpdateHeader) }
	if w := e.do("GET", "/api/updates", "", noHeader); w.Code != http.StatusForbidden {
		t.Fatalf("GET /api/updates without the header = %d, want 403", w.Code)
	}
	got := scrapeRefused(t, e)
	if len(got) != 1 || got["/api/updates\x00update_header"] != 1 {
		t.Fatalf("after ONE refusal /metrics carries %v, want exactly {/api/updates, update_header} = 1", got)
	}
}

// The category map is CLOSED: every refusal reason the code can produce maps
// to a category other than "unmapped". The reasons are read off the source
// (every string literal returned by updatesRefusal and passed to
// updatesForbid, including the prefix of a concatenation) plus
// config.JWTSecretUnfitForUpdates' own answers.
func TestEveryUpdatesRefusalReasonHasACategory(t *testing.T) {
	f, err := parser.ParseFile(token.NewFileSet(), "handler_updates.go", nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	lit := func(x ast.Expr) (string, bool) {
		for {
			if b, ok := x.(*ast.BinaryExpr); ok && b.Op == token.ADD {
				x = b.X
				continue
			}
			break
		}
		bl, ok := x.(*ast.BasicLit)
		if !ok || bl.Kind != token.STRING {
			return "", false
		}
		s, err := strconv.Unquote(bl.Value)
		return s, err == nil
	}
	var reasons []string
	ast.Inspect(f, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.FuncDecl:
			if x.Name.Name == "updatesRefusal" {
				ast.Inspect(x.Body, func(m ast.Node) bool {
					if rs, ok := m.(*ast.ReturnStmt); ok && len(rs.Results) == 1 {
						if s, ok := lit(rs.Results[0]); ok && s != "" {
							reasons = append(reasons, s)
						}
					}
					return true
				})
			}
		case *ast.CallExpr:
			name := ""
			switch fn := x.Fun.(type) {
			case *ast.SelectorExpr:
				name = fn.Sel.Name
			case *ast.Ident:
				name = fn.Name
			}
			if name == "updatesForbid" && len(x.Args) == 2 {
				if s, ok := lit(x.Args[1]); ok {
					reasons = append(reasons, s)
				}
			}
		}
		return true
	})
	if len(reasons) < 25 {
		t.Fatalf("read only %d refusal reasons off handler_updates.go — the scan is not seeing the gate", len(reasons))
	}
	for _, secret := range [][]byte{nil, []byte("default-jwt-secret-change-in-production"), []byte("short")} {
		if why := config.JWTSecretUnfitForUpdates(secret); why != "" {
			reasons = append(reasons, why)
		}
	}
	okCat := regexp.MustCompile(`^[a-z][a-z_]{1,39}$`)
	for _, r := range reasons {
		c := updatesRefusalCategory(r)
		if c == updatesRefusalUnmapped || !okCat.MatchString(c) {
			t.Errorf("refusal reason %q maps to category %q — add it to the closed map", r, c)
		}
	}
	// and the forwarded prefix maps whatever header name follows it
	if c := updatesRefusalCategory("forwarded request (x-forwarded-*)"); c != "forwarded" {
		t.Fatalf("forwarded request category = %q", c)
	}
	if c := updatesRefusalCategory("some reason nobody mapped"); c != updatesRefusalUnmapped {
		t.Fatalf("an unknown reason maps to %q, want %q (never a free-text label)", c, updatesRefusalUnmapped)
	}
}
