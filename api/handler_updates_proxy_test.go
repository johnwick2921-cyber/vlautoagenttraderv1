package api

// W-ONE-BUTTON M3 red-team fold M3 (red-2 #2): the loopback check judges the
// socket peer, and a reverse proxy or tunnel ON THE SAME BOX is a loopback
// peer for every client it relays — config.go itself tells an operator who
// binds off-loopback to put "a firewall/reverse proxy" in front. So
// /api/updates* refuse any request that says it was forwarded: Forwarded,
// X-Forwarded-*, X-Real-IP, Via and the vendor client-IP headers. No browser
// and no local UI sends any of them.
//
// Known limit (named, not pinned): a proxy that strips or never adds these
// headers (nginx's default proxy_pass adds none; an SSH -L or socat tunnel
// adds none) is indistinguishable from a local client. The runbook says:
// updates are loopback-DIRECT only.

import (
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"strings"
	"testing"
)

var forwardingHeaderNames = []string{
	"Forwarded", "X-Forwarded-For", "X-Forwarded-Host", "X-Forwarded-Proto",
	"X-Forwarded-Port", "X-Forwarded-Server", "X-Forwarded-Anything", "X-Real-IP", "Via",
	"CF-Connecting-IP", "True-Client-IP", "X-Client-IP", "X-Cluster-Client-IP",
}

// Every forwarding header — canonical, lower-case, underscore-spelled, with an
// empty value or a loopback-looking value — turns an otherwise admitted
// loopback request into the uniform 403 on all five routes.
func TestUpdatesRefuseRequestsCarryingAForwardingHeader(t *testing.T) {
	e := newUpdEnv(t)
	e.expectAllAdmitted("no forwarding header")
	for _, name := range forwardingHeaderNames {
		for label, mut := range map[string]func(*http.Request){
			"canonical":  func(r *http.Request) { r.Header.Set(name, "127.0.0.1") },
			"lowercase":  func(r *http.Request) { r.Header[strings.ToLower(name)] = []string{"127.0.0.1"} },
			"underscore": func(r *http.Request) { r.Header[strings.ReplaceAll(name, "-", "_")] = []string{"127.0.0.1"} },
			"empty":      func(r *http.Request) { r.Header[http.CanonicalHeaderKey(name)] = []string{""} },
		} {
			e.expectAllForbidden(name+" ("+label+")", mut)
		}
	}
	// positive control: a header that merely resembles one is not refused
	e.expectAllAdmitted("unrelated header", func(r *http.Request) { r.Header.Set("X-Forwarding-Note", "none") })
}

// Real sockets: the production router listens on 127.0.0.1; a same-box
// reverse proxy (net/http/httputil — Rewrite+SetXForwarded, and the legacy
// Director form, which appends X-Forwarded-For) relays a client. The proxy's
// connection is a loopback peer and it rewrites Host to the upstream's, so
// only the forwarding headers betray it: every route is refused. Positive
// control: the same client talking to the router directly is admitted.
func TestUpdatesRefuseAClientRelayedByASameBoxReverseProxy(t *testing.T) {
	e := newUpdEnv(t)
	app := httptest.NewServer(e.s.router)
	defer app.Close()
	up, err := url.Parse(app.URL)
	if err != nil {
		t.Fatal(err)
	}
	proxies := map[string]*httputil.ReverseProxy{
		"Rewrite+SetXForwarded": {Rewrite: func(pr *httputil.ProxyRequest) { pr.SetURL(up); pr.SetXForwarded() }},
		"legacy Director":       httputil.NewSingleHostReverseProxy(up),
	}
	call := func(base, method, path, body string) (int, string) {
		t.Helper()
		req, err := http.NewRequest(method, base+path, strings.NewReader(body))
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set(UpdateHeader, "1")
		req.Header.Set("Authorization", "Bearer "+e.tok)
		if body != "" {
			req.Header.Set("Content-Type", "application/json")
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		var sb strings.Builder
		buf := make([]byte, 512)
		for {
			n, rerr := resp.Body.Read(buf)
			sb.Write(buf[:n])
			if rerr != nil {
				break
			}
		}
		return resp.StatusCode, sb.String()
	}
	for _, rt := range e.allRoutes() {
		if code, body := call(app.URL, rt.method, rt.path, rt.body); code != admittedStatus[rt.path] {
			t.Fatalf("positive control (direct): %s %s = %d %s, want %d", rt.method, rt.path, code, body, admittedStatus[rt.path])
		}
	}
	for name, px := range proxies {
		ps := httptest.NewServer(px)
		for _, rt := range e.allRoutes() {
			if code, body := call(ps.URL, rt.method, rt.path, rt.body); code != http.StatusForbidden || body != forbiddenBody {
				t.Errorf("%s proxy: %s %s = %d %s, want 403 %s", name, rt.method, rt.path, code, body, forbiddenBody)
			}
		}
		ps.Close()
	}
}
