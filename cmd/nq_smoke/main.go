// Command nq_smoke is a standalone single-cycle runner for the NQ trading slice.
//
// Pipeline: Databento OHLCV → indicators → futures prompt → stdin-paste AI
// decision → CSV signal → NT fill → console output.
//
// Requires:
//   - NT 8 running on Windows host with vltrader.cs on an MNQ chart in SIM
//   - .env populated with DATABENTO_API_KEY + NINJATRADER_DATA_DIR
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"

	"nofx/kernel"
	"nofx/market"
	"nofx/provider/databento"
	"nofx/provider/ninjatrader"
)

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "databento":
			runDatabentoSmoke()
			return
		case "resolver":
			runResolverSmoke()
			return
		case "prompt":
			runPromptSmoke()
			return
		case "roundtrip":
			runRoundtripSmoke()
			return
		case "tcp":
			runTCPSmoke()
			return
		case "all":
			runAllSmokes()
			return
		case "help", "-h", "--help":
			printHelp()
			return
		}
	}
	_ = godotenv.Load() // .env from the working directory (run from the repo root)

	dbKey := os.Getenv("DATABENTO_API_KEY")
	if dbKey == "" {
		log.Fatal("DATABENTO_API_KEY not set in .env")
	}
	ntDir := os.Getenv("NINJATRADER_DATA_DIR")
	if ntDir == "" {
		log.Fatal("NINJATRADER_DATA_DIR not set in .env")
	}

	// 1. Fetch NQ bars
	db := databento.NewClient("", dbKey)
	const (
		// SMOKE TEST CONFIG — uses 96h (4-day) lookback to accommodate:
		//   1. Databento Historical tier ~3h availability lag
		//   2. CME weekend gap (Fri 16:00 CT → Sun 17:00 CT, ~49h)
		//   3. Single-day holiday extensions (~73h)
		//   4. Standard 3-day market holidays (Memorial Day, MLK, etc., ~97h)
		//
		// Does NOT cover 5+ day exchange closures (year-end Christmas).
		// Defer smoke test during those rare periods.
		//
		// Production Plan 1.5 will use real-time embargo (15min + 2min
		// safety). See Plan 1.5 Cold Start Sequence spec (commit e15c1dc0).
		smokeLookbackEnd = 96 * time.Hour

		// 90 minutes window: gives 90 bars at 1m granularity.
		// EMA50 (longest indicator) needs 50 bars minimum. 90 provides
		// 40-bar headroom for session-boundary gaps and any sparse
		// OHLCV records during low-liquidity periods.
		smokeLookbackRange = 90 * time.Minute
	)
	end := time.Now().UTC().Add(-smokeLookbackEnd)
	start := end.Add(-smokeLookbackRange)
	bars, err := db.GetOHLCV("NQ.c.0", "1m", start, end)
	if err != nil {
		log.Fatalf("databento: %v", err)
	}
	if len(bars) < 50 {
		log.Printf("got %d bars (need 50); window=[%s, %s]",
			len(bars), start.Format(time.RFC3339), end.Format(time.RFC3339))
		os.Exit(1)
	}
	fmt.Printf("✓ Fetched %d 1m NQ bars\n", len(bars))

	// 2. Convert to klines and compute indicators
	klines := market.BarsToKlines(bars)
	ctx := kernel.FuturesContext{
		Symbol:       "MNQ",
		CurrentPrice: klines[len(klines)-1].Close,
		EMA20:        market.ExportCalculateEMA(klines, 20),
		EMA50:        market.ExportCalculateEMA(klines, 50),
		RSI14:        market.ExportCalculateRSI(klines, 14),
		MACD:         market.ExportCalculateMACD(klines),
		ATR14:        market.ExportCalculateATR(klines, 14),
	}
	upper, _, lower := market.ExportCalculateBOLL(klines, 20, 2.0)
	ctx.BollUpper = upper
	ctx.BollLower = lower

	fmt.Printf("✓ Indicators computed. Current price: %.2f\n", ctx.CurrentPrice)

	// 3. Build prompts
	sysP := kernel.BuildFuturesSystemPrompt(kernel.FuturesPromptConfig{
		Symbol:             "MNQ",
		ContractMultiplier: 2.0,
		TickSize:           0.25,
		MinStopPoints:      15,
		MaxStopPoints:      50,
		MinRiskReward:      1.5,
	})
	userP := kernel.BuildFuturesUserPrompt(ctx)
	fmt.Println("\n--- SYSTEM PROMPT ---")
	fmt.Println(sysP)
	fmt.Println("\n--- USER PROMPT ---")
	fmt.Println(userP)

	// 4. AI call — STUBBED. Paste a decision JSON via stdin.
	fmt.Println("\n>>> Paste an AI decision JSON (or press Ctrl-C to abort):")
	var decision struct {
		Action     string  `json:"action"`
		Entry      float64 `json:"entry"`
		StopLoss   float64 `json:"stop_loss"`
		TakeProfit float64 `json:"take_profit"`
		Reasoning  string  `json:"reasoning"`
	}
	dec := json.NewDecoder(os.Stdin)
	if err := dec.Decode(&decision); err != nil {
		log.Fatalf("decode decision: %v", err)
	}
	fmt.Printf("✓ Decision: %s entry=%.2f sl=%.2f tp=%.2f\n", decision.Action, decision.Entry, decision.StopLoss, decision.TakeProfit)

	if decision.Action == "NONE" {
		fmt.Println("Action=NONE; nothing to write. Exiting.")
		return
	}

	// 5. Write signal
	w := ninjatrader.NewCSVWriter(ntDir)
	sig := ninjatrader.SignalRow{
		DateTime:   time.Now().Format("01/02/2006 15:04:05"),
		Direction:  decision.Action,
		EntryPrice: decision.Entry,
		StopLoss:   decision.StopLoss,
		TakeProfit: decision.TakeProfit,
	}
	if err := w.WriteSignal(sig); err != nil {
		log.Fatalf("write signal: %v", err)
	}
	fmt.Println("✓ Signal written to", w.SignalsPath())

	// 6. Tail fills for 30 seconds
	fmt.Println("\nTailing trades_taken.csv for 30s — watch for fill...")
	tailer := ninjatrader.NewCSVTailer(ntDir, time.Second)
	ctxT, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	_ = tailer.TailFills(ctxT, func(f ninjatrader.FillRow) {
		fmt.Printf("  >>> FILL: %s %s @ %.2f\n", f.DateTime, f.Direction, f.EntryPrice)
	})
	fmt.Println("Done.")
}
