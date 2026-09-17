package kernel

import (
	"testing"
	"time"

	"nofx/market"
)

// TestDetectHTFLevelsExportPinned — F0 (CTO review 2026-09-17): the replay
// harness calls DetectHTFLevelsExport, so the symbol must exist as a REAL
// export (not a _test-only helper) and keep working. This pin is the reason the
// export lives in a non-test file.
func TestDetectHTFLevelsExportPinned(t *testing.T) {
	bars := func(tf string, count int) []market.Kline {
		return []market.Kline{
			{OpenTime: time.Date(2026, 9, 16, 14, 0, 0, 0, time.UTC).UnixMilli(), Open: 100, High: 101, Low: 99, Close: 100.5},
		}
	}
	now := time.Date(2026, 9, 16, 16, 31, 0, 0, time.UTC)
	levels, rep := DetectHTFLevelsExport(bars, []string{"1h"}, "MNQ", now)
	if rep == nil {
		t.Fatal("export must return a non-nil detection report")
	}
	_ = levels // the detector returns its honest-empty result; the pin is the export existing and running
}
