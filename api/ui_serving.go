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

// UIServingBootLineAt is the UI's boot line: what serves it and how old the
// bundle is. Every field is READ — there is no literal in it.
//
// The binary's build time is PASSED IN rather than read here. kernel already
// owns the one reader of debug.ReadBuildInfo (buildStamp, surfaced as
// BootIntegrity.BuildTime) and main.go holds that value at boot; a second
// reader in this package would be a second source of the same truth, free to
// disagree with the boot-integrity line printed three lines above it.
// binaryAt is the build time of the running binary; the bundle is STALE when it
// predates it — exactly the state found at the 1D cutover (a 2026-08-31 dist
// under a 2026-09-03 binary), and the state this line exists to make impossible
// to miss. A zero binaryAt disables the comparison rather than calling
// everything stale.
func UIServingBootLineAt(dist string, binaryAt time.Time) string {
	st, err := os.Stat(filepath.Join(dist, "index.html"))
	if err != nil {
		// A field the process cannot know prints n/a — never a zero time, which
		// would render as 0001-01-01 and read as a real (very old) build.
		return "ui: served-by=none build=n/a — no bundle at " + dist +
			" (run `cd web && npm ci && npm run build`); the API is unaffected"
	}
	built := st.ModTime().UTC()
	line := "ui: served-by=go-static build=" + built.Format(time.RFC3339)
	if !binaryAt.IsZero() && built.Before(binaryAt.UTC()) {
		line += " STALE — the bundle predates this binary by " +
			binaryAt.UTC().Sub(built).Round(time.Minute).String() +
			"; the UI is not showing this build"
	}
	return line
}
