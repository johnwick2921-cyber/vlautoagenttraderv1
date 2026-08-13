package kernel

import (
	"strings"
	"testing"

	"nofx/store"
)

// TestFormatKlineTimeframes_Golden locks the "Available Data" timeframe summary
// line. The empty-SelectedTimeframes cases MUST stay byte-identical to the
// pre-fix wording (legacy configs unchanged); the populated cases list every
// selected timeframe (the fix — previously only primary + longer were named).
func TestFormatKlineTimeframes_Golden(t *testing.T) {
	cases := []struct {
		name string
		in   store.KlineConfig
		want string
	}{
		{
			name: "legacy empty multi (byte-identical fallback)",
			in:   store.KlineConfig{PrimaryTimeframe: "5m", LongerTimeframe: "1h", EnableMultiTimeframe: true},
			want: "- 5m price series + 1h K-line series\n",
		},
		{
			name: "legacy empty single (byte-identical fallback)",
			in:   store.KlineConfig{PrimaryTimeframe: "3m", EnableMultiTimeframe: false},
			want: "- 3m price series\n",
		},
		{
			name: "three selected (MNQ SIM Default 5m/15m/1h)",
			in: store.KlineConfig{
				PrimaryTimeframe: "5m", LongerTimeframe: "1h", EnableMultiTimeframe: true,
				SelectedTimeframes: []string{"5m", "15m", "1h"},
			},
			want: "- 5m, 15m, 1h price series\n",
		},
		{
			name: "three selected (longer=4h not in set: 15m/1h/4h)",
			in: store.KlineConfig{
				PrimaryTimeframe: "15m", LongerTimeframe: "4h", EnableMultiTimeframe: true,
				SelectedTimeframes: []string{"15m", "1h", "4h"},
			},
			want: "- 15m, 1h, 4h price series\n",
		},
		{
			name: "single selected",
			in:   store.KlineConfig{PrimaryTimeframe: "5m", SelectedTimeframes: []string{"5m"}},
			want: "- 5m price series\n",
		},
		{
			// De-cap proof (>4, above the old artificial max): every selected
			// timeframe must appear in the line — nothing is truncated to 4.
			name: "seven selected (above the old max-4 cap)",
			in: store.KlineConfig{
				PrimaryTimeframe: "5m", EnableMultiTimeframe: true,
				SelectedTimeframes: []string{"1m", "5m", "15m", "1h", "4h", "1d", "1w"},
			},
			want: "- 1m, 5m, 15m, 1h, 4h, 1d, 1w price series\n",
		},
		{
			// B10 (audit F7): primary_timeframe is always fetched + rendered, so it
			// must be ANNOUNCED even when the user omitted it from
			// selected_timeframes (the 15m case from the rules audit). Appended.
			name: "primary not in selected (F7: 15m primary announced)",
			in: store.KlineConfig{
				PrimaryTimeframe: "15m", EnableMultiTimeframe: true,
				SelectedTimeframes: []string{"5m", "1h", "3m", "4h", "1d"},
			},
			want: "- 5m, 1h, 3m, 4h, 1d, 15m price series\n",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := formatKlineTimeframes(tc.in)
			if got != tc.want {
				t.Fatalf("formatKlineTimeframes(%+v)\n got:  %q\n want: %q", tc.in, got, tc.want)
			}
		})
	}
}

// TestWriteAvailableIndicators_TimeframeLine proves the fix end-to-end through
// the real engine method: a 3-timeframe selection lists all three (and never the
// old 2-timeframe "primary + longer" form), while a legacy config (no
// SelectedTimeframes) keeps the exact pre-fix line — the golden invariant.
func TestWriteAvailableIndicators_TimeframeLine(t *testing.T) {
	// MNQ SIM Default shape: 5m/15m/1h selected.
	cfg := &store.StrategyConfig{
		Indicators: store.IndicatorConfig{
			Klines: store.KlineConfig{
				PrimaryTimeframe:     "5m",
				LongerTimeframe:      "1h",
				EnableMultiTimeframe: true,
				SelectedTimeframes:   []string{"5m", "15m", "1h"},
			},
		},
	}
	var sb strings.Builder
	NewStrategyEngine(cfg).writeAvailableIndicators(&sb)
	out := sb.String()
	if !strings.Contains(out, "- 5m, 15m, 1h price series\n") {
		t.Fatalf("expected all 3 selected timeframes listed; got:\n%s", out)
	}
	if strings.Contains(out, "5m price series + 1h K-line series") {
		t.Fatalf("must NOT emit the legacy 2-timeframe form when 3 are selected; got:\n%s", out)
	}

	// Legacy config (no SelectedTimeframes) → byte-identical old wording.
	legacy := &store.StrategyConfig{
		Indicators: store.IndicatorConfig{
			Klines: store.KlineConfig{
				PrimaryTimeframe:     "5m",
				LongerTimeframe:      "4h",
				EnableMultiTimeframe: true,
			},
		},
	}
	var lb strings.Builder
	NewStrategyEngine(legacy).writeAvailableIndicators(&lb)
	if !strings.Contains(lb.String(), "- 5m price series + 4h K-line series\n") {
		t.Fatalf("legacy empty-SelectedTimeframes config must keep old wording; got:\n%s", lb.String())
	}
}
