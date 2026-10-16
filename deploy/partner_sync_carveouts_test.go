package deploy

// partner_sync_carveouts_test pins deploy/partner_sync_carveouts: the list an
// owner→partner sync reads to decide which files it must NEVER overwrite. A
// partner-only file missing from the list is a bug the next sync will silently
// clobber; a listed path that does not exist is a stale entry. Both are refused
// here.
//
// release #32: deploy/partner-first-time-setup.sh is partner-only (it targets
// the partner repo and the partner machine's one-time setup) and is pinned IN
// this list, so an owner-repo sync never replaces it with an owner variant.

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

func readCarveouts(t *testing.T) []string {
	t.Helper()
	b, err := os.ReadFile("partner_sync_carveouts")
	if err != nil {
		t.Fatalf("read deploy/partner_sync_carveouts: %v", err)
	}
	var out []string
	for _, line := range strings.Split(string(b), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		out = append(out, line)
	}
	return out
}

func TestPartnerSyncCarveoutsPinned(t *testing.T) {
	list := readCarveouts(t)
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root := filepath.Join(wd, "..") // paths in the list are repo-root-relative
	seen := map[string]bool{}
	for _, p := range list {
		if seen[p] {
			t.Fatalf("duplicate carve-out %q", p)
		}
		seen[p] = true
		if _, err := os.Stat(filepath.Join(root, p)); err != nil {
			t.Fatalf("carve-out %q does not exist in the tree: %v", p, err)
		}
	}
	if !sort.StringsAreSorted(list) {
		t.Fatalf("carve-outs are not sorted: %v", list)
	}
	// The release #32 script and its test are partner-only — a sync must keep
	// the partner copy.
	for _, want := range []string{
		"deploy/partner-first-time-setup.sh",
		"deploy/partner_first_time_setup_test.go",
		"deploy/release_allowed_signers",
	} {
		if !seen[want] {
			t.Fatalf("%s is not in deploy/partner_sync_carveouts", want)
		}
	}
}
