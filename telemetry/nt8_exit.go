// W117 a3 — the ordered worker's final-failure net for a close that met SQLite
// lock contention. The exit is never dropped: the broker's price is parked
// (reconcile orphan close), the flat signal is dropped, the receipt is
// persisted, and RetryPendingNT8Exits applies it later.
package telemetry

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// NT8ExitBusyParksTotal counts exit receipts parked after ApplyNT8Exit
// exhausted its bounded busy retries (the receipt is persisted and retried by
// RetryPendingNT8Exits — this is the never-dropped net, not a loss).
var NT8ExitBusyParksTotal = promauto.NewCounter(
	prometheus.CounterOpts{
		Name: "nofx_nt8_exit_busy_parks_total",
		Help: "NT8 exit receipts parked (priced + persisted) after ApplyNT8Exit busy retries exhausted.",
	},
)

// NT8ExitBusyParkPersistFailuresTotal counts the subset of parked exits whose
// small receipt-persist write also failed (the 2-minute priced park is then the
// only net until the next frame).
var NT8ExitBusyParkPersistFailuresTotal = promauto.NewCounter(
	prometheus.CounterOpts{
		Name: "nofx_nt8_exit_busy_park_persist_failures_total",
		Help: "NT8 exit receipts whose post-busy small persist write failed.",
	},
)

// IncNT8ExitBusyPark records one exit receipt parked after busy retries.
func IncNT8ExitBusyPark() { NT8ExitBusyParksTotal.Inc() }

// IncNT8ExitBusyParkPersistFailure records one failed post-busy persist write.
func IncNT8ExitBusyParkPersistFailure() { NT8ExitBusyParkPersistFailuresTotal.Inc() }
