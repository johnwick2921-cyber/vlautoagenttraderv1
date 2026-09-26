package trader

import (
	"strings"
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

// CTO ruling on push 4: UNKNOWN identity does not drop the frame, but the
// window must be MEASURED rather than silent — a counter, the boot line, and
// one WARN per window.

func TestPictureHtf_UnknownContractIsCountedAndBootLineReadsIt(t *testing.T) {
	env := newPictureHtfEnv(t, store.PictureHtfConfig{Enabled: true, MinRR: 2.5})
	env.seedPictureTape()
	ev := env.at.pictureHtfEvaluator()
	if ev == nil {
		t.Fatalf("fixture: the trader must own an evaluator")
	}
	orig := pictureHtfContractOf
	// No subscribed ACK yet: the trader's own side is unknown.
	pictureHtfContractOf = func(_ *AutoTrader, _ string) (string, string) { return "", "none" }
	t.Cleanup(func() { pictureHtfContractOf = orig })

	frame := tailOf(market.FuturesBarsProvider("MNQ", "5m", 28), 1)
	for i := range frame {
		frame[i].Contract = "MNQ 12-26"
	}
	ev.OnBars("MNQ", "5m", frame, env.now)

	if got := ev.UnknownContractFrames(); got != 1 {
		t.Fatalf("a frame evaluated without contract identity must be counted, got %d", got)
	}
	line := env.at.pictureHtfBootLineAt(env.now)
	for _, want := range []string{"contract=n/a", "foreign=0", "unknown=1"} {
		if !strings.Contains(line, want) {
			t.Fatalf("the boot line must READ %q; got %q", want, line)
		}
	}
}

func TestPictureHtf_UnknownContractWarnsOncePerWindow(t *testing.T) {
	env := newPictureHtfEnv(t, store.PictureHtfConfig{Enabled: true, MinRR: 2.5})
	env.seedPictureTape()
	ev := env.at.pictureHtfEvaluator()
	orig := pictureHtfContractOf
	mine := ""
	pictureHtfContractOf = func(_ *AutoTrader, _ string) (string, string) { return mine, "test" }
	t.Cleanup(func() { pictureHtfContractOf = orig })

	frame := tailOf(market.FuturesBarsProvider("MNQ", "5m", 28), 1)
	for i := range frame {
		frame[i].Contract = "MNQ 12-26"
	}
	ev.OnBars("MNQ", "5m", frame, env.now) // window opens, no warn yet
	if got := ev.UnknownContractWarnings(); got != 0 {
		t.Fatalf("no warning before the window elapses, got %d", got)
	}
	ev.OnBars("MNQ", "5m", frame, env.now.Add(61*time.Second))
	if got := ev.UnknownContractWarnings(); got != 1 {
		t.Fatalf("one warning once the window elapses, got %d", got)
	}
	ev.OnBars("MNQ", "5m", frame, env.now.Add(200*time.Second))
	if got := ev.UnknownContractWarnings(); got != 1 {
		t.Fatalf("exactly ONE warning per window, got %d", got)
	}
	// Identity returns: the window closes and re-arms for the next outage.
	mine = "MNQ 12-26"
	ev.OnBars("MNQ", "5m", frame, env.now.Add(210*time.Second))
	mine = ""
	ev.OnBars("MNQ", "5m", frame, env.now.Add(220*time.Second))
	ev.OnBars("MNQ", "5m", frame, env.now.Add(300*time.Second))
	if got := ev.UnknownContractWarnings(); got != 2 {
		t.Fatalf("a NEW unknown window must warn again, got %d", got)
	}
}
