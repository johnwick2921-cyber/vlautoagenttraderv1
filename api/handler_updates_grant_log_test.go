package api

// W-ONE-BUTTON M3 red-team fold (red-3 #6) at the production call site: the
// install handler logs the M4 hand-off's error with %v
// (handler_updates.go "hand-off refused"). A starter whose error carries the
// Grant — the most natural thing for an M4 worker client to write — must not
// put the live MAC into data/nofx_*.log or log_events.

import (
	"bytes"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"testing"

	"nofx/internal/updateauth"
	"nofx/logger"

	"github.com/gin-gonic/gin"
)

func TestInstallHandOffErrorNeverLogsTheGrantMAC(t *testing.T) {
	var buf bytes.Buffer
	var mu sync.Mutex
	sink := writerFunc(func(p []byte) (int, error) { mu.Lock(); defer mu.Unlock(); return buf.Write(p) })
	prevGin := gin.DefaultWriter
	gin.DefaultWriter = sink
	logger.Log.SetOutput(sink)
	t.Cleanup(func() { gin.DefaultWriter = prevGin; logger.Log.SetOutput(os.Stdout) })

	e := newUpdEnv(t)
	e.s.SetUpdateVerifier(fakeVerifier{})
	e.s.SetUpdateStarter(func(g updateauth.Grant, _ updateauth.Manifest) error {
		return fmt.Errorf("worker refused %v / %+v / %#v", g, g, &g)
	})
	g := e.grant(updRelease)
	if w := e.do("POST", "/api/updates/install", grantBody(g)); w.Code != http.StatusServiceUnavailable {
		t.Fatalf("install with a failing starter = %d %s, want 503", w.Code, w.Body.String())
	}
	mu.Lock()
	logs := buf.String()
	mu.Unlock()
	if !strings.Contains(logs, "hand-off refused") || !strings.Contains(logs, g.JobID) {
		t.Fatalf("positive control: the hand-off error line (naming the job) was not captured:\n%s", logs)
	}
	if strings.Contains(logs, g.HMAC) {
		t.Fatal("the hand-off error log carries the grant's MAC")
	}
}
