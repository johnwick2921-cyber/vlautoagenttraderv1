package api

// Parity test for the POST /api/updates/check answers (P0 field-names hotfix,
// owner 10-02 12:2x CT; no-write fix 13:2x CT). The wire shape is built by
// internal/updatescheck — the same builders the handler answers with. This test
// never WRITES anything: it compares the builders' canonical output
// byte-for-byte against the checked-in fixture the web vitest consumes, and
// pins the exact key sets so a rename (tag -> release_id) fails here.
//
// Regenerate the fixture with:  go run ./cmd/gen-updates-check-fixture

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"vl/internal/updatescheck"
)

func TestCheckAnswersKeysAndFixtureCannotDrift(t *testing.T) {
	pair := updatescheck.FixturePair()
	vr := pair["verified_ready"].(map[string]any)
	ud := pair["up_to_date"].(map[string]any)

	wantVR := []string{"available", "checked", "ready", "reason", "source_sha", "tag", "target_commitish"}
	wantUD := []string{"available", "checked", "ready", "reason", "tag", "target_commitish"}
	if !equalSorted(keys(vr), wantVR) {
		t.Fatalf("verified_ready answer keys = %v, want %v — the web reads `tag`/`available`/`ready`; a rename breaks the button", keys(vr), wantVR)
	}
	if !equalSorted(keys(ud), wantUD) {
		t.Fatalf("up_to_date answer keys = %v, want %v", keys(ud), wantUD)
	}

	got, err := json.MarshalIndent(pair, "", "  ")
	if err != nil {
		t.Fatalf("marshal pair: %v", err)
	}
	got = append(got, '\n')

	fixturePath := filepath.Join("..", "web", "src", "lib", "api", "fixtures", "updates-check.json")
	want, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Fatalf("read fixture %s: %v — regenerate it with: go run ./cmd/gen-updates-check-fixture", fixturePath, err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("the checked-in fixture %s is STALE (drift between the Go wire shape and the web fixture). Regenerate with:\n\n  go run ./cmd/gen-updates-check-fixture\n\nbuilt:\n%s\n\nchecked in:\n%s", fixturePath, got, want)
	}
}

func keys(m map[string]any) []string {
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
