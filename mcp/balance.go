package mcp

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"
)

// P5 (ledger-close 2026-08-19) — DeepSeek balance endpoint.
//
// DeepSeek exposes GET https://api.deepseek.com/user/balance (Bearer auth):
//   {"is_available":true,"balance_infos":[{"currency":"USD",
//    "total_balance":"12.34","granted_balance":"0.00","topped_up_balance":"12.34"}]}
// (documented at api-docs.deepseek.com; parsed defensively — any shape drift
// degrades to an error, never a panic). Used ONLY by the optional daily
// AI_BALANCE_WARN poll; nothing on the decision path calls this.

const deepSeekBalanceURL = "https://api.deepseek.com/user/balance"

type deepSeekBalanceResp struct {
	IsAvailable  bool `json:"is_available"`
	BalanceInfos []struct {
		Currency     string `json:"currency"`
		TotalBalance string `json:"total_balance"`
	} `json:"balance_infos"`
}

// DeepSeekBalance returns (total balance, currency) for the given API key.
func DeepSeekBalance(apiKey string) (float64, string, error) {
	req, err := http.NewRequest(http.MethodGet, deepSeekBalanceURL, nil)
	if err != nil {
		return 0, "", err
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return 0, "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<16))
	if err != nil {
		return 0, "", err
	}
	if resp.StatusCode != http.StatusOK {
		return 0, "", fmt.Errorf("balance endpoint returned status %d: %s", resp.StatusCode, truncate(string(body), 200))
	}
	var parsed deepSeekBalanceResp
	if err := json.Unmarshal(body, &parsed); err != nil {
		return 0, "", fmt.Errorf("balance response unparseable: %w", err)
	}
	if len(parsed.BalanceInfos) == 0 {
		return 0, "", fmt.Errorf("balance response carried no balance_infos")
	}
	bal, err := strconv.ParseFloat(parsed.BalanceInfos[0].TotalBalance, 64)
	if err != nil {
		return 0, "", fmt.Errorf("total_balance %q unparseable: %w", parsed.BalanceInfos[0].TotalBalance, err)
	}
	return bal, parsed.BalanceInfos[0].Currency, nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
