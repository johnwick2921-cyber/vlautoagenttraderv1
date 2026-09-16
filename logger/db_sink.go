package logger

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"regexp"
	"sync/atomic"

	"github.com/sirupsen/logrus"
)

// P6 (ledger-close 2026-08-19) — the WARN+ERROR→DB tee.
//
// A logrus Hook on the global Log captures every Warnf/Errorf (and the trader
// wrappers logWarnf/logErrorf, which delegate here) WITHOUT touching any call
// site — the repo had zero hooks before this. INFO stays journal-only (the
// per-frame flood is exactly what rotates journald in hours).
//
// The sink function is injected AFTER the store exists (main.go), because the
// logger initializes before the DB. Lines logged before attachment are
// unshipped by design (boot prints are re-emitted every boot anyway).
// Recursion safety: the sink's implementation must never log through logrus —
// the store-side shipper only bumps counters on its own failures.

// ShipFunc receives one log line for durable storage. It MUST be non-blocking.
type ShipFunc func(tsMs int64, level, component, traderID, message, fieldsJSON string)

var (
	shipFn      atomic.Value // ShipFunc
	hookAttached atomic.Bool
)

// traderTagRe extracts the trader id the per-trader wrappers prepend:
// "[trader_id=... trader_name=...]".
var traderTagRe = regexp.MustCompile(`\[trader_id=([^ \]]+)`)

// AttachDBSink installs the WARN+ hook with the given sink. Idempotent for the
// hook itself; the newest sink wins.
func AttachDBSink(fn ShipFunc) {
	shipFn.Store(fn)
	if hookAttached.CompareAndSwap(false, true) {
		Log.AddHook(&dbHook{})
	}
}

type dbHook struct{}

// Levels ships WARN and up. "CRITICAL" lines in this repo are Errorf text —
// covered by ErrorLevel.
func (h *dbHook) Levels() []logrus.Level {
	return []logrus.Level{logrus.WarnLevel, logrus.ErrorLevel, logrus.FatalLevel, logrus.PanicLevel}
}

// Fire forwards the entry to the sink. Never returns an error (a logging tee
// must never fail the log call) and never blocks (the sink contract).
func (h *dbHook) Fire(e *logrus.Entry) error {
	fnAny := shipFn.Load()
	if fnAny == nil {
		return nil
	}
	fn, ok := fnAny.(ShipFunc)
	if !ok {
		return nil
	}
	component := ""
	if e.Caller != nil {
		component = fmt.Sprintf("%s:%d", filepath.Base(e.Caller.File), e.Caller.Line)
	}
	traderID := ""
	if m := traderTagRe.FindStringSubmatch(e.Message); m != nil {
		traderID = m[1]
	}
	fieldsJSON := "{}"
	if len(e.Data) > 0 {
		if b, err := json.Marshal(e.Data); err == nil {
			fieldsJSON = string(b)
		}
	}
	fn(e.Time.UnixMilli(), e.Level.String(), component, traderID, e.Message, fieldsJSON)
	return nil
}
