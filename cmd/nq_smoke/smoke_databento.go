package main

import (
	"fmt"
	"os"
	"time"

	"github.com/joho/godotenv"

	"nofx/provider/databento"
)

// runDatabentoSmoke fetches ~90min of 1m NQ.c.0 bars from live Databento and
// verifies the response shape (bar count, monotonic timestamps, non-zero OHLC).
// Skips cleanly when DATABENTO_API_KEY is not set so this can run in CI without
// network credentials.
func runDatabentoSmoke() {
	_ = godotenv.Load() // .env from the working directory (run from the repo root)
	apiKey := os.Getenv("DATABENTO_API_KEY")
	if apiKey == "" {
		fmt.Println("SKIP databento: DATABENTO_API_KEY not set")
		return
	}

	client := databento.NewClient("", apiKey)

	// Use a 96h-old window to dodge the Databento Historical tier ~3h lag and
	// the standard CME weekend gap. Matches the default smoke flow in this
	// package (see main.go smokeLookbackEnd).
	end := time.Now().UTC().Add(-96 * time.Hour)
	start := end.Add(-90 * time.Minute)

	bars, err := client.GetOHLCV("NQ.c.0", "1m", start, end)
	if err != nil {
		fmt.Printf("FAIL databento: %v\n", err)
		os.Exit(1)
	}
	if len(bars) < 40 {
		fmt.Printf("FAIL databento: too few bars: %d (expected >=40); window=[%s, %s]\n",
			len(bars), start.Format(time.RFC3339), end.Format(time.RFC3339))
		os.Exit(1)
	}

	for i := 1; i < len(bars); i++ {
		if !bars[i].Timestamp.After(bars[i-1].Timestamp) {
			fmt.Printf("FAIL databento: non-monotonic timestamps at i=%d: %s !> %s\n",
				i, bars[i].Timestamp.Format(time.RFC3339), bars[i-1].Timestamp.Format(time.RFC3339))
			os.Exit(1)
		}
		if bars[i].Open == 0 || bars[i].Close == 0 || bars[i].High == 0 || bars[i].Low == 0 {
			fmt.Printf("FAIL databento: zero OHLC at i=%d: O=%.2f H=%.2f L=%.2f C=%.2f\n",
				i, bars[i].Open, bars[i].High, bars[i].Low, bars[i].Close)
			os.Exit(1)
		}
	}

	fmt.Printf("OK databento: fetched %d NQ.c.0 1m bars over 90min window ending %s\n",
		len(bars), end.Format(time.RFC3339))
}
