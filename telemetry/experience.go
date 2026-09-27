// Package telemetry handles product telemetry
package telemetry

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

var telemetryEndpoint = "https://www.google-analytics.com/mp/collect"

const (
	tid = "G-14J8SY6F0J"
	tk  = "sgPLmshGTPiF-X57rzEIKA"
)

var (
	client     *Client
	clientOnce sync.Once
	httpClient = &http.Client{Timeout: 5 * time.Second}

	// ga4Failures counts every GA4 send that failed: transport error, request
	// construction error, or a non-2xx response. P2-9: before this counter a
	// 100%-dead GA4 pipe was indistinguishable from a healthy one.
	ga4Failures atomic.Int64
)

// IncGA4Failure records one failed GA4 send (any class).
func IncGA4Failure() { ga4Failures.Add(1) }

// GA4Failures returns the lifetime GA4 send-failure count.
func GA4Failures() int64 { return ga4Failures.Load() }

type Client struct {
	enabled        bool
	installationID string
	mu             sync.RWMutex
}

type TradeEvent struct {
	Exchange  string
	TradeType string
	Symbol    string
	AmountUSD float64
	Leverage  int
	UserID    string
	TraderID  string
}

type AIUsageEvent struct {
	UserID        string
	TraderID      string
	ModelProvider string // openai, deepseek, anthropic, etc.
	ModelName     string // gpt-4o, deepseek-chat, claude-3, etc.
	Channel       string // payment channel: "claw402" or "native"
	InputTokens   int
	OutputTokens  int
}

type telemetryPayload struct {
	ClientID string           `json:"client_id"`
	Events   []telemetryEvent `json:"events"`
}

type telemetryEvent struct {
	Name   string                 `json:"name"`
	Params map[string]interface{} `json:"params"`
}

func Init(enabled bool, installationID string) {
	clientOnce.Do(func() {
		client = &Client{
			enabled:        enabled,
			installationID: installationID,
		}
	})
}

func SetInstallationID(id string) {
	if client == nil {
		return
	}
	client.mu.Lock()
	defer client.mu.Unlock()
	client.installationID = id
}

func GetInstallationID() string {
	if client == nil {
		return ""
	}
	client.mu.RLock()
	defer client.mu.RUnlock()
	return client.installationID
}

func SetEnabled(enabled bool) {
	if client == nil {
		return
	}
	client.mu.Lock()
	defer client.mu.Unlock()
	client.enabled = enabled
}

func IsEnabled() bool {
	if client == nil {
		return false
	}
	client.mu.RLock()
	defer client.mu.RUnlock()
	return client.enabled
}

func TrackTrade(event TradeEvent) {
	if client == nil || !IsEnabled() {
		return
	}

	// Send asynchronously to not block trading
	goTracked("ga4-trade", func() {
		if err := sendTradeEvent(event); err != nil {
			IncGA4Failure()
		}
	})
}

// sendTradeEvent sends the trade event to GA4
func sendTradeEvent(event TradeEvent) error {
	client.mu.RLock()
	installationID := client.installationID
	client.mu.RUnlock()

	payload := telemetryPayload{
		ClientID: installationID,
		Events: []telemetryEvent{
			{
				Name: "trade",
				Params: map[string]interface{}{
					"exchange":             event.Exchange,
					"trade_type":           event.TradeType,
					"symbol":               event.Symbol,
					"amount_usd":           event.AmountUSD,
					"leverage":             event.Leverage,
					"installation_id":      installationID, // For counting active installations
					"user_id":              event.UserID,   // For counting active users
					"trader_id":            event.TraderID, // For counting active traders
					"engagement_time_msec": 1,              // Required by GA4
				},
			},
		},
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	url := telemetryEndpoint + "?measurement_id=" + tid + "&api_secret=" + tk
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("GA4 non-2xx: %d", resp.StatusCode)
	}

	return nil
}

func TrackStartup(version string) {
	if client == nil || !IsEnabled() {
		return
	}

	goTracked("ga4-event", func() {
		client.mu.RLock()
		installationID := client.installationID
		client.mu.RUnlock()

		payload := telemetryPayload{
			ClientID: installationID,
			Events: []telemetryEvent{
				{
					Name: "app_startup",
					Params: map[string]interface{}{
						"version":              version,
						"installation_id":      installationID,
						"engagement_time_msec": 1,
					},
				},
			},
		}

		if err := postTelemetryEvent(payload); err != nil {
			IncGA4Failure()
		}
	})
}

func TrackAIUsage(event AIUsageEvent) {
	if client == nil || !IsEnabled() {
		return
	}

	goTracked("ga4-event", func() {
		client.mu.RLock()
		installationID := client.installationID
		client.mu.RUnlock()

		payload := telemetryPayload{
			ClientID: installationID,
			Events: []telemetryEvent{
				{
					Name: "ai_usage",
					Params: map[string]interface{}{
						"model_provider":       event.ModelProvider,
						"model_name":           event.ModelName,
						"channel":              event.Channel,
						"input_tokens":         event.InputTokens,
						"output_tokens":        event.OutputTokens,
						"total_tokens":         event.InputTokens + event.OutputTokens,
						"installation_id":      installationID,
						"user_id":              event.UserID,
						"trader_id":            event.TraderID,
						"engagement_time_msec": 1,
					},
				},
			},
		}

		if err := postTelemetryEvent(payload); err != nil {
			IncGA4Failure()
		}
	})
}

// goTracked runs a fire-and-forget telemetry post in its own goroutine under
// the panic net: a panic counts a GA4 failure and an error, never kills the
// process. safe.GoNet cannot be imported here (safe counts via telemetry —
// an import cycle), so the recover lives inline.
func goTracked(name string, fn func()) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				IncGA4Failure()
				RecordError("", "goroutine_panic", name+": "+fmt.Sprint(r), CostNone)
			}
		}()
		fn()
	}()
}

// postTelemetryEvent marshals and POSTs one GA4 payload; ANY failure (marshal,
// request construction, transport, non-2xx) returns an error so the caller
// counts it — a swallowed send is a silent dead pipe.
func postTelemetryEvent(payload telemetryPayload) error {
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	url := telemetryEndpoint + "?measurement_id=" + tid + "&api_secret=" + tk
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("GA4 non-2xx: %d", resp.StatusCode)
	}
	return nil
}
