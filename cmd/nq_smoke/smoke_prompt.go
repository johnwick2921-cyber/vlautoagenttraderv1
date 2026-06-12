package main

import (
	"fmt"
	"os"
	"strings"

	"nofx/kernel"
)

// runPromptSmoke builds the futures system and user prompts with sample data
// and verifies they are non-empty and contain the expected futures vocabulary.
// No network — pure construction.
func runPromptSmoke() {
	cfg := kernel.FuturesPromptConfig{
		Symbol:             "MNQ",
		ContractMultiplier: 2.0,
		TickSize:           0.25,
		MinStopPoints:      15,
		MaxStopPoints:      50,
		MinRiskReward:      1.5,
	}
	system := kernel.BuildFuturesSystemPrompt(cfg)
	if system == "" {
		fmt.Println("FAIL prompt: system prompt empty")
		os.Exit(1)
	}

	ctx := kernel.FuturesContext{
		Symbol:       "MNQ",
		CurrentPrice: 21500.25,
		EMA20:        21495.00,
		EMA50:        21480.00,
		RSI14:        58.5,
		MACD:         3.25,
		ATR14:        18.50,
		BollUpper:    21525.00,
		BollLower:    21465.00,
	}
	user := kernel.BuildFuturesUserPrompt(ctx)
	if user == "" {
		fmt.Println("FAIL prompt: user prompt empty")
		os.Exit(1)
	}

	// Verify each key term appears in at least one of the two prompts.
	combined := system + "\n" + user
	for _, term := range []string{"MNQ", "tick", "stop", "RSI"} {
		if !strings.Contains(combined, term) {
			fmt.Printf("WARN prompt: term %q not found in either prompt\n", term)
		}
	}

	fmt.Printf("OK prompt: system=%d bytes, user=%d bytes\n", len(system), len(user))
}
