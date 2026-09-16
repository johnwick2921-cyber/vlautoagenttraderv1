// F12 — the build-id boot line.
//
// CLASS 6: a distributed change is proven only by a RECEIVED far-side frame.
// The AddOn's VL_BUILD_ID has been bumped in OUR source since 2026-08-30-e7,
// but NT8 keeps running whatever DLL was last compiled — so a line that read
// the source constant would report success for a change that never landed.
// These pin that the line reports what was RECEIVED, and says NO until it
// matches.

package ninjatrader

import (
	"strings"
	"testing"
	"time"
)

func TestBuildIDLineSaysNoWhenNothingWasReceived(t *testing.T) {
	line := AddonBuildLine("", ExpectedAddonBuild)
	if !strings.Contains(line, "build_id=none") {
		t.Errorf("no received build must read none, got %q", line)
	}
	if !strings.Contains(line, "match=NO") {
		t.Errorf("unknown must never read as a match: %q", line)
	}
	// It must NOT quietly report the source constant as if it were received.
	if strings.Count(line, ExpectedAddonBuild) != 1 {
		t.Errorf("the expected build may appear only as `expected=`, got %q", line)
	}
}

func TestBuildIDLineSaysNoOnAStaleAddOn(t *testing.T) {
	line := AddonBuildLine("2026-08-30-e7", "2026-09-03-f12")
	if !strings.Contains(line, "match=NO") {
		t.Fatalf("a stale DLL must read NO: %q", line)
	}
	if !strings.Contains(line, "2026-08-30-e7") || !strings.Contains(line, "2026-09-03-f12") {
		t.Errorf("the line must quote BOTH builds so the reader can see the gap: %q", line)
	}
}

func TestBuildIDLineSaysYesOnlyOnAnExactMatch(t *testing.T) {
	line := AddonBuildLine("2026-09-03-f12", "2026-09-03-f12")
	if !strings.Contains(line, "match=yes") {
		t.Fatalf("an exact match must read yes: %q", line)
	}
}

// The snapshot half of the line: age and count READ from the cache, and an
// explicit "none" when there is no book — never age=0, which would read as a
// book received this instant.
func TestSnapshotLineSaysNoneWithNoBook(t *testing.T) {
	c := NewOrderSnapshotCache()
	now := time.Date(2026, 9, 3, 20, 0, 0, 0, time.UTC)
	line := OrderSnapshotLineAt(c, "Sim101", "MNQ", now)
	if !strings.Contains(line, "none") {
		t.Errorf("no book must say none, got %q", line)
	}
	if strings.Contains(line, "age=0") {
		t.Errorf("a missing book must never render age=0: %q", line)
	}
	if !strings.Contains(line, "source=ledger") {
		t.Errorf("with no book leg 4 is the ledger's answer and the line must say so: %q", line)
	}
}

func TestSnapshotLineReportsAgeCountAndBrokerSource(t *testing.T) {
	c := NewOrderSnapshotCache()
	now := time.Date(2026, 9, 3, 20, 0, 0, 0, time.UTC)
	p, err := ParseOrderSnapshot([]byte(`{"account":"Sim101","build_id":"2026-09-03-f12","emitted_at_ms":1,"orders":[
	  {"order_id":"1","state":"Working","type":"stop","symbol":"MNQ","quantity":1},
	  {"order_id":"2","state":"Filled","type":"limit","symbol":"MNQ","quantity":1}]}`))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	c.PutAt(p, now.Add(-12*time.Second))

	line := OrderSnapshotLineAt(c, "Sim101", "MNQ", now)
	if !strings.Contains(line, "age=12s") {
		t.Errorf("age must be READ from the cache: %q", line)
	}
	// orders= is the WORKING count, not the raw list length: a filled order is
	// not a working order and leg 4 counts the same subset.
	if !strings.Contains(line, "orders=1") {
		t.Errorf("orders must be the working count (1 of 2), got %q", line)
	}
	if !strings.Contains(line, "source=broker") {
		t.Errorf("a fresh book means leg 4 reads the broker: %q", line)
	}
}
