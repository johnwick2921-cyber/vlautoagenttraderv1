package api

// PR #200 fold F9 (CTO 1790252194343 #21): PUT /api/user/password compares
// current_password with no limiter. The fold: a fixed 1 s delay after a
// FAILED compare, before the 403 is written, plus a counted refusal (the B6
// gate-block table, process-wide key "") beside the WARN line. A right
// current_password is not delayed, and neither is any other refusal on the
// route (they never reach the compare). Driven at the PRODUCTION router
// (canon 53); the counter is read back through GET /api/risk/gate-blocks.
// The delay seam is recorded, never slept (testmain_test.go).

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"
)

// wrongPasswordCount reads the counter through the production route.
func wrongPasswordCount(t *testing.T, e *updEnv, tok string) int {
	t.Helper()
	w := credCall(t, e, "GET", "/api/risk/gate-blocks", tok, "")
	if w.Code != http.StatusOK {
		t.Fatalf("GET /api/risk/gate-blocks = %d %s", w.Code, w.Body.String())
	}
	var out struct {
		ByTrader map[string]map[string]int `json:"by_trader"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	return out.ByTrader[""][currentPasswordWrongGate]
}

func TestWrongCurrentPasswordIsDelayedAndCounted(t *testing.T) {
	// The production values: a real 1 s time.Sleep.
	if currentPasswordFailDelay != time.Second {
		t.Fatalf("currentPasswordFailDelay = %v — the ruled production delay is 1 s", currentPasswordFailDelay)
	}
	if reflect.ValueOf(prodCurrentPasswordFailSleep).Pointer() != reflect.ValueOf(time.Sleep).Pointer() {
		t.Fatal("the production delay seam is not time.Sleep")
	}

	logs := captureLogs(t)
	e := newUpdEnv(t)
	owner, code := credLogin(t, e, updAdminEmail, updAdminPass)
	if code != http.StatusOK {
		t.Fatalf("owner login = %d", code)
	}

	var w *httptest.ResponseRecorder
	var calls []time.Duration
	var bodyAtSleep []int
	prev := currentPasswordFailSleep
	currentPasswordFailSleep = func(d time.Duration) {
		calls = append(calls, d)
		bodyAtSleep = append(bodyAtSleep, w.Body.Len())
	}
	t.Cleanup(func() { currentPasswordFailSleep = prev })
	put := func(tok, body string) *httptest.ResponseRecorder {
		r := httptest.NewRequest("PUT", "/api/user/password", strings.NewReader(body))
		r.RemoteAddr, r.Host = "127.0.0.1:52000", "127.0.0.1:8080"
		r.Header.Set("Authorization", "Bearer "+tok)
		r.Header.Set("Content-Type", "application/json")
		w = httptest.NewRecorder()
		e.s.router.ServeHTTP(w, r)
		return w
	}

	// A WRONG current_password: delayed once by exactly 1 s, BEFORE the 403
	// is written; counted; the WARN line.
	before := wrongPasswordCount(t, e, owner)
	mark := len(logs())
	rec := put(owner, `{"current_password":"not-the-password","new_password":"owner-new-password-8"}`)
	if rec.Code != http.StatusForbidden || !strings.Contains(rec.Body.String(), "current password is incorrect") {
		t.Fatalf("wrong current_password = %d %s — want 403 current password is incorrect", rec.Code, rec.Body.String())
	}
	if len(calls) != 1 || calls[0] != time.Second {
		t.Fatalf("delay seam calls = %v — want exactly one, of 1s", calls)
	}
	if bodyAtSleep[0] != 0 {
		t.Fatal("the 403 was written BEFORE the delay — the delay must hold the answer")
	}
	if got := wrongPasswordCount(t, e, owner); got != before+1 {
		t.Fatalf("gate-block %s = %d after one wrong compare — want %d", currentPasswordWrongGate, got, before+1)
	}
	if tail := logs()[mark:]; !strings.Contains(tail, "[WARN]") || !strings.Contains(tail, "refused PUT /api/user/password from 127.0.0.1: current password is incorrect") {
		t.Fatalf("no WARN line for the wrong current_password:\n%s", tail)
	}

	// Refusals that never reach the compare: not delayed, not counted.
	calls = nil
	before = wrongPasswordCount(t, e, owner)
	if rec := put(mustBotToken(t), `{"current_password":"not-the-password","new_password":"machine-chosen-pass-8"}`); rec.Code != http.StatusForbidden {
		t.Fatalf("machine token = %d, want 403", rec.Code)
	}
	if rec := put(owner, `{"new_password":"owner-new-password-8"}`); rec.Code != http.StatusBadRequest {
		t.Fatalf("no current_password = %d, want 400", rec.Code)
	}
	if len(calls) != 0 || wrongPasswordCount(t, e, owner) != before {
		t.Fatalf("a refusal that never compared was delayed (%v) or counted", calls)
	}

	// The RIGHT current_password: no delay, not counted, 200.
	if rec := put(owner, `{"current_password":"`+updAdminPass+`","new_password":"owner-new-password-8"}`); rec.Code != http.StatusOK {
		t.Fatalf("right current_password = %d %s — want 200", rec.Code, rec.Body.String())
	}
	if len(calls) != 0 {
		t.Fatalf("a RIGHT current_password was delayed: %v", calls)
	}
}
