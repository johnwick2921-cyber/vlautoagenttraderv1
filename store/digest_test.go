package store

import (
	"path/filepath"
	"testing"
)

func TestDigestStore(t *testing.T) {
	st, err := New(filepath.Join(t.TempDir(), "t.db"))
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	dg := st.Digest()

	// session digests (idempotent).
	wrote, _ := dg.SaveIfAbsent(&DigestDB{Symbol: "MNQ", TradeDate: "2026-08-14", Session: "NY", Kind: "session", Text: "NY digest"})
	if !wrote {
		t.Fatalf("first session digest should write")
	}
	again, _ := dg.SaveIfAbsent(&DigestDB{Symbol: "MNQ", TradeDate: "2026-08-14", Session: "NY", Kind: "session", Text: "changed"})
	if again {
		t.Fatalf("duplicate session digest must be a no-op")
	}
	sd, _ := dg.SessionDigests("MNQ", "2026-08-14")
	if len(sd) != 1 || sd[0] != "NY digest" {
		t.Fatalf("session digests = %v", sd)
	}

	// dailies newest-first.
	for _, d := range []string{"2026-08-11", "2026-08-13", "2026-08-12"} {
		dg.SaveIfAbsent(&DigestDB{Symbol: "MNQ", TradeDate: d, Kind: "daily", Text: d})
	}
	dailies, _ := dg.RecentDailies("MNQ", 7)
	if len(dailies) != 3 || dailies[0] != "2026-08-13" {
		t.Fatalf("recent dailies newest-first wrong: %v", dailies)
	}
}
