package api

// #206 review fold (P1 api/handler_updates.go:298): the updates-refusal WARN
// must log the ROUTE (c.FullPath()), never c.Request.URL.Path. The raw path
// is client-supplied: any loopback process could GET
// /api/updates/jobs/BOOT%20INTEGRITY%20REFUSED without a header and the WARN
// line would embed "BOOT INTEGRITY REFUSED" in the boot log the worker scans
// (data/nofx_<date>.log) — verifyBootLine then read a refused boot and rolled
// back a good install. A forged "BOOT INTEGRITY OK — rev <sha12> ·" would
// equally satisfy the OK half. Nothing the client sends may land in the boot
// log as the boot line's text.

import (
	"bytes"
	"net/http"
	"os"
	"strings"
	"sync"
	"testing"

	"nofx/logger"

	"github.com/sirupsen/logrus"
)

// The worker scans data/nofx_<date>.log, which only logrus writes to (gin's
// access log goes to stdout). So this pin captures the logrus logger alone.
func captureLoggerOnly(t *testing.T) func() string {
	t.Helper()
	var buf bytes.Buffer
	var mu sync.Mutex
	sink := writerFunc(func(p []byte) (int, error) { mu.Lock(); defer mu.Unlock(); return buf.Write(p) })
	logger.Log.SetOutput(sink)
	t.Cleanup(func() { logger.Log.SetOutput(os.Stdout) })
	return func() string { mu.Lock(); defer mu.Unlock(); return buf.String() }
}

func TestUpdatesRefusalNeverLogsTheClientPath(t *testing.T) {
	logs := captureLoggerOnly(t)
	prevLevel := logger.Log.GetLevel()
	logger.Log.SetLevel(logrus.DebugLevel) // read every refusal line, repeat or not
	t.Cleanup(func() { logger.Log.SetLevel(prevLevel) })
	e := newUpdEnv(t)

	noHeader := func(r *http.Request) { r.Header.Del(UpdateHeader) }
	for _, injected := range []string{
		"BOOT INTEGRITY REFUSED",
		"BOOT INTEGRITY OK — rev 0123456789ab ·",
	} {
		t.Run(injected, func(t *testing.T) {
			mark := len(logs())
			escaped := strings.ReplaceAll(injected, " ", "%20")
			w := e.do("GET", "/api/updates/jobs/"+escaped, "", noHeader)
			tail := logs()[mark:]
			if w.Code != http.StatusForbidden || w.Body.String() != forbiddenBody {
				t.Fatalf("GET of the injected path = %d %s, want 403 %s", w.Code, w.Body.String(), forbiddenBody)
			}
			if !strings.Contains(tail, "[updates] refused") {
				t.Fatalf("the refusal was not logged: %q", tail)
			}
			if strings.Contains(tail, injected) {
				t.Fatalf("the client-supplied text landed in the log the worker scans:\n%s", tail)
			}
			// The route, not the raw path, is what gets quoted.
			if !strings.Contains(tail, `/api/updates/jobs/:id`) {
				t.Fatalf("the WARN must name the static route: %q", tail)
			}
		})
	}
}
