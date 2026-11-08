package deploy

// m470 (partner-only): the auto-sync must only take a vl tag that vl
// PUBLISHED. Since vl release #63 a tag can be refused by vl's release quality
// gate (its test suite red on the tagged code) and a HOLD is a pre-release;
// before m470 the sync took the newest v* tag by name and shipped it anyway.
//
// These tests run the real deploy/vl-published-tag.sh with a stand-in `gh`
// that answers from a table, so they need bash but no network.

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// fakeGhReleaseTags writes a stand-in gh that answers `gh api repos/<slug>/releases/tags/<tag> --jq ...`
// from states: "published" -> "false false", "held" -> "false true",
// "missing" -> HTTP 404, "error" -> a 500.
func fakeGhReleaseTags(t *testing.T, states map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	var b strings.Builder
	b.WriteString("#!/usr/bin/env bash\nurl=\"$2\"; tag=\"${url##*/}\"\ncase \"$tag\" in\n")
	for tag, st := range states {
		switch st {
		case "published":
			b.WriteString("  " + tag + ") echo 'false false'; exit 0;;\n")
		case "held":
			b.WriteString("  " + tag + ") echo 'false true'; exit 0;;\n")
		case "error":
			b.WriteString("  " + tag + ") echo 'gh: Server Error (HTTP 500)' >&2; exit 1;;\n")
		}
	}
	b.WriteString("  *) echo 'gh: Not Found (HTTP 404)' >&2; exit 1;;\nesac\n")
	p := filepath.Join(dir, "gh")
	if err := os.WriteFile(p, []byte(b.String()), 0o755); err != nil {
		t.Fatal(err)
	}
	return p
}

// pickNewestPublished runs newest_published_vl_tag over the given tags.
func pickNewestPublished(t *testing.T, gh string, tags []string) (string, string, int) {
	t.Helper()
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("bash not available")
	}
	script, err := filepath.Abs("vl-published-tag.sh")
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("bash", "-c", `. "$1"; newest_published_vl_tag`, "x", script)
	cmd.Env = append(os.Environ(), "GH_BIN="+gh, "VL_SLUG=johnwick2921-cyber/vl")
	cmd.Stdin = strings.NewReader(strings.Join(tags, "\n") + "\n")
	var out, errb strings.Builder
	cmd.Stdout, cmd.Stderr = &out, &errb
	rc := 0
	if err := cmd.Run(); err != nil {
		ee, ok := err.(*exec.ExitError)
		if !ok {
			t.Fatal(err)
		}
		rc = ee.ExitCode()
	}
	return strings.TrimSpace(out.String()), errb.String(), rc
}

func TestAutoSyncSkipsATagVlDidNotPublish(t *testing.T) {
	gh := fakeGhReleaseTags(t, map[string]string{
		"v2026.11.04.1": "published",
		"v2026.11.07.1": "missing", // refused by the vl quality gate
		"v2026.11.06.1": "held",    // HOLD = pre-release
	})
	got, logs, rc := pickNewestPublished(t, gh, []string{"v2026.11.04.1", "v2026.11.07.1", "v2026.11.06.1", "v2026.10.31.1"})
	if rc != 0 || got != "v2026.11.04.1" {
		t.Fatalf("picked %q rc=%d, want v2026.11.04.1 (the newest PUBLISHED tag); log:\n%s", got, rc, logs)
	}
	for _, want := range []string{"v2026.11.07.1 is unpublished", "v2026.11.06.1 is held"} {
		if !strings.Contains(logs, want) {
			t.Fatalf("run log must say why a newer tag was skipped (%q); got:\n%s", want, logs)
		}
	}
}

func TestAutoSyncSortsTagsByVersionNotByName(t *testing.T) {
	gh := fakeGhReleaseTags(t, map[string]string{"v2026.11.10.1": "published", "v2026.11.9.1": "published"})
	got, _, rc := pickNewestPublished(t, gh, []string{"v2026.11.9.1", "v2026.11.10.1"})
	if rc != 0 || got != "v2026.11.10.1" {
		t.Fatalf("picked %q rc=%d, want v2026.11.10.1", got, rc)
	}
}

// A read failure that is not a 404 must stop the sync, never be read as
// "unpublished" (which would quietly fall back to an older tag).
func TestAutoSyncRefusesWhenVlReleasesCannotBeRead(t *testing.T) {
	gh := fakeGhReleaseTags(t, map[string]string{"v2026.11.07.1": "error", "v2026.11.04.1": "published"})
	got, _, rc := pickNewestPublished(t, gh, []string{"v2026.11.04.1", "v2026.11.07.1"})
	if rc != 2 || got != "" {
		t.Fatalf("got %q rc=%d, want rc=2 and no tag", got, rc)
	}
}

func TestAutoSyncNothingPublishedReturnsOne(t *testing.T) {
	gh := fakeGhReleaseTags(t, map[string]string{})
	got, _, rc := pickNewestPublished(t, gh, []string{"v2026.11.07.1"})
	if rc != 1 || got != "" {
		t.Fatalf("got %q rc=%d, want rc=1", got, rc)
	}
}

// The sync script must use the published-tag picker on both the automatic and
// the explicit NEW_TAG paths.
func TestAutoSyncScriptUsesThePublishedPicker(t *testing.T) {
	s := repoFile(t, "deploy/vl-auto-sync.sh")
	for _, want := range []string{
		`. "$ROOT/deploy/vl-published-tag.sh"`,
		"newest_published_vl_tag",
		`vl_release_state "$NEW_TAG"`,
	} {
		if !strings.Contains(s, want) {
			t.Fatalf("deploy/vl-auto-sync.sh must contain %q (m470)", want)
		}
	}
	if strings.Contains(s, "sort -V | tail -1") {
		t.Fatal("deploy/vl-auto-sync.sh still picks the newest tag by name (sort -V | tail -1)")
	}
}
