package api

// P0 field-names hotfix (owner 10-02 12:2x CT): the parity test that cannot
// drift. This test serialises the REAL verified_ready / up_to_date answer
// builders into web/src/lib/api/fixtures/updates-check.json — the exact JSON
// the handler puts on the wire — and the vitest side consumes that fixture to
// drive the Updates page. It also pins the exact key sets, so a rename
// (tag -> release_id) goes RED here before the web ever reads a stale name.

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"testing"
)

func TestCheckAnswersKeysAndFixtureCannotDrift(t *testing.T) {
	vr := checkAnswerVerifiedReady(checkDetail{
		Tag:             "v2026.10.02.3",
		TargetCommitish: "dev",
		SourceSHA:       "0123456789abcdef",
	})
	ud := checkAnswerUpToDate(checkDetail{
		Tag:             "v2026.10.02.3",
		TargetCommitish: "dev",
	})

	vrKeys := keys(vr)
	udKeys := keys(ud)

	wantVR := []string{"available", "checked", "ready", "reason", "source_sha", "tag", "target_commitish"}
	wantUD := []string{"available", "checked", "ready", "reason", "tag", "target_commitish"}
	if !equalSorted(vrKeys, wantVR) {
		t.Fatalf("verified_ready answer keys = %v, want %v — the web reads `tag`/`available`/`ready`; a rename breaks the button", vrKeys, wantVR)
	}
	if !equalSorted(udKeys, wantUD) {
		t.Fatalf("up_to_date answer keys = %v, want %v", udKeys, wantUD)
	}
	if vr["tag"] != "v2026.10.02.3" || ud["available"] != false || vr["available"] != true || vr["ready"] != true {
		t.Fatalf("answer values wrong: vr=%v ud=%v", vr, ud)
	}

	fixture := map[string]interface{}{
		"verified_ready": vr,
		"up_to_date":     ud,
	}
	b, err := json.MarshalIndent(fixture, "", "  ")
	if err != nil {
		t.Fatalf("marshal fixture: %v", err)
	}
	b = append(b, '\n')
	dir := filepath.Join("..", "web", "src", "lib", "api", "fixtures")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir fixture dir: %v", err)
	}
	path := filepath.Join(dir, "updates-check.json")
	if err := os.WriteFile(path, b, 0o644); err != nil {
		t.Fatalf("write fixture %s: %v", path, err)
	}
}

func keys(m map[string]interface{}) []string {
	var ks []string
	for k := range m {
		ks = append(ks, k)
	}
	sort.Strings(ks)
	return ks
}

func equalSorted(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
