package ninjatrader

import (
	"strings"
	"testing"
)

// E1 — the boot summary renders RESOLVED values from a fake ring for all five
// tfs: received/asked per tf, n/a for a tf with no frame, and a ZERO frame is
// called out as the 09-16 shape.
func TestHistoryAtSubscribeLineRendersResolvedValues(t *testing.T) {
	var h historyAtSubscribe
	h.note("MNQ", "1m", 2000)
	h.note("MNQ", "5m", 2000)
	h.note("MNQ", "15m", 2000)
	h.note("MNQ", "1h", 1536)
	h.note("MNQ", "4h", 0) // a reconnect that ran while the feed was down
	// 1d: no frame at all

	tfs := ladderOrder([]string{"4h", "1m", "1d", "5m", "1h", "15m"})
	line := HistoryAtSubscribeLine("MNQ", 2000, h.received, h.frames, tfs)

	for _, want := range []string{"1m=2000/2000", "5m=2000/2000", "15m=2000/2000", "1h=1536/2000", "4h=0/2000", "1d=n/a/2000"} {
		if !strings.Contains(line, want) {
			t.Errorf("line missing %q: %s", want, line)
		}
	}
	if !strings.Contains(line, "1 tf(s) received ZERO") {
		t.Errorf("a zero frame must be called out: %s", line)
	}
	// ladder order: 1m before 5m before 15m before 1h before 4h before 1d
	if strings.Index(line, "1m=") > strings.Index(line, "5m=") || strings.Index(line, "4h=") > strings.Index(line, "1d=") {
		t.Errorf("not in ladder order: %s", line)
	}
}

// A tf that never received a frame renders n/a, never 0 — an absence is not a
// measurement (A24).
func TestHistoryAtSubscribeAbsentIsNotZero(t *testing.T) {
	var h historyAtSubscribe
	line := HistoryAtSubscribeLine("MNQ", 2000, h.received, h.frames, []string{"5m"})
	if strings.Contains(line, "5m=0/") || !strings.Contains(line, "5m=n/a/2000") {
		t.Fatalf("absent must be n/a: %s", line)
	}
}
