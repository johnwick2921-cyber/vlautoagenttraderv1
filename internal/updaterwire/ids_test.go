package updaterwire

import (
	"strings"
	"testing"
)

// The id allow-lists are the ONE source for M3 (internal/updateauth does not
// exist at this base; the auth builder imports these). A MAC over
// release_id|job_id|expires_at is unambiguous only because neither id can
// hold "|"; a release id reaches a filesystem join in M4, so nothing
// path-shaped may pass.
func TestJobIDAllowList(t *testing.T) {
	for _, ok := range []string{"job-0001abcd", "abcdefgh", "0123456789abcdef", strings.Repeat("a", 64), "a-------"} {
		if !ValidJobID(ok) {
			t.Fatalf("positive control: %q must be a valid job id", ok)
		}
	}
	for _, bad := range []string{
		"", "short", "abcdefg", strings.Repeat("a", 65), "-abcdefgh", "JOB-0001ABCD", "job_0001abcd",
		"job 0001abcd", "job|0001abcd", "job/0001abcd", "../job0001", "job.0001abcd", "job\x000001abcd",
		"job-0001abcd\n", "jób-0001abcd",
	} {
		if ValidJobID(bad) {
			t.Fatalf("%q must NOT be a valid job id", bad)
		}
	}
}

func TestReleaseIDAllowList(t *testing.T) {
	for _, ok := range []string{"v1.4.2", "1.0.0", "v2026.09.24-rc1", "a", "V1_2", strings.Repeat("a", 64)} {
		if !ValidReleaseID(ok) {
			t.Fatalf("positive control: %q must be a valid release id", ok)
		}
	}
	for _, bad := range []string{
		"", "..", ".", "../../etc/passwd", "/abs", "a/b", `a\b`, "https://x/y", "file:x", "%2e%2e",
		"v1\x00", strings.Repeat("a", 65), ".hidden", "-v1", "a..b", "v1..", "a|b", "v1 2", "v1\n",
		"~root", "$(id)", "`id`", "v1;rm", "v1.4.2 ", "ｖ1",
	} {
		if ValidReleaseID(bad) {
			t.Fatalf("%q must NOT be a valid release id", bad)
		}
	}
}

func TestNoValidIDCanHoldTheMACSeparator(t *testing.T) {
	// release="a|b",job=... vs release="a",job="b|..." would otherwise MAC the same bytes
	if ValidReleaseID("a|b") || ValidJobID("b|cdefghij") {
		t.Fatal("the MAC field separator '|' passed an id allow-list — fields could be reframed")
	}
}
