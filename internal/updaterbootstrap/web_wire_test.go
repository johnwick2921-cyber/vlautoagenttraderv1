package updaterbootstrap

// PR #200 fold F1 (CTO 1790252194343): the Updates page must accept EXACTLY
// what `authorize` prints, and the web client must send it back unchanged.
// The one byte string both sides are pinned to is the committed wire fixture
// web/src/lib/api/testdata/updates-install-body.wire.txt (the web client
// puts it on the wire byte for byte — web/src/lib/api/updates.header.test.ts;
// the production router takes it past the parse —
// api/handler_updates_web_body_test.go). Here the REAL CLI entry (Run) prints
// a grant at this file's clock and release, and that line — with its two
// random values (job_id, hmac) swapped for the fixture's — IS the fixture:
// same keys, same order, expires_at a bare JSON number, same bytes.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"nofx/internal/updateauth"
)

const webInstallBodyFixture = "web/src/lib/api/testdata/updates-install-body.wire.txt"

func TestAuthorizePrintsTheBytesTheWebClientSends(t *testing.T) {
	attended(t)
	inst := enrolledInstall(t)
	rc, out, errb := run(inst, authorizeLine(bRel), "authorize", bRel)
	if rc != 0 {
		t.Fatalf("rc=%d %s", rc, errb)
	}
	if !strings.HasSuffix(out, "\n") || strings.Count(out, "\n") != 1 {
		t.Fatalf("stdout is not exactly one line: %q", out)
	}
	line := strings.TrimSuffix(out, "\n")
	g, err := updateauth.ParseInstallRequest(strings.NewReader(line))
	if err != nil {
		t.Fatalf("the printed grant is not a valid install body: %v", err)
	}

	fixture, err := os.ReadFile(filepath.Join("..", "..", filepath.FromSlash(webInstallBodyFixture)))
	if err != nil {
		t.Fatalf("read %s: %v", webInstallBodyFixture, err)
	}
	f, err := updateauth.ParseInstallRequest(strings.NewReader(string(fixture)))
	if err != nil {
		t.Fatalf("%s is refused by the production parser: %v", webInstallBodyFixture, err)
	}
	// the fixture is pinned at this file's release and clock, so only the
	// two random values differ
	if f.ReleaseID != bRel || f.ExpiresAt != bNow.Add(updateauth.MaxAuthorizationWindow).Unix() {
		t.Fatalf("%s names release %q expiring %d; this pin prints %q expiring %d — re-pin both together",
			webInstallBodyFixture, f.ReleaseID, f.ExpiresAt, bRel, bNow.Add(updateauth.MaxAuthorizationWindow).Unix())
	}
	swapped := strings.Replace(line, `"job_id":"`+g.JobID+`"`, `"job_id":"`+f.JobID+`"`, 1)
	swapped = strings.Replace(swapped, `"hmac":"`+g.HMAC+`"`, `"hmac":"`+f.HMAC+`"`, 1)
	if swapped != string(fixture) {
		t.Fatalf("authorize prints (random values swapped for the fixture's)\n  %s\nthe web client sends\n  %s", swapped, fixture)
	}
}
