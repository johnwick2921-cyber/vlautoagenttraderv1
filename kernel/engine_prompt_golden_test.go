package kernel

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var update = flag.Bool("update", false, "update golden files in kernel/testdata/golden/")

// aggressiveFuturesConfig is a higher-risk-tolerance configuration: wider stop
// envelope, lower minimum R/R. Used to snapshot how the system prompt varies
// when the strategist accepts more risk per trade.
func aggressiveFuturesConfig() FuturesPromptConfig {
	return FuturesPromptConfig{
		Symbol:             "NQ",
		ContractMultiplier: 20.0, // NQ = $20/point
		TickSize:           0.25,
		MinStopPoints:      20,
		MaxStopPoints:      80,
		MinRiskReward:      1.2,
	}
}

// conservativeFuturesConfig is a lower-risk-tolerance configuration: tighter
// stops, higher minimum R/R. Used to snapshot the conservative variant.
func conservativeFuturesConfig() FuturesPromptConfig {
	return FuturesPromptConfig{
		Symbol:             "MNQ",
		ContractMultiplier: 2.0, // MNQ = $2/point
		TickSize:           0.25,
		MinStopPoints:      10,
		MaxStopPoints:      30,
		MinRiskReward:      2.0,
	}
}

// basicFuturesContext is a representative single-bar snapshot of MNQ near the
// 21500 area with indicators populated to plausible mid-session values.
func basicFuturesContext() FuturesContext {
	return FuturesContext{
		Symbol:       "MNQM6",
		CurrentPrice: 21512.75,
		EMA20:        21498.50,
		EMA50:        21472.25,
		RSI14:        62.4,
		MACD:         4.85,
		ATR14:        11.75,
		BollUpper:    21545.00,
		BollLower:    21455.50,
	}
}

func TestBuildFuturesSystemPrompt_Aggressive(t *testing.T) {
	got := BuildFuturesSystemPrompt(aggressiveFuturesConfig())
	assertGolden(t, "futures_system_aggressive.txt", got)
}

func TestBuildFuturesSystemPrompt_Conservative(t *testing.T) {
	got := BuildFuturesSystemPrompt(conservativeFuturesConfig())
	assertGolden(t, "futures_system_conservative.txt", got)
}

func TestBuildFuturesUserPrompt_Basic(t *testing.T) {
	got := BuildFuturesUserPrompt(basicFuturesContext())
	assertGolden(t, "futures_user_basic.txt", got)
}

func assertGolden(t *testing.T, name, got string) {
	t.Helper()
	path := filepath.Join("testdata", "golden", name)
	if *update {
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(got), 0644); err != nil {
			t.Fatal(err)
		}
		t.Logf("updated %s (%d bytes)", path, len(got))
		return
	}
	wantBytes, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read golden %s: %v\n(run `go test ./kernel/... -run %s -update` to create)", path, err, t.Name())
	}
	want := string(wantBytes)
	if got != want {
		t.Fatalf("prompt mismatch for %s. Run with -update if intentional.\n--- diff (first 30 differing lines) ---\n%s", name, lineDiff(got, want, 30))
	}
}

func lineDiff(got, want string, maxLines int) string {
	gotLines := strings.Split(got, "\n")
	wantLines := strings.Split(want, "\n")
	var b strings.Builder
	n := len(gotLines)
	if len(wantLines) > n {
		n = len(wantLines)
	}
	shown := 0
	for i := 0; i < n && shown < maxLines; i++ {
		var g, w string
		if i < len(gotLines) {
			g = gotLines[i]
		}
		if i < len(wantLines) {
			w = wantLines[i]
		}
		if g != w {
			b.WriteString(fmt.Sprintf("L%d:\n  got:  %q\n  want: %q\n", i+1, g, w))
			shown++
		}
	}
	return b.String()
}
