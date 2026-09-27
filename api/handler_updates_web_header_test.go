package api

// W-ONE-BUTTON M3 × M5 wire pin (CTO ruling 1790243172089: M3 owns the wire
// contract). The gate refuses every /api/updates* request without exactly one
// UpdateHeader: 1, before it reads the JWT. The M5 web readers never sent it:
// M5's shape pins mocked the transport and M3's pins drove the server with Go
// requests, so neither side exercised the call site across the wire and the
// enrolled admin's own page read 403 forever.
//
// This pin reads the web client's source against THIS package's constant:
// renaming the header on either side, or adding an /updates call that does
// not pass the header object, goes RED here. What axios actually puts on the
// wire is pinned at the web call site (web/src/lib/api/updates.header.test.ts).

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

const webUpdatesClient = "web/src/lib/api/updates.ts"

func TestWebUpdatesClientSendsTheGatesHeader(t *testing.T) {
	b, err := os.ReadFile(filepath.Join("..", filepath.FromSlash(webUpdatesClient)))
	if err != nil {
		t.Fatalf("read %s: %v", webUpdatesClient, err)
	}
	src := string(b)

	// 1. The ONE header object spells this package's header name, value "1".
	decl := regexp.MustCompile(`const UPDATE_HEADERS = \{ '([^']+)': '([^']*)' \}`).FindAllStringSubmatch(src, -1)
	if len(decl) != 1 {
		t.Fatalf("%s: want exactly one `const UPDATE_HEADERS = { '<name>': '<value>' }`, found %d", webUpdatesClient, len(decl))
	}
	if decl[0][1] != UpdateHeader || decl[0][2] != "1" {
		t.Fatalf("%s sends %q: %q, the gate requires %q: \"1\" (api.UpdateHeader) — every /updates call would be 403 \"update header missing or wrong\"",
			webUpdatesClient, decl[0][1], decl[0][2], UpdateHeader)
	}

	// 2. Every /api/updates* request the client makes passes that object.
	// A call is `httpClient.request<…>(` whose URL is `${API_BASE}/updates…`;
	// its options run to the call's closing `)`.
	calls := regexp.MustCompile(`httpClient\.request<[^(]*\(\s*`+"`"+`\$\{API_BASE\}/updates[^`+"`"+`]*`+"`"+`,\s*\{([^}]*)\}\s*\)`).FindAllStringSubmatch(src, -1)
	all := strings.Count(src, "${API_BASE}/updates")
	if all == 0 || len(calls) != all {
		t.Fatalf("%s: %d spellings of ${API_BASE}/updates but %d parsed request calls — a new call shape this pin cannot read; extend it", webUpdatesClient, all, len(calls))
	}
	for _, c := range calls {
		if !strings.Contains(c[1], "headers: UPDATE_HEADERS") {
			t.Fatalf("%s: an /updates request without `headers: UPDATE_HEADERS`: {%s} — the gate refuses it (403)", webUpdatesClient, strings.TrimSpace(c[1]))
		}
	}
}
