package trader

import (
	"testing"

	"nofx/market"
	"nofx/store"
)

// W-EXEC-TRUTH W4 / D25 — a stopped trader keeps receiving frames, and an
// evaluation already in flight can reach the wire after its trader is gone.

func TestPictureHtf_StopUnregistersFromTheLiveRegistry(t *testing.T) {
	at, _ := resetTrader(t, store.StrategyConfig{DayPlan: &store.DayPlanConfig{PictureHtf: &store.PictureHtfConfig{Enabled: true}, PlanEnabled: true}})
	at.isRunningMutex.Lock()
	at.isRunning = true
	at.isRunningMutex.Unlock()
	at.registerPictureHtf()
	if _, ok := pictureHtfTraders.Load(at.id); !ok {
		t.Fatalf("fixture: the trader must be registered")
	}
	t.Cleanup(func() { pictureHtfTraders.Delete(at.id) })

	at.unregisterPictureHtf()
	if _, ok := pictureHtfTraders.Load(at.id); ok {
		t.Fatalf("a stopped trader must not stay in the live-bar registry")
	}
}

// CompareAndDelete, not Delete: a RESTARTED trader has already re-registered
// under the same id, and a late Stop from the OLD instance must not evict it.
// The registry is also W3's armed-kick registry, so evicting the wrong entry
// would silently stop the armed event pass for a LIVE trader.
func TestPictureHtf_ALateStopNeverEvictsTheRestartedTrader(t *testing.T) {
	old, _ := resetTrader(t, store.StrategyConfig{DayPlan: &store.DayPlanConfig{PictureHtf: &store.PictureHtfConfig{Enabled: true}, PlanEnabled: true}})
	old.registerPictureHtf()
	t.Cleanup(func() { pictureHtfTraders.Delete(old.id) })

	// The restart: a NEW AutoTrader value registers under the same id.
	restarted, _ := resetTrader(t, store.StrategyConfig{DayPlan: &store.DayPlanConfig{PictureHtf: &store.PictureHtfConfig{Enabled: true}, PlanEnabled: true}})
	restarted.id = old.id
	restarted.registerPictureHtf()

	old.unregisterPictureHtf() // the late Stop from the OLD instance

	v, ok := pictureHtfTraders.Load(old.id)
	if !ok {
		t.Fatalf("the RESTARTED trader was evicted by the old instance's Stop — W3's armed kicks would stop with it")
	}
	if v.(*AutoTrader) != restarted {
		t.Fatalf("the registry must hold the restarted trader, not the stopped one")
	}
}

// The last gate before the wire: an evaluation that began under one generation
// must not send after a Stop or a restart. The generation is read through a
// seam so the test can move it BETWEEN the evaluation's start and its send —
// the interleaving that makes this bug real and that a wall-clock test cannot
// reproduce.
func TestPictureHtf_AGenerationChangeBeforeTheWireRefusesTheSend(t *testing.T) {
	env := newPictureHtfEnv(t, store.PictureHtfConfig{Enabled: true, MinRR: 2.5})
	env.seedPictureTape()

	orig := pictureTraderGenerationOf
	calls := 0
	pictureTraderGenerationOf = func(at *AutoTrader) int64 {
		calls++
		if calls == 1 {
			return 1 // the generation the evaluation BEGINS under
		}
		return 2 // ...restarted before it reaches the wire
	}
	t.Cleanup(func() { pictureTraderGenerationOf = orig })

	frame := tailOf(market.FuturesBarsProvider("MNQ", "5m", 28), 1)
	for i := range frame {
		frame[i].Final = true
		frame[i].EmittedAt = env.now.UnixMilli()
	}
	env.eval.OnBars("MNQ", "5m", frame, env.now)

	if len(env.submits) != 0 {
		t.Fatalf("an evaluation whose trader was restarted before the wire must NOT send, got %d", len(env.submits))
	}
	if got := env.eval.StaleTraderSends(); got != 1 {
		t.Fatalf("the refused send must be counted, got %d", got)
	}
}
