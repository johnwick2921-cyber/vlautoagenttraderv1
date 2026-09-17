// UI SERVING PATH (owner ruling 2026-09-03).
//
// THE RULING: the bot's own process serves the UI. Rejected alternative: a
// systemd unit keeping `npm run dev` alive. Reasons, in order —
//
//  1. A dev server is not a production server. `vite dev` ships unminified
//     modules, holds an HMR websocket open, and rebuilds on filesystem events;
//     supervising it does not make it a production server, it makes an
//     unsupervised dev server a supervised one.
//  2. Fewer moving parts. One process means the UI cannot outlive the API or
//     predecease it, and there is no second unit to install — installing units
//     needs sudo, which the agent lane does not have, so a vite unit would have
//     stayed a plan rather than becoming a fact.
//  3. It survives a reboot by construction: whatever brings the bot back brings
//     the UI back, because they are the same process.
//
// COST, stated plainly: the production UI moves from :3000 to :8080. Vite stays
// available for development and keeps its /api proxy; it is no longer what the
// owner depends on. The Guide says :8080.
package api

import (
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// UIDistDir is where the built bundle lives, relative to the process working
// directory — the same convention data/data.db already uses.
const UIDistDir = "web/dist"

// MountUI registers the static bundle and the SPA fallback on r.
//
// It is deliberately a free function taking the directory: the tests mount it
// on a bare engine over a temp dir, so what is pinned is the ROUTING, not a
// server construction path they would otherwise have to fake.
func MountUI(r *gin.Engine, dist string) {
	index := filepath.Join(dist, "index.html")
	if _, err := os.Stat(index); err != nil {
		// No bundle: leave routing untouched. The API must keep working and the
		// boot line says served-by=none, so this degrades loudly elsewhere
		// rather than crashing here (class 23 — a read surface never stops the
		// process).
		return
	}

	fs := http.Dir(dist)
	fileServer := http.StripPrefix("/", http.FileServer(fs))

	r.NoRoute(func(c *gin.Context) {
		p := c.Request.URL.Path

		// /api KEEPS ITS EXACT PREVIOUS BEHAVIOUR. Without this an unknown
		// endpoint would return the SPA shell with status 200, turning a broken
		// route into a blank screen and a 404 into a lie.
		if strings.HasPrefix(p, "/api/") || p == "/api" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Not found"})
			return
		}

		// Only GET/HEAD may reach the bundle; a POST to an unknown path is a
		// client bug, not a page request.
		if c.Request.Method != http.MethodGet && c.Request.Method != http.MethodHead {
			c.JSON(http.StatusNotFound, gin.H{"error": "Not found"})
			return
		}

		// path.Clean inside http.FileServer already resolves "..", and
		// http.Dir refuses to escape its root — but the check is explicit
		// because the bundle's parent directory holds .env and data/data.db,
		// and "the library handles it" is not something to leave implicit at
		// that blast radius.
		clean := filepath.Clean(p)
		if strings.Contains(clean, "..") {
			c.JSON(http.StatusNotFound, gin.H{"error": "Not found"})
			return
		}

		// A real file wins; anything else is a client route and gets the shell,
		// so a reload of /studio/expectancy does not 404.
		if f, err := fs.Open(clean); err == nil {
			st, serr := f.Stat()
			_ = f.Close()
			if serr == nil && !st.IsDir() {
				fileServer.ServeHTTP(c.Writer, c.Request)
				return
			}
		}
		c.File(index)
	})
}

// UIServingBootLine is the UI's boot line, judged by REV (2026-09-16): the
// served entry bundle carries GUIDE_BUILT_REV (web/src/guide/types.ts) as a
// 40-hex literal, and the bundle is STALE when that rev is not the binary's.
// The former timestamp rule (removed) was right at the c6579347 boot by
// luck — the wrong bundle also happened to be older — and would have called
// the RIGHT bundle stale had it been installed (built 17 minutes before the
// binary). The build time stays on the line as a secondary field. A bundle
// carrying no 40-hex literal prints bundle-rev=UNKNOWN and is not judged;
// an empty binaryRev disables the comparison rather than calling everything
// stale (A24: unknown is not stale, and not fresh).
func UIServingBootLine(dist string, binaryAt time.Time, binaryRev string) string {
	st, err := os.Stat(filepath.Join(dist, "index.html"))
	if err != nil {
		return "ui: served-by=none build=n/a — no bundle at " + dist +
			" (run `cd web && npm ci && npm run build`); the API is unaffected"
	}
	built := st.ModTime().UTC()
	entry, revs := servedBundleRevs(dist)
	line := "ui: served-by=go-static build=" + built.Format(time.RFC3339)
	if entry != "" {
		line += " bundle=" + entry
	}
	short := func(r string) string {
		if len(r) > 8 {
			return r[:8]
		}
		return r
	}
	switch {
	case len(revs) == 0:
		line += " bundle-rev=UNKNOWN (no 40-hex literal in the served entry bundle — not judged)"
	case binaryRev == "":
		line += " bundle-rev=" + short(revs[0]) + " (binary rev unknown — not judged)"
	default:
		match := false
		for _, r := range revs {
			if strings.EqualFold(r, binaryRev) {
				match = true
				break
			}
		}
		if match {
			line += " bundle-rev=" + short(binaryRev) + " matches the binary"
		} else {
			line += " bundle-rev=" + short(revs[0]) + " STALE — built for another rev than binary " + short(binaryRev) +
				"; the UI is not showing this build (install the dist built at " + short(binaryRev) + ")"
		}
	}
	if !binaryAt.IsZero() && built.Before(binaryAt.UTC()) {
		line += " · bundle mtime predates the binary by " + binaryAt.UTC().Sub(built).Round(time.Minute).String()
	}
	return line
}

var (
	entryBundleRe = regexp.MustCompile(`assets/(index-[A-Za-z0-9_-]+\.js)`)
	hex40Re       = regexp.MustCompile(`[0-9a-f]{40}`)
)

// servedBundleRevs finds the entry bundle index.html references and every
// distinct 40-hex literal in it, in order of appearance.
func servedBundleRevs(dist string) (entry string, revs []string) {
	index, err := os.ReadFile(filepath.Join(dist, "index.html"))
	if err != nil {
		return "", nil
	}
	m := entryBundleRe.FindSubmatch(index)
	if m == nil {
		return "", nil
	}
	entry = string(m[1])
	js, err := os.ReadFile(filepath.Join(dist, "assets", entry))
	if err != nil {
		return entry, nil
	}
	seen := map[string]bool{}
	for _, h := range hex40Re.FindAll(js, -1) {
		r := string(h)
		if !seen[r] {
			seen[r] = true
			revs = append(revs, r)
		}
	}
	return entry, revs
}
