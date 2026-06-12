package kernel

import (
	"strings"
	"testing"
	"time"
)

func TestRiskLimitsCheckPreTrade(t *testing.T) {
	base := RiskLimits{
		MaxDailyLossUSD:      500,
		MaxConcurrentTrades:  2,
		MaxNotionalUSD:       50_000,
		MaxContractsPerOrder: 5,
	}

	cases := []struct {
		name              string
		limits            RiskLimits
		totalPnL          float64
		openPositions     int
		requestedNotional float64
		existingNotional  float64
		wantErr           bool
		wantSubstr        string
	}{
		{
			name:              "pnl_under_limit_no_error",
			limits:            base,
			totalPnL:          -200,
			openPositions:     0,
			requestedNotional: 10_000,
			wantErr:           false,
		},
		{
			name:              "pnl_at_negative_limit_boundary_no_error",
			limits:            base,
			totalPnL:          -500, // == -MaxDailyLossUSD, plan's `<` semantics → permissive
			openPositions:     0,
			requestedNotional: 10_000,
			wantErr:           false,
		},
		{
			name:              "pnl_just_past_limit_triggers",
			limits:            base,
			totalPnL:          -500.01,
			openPositions:     0,
			requestedNotional: 10_000,
			wantErr:           true,
			wantSubstr:        "daily loss",
		},
		{
			name:              "pnl_well_past_limit_triggers",
			limits:            base,
			totalPnL:          -10_000,
			openPositions:     0,
			requestedNotional: 0,
			wantErr:           true,
			wantSubstr:        "daily loss",
		},
		{
			name:              "concurrent_at_cap_triggers",
			limits:            base,
			totalPnL:          0,
			openPositions:     2,
			requestedNotional: 1_000,
			wantErr:           true,
			wantSubstr:        "concurrent",
		},
		{
			name:              "concurrent_below_cap_ok",
			limits:            base,
			totalPnL:          0,
			openPositions:     1,
			requestedNotional: 1_000,
			wantErr:           false,
		},
		{
			name:              "notional_exceeded_triggers",
			limits:            base,
			totalPnL:          0,
			openPositions:     0,
			requestedNotional: 30_000,
			existingNotional:  25_000,
			wantErr:           true,
			wantSubstr:        "notional",
		},
		{
			name:              "notional_exactly_at_cap_no_error",
			limits:            base,
			totalPnL:          0,
			openPositions:     0,
			requestedNotional: 25_000,
			existingNotional:  25_000, // total = 50_000 == cap, strict-exceed semantics
			wantErr:           false,
		},
		{
			name:              "all_in_range_no_error",
			limits:            base,
			totalPnL:          100,
			openPositions:     1,
			requestedNotional: 5_000,
			existingNotional:  5_000,
			wantErr:           false,
		},
		{
			name:              "zero_limits_disabled_no_error",
			limits:            RiskLimits{},
			totalPnL:          -1_000_000,
			openPositions:     999,
			requestedNotional: 999_999_999,
			existingNotional:  999_999_999,
			wantErr:           false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.limits.CheckPreTrade(tc.totalPnL, tc.openPositions, tc.requestedNotional, tc.existingNotional)
			if tc.wantErr && err == nil {
				t.Fatalf("expected error containing %q, got nil", tc.wantSubstr)
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if tc.wantErr && tc.wantSubstr != "" && !strings.Contains(err.Error(), tc.wantSubstr) {
				t.Fatalf("error %q missing substring %q", err.Error(), tc.wantSubstr)
			}
		})
	}
}

func TestRiskLimitsClassify(t *testing.T) {
	r := RiskLimits{
		MaxDailyLossUSD:     500,
		MaxConcurrentTrades: 2,
		MaxNotionalUSD:      50_000,
	}

	t.Run("allow_when_clean", func(t *testing.T) {
		d, err := r.Classify(0, 0, 1_000, 0)
		if d != RiskAllow || err != nil {
			t.Fatalf("want Allow nil, got %d %v", d, err)
		}
	})

	t.Run("force_flat_on_daily_loss", func(t *testing.T) {
		d, err := r.Classify(-600, 0, 0, 0)
		if d != RiskForceFlat || err == nil {
			t.Fatalf("want ForceFlat err, got %d %v", d, err)
		}
	})

	t.Run("block_entry_on_concurrent_cap", func(t *testing.T) {
		d, err := r.Classify(0, 2, 1_000, 0)
		if d != RiskBlockEntry || err == nil {
			t.Fatalf("want BlockEntry err, got %d %v", d, err)
		}
	})

	t.Run("nil_receiver_allows", func(t *testing.T) {
		var nilR *RiskLimits
		d, err := nilR.Classify(-99999, 99, 99999, 99999)
		if d != RiskAllow || err != nil {
			t.Fatalf("nil receiver should allow, got %d %v", d, err)
		}
	})
}

func TestMaybeResetDaily(t *testing.T) {
	// First call always resets (last == "").
	dailyResetMu.Lock()
	lastDailyResetDate = ""
	dailyResetMu.Unlock()

	now := time.Date(2026, 5, 26, 12, 0, 0, 0, time.UTC)
	if !MaybeResetDaily(now) {
		t.Fatalf("first call should fire reset")
	}
	if MaybeResetDaily(now) {
		t.Fatalf("second call same day should not reset")
	}
	tomorrow := now.Add(24 * time.Hour)
	if !MaybeResetDaily(tomorrow) {
		t.Fatalf("next day should reset")
	}
}
