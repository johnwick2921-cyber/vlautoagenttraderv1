package api

// PR #200 fold F1 (CTO 1790252194343, class-260 instance #2): the M5 web
// client typed the install body's expires_at as a STRING. The server parses
// the RAW bytes (internal/updateauth/strict.go rawUnixSeconds): expires_at is
// a canonical JSON NUMBER and nothing else — the MAC is over its decimal
// text, so a quoted "1800000300" must never alias it. A UI following that
// type would send the quoted form and every install would be 400.
//
// ONE byte string joins the three parties: the committed wire fixture
// web/src/lib/api/testdata/updates-install-body.wire.txt (no trailing
// newline — the file IS the request body).
//   - the web client puts exactly these bytes on the wire for the typed body
//     and for a pasted authorize line (web/src/lib/api/updates.header.test.ts,
//     real httpClient + capturing axios adapter);
//   - `updater-bootstrap authorize` prints exactly these bytes, modulo its two
//     random values (internal/updaterbootstrap, the real Run entry);
//   - here: the production parser and the production router take them past
//     the parse, and refuse the quoted form with 400.
// The fourth pin reads the web client's declared install-body types and
// checks each against the kinds ParseInstallRequest ACCEPTS, derived by
// re-encoding that one field of the fixture — no expectation is hand-kept.

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"nofx/internal/updateauth"
	"nofx/logger"
)

const webInstallBodyFixture = "web/src/lib/api/testdata/updates-install-body.wire.txt"

func readRepoFile(t *testing.T, rel string) []byte {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("..", filepath.FromSlash(rel)))
	if err != nil {
		t.Fatalf("read %s: %v", rel, err)
	}
	return b
}

type wireField struct {
	key string
	val json.RawMessage
}

// wireFields decodes one JSON object into its fields IN ORDER, values raw.
func wireFields(t *testing.T, b []byte) []wireField {
	t.Helper()
	dec := json.NewDecoder(bytes.NewReader(b))
	if tok, err := dec.Token(); err != nil || tok != json.Delim('{') {
		t.Fatalf("not a JSON object: %q", b)
	}
	var out []wireField
	for dec.More() {
		kt, err := dec.Token()
		if err != nil {
			t.Fatal(err)
		}
		var v json.RawMessage
		if err := dec.Decode(&v); err != nil {
			t.Fatal(err)
		}
		out = append(out, wireField{kt.(string), v})
	}
	return out
}

// encodeWire is the compact encoding JSON.stringify and json.Marshal share.
func encodeWire(fs []wireField) []byte {
	var b bytes.Buffer
	b.WriteByte('{')
	for i, f := range fs {
		if i > 0 {
			b.WriteByte(',')
		}
		k, _ := json.Marshal(f.key)
		b.Write(k)
		b.WriteByte(':')
		b.Write(f.val)
	}
	b.WriteByte('}')
	return b.Bytes()
}

// withField re-encodes ONE field's value; every other byte is unchanged.
func withField(fs []wireField, key string, val string) []byte {
	cp := make([]wireField, len(fs))
	copy(cp, fs)
	for i := range cp {
		if cp[i].key == key {
			cp[i].val = json.RawMessage(val)
		}
	}
	return encodeWire(cp)
}

func wireKind(v json.RawMessage) string {
	switch t := bytes.TrimSpace(v); {
	case len(t) > 0 && t[0] == '"':
		return "string"
	case len(t) > 0 && (t[0] == '-' || (t[0] >= '0' && t[0] <= '9')):
		return "number"
	}
	return "other"
}

// TestWebInstallBodyFixtureIsTheGrantAuthorizePrints: the fixture is a
// valid install body, expires_at is a bare JSON number in it, and the
// serializer `updater-bootstrap authorize` prints with (json.Marshal of
// updateauth.Grant, internal/updaterbootstrap/bootstrap.go authorize)
// reproduces it byte for byte — so what the page is pasted and what the web
// client sends are the same bytes.
func TestWebInstallBodyFixtureIsTheGrantAuthorizePrints(t *testing.T) {
	fixture := readRepoFile(t, webInstallBodyFixture)
	fs := wireFields(t, fixture)
	if !bytes.Equal(encodeWire(fs), fixture) {
		t.Fatalf("%s is not compact single-object JSON (JSON.stringify's form): %q", webInstallBodyFixture, fixture)
	}
	g, err := updateauth.ParseInstallRequest(bytes.NewReader(fixture))
	if err != nil {
		t.Fatalf("%s is refused by the production parser: %v", webInstallBodyFixture, err)
	}
	for _, f := range fs {
		if f.key == "expires_at" && wireKind(f.val) != "number" {
			t.Fatalf("%s carries expires_at as %s %s, want a bare JSON number", webInstallBodyFixture, wireKind(f.val), f.val)
		}
	}
	printed, err := json.Marshal(g)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(printed, fixture) {
		t.Fatalf("authorize's serializer prints\n  %s\nthe web client sends\n  %s", printed, fixture)
	}
}

// TestWebInstallBodyFixturePassesTheParseAtTheRouter drives the production
// router (NewServer, real middleware): the fixture's bytes get past the
// parse AND the expiry window and are judged by the MAC (403, "install: MAC
// mismatch" — the fixture's hmac is not minted under this test's random
// device key); the same bytes with expires_at quoted — what a string-typed
// client sends — are 400 "bad request" at the parse. Positive control: a
// real authorized grant in the same byte layout reaches the stub (422).
func TestWebInstallBodyFixturePassesTheParseAtTheRouter(t *testing.T) {
	var buf bytes.Buffer
	var mu sync.Mutex
	logger.Log.SetOutput(writerFunc(func(p []byte) (int, error) { mu.Lock(); defer mu.Unlock(); return buf.Write(p) }))
	t.Cleanup(func() { logger.Log.SetOutput(os.Stdout) })
	logs := func() string { mu.Lock(); defer mu.Unlock(); s := buf.String(); buf.Reset(); return s }

	e := newUpdEnv(t)
	fixture := readRepoFile(t, webInstallBodyFixture)
	g, err := updateauth.ParseInstallRequest(bytes.NewReader(fixture))
	if err != nil {
		t.Fatalf("fixture: %v", err)
	}
	inWindow := time.Unix(g.ExpiresAt-60, 0)
	e.s.updatesNow = func() time.Time { return inWindow }

	logs()
	w := e.do("POST", "/api/updates/install", string(fixture))
	if w.Code != http.StatusForbidden || w.Body.String() != forbiddenBody {
		t.Fatalf("the web client's install body = %d %s, want 403 %s at the MAC stage (past the parse)", w.Code, w.Body.String(), forbiddenBody)
	}
	if l := logs(); !strings.Contains(l, "install: MAC mismatch") {
		t.Fatalf("the web client's install body was refused before the MAC stage; logs: %s", l)
	}

	quoted := withField(wireFields(t, fixture), "expires_at", `"`+string(mustRaw(t, fixture, "expires_at"))+`"`)
	w = e.do("POST", "/api/updates/install", string(quoted))
	if w.Code != http.StatusBadRequest || w.Body.String() != `{"error":"bad request"}` {
		t.Fatalf("quoted expires_at %s = %d %s, want 400 {\"error\":\"bad request\"}", quoted, w.Code, w.Body.String())
	}
	if l := logs(); !strings.Contains(l, "install: bad request") {
		t.Fatalf("quoted expires_at was not refused at the parse; logs: %s", l)
	}

	e.s.updatesNow = nil
	if w := e.do("POST", "/api/updates/install", grantBody(e.grant(updRelease))); w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("positive control: a real grant = %d %s, want 422", w.Code, w.Body.String())
	}
}

func mustRaw(t *testing.T, b []byte, key string) json.RawMessage {
	t.Helper()
	for _, f := range wireFields(t, b) {
		if f.key == key {
			return f.val
		}
	}
	t.Fatalf("no %q in %s", key, b)
	return nil
}

// webInstallBodyTypes reads the install body the web client declares:
// `async install(body: { k: T … })` in web/src/lib/api/updates.ts.
func webInstallBodyTypes(t *testing.T) map[string]string {
	t.Helper()
	src := string(readRepoFile(t, webUpdatesClient))
	m := regexp.MustCompile(`async install\(body: \{([^}]*)\}\)`).FindAllStringSubmatch(src, -1)
	if len(m) != 1 || strings.Count(src, "async install(") != 1 {
		t.Fatalf("%s: want exactly one `async install(body: { … })`, found %d — a shape this pin cannot read; extend it", webUpdatesClient, len(m))
	}
	decl := regexp.MustCompile(`^([A-Za-z_][A-Za-z0-9_]*)(\??)\s*:\s*([A-Za-z]+)$`)
	out := map[string]string{}
	for _, part := range strings.FieldsFunc(m[0][1], func(r rune) bool { return r == '\n' || r == ';' || r == ',' }) {
		s := strings.TrimSpace(part)
		if s == "" || strings.HasPrefix(s, "//") {
			continue
		}
		p := decl.FindStringSubmatch(s)
		if p == nil {
			t.Fatalf("%s: install body member %q is a shape this pin cannot read; extend it", webUpdatesClient, s)
		}
		if p[2] == "?" {
			t.Fatalf("%s: install body member %s is optional, but the server requires every key (a missing one is 400)", webUpdatesClient, p[1])
		}
		if _, dup := out[p[1]]; dup {
			t.Fatalf("%s: install body member %s declared twice", webUpdatesClient, p[1])
		}
		out[p[1]] = p[3]
	}
	return out
}

// TestWebInstallBodyTypesAreWhatTheServerParses: the web client's declared
// install body has exactly the server's keys, and every member's TS type is
// a JSON kind ParseInstallRequest ACCEPTS for that key. The accepted kinds
// are derived from the production parser by re-encoding that one field of
// the fixture as a string and as a number.
func TestWebInstallBodyTypesAreWhatTheServerParses(t *testing.T) {
	fixture := readRepoFile(t, webInstallBodyFixture)
	fs := wireFields(t, fixture)
	if _, err := updateauth.ParseInstallRequest(bytes.NewReader(fixture)); err != nil {
		t.Fatalf("positive control: the fixture is refused: %v", err)
	}

	accepted := map[string][]string{}
	for _, f := range fs {
		// the server requires the key: the fixture without it is malformed
		var rest []wireField
		for _, o := range fs {
			if o.key != f.key {
				rest = append(rest, o)
			}
		}
		if _, err := updateauth.ParseInstallRequest(bytes.NewReader(encodeWire(rest))); !errors.Is(err, updateauth.ErrMalformed) {
			t.Fatalf("the fixture without %q = %v, want ErrMalformed (the server requires it)", f.key, err)
		}
		enc := map[string]string{}
		switch wireKind(f.val) {
		case "string":
			enc["string"] = string(f.val)
			enc["number"] = "1"
		case "number":
			enc["number"] = string(f.val)
			enc["string"] = `"` + string(f.val) + `"`
		default:
			t.Fatalf("fixture %s is %s, neither string nor number", f.key, f.val)
		}
		for _, kind := range []string{"number", "string"} {
			_, err := updateauth.ParseInstallRequest(bytes.NewReader(withField(fs, f.key, enc[kind])))
			switch {
			case err == nil:
				accepted[f.key] = append(accepted[f.key], kind)
			case !errors.Is(err, updateauth.ErrMalformed):
				t.Fatalf("%s as %s: %v, want nil or ErrMalformed", f.key, kind, err)
			}
		}
	}

	declared := webInstallBodyTypes(t)
	var serverKeys, webKeys []string
	for k := range accepted {
		serverKeys = append(serverKeys, k)
	}
	for k := range declared {
		webKeys = append(webKeys, k)
	}
	sort.Strings(serverKeys)
	sort.Strings(webKeys)
	if strings.Join(webKeys, ",") != strings.Join(serverKeys, ",") {
		t.Fatalf("%s install body declares %v, the server parses exactly %v (unknown or missing keys are 400)", webUpdatesClient, webKeys, serverKeys)
	}
	for _, k := range serverKeys {
		ts := declared[k]
		if ts != "string" && ts != "number" {
			t.Fatalf("%s install body types %s as %q, a type this pin cannot map to a JSON kind; extend it", webUpdatesClient, k, ts)
		}
		ok := false
		for _, kind := range accepted[k] {
			ok = ok || kind == ts
		}
		if !ok {
			t.Errorf("%s install body types %s as %s, but ParseInstallRequest accepts it only as a JSON %s — the %s form is 400 \"bad request\", so every UI install would be 400",
				webUpdatesClient, k, ts, strings.Join(accepted[k], "/"), ts)
		}
	}
}
