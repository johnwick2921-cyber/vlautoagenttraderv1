package deploy

// partner_sync_carveouts_test pins deploy/partner_sync_carveouts: the list an
// owner→partner sync reads to decide which files it must NEVER overwrite. A
// partner-only file missing from the list is a bug the next sync will silently
// clobber; a listed path that does not exist is a stale entry. Both are refused
// here.
//
// new-install (OWNER ORDER 10-16): deploy/new-install.sh and its test are
// partner-only (they target the partner repo and the partner machine's fresh
// install) and are pinned IN this list, so an owner-repo sync never replaces
// them with an owner variant.

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
	// The new-install script and its test are partner-only — a sync must keep
	// the partner copy.
	for _, want := range []string{
		"deploy/new-install.sh",
		"deploy/new_install_test.go",
		"deploy/release_allowed_signers",
	} {
		if !seen[want] {
			t.Fatalf("%s is not in deploy/partner_sync_carveouts", want)
		}
	}
}
