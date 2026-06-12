package telemetry

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Plan 4 Task 25 — Prometheus metrics for the NQ trading system.
//
// Collectors are registered via promauto at package init; they appear in
// the global registry consumed by promhttp.Handler() exposed at /metrics.
//
// Naming convention: nofx_<subject>_<unit_or_total>. Labels stay low-
// cardinality (no symbol-per-label, no per-request-id) so Prometheus
// scrape and storage costs stay bounded.

var (
	// DecisionsTotal counts AI decisions, partitioned by trader, action
	// (LONG/SHORT/HOLD), and execution status (queued/filled/rejected/blocked).
	DecisionsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "nofx_decisions_total",
			Help: "Total number of AI trading decisions.",
		},
		[]string{"trader_id", "action", "status"},
	)

	// DecisionLatency tracks end-to-end decision time (request → AI → parsed).
	DecisionLatency = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "nofx_decision_latency_seconds",
			Help:    "Latency of AI decision cycle in seconds.",
			Buckets: prometheus.DefBuckets, // 0.005..10s
		},
		[]string{"trader_id"},
	)

	// FillLatency tracks time from order submission to fill confirmation.
	// Buckets span sub-second to minutes to fit both crypto (fast) and
	// futures via NinjaTrader CSV bridge (slow file-watch tail).
	FillLatency = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "nofx_fill_latency_seconds",
			Help:    "Latency from order signal to fill confirmation in seconds.",
			Buckets: []float64{0.1, 0.5, 1, 2, 5, 10, 30, 60, 120},
		},
		[]string{"exchange"},
	)

	// DatabentoErrorsTotal counts HTTP errors from Databento client
	// (5xx responses after retries are exhausted, plus network failures).
	DatabentoErrorsTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "nofx_databento_errors_total",
			Help: "Total Databento HTTP errors (5xx + network failures after retry).",
		},
	)

	// RiskGateTrips counts how often each Plan 2+3 gate has blocked a cycle.
	// gate_name values:
	//   - "task18_cme_closed"      — CME session-closed gate (Plan 2 T18)
	//   - "task19_contract_roll"   — contract-expiry / roll gate (Plan 2 T19)
	//   - "task21_risk_limit"      — daily-loss / concurrent-position cap (Plan 3 T21)
	//   - "task22_drift"           — stale-data / suspicious-drift gate (Plan 3 T22)
	RiskGateTrips = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "nofx_risk_gate_trips_total",
			Help: "Number of times each safety gate has tripped (skip cycle / block entry).",
		},
		[]string{"gate_name"},
	)
)
