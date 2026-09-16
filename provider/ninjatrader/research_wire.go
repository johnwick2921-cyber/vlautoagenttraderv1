package ninjatrader

import (
	"encoding/json"
	"fmt"
	"nofx/researchsnapshot"
	"time"
)

// One state per connection, accessed only by the research worker. Metadata
// belongs to the frames preceding the observation, never a later live cache.
type researchWire struct {
	contract, build string
	previous        map[string]Bar
}

func newResearchWire() *researchWire { return &researchWire{previous: map[string]Bar{}} }
func (w *researchWire) observe(kind FrameType, payload json.RawMessage) {
	w.observeAt(kind, payload, time.Now())
}
func (w *researchWire) observeAt(kind FrameType, payload json.RawMessage, received time.Time) {
	defer researchsnapshot.Contain(" recording")
	switch kind {
	case FrameHello, FrameSubscribed, FrameBarsHistorical, FrameBarUpdate, FrameFill, FrameOrderUpdate, FrameOrderSnapshot, FramePositionClose:
	default:
		return
	}
	researchsnapshot.Record("wire:"+string(kind), func() []researchsnapshot.Fact { return w.facts(kind, payload, received) })
}
func (w *researchWire) facts(kind FrameType, payload json.RawMessage, received time.Time) []researchsnapshot.Fact {
	var m map[string]json.RawMessage
	if err := json.Unmarshal(payload, &m); err != nil {
		panic("research wire JSON decode")
	}
	text := func(key string) string { var v string; _ = json.Unmarshal(m[key], &v); return v }
	if kind == FrameHello {
		w.build = text("build_id")
		return nil
	}
	if kind == FrameSubscribed {
		if text("symbol") == "MNQ" {
			w.contract = text("resolved_contract")
		}
		return nil
	}
	if kind == FrameBarsHistorical || kind == FrameBarUpdate {
		if text("symbol") != "MNQ" {
			return nil
		}
		var bars []Bar
		if err := json.Unmarshal(m["bars"], &bars); err != nil {
			panic("research bars decode")
		}
		var rawBars []map[string]json.RawMessage
		if err := json.Unmarshal(m["bars"], &rawBars); err != nil {
			panic("research raw bars decode")
		}
		tf := text("timeframe")
		opened := OpenStampBars(bars, tf)
		out := make([]researchsnapshot.Fact, 0, len(bars))
		for i, b := range bars {
			clocks := researchsnapshot.Clocks{ReceiptMS: researchsnapshot.Value(received.UnixMilli())}
			if b.T > 0 {
				clocks.ObservationMS = researchsnapshot.Value(b.T)
			}
			f := researchsnapshot.NewFact("market", string(kind), nil, clocks)
			f.Set("root_symbol", "MNQ")
			f.Set("feed", "NT8 TCP")
			f.Set("source_timezone", "UTC epoch milliseconds")
			f.Set("timeframe", tf)
			if b.T > 0 {
				f.Set("source_stamp_ms", b.T)
				f.Set("bar_open_ms", opened[i].T)
				f.Set("bar_close_ms", b.T)
				f.Set("finalized", b.T <= received.UnixMilli())
				f.Set("forming", b.T > received.UnixMilli())
			}
			for dest, src := range map[string]string{"open": "o", "high": "h", "low": "l", "close": "c", "volume": "v"} {
				if value, ok := rawBars[i][src]; ok {
					f.Set(dest, value)
				}
			}

			if w.contract != "" {
				f.Set("contract", w.contract)
				f.Set("contract_basis", "preceding received subscription acknowledgement; historical per-bar contract UNKNOWN")
			}
			if w.build != "" {
				f.Set("source_build_id", w.build)
			}
			f.Unknown("price_scale", "wire has no merge/back-adjust policy; cannot certify historical contract prices")
			key := fmt.Sprintf("%s:%d", tf, b.T)
			complete := b.T > 0
			for _, key := range []string{"o", "h", "l", "c", "v"} {
				if _, ok := rawBars[i][key]; !ok {
					complete = false
				}
			}
			if prev, ok := w.previous[key]; ok && complete {
				f.Set("previous_observation", prev)
				f.Set("correction", prev != b)
			} else {
				f.Unknown("correction", "no prior captured version in this connection's bounded comparison window")
			}
			if len(w.previous) >= 12000 {
				w.previous = map[string]Bar{}
			}
			if complete {
				w.previous[key] = b
			}
			out = append(out, f)
		}
		return out
	}
	// Select only non-account fields. Never archive credentials or account names.
	f := researchsnapshot.NewFact("exec", string(kind), nil, researchsnapshot.Clocks{ReceiptMS: researchsnapshot.Value(received.UnixMilli())})
	var emitted int64
	_ = json.Unmarshal(m["emitted_at_ms"], &emitted)
	if emitted > 0 {
		f.Clocks.ObservationMS = researchsnapshot.Value(emitted)
		delete(f.Missing, "observation_ms")
	}
	if stamp := text("fill_time"); stamp != "" {
		if t, e := time.Parse(time.RFC3339Nano, stamp); e == nil {
			f.Clocks.ObservationMS = researchsnapshot.Value(t.UnixMilli())
			delete(f.Missing, "observation_ms")
		}
	}
	selected := map[string]json.RawMessage{}
	for _, key := range []string{"signal_id", "symbol", "order_name", "state", "fill_price", "fill_time", "side", "quantity", "slippage_ticks", "status", "seq", "orders", "build_id", "emitted_at_ms", "reason", "exit_price", "exit_time", "position_id"} {
		if v, ok := m[key]; ok {
			selected[key] = v
		}
	}
	f.Set("broker_frame", selected)
	for dest, src := range map[string]string{"root_symbol": "symbol", "signal_id": "signal_id", "size": "quantity", "reason": "reason", "source_build_id": "build_id", "position_id": "position_id", "exit_price": "exit_price"} {
		if v, ok := m[src]; ok {
			f.Set(dest, v)
		}
	}
	if kind == FrameFill {
		f.Set("fills", selected)
		f.Set("ambiguity", []string{"entry/exit role requires signal and order linkage"})
		f.Unknown("attainable_entry", "fill price retained in fills; entry versus exit role requires signal/order linkage")
	}
	if text("reason") == "" {
		f.Unknown("reason", "received frame omits reason; h1 does not transmit rejection check")
	}
	f.Unknown("costs", "no broker commission or cost field in this received frame")
	out := []researchsnapshot.Fact{f}
	if kind == FrameOrderSnapshot {
		var orders []map[string]json.RawMessage
		if e := json.Unmarshal(m["orders"], &orders); e != nil {
			panic("research book decode")
		}
		for _, order := range orders {
			of := researchsnapshot.NewFact("exec", "broker_book_order", nil, f.Clocks)
			for dest, src := range map[string]string{"root_symbol": "symbol", "order_id": "order_id", "side": "action", "order_type": "type", "oco_id": "oco", "tif": "tif", "size": "quantity"} {
				if value, ok := order[src]; ok {
					of.Set(dest, value)
				}
			}
			prices := map[string]json.RawMessage{}
			for _, key := range []string{"limit_price", "stop_price"} {
				if v, ok := order[key]; ok {
					prices[key] = v
				}
			}
			if len(prices) > 0 {
				of.Set("order_semantics", map[string]any{"prices": prices, "basis": "prices present in received broker book; order role/state retained, not a fill"})
			}
			// Retain only known order-schema keys; strip account/unknown extensions.
			clean := map[string]json.RawMessage{}
			for _, key := range []string{"order_id", "symbol", "name", "action", "type", "limit_price", "stop_price", "quantity", "filled", "state", "oco", "tif", "time_ms"} {
				if v, ok := order[key]; ok {
					clean[key] = v
				}
			}
			of.Set("broker_frame", clean)
			if build := text("build_id"); build != "" {
				of.Set("source_build_id", build)
			}
			out = append(out, of)
		}
	}
	return out
}

func recordResearchSignal(p SignalPayload, err error) {
	recordResearchSignalAt(p, err, time.Now())
}
func recordResearchSignalAt(p SignalPayload, err error, now time.Time) {
	defer researchsnapshot.Contain("signal recording")
	// Copy only the authorized wire semantics, never the account or trader binding.
	reason := "queued for transport; broker acceptance unproven"
	if err != nil {
		reason = err.Error()
	}
	symbol, signal, side, kind, seq := p.Symbol, p.SignalID, p.Side, p.OrderType, p.Seq
	entry, limit, stop, sl, tp, size, stamp := p.Entry, p.LimitPrice, p.StopPrice, p.StopLoss, p.TakeProfit, p.Quantity, p.Timestamp
	researchsnapshot.Record("exec:signal_transport", func() []researchsnapshot.Fact {
		c := researchsnapshot.Clocks{ReceiptMS: researchsnapshot.Value(now.UnixMilli())}
		if t, e := time.Parse(time.RFC3339Nano, stamp); e == nil {
			c.ObservationMS = researchsnapshot.Value(t.UnixMilli())
		}
		f := researchsnapshot.NewFact("exec", "signal_transport", nil, c)
		f.Set("root_symbol", symbol)
		f.Set("signal_id", signal)
		f.Set("side", side)
		f.Set("order_type", kind)
		f.Set("size", size)
		f.Set("reason", reason)
		f.Set("composed_entry", map[string]any{"entry": entry, "limit_price": limit, "stop_price": stop, "stop_loss": sl, "take_profit": tp, "sequence": seq})
		f.Unknown("accepted_entry", "transport result is not a received broker acceptance")
		return []researchsnapshot.Fact{f}
	})
}
