package ninjatrader

import (
	"bytes"
	"encoding/json"
	"testing"
)

// TestPurgeSymbol locks the P5.3 clean-teardown rule: removing a symbol purges
// EVERY timeframe's cached bars for that symbol (case-insensitive) and ONLY
// that symbol — the other roots' caches are untouched.
func TestPurgeSymbol(t *testing.T) {
	c := NewBarCache(0)
	bars := []Bar{{T: 1000, O: 1, H: 2, L: 0.5, C: 1.5, V: 10}}
	c.SeedHistorical("MNQ", "5m", bars)
	c.SeedHistorical("ES", "5m", bars)
	c.SeedHistorical("ES", "1h", bars)

	if removed := c.PurgeSymbol("es"); removed != 2 {
		t.Fatalf("PurgeSymbol(es) removed %d entries; want 2 (5m+1h)", removed)
	}
	if got := c.Get("ES", "5m"); got != nil {
		t.Fatalf("ES|5m must be purged; got %d bars", len(got))
	}
	if got := c.Get("MNQ", "5m"); len(got) != 1 {
		t.Fatalf("MNQ|5m must be UNTOUCHED; got %d bars", len(got))
	}
	if removed := c.PurgeSymbol(""); removed != 0 {
		t.Fatalf("PurgeSymbol(\"\") must be a no-op")
	}
}

// TestSubscribeBarsSymbol_RuntimeAdd locks the runtime-ADD semantics: the
// symbol registers (pending state; survives reconnect) even when disconnected
// (send error reported); the PRIMARY is refused; the state map exposes it.
func TestSubscribeBarsSymbol_RuntimeAdd(t *testing.T) {
	s := NewTCPServer(nil)
	s.SetBarsSubscribeSymbol("MNQ")

	if err := s.SubscribeBarsSymbol("MNQ"); err == nil {
		t.Fatal("subscribing the PRIMARY must be refused")
	}
	// Disconnected: registered + pending, send error returned.
	if err := s.SubscribeBarsSymbol("NQ"); err == nil {
		t.Fatal("expected a not-connected send error (state still registered)")
	}
	got := s.barsSubscribePayloads()
	if len(got) != 2 || got[1].Symbol != "NQ" {
		t.Fatalf("NQ must be registered for reconnect re-subscribe; got %+v", got)
	}
	states := s.BarsSubscriptionStates()
	if st, ok := states["NQ"]; !ok || st.State != "pending" {
		t.Fatalf("NQ state must be pending; got %+v", states)
	}
	// The primary appears as a pending placeholder too (no ack yet).
	if _, ok := states["MNQ"]; !ok {
		t.Fatalf("primary must appear in the state map; got %+v", states)
	}
}

// TestSubscriptionAckFraming_RoundTrip locks the P5.3 ack frames' encodings.
func TestSubscriptionAckFraming_RoundTrip(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteFrame(&buf, FrameSubscribed, SubscribedPayload{Symbol: "NQ", ResolvedContract: "NQ 06-26"}); err != nil {
		t.Fatal(err)
	}
	if err := WriteFrame(&buf, FrameSubscribeError, SubscribeErrorPayload{Symbol: "XYZ", Reason: "not in NT8 DB"}); err != nil {
		t.Fatal(err)
	}
	if err := WriteFrame(&buf, FrameUnsubscribed, UnsubscribedPayload{Symbol: "NQ", Removed: 14}); err != nil {
		t.Fatal(err)
	}

	env, _ := ReadFrame(&buf)
	var sub SubscribedPayload
	if env.Type != FrameSubscribed || json.Unmarshal(env.Payload, &sub) != nil || sub.ResolvedContract != "NQ 06-26" {
		t.Fatalf("subscribed round-trip failed: %+v %+v", env, sub)
	}
	env, _ = ReadFrame(&buf)
	var serr SubscribeErrorPayload
	if env.Type != FrameSubscribeError || json.Unmarshal(env.Payload, &serr) != nil || serr.Reason == "" {
		t.Fatalf("subscribe_error round-trip failed: %+v %+v", env, serr)
	}
	env, _ = ReadFrame(&buf)
	var uns UnsubscribedPayload
	if env.Type != FrameUnsubscribed || json.Unmarshal(env.Payload, &uns) != nil || uns.Removed != 14 {
		t.Fatalf("unsubscribed round-trip failed: %+v %+v", env, uns)
	}
}

// TestUnsubscribePurgesCache locks remove → cache purge (the teardown leaves
// no stale data for the chart/API).
func TestUnsubscribePurgesCache(t *testing.T) {
	s := NewTCPServer(nil)
	s.SetBarsSubscribeSymbol("MNQ")
	s.AddBarsSubscribeSymbols("ES")
	s.barCache.SeedHistorical("ES", "5m", []Bar{{T: 1, C: 1}})
	s.barCache.SeedHistorical("MNQ", "5m", []Bar{{T: 1, C: 1}})

	_ = s.UnsubscribeBarsSymbol("ES") // send fails (not connected) — teardown still applies
	if got := s.barCache.Get("ES", "5m"); got != nil {
		t.Fatalf("ES cache must be purged on unsubscribe; got %d bars", len(got))
	}
	if got := s.barCache.Get("MNQ", "5m"); len(got) != 1 {
		t.Fatalf("MNQ (primary) cache must be untouched; got %d", len(got))
	}
	if st := s.BarsSubscriptionStates()["ES"]; st.State != "unsubscribed" {
		t.Fatalf("ES state must be unsubscribed; got %+v", st)
	}
}
