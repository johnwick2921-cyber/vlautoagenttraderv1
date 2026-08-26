package logger

import (
	"sync"
	"testing"
)

// P6 (ledger-close 2026-08-19) — the WARN+ hook fires for Warnf/Errorf (incl.
// the trader-wrapper message shape) and NOT for Infof.
func TestDBSinkShipsWarnPlusOnly(t *testing.T) {
	Init(nil)

	var mu sync.Mutex
	type shipped struct{ level, component, traderID, message string }
	var got []shipped
	AttachDBSink(func(tsMs int64, level, component, traderID, message, fieldsJSON string) {
		mu.Lock()
		defer mu.Unlock()
		got = append(got, shipped{level, component, traderID, message})
	})

	Infof("info line must NOT ship")
	Warnf("[trader_id=t9 trader_name=hoang] ⛔ feed-gate: skipped")
	Errorf("🚨 CLOCK CRITICAL [test]: drift")

	mu.Lock()
	defer mu.Unlock()
	if len(got) != 2 {
		t.Fatalf("want exactly the 2 WARN+ lines shipped, got %d: %+v", len(got), got)
	}
	if got[0].level != "warning" || got[0].traderID != "t9" {
		t.Errorf("trader id must parse from the wrapper prefix: %+v", got[0])
	}
	if got[1].level != "error" || got[1].traderID != "" {
		t.Errorf("untagged error line ships with empty trader id: %+v", got[1])
	}
	for _, g := range got {
		if g.component == "" {
			t.Errorf("component (file:line) must be present, got %+v", g)
		}
	}
}
