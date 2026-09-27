package api

// W-ONE-BUTTON M3 — the install stage ORDER, at the production router
// (canon 53): gate 403 → strict parse 400 → expiry 403 → HMAC 403 → job-id
// replay 409 → manifest 422. Each pair below carries TWO defects and asserts
// the EARLIER stage answers, with the one-defect request as the positive
// control. The order is a security property, not style:
//   - gate before parse: a caller that is not the enrolled admin learns
//     nothing about the body grammar (403, never 400);
//   - expiry before consume: an expired grant never spends its job id;
//   - HMAC before replay: only a holder of device.key can learn whether a
//     job id was spent (409) — anyone else gets the uniform 403;
//   - replay before the verifier: a spent job id never reaches the manifest
//     verifier (in M4, never a second job).

import (
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"nofx/internal/updateauth"
	"nofx/store"
)

func TestInstallStagesRunInOrderAndEachRefusalStopsTheNext(t *testing.T) {
	e := newUpdEnv(t)
	spy := &spyVerifier{}
	e.s.SetUpdateVerifier(spy)
	const malformed = `{"release_id":"v1","job_id":"0123456789abcdef","expires_at":1,"hmac":"x","extra":1}`

	// gate before parse
	if w := e.do("POST", "/api/updates/install", malformed); w.Code != http.StatusBadRequest {
		t.Fatalf("positive control: admin + malformed body = %d, want 400", w.Code)
	}
	past := time.Now().Add(-time.Hour).UTC()
	if err := e.st.User().Create(&store.User{ID: updOtherID, Email: updOtherEmail, PasswordHash: "x", CreatedAt: past, UpdatedAt: past}); err != nil {
		t.Fatal(err)
	}
	other := mintJWT(t, updOtherID, updOtherEmail, time.Now().Add(-5*time.Second), time.Now().Add(time.Hour), updSecret)
	for name, mut := range map[string]func(*http.Request){
		"no token":        func(r *http.Request) { r.Header.Del("Authorization") },
		"ordinary user":   withToken(other),
		"no update hdr":   func(r *http.Request) { r.Header.Del(UpdateHeader) },
		"cross origin":    func(r *http.Request) { r.Header.Set("Origin", "http://evil.test") },
		"non-loopback":    func(r *http.Request) { r.RemoteAddr = "192.168.1.5:4000" },
		"rebinding host":  func(r *http.Request) { r.Host = "evil.test:8080" },
		"garbage bearer":  withToken("not.a.jwt"),
		"cross-site sfs":  func(r *http.Request) { r.Header.Set("Sec-Fetch-Site", "cross-site") },
		"two auth values": func(r *http.Request) { r.Header.Add("Authorization", "Bearer x") },
	} {
		if w := e.do("POST", "/api/updates/install", malformed, mut); w.Code != http.StatusForbidden || w.Body.String() != forbiddenBody {
			t.Errorf("%s + malformed body = %d %s, want 403 %s (the gate answers before the parser)", name, w.Code, w.Body.String(), forbiddenBody)
		}
	}

	// parse before expiry: an expired grant with an extra field is 400
	key := mustKey(t, e.dataDir)
	expired := grantBodyUnder(t, key, updRelease, "order-expired-0001", time.Now().Unix()-5)
	if w := e.do("POST", "/api/updates/install", strings.TrimSuffix(expired, "}")+`,"extra":1}`); w.Code != http.StatusBadRequest {
		t.Errorf("expired + malformed = %d, want 400", w.Code)
	}
	// expiry before consume: the expired grant (valid MAC) is 403 and its
	// job id is not spent
	if w := e.do("POST", "/api/updates/install", expired); w.Code != http.StatusForbidden || w.Body.String() != forbiddenBody {
		t.Errorf("expired grant = %d %s, want 403", w.Code, w.Body.String())
	}
	if b, _ := os.ReadFile(updateauth.SeenPath(e.dataDir)); strings.Contains(string(b), "order-expired-0001") {
		t.Error("an expired grant spent its job id")
	}

	// HMAC before replay: spend g, then replay it with a wrong MAC ⇒ 403
	// (no replay oracle without the key); with the right MAC ⇒ 409
	g := e.grant(updRelease)
	if w := e.do("POST", "/api/updates/install", grantBody(g)); w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("positive control: first use = %d", w.Code)
	}
	calls := len(spy.calls)
	bad := g
	bad.HMAC = strings.Repeat("0", 64)
	if w := e.do("POST", "/api/updates/install", grantBody(bad)); w.Code != http.StatusForbidden || w.Body.String() != forbiddenBody {
		t.Errorf("spent job id + wrong MAC = %d %s, want 403 %s (a key-less caller must not learn the id was spent)", w.Code, w.Body.String(), forbiddenBody)
	}
	// replay before the verifier: the 409 never reaches it
	if w := e.do("POST", "/api/updates/install", grantBody(g)); w.Code != http.StatusConflict {
		t.Errorf("spent job id + right MAC = %d, want 409", w.Code)
	}
	if len(spy.calls) != calls {
		t.Errorf("the verifier was called %d more time(s) after the first use; a refused request must never reach it", len(spy.calls)-calls)
	}
}
