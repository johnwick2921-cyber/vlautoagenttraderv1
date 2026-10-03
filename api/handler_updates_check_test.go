package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"vl/internal/updaterwire"
	"vl/internal/updaterwire/wireserver"
)

// checkWorker is a real wireserver on <dataDir>/updater/worker.sock whose
// check verb answers what onCheck returns; every other verb is recorded and
// refused.
type checkWorker struct {
	onCheck func() updaterwire.Response
}

// startCheckWorker listens with the production listener (fold A1 relay test).
func startCheckWorker(t *testing.T, dataDir string, onCheck func() updaterwire.Response) {
	t.Helper()
	path, err := updaterwire.SocketPath(dataDir)
	if err != nil {
		t.Fatalf("SocketPath: %v", err)
	}
	ln, err := wireserver.Listen(path, t.Logf)
	if err != nil {
		t.Fatalf("wireserver.Listen(%s): %v", path, err)
	}
	done := make(chan struct{})
	go func() {
		defer close(done)
		_ = ln.Serve(func(req updaterwire.Request) updaterwire.Response {
			if req.Verb == updaterwire.VerbCheck && req.Check != nil {
				return onCheck()
			}
			return updaterwire.Response{OK: false, Error: "unexpected verb"}
		})
	}()
	t.Cleanup(func() { _ = ln.Close(); <-done })
}

func decodeCheckBody(t *testing.T, w *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &m); err != nil {
		t.Fatalf("check body is not JSON: %q (%v)", w.Body.String(), err)
	}
	return m
}

func TestUpdatesCheckRelayWorkerUnreachable(t *testing.T) {
	e := newUpdEnv(t)
	// No wireserver in this data dir: the relay answers checked:false.
	w := e.do("POST", "/api/updates/check", "{}")
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	m := decodeCheckBody(t, w)
	if m["checked"] != false || m["reason"] == "" {
		t.Fatalf("body = %v, want checked:false with a reason", m)
	}
}

func TestUpdatesCheckRelayStates(t *testing.T) {
	cases := []struct {
		name  string
		resp  updaterwire.Response
		check func(t *testing.T, m map[string]any)
	}{
		{
			"knob off keeps the M3 answer",
			updaterwire.Response{OK: true, State: "off"},
			func(t *testing.T, m map[string]any) {
				if m["checked"] != false || m["reason"] != "no release source in this build" {
					t.Fatalf("body = %v", m)
				}
			},
		},
		{
			"rate limited",
			updaterwire.Response{OK: true, State: "rate_limited", Detail: `{"reason":"rate limited, try later"}`},
			func(t *testing.T, m map[string]any) {
				if m["checked"] != false || m["reason"] != "rate limited, try later" {
					t.Fatalf("body = %v", m)
				}
			},
		},
		{
			"up to date",
			updaterwire.Response{OK: true, State: "up_to_date", Detail: `{"available":false,"ready":false,"tag":"v9","target_commitish":"` + commitish + `"}`},
			func(t *testing.T, m map[string]any) {
				if m["checked"] != true || m["available"] != false || m["tag"] != "v9" {
					t.Fatalf("body = %v", m)
				}
			},
		},
		{
			"verified ready",
			updaterwire.Response{OK: true, State: "verified_ready", Detail: `{"available":true,"ready":true,"tag":"v9","target_commitish":"` + commitish + `","source_sha":"` + commitish + `"}`},
			func(t *testing.T, m map[string]any) {
				if m["checked"] != true || m["available"] != true || m["ready"] != true ||
					m["tag"] != "v9" || m["target_commitish"] != commitish || m["source_sha"] != commitish {
					t.Fatalf("body = %v", m)
				}
			},
		},
		{
			"worker error carries its reason",
			updaterwire.Response{OK: true, State: "error", Detail: `{"reason":"verification failed"}`},
			func(t *testing.T, m map[string]any) {
				if m["checked"] != false || m["reason"] != "verification failed" {
					t.Fatalf("body = %v", m)
				}
			},
		},
		{
			"refused frame",
			updaterwire.Response{OK: false, Error: "rejected"},
			func(t *testing.T, m map[string]any) {
				if m["checked"] != false || m["reason"] != "check failed" {
					t.Fatalf("body = %v", m)
				}
			},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			e := newUpdEnv(t)
			startCheckWorker(t, e.dataDir, func() updaterwire.Response { return c.resp })
			w := e.do("POST", "/api/updates/check", "{}")
			if w.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200 (body %s)", w.Code, w.Body.String())
			}
			c.check(t, decodeCheckBody(t, w))
		})
	}
}

const commitish = "0123456789abcdef0123456789abcdef01234567"
