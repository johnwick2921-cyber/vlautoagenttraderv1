package trader

import (
	"testing"

	"nofx/market"
	"nofx/store"
)

// realDailyMNQ — 20 real MNQ daily bars (07-20..08-14, Globex full-session, from
// decision_records) used to render a REAL INDICATORS mirror sample for the report.
func realDailyMNQ() []market.Kline {
	rows := [][4]float64{
		{28747.50, 29192.50, 28706.75, 28778.75}, {28783.25, 29363.50, 28700.00, 29316.00},
		{29309.00, 29342.50, 28961.25, 29181.25}, {29107.50, 29283.00, 28432.50, 28620.75},
		{28708.00, 28734.75, 28212.50, 28282.25}, {28501.50, 28763.75, 27938.50, 28190.00},
		{28210.50, 28229.00, 27603.25, 27922.00}, {27962.25, 28177.25, 27200.00, 27259.75},
		{27208.00, 28410.00, 27204.75, 28237.75}, {28317.25, 28725.75, 28079.75, 28404.25},
		{28567.50, 28965.00, 28313.00, 28891.75}, {28930.00, 29956.50, 28831.50, 29863.50},
		{29781.25, 30073.25, 29530.75, 29615.00}, {29576.00, 29686.25, 29241.00, 29488.25},
		{29515.00, 29867.25, 29455.00, 29834.75}, {29851.50, 29985.00, 29719.00, 29737.00},
		{29764.25, 29887.00, 29533.50, 29626.00}, {29663.00, 30001.50, 29625.00, 29853.25},
		{29820.25, 30273.25, 29780.50, 30188.50}, {30148.75, 30287.25, 30025.00, 30147.25},
	}
	var out []market.Kline
	t := int64(1_700_000_000_000)
	for _, r := range rows {
		out = append(out, market.Kline{OpenTime: t, Open: r[0], High: r[1], Low: r[2], Close: r[3], CloseTime: t + 86_400_000 - 1})
		t += 86_400_000
	}
	return out
}

// TestW11SampleIndicatorBlock logs a REAL assembled INDICATORS mirror (real MNQ
// daily bars → real EMA/ATR/RSI via the executor's compute path) for the report.
func TestW11SampleIndicatorBlock(t *testing.T) {
	prev := market.FuturesBarsProvider
	defer func() { market.FuturesBarsProvider = prev }()
	market.FuturesBarsProvider = func(symbol, tf string, count int) []market.Kline {
		if tf == "1d" {
			return realDailyMNQ()
		}
		return nil
	}
	at := &AutoTrader{
		id: "t1", exchange: "ninjatrader",
		config: AutoTraderConfig{
			NinjaTraderSymbol: "MNQ",
			StrategyConfig: &store.StrategyConfig{
				DayPlan: &store.DayPlanConfig{PlanEnabled: true},
				Indicators: store.IndicatorConfig{
					EnableEMA: true, EnableATR: true, EnableRSI: true,
					EMAPeriods: []int{20}, ATRPeriods: []int{14}, RSIPeriods: []int{14},
					Klines: store.KlineConfig{PrimaryTimeframe: "1d", PrimaryCount: 20, SelectedTimeframes: []string{"1d"}},
				},
			},
		},
	}
	block, hash := at.renderIndicatorMirror("MNQ")
	if block == "" {
		t.Fatal("expected a non-empty indicator block from real bars")
	}
	t.Logf("ai_config=%s\n---INDICATORS BLOCK---\n%s\n---END---", hash, block)
}
