package trader

import (
	"testing"
	"time"

	"nofx/market"
	"nofx/store"
)

// W-EXEC-TRUTH W4 / D21 identity — bars belonging to ANOTHER contract must
// never drive an evaluation. The AddOn names the contract on every bar frame;
// until this wave Go parsed it nowhere, so a frame arriving for the new front
// month during a roll (or after the reconnect-without-ACK path, F7) was
// indistinguishable from this trader's own tape.

func TestPictureHtf_ForeignContractFrameNeverEvaluates(t *testing.T) {
	env := newPictureHtfEnv(t, store.PictureHtfConfig{Enabled: true, MinRR: 2.5})
	env.seedPictureTape()
	orig := pictureHtfContractOf
	pictureHtfContractOf = func(_ *AutoTrader, _ string) (string, string) { return "MNQ 12-26", "test" }
	t.Cleanup(func() { pictureHtfContractOf = orig })

	// A 5m frame stamped with a DIFFERENT contract than the trader's.
	frame := tailOf(market.FuturesBarsProvider("MNQ", "5m", 28), 1)
	for i := range frame {
		frame[i].Contract = "MNQ 09-26"
	}
	env.eval.OnBars("MNQ", "5m", frame, env.now)

	if got := env.eval.ForeignContractFrames(); got != 1 {
		t.Fatalf("a foreign-contract frame must be counted, got %d", got)
	}
	if len(env.submits) != 0 {
		t.Fatalf("a foreign-contract frame must never submit, got %d", len(env.submits))
	}
}

// The mirror: the trader's OWN contract passes, so identity cannot be
// satisfied by dropping everything.
func TestPictureHtf_OwnContractFrameStillEvaluates(t *testing.T) {
	env := newPictureHtfEnv(t, store.PictureHtfConfig{Enabled: true, MinRR: 2.5})
	env.seedPictureTape()
	orig := pictureHtfContractOf
	pictureHtfContractOf = func(_ *AutoTrader, _ string) (string, string) { return "MNQ 12-26", "test" }
	t.Cleanup(func() { pictureHtfContractOf = orig })

	frame := tailOf(market.FuturesBarsProvider("MNQ", "5m", 28), 1)
	for i := range frame {
		frame[i].Contract = "MNQ 12-26"
	}
	env.eval.OnBars("MNQ", "5m", frame, env.now)
	if got := env.eval.ForeignContractFrames(); got != 0 {
		t.Fatalf("the trader's own contract must not be counted foreign, got %d", got)
	}
	if len(env.submits) != 1 {
		t.Fatalf("the own-contract frame must evaluate as before, got %d submit(s)", len(env.submits))
	}
}

var _ = time.Second
