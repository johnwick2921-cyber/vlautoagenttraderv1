package ninjatrader

import (
	"bytes"
	"strings"
	"testing"

	"nofx/logger"
)

// P2-18 RED→GREEN — an UNSET NT_TRANSPORT must not fall back silently: the
// production call site NewTraderFromEnv prints a WARN boot line naming the
// effective (deprecated CSV) transport, while the constructed trader is still
// the CSV one (behaviour preserved — only the silence is fixed). RED: remove
// the WARN and the captured log lacks the line.
func TestNewTraderFromEnvUnsetTransportWarns(t *testing.T) {
	t.Setenv(TransportEnvVar, "")

	var buf bytes.Buffer
	prev := logger.Log.Out
	logger.Log.SetOutput(&buf)
	t.Cleanup(func() { logger.Log.SetOutput(prev) })

	tr, err := NewTraderFromEnv(Config{Symbol: "MNQ", Account: "Sim101"})
	if err != nil {
		t.Fatalf("unset transport must still construct (today's default): %v", err)
	}
	if _, ok := tr.(*Trader); !ok {
		t.Fatalf("unset transport must resolve to the CSV trader (today's behavior), got %T", tr)
	}
	out := buf.String()
	if !strings.Contains(out, "NT_TRANSPORT") || !strings.Contains(out, "UNSET") {
		t.Fatalf("unset transport did not warn naming the effective transport; log: %q", out)
	}
	if !strings.Contains(out, "effective transport: csv") {
		t.Fatalf("warn must NAME the effective transport as csv; log: %q", out)
	}
}

// P2-18 — an EXPLICIT csv stays silent (the operator chose it) and unknown
// values keep failing fast.
func TestNewTraderFromEnvExplicitCSVSilentAndUnknownFails(t *testing.T) {
	t.Setenv(TransportEnvVar, "csv")
	var buf bytes.Buffer
	prev := logger.Log.Out
	logger.Log.SetOutput(&buf)
	t.Cleanup(func() { logger.Log.SetOutput(prev) })

	tr, err := NewTraderFromEnv(Config{Symbol: "MNQ", Account: "Sim101"})
	if err != nil {
		t.Fatalf("explicit csv must construct: %v", err)
	}
	if _, ok := tr.(*Trader); !ok {
		t.Fatalf("explicit csv must be the CSV trader, got %T", tr)
	}
	if strings.Contains(buf.String(), "UNSET") {
		t.Fatalf("explicit csv must not warn about an unset transport; log: %q", buf.String())
	}

	t.Setenv(TransportEnvVar, "carrier-pigeon")
	if _, err := NewTraderFromEnv(Config{Symbol: "MNQ", Account: "Sim101"}); err == nil {
		t.Fatal("unknown transport must fail fast")
	}
}
