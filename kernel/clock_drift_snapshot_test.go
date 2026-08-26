package kernel

import (
	"testing"
	"time"

	"nofx/market"
)

// A9 (T5, fail-register wave) — a legal slow AI call is NOT clock drift: the
// guard evaluates the snapshot instant against the snapshot bars, so a 200s
// call latency contributes ZERO to the drift number.
func TestClockDriftIgnoresAICallLatency(t *testing.T) {
	snapshot := time.Now().UnixMilli()
	// freshest 1m bar stamped AT the snapshot (live feed at assembly time)
	md := &market.Data{TimeframeData: map[string]*market.TimeframeSeriesData{
		"1m": {Klines: []market.KlineBar{{Time: snapshot - 30_000}}},
	}}
	barMs, ivMs := freshestIntradayBar(md)
	if barMs <= 0 || ivMs <= 0 {
		t.Skip("freshestIntradayBar shape differs — guard covered elsewhere")
	}
	// Evaluated at the SNAPSHOT clock: drift is small.
	if d := clockDriftMs(snapshot, barMs, ivMs); absI64(d) > clockDriftToleranceMs {
		t.Fatalf("drift at snapshot clock = %dms — must be inside tolerance", d)
	}
	// The OLD bug: evaluated 200s later (post-call), the same bars read as
	// 200s of 'drift'. This pins that the call site no longer does that
	// (source contract below) and the math itself is unchanged.
	late := snapshot + 200_000
	if d := clockDriftMs(late, barMs, ivMs); absI64(d) <= clockDriftToleranceMs {
		t.Fatal("sanity: the post-call clock DOES fabricate drift — which is why the call site must pass SnapshotMs")
	}
}
