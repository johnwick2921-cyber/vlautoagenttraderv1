package ninjatrader

import (
	"bytes"
	"encoding/json"
	"reflect"
	"testing"
)

// TestBarsSubscribePayloads_PrimaryOnly_ByteIdentical is the P5.1 golden: with
// NO extra symbols registered, the auto-subscribe payload list is exactly ONE
// frame whose content equals the pre-P5 single payload (currentBarsSubscribe).
// The single-symbol [MNQ] deployment therefore behaves byte-identically.
func TestBarsSubscribePayloads_PrimaryOnly_ByteIdentical(t *testing.T) {
	s := NewTCPServer(nil)
	s.SetBarsSubscribeSymbol("MNQ")

	got := s.barsSubscribePayloads()
	if len(got) != 1 {
		t.Fatalf("no extras must yield exactly 1 subscribe frame; got %d: %+v", len(got), got)
	}
	want := s.currentBarsSubscribe() // the pre-P5 payload shape
	if !reflect.DeepEqual(got[0], want) {
		t.Fatalf("primary frame must equal the legacy payload\n got:  %+v\n want: %+v", got[0], want)
	}
	if got[0].Symbol != "MNQ" || len(got[0].Timeframes) == 0 || got[0].BarsBack <= 0 {
		t.Fatalf("primary frame malformed: %+v", got[0])
	}
}

// TestAddBarsSubscribeSymbols locks the extras semantics: blanks, duplicates
// (case-insensitive), and the primary itself are skipped; each extra clones the
// primary's timeframes + bars-back with only the symbol swapped; the primary is
// always FIRST (the trading feed subscribes before any extra).
func TestAddBarsSubscribeSymbols(t *testing.T) {
	s := NewTCPServer(nil)
	s.SetBarsSubscribeSymbol("MNQ")
	s.AddBarsSubscribeSymbols("ES", " es ", "", "mnq", "NQ", "ES")

	got := s.barsSubscribePayloads()
	if len(got) != 3 {
		t.Fatalf("want primary+ES+NQ (3 frames); got %d: %+v", len(got), got)
	}
	if got[0].Symbol != "MNQ" || got[1].Symbol != "ES" || got[2].Symbol != "NQ" {
		t.Fatalf("order must be primary first then extras in add order; got %s,%s,%s",
			got[0].Symbol, got[1].Symbol, got[2].Symbol)
	}
	for i := 1; i < len(got); i++ {
		if !reflect.DeepEqual(got[i].Timeframes, got[0].Timeframes) || got[i].BarsBack != got[0].BarsBack {
			t.Fatalf("extra %s must clone the primary's timeframes+bars_back; got %+v vs primary %+v",
				got[i].Symbol, got[i], got[0])
		}
	}
}

// TestSetExtraBarsSymbols_ReplaceSemantics locks the P5.4 owner-keyed reload
// rule: a trader's (re)load REPLACES only ITS OWN config extras — a second
// owner's load can't wipe the first's, and env/runtime adds (extraBarsSymbols)
// survive config replaces.
func TestSetExtraBarsSymbols_ReplaceSemantics(t *testing.T) {
	s := NewTCPServer(nil)
	s.SetBarsSubscribeSymbol("MNQ")
	s.SetExtraBarsSymbolsFor("MNQ", "ES", "NQ") // owner MNQ's config extras
	s.SetExtraBarsSymbolsFor("MNQ", "RTY")      // MNQ reloads: only RTY now

	got := s.barsSubscribePayloads()
	if len(got) != 2 || got[0].Symbol != "MNQ" || got[1].Symbol != "RTY" {
		t.Fatalf("an owner's reload must REPLACE its own extras (want MNQ+RTY); got %+v", got)
	}
	// A SECOND owner's load must NOT wipe the first's extras.
	s.SetBarsSubscribeSymbol("ES") // trader 2 registers (P5.4: appends, no overwrite)
	s.SetExtraBarsSymbolsFor("ES") // trader 2 has no extras
	got = s.barsSubscribePayloads()
	want := map[string]bool{"MNQ": true, "RTY": true, "ES": true}
	if len(got) != 3 {
		t.Fatalf("trader2's load must not wipe trader1's extras; got %+v", got)
	}
	for _, p := range got {
		if !want[p.Symbol] {
			t.Fatalf("unexpected root %q in %+v", p.Symbol, got)
		}
	}
	// Replace-with-nothing for owner MNQ → its extras gone; ES (trading) stays.
	s.SetExtraBarsSymbolsFor("MNQ")
	if got := s.barsSubscribePayloads(); len(got) != 2 {
		t.Fatalf("clearing MNQ's extras must leave MNQ+ES; got %+v", got)
	}
}

// TestTradingSymbolRegistry locks the P5.4 no-overwrite rule: the first
// registration claims the primary slot (replacing the hardwired default);
// a second trader's registration APPENDS (the first trader's feed survives);
// re-registration is idempotent; trading roots are unsubscribe-refused.
func TestTradingSymbolRegistry(t *testing.T) {
	s := NewTCPServer(nil)
	s.SetBarsSubscribeSymbol("MNQ") // trader 1 — claims primary (replaces default)
	s.SetBarsSubscribeSymbol("ES")  // trader 2 — APPENDS (the P5.4 fix)
	s.SetBarsSubscribeSymbol("mnq") // trader 1 reload — idempotent

	if got := s.BarsSubscribeSymbol(); got != "MNQ" {
		t.Fatalf("primary slot must stay the FIRST registration; got %q", got)
	}
	roots := s.TradingSymbols()
	if len(roots) != 2 || roots[0] != "MNQ" || roots[1] != "ES" {
		t.Fatalf("registry must hold both trading roots; got %v", roots)
	}
	got := s.barsSubscribePayloads()
	if len(got) != 2 || got[0].Symbol != "MNQ" || got[1].Symbol != "ES" {
		t.Fatalf("both trading roots must subscribe (primary first); got %+v", got)
	}
	// BOTH trading roots are unsubscribe-refused (not just the primary).
	if err := s.UnsubscribeBarsSymbol("ES"); err == nil {
		t.Fatal("unsubscribing a registered TRADING symbol must be refused")
	}
	if err := s.UnsubscribeBarsSymbol("MNQ"); err == nil {
		t.Fatal("unsubscribing the primary must be refused")
	}
}

// TestIsSubscribedBarsSymbol_SplitBrainDefense locks the P5.2 mistagged-bar
// rejection predicate: only the primary + registered extras pass; an
// unsubscribed/mistagged symbol (or empty) is rejected.
func TestIsSubscribedBarsSymbol_SplitBrainDefense(t *testing.T) {
	s := NewTCPServer(nil)
	s.SetBarsSubscribeSymbol("MNQ")
	s.AddBarsSubscribeSymbols("ES")

	for _, ok := range []string{"MNQ", "mnq", "ES", "es"} {
		if !s.isSubscribedBarsSymbol(ok) {
			t.Fatalf("%q must be accepted (subscribed)", ok)
		}
	}
	for _, bad := range []string{"NQ", "BTCUSDT", ""} {
		if s.isSubscribedBarsSymbol(bad) {
			t.Fatalf("%q must be REJECTED (not subscribed — split-brain defense)", bad)
		}
	}
}

// TestHelloFraming_RoundTrip locks the P5.2 handshake encoding: a hello frame
// survives Write→Read with the version + source intact, and the current
// ProtocolVersion constant is 2 (bump ONLY with a lockstep C#+Go ship).
func TestHelloFraming_RoundTrip(t *testing.T) {
	if ProtocolVersion != 2 {
		t.Fatalf("ProtocolVersion = %d; P5.2 ships v2 — a bump requires a lockstep C#+Go deploy", ProtocolVersion)
	}
	var buf bytes.Buffer
	if err := WriteFrame(&buf, FrameHello, HelloPayload{ProtocolVersion: ProtocolVersion, Source: "vltrader-addon"}); err != nil {
		t.Fatal(err)
	}
	env, err := ReadFrame(&buf)
	if err != nil {
		t.Fatal(err)
	}
	if env.Type != FrameHello {
		t.Fatalf("type = %s; want hello", env.Type)
	}
	var p HelloPayload
	if err := json.Unmarshal(env.Payload, &p); err != nil {
		t.Fatal(err)
	}
	if p.ProtocolVersion != 2 || p.Source != "vltrader-addon" {
		t.Fatalf("payload round-trip mismatch: %+v", p)
	}
}

// TestUnsubscribeBarsSymbol locks the teardown semantics: the PRIMARY symbol is
// refused (never tear down the live trading feed); an extra is removed from the
// auto-subscribe list even when disconnected (so a reconnect won't re-subscribe
// it), with the send error reported.
func TestUnsubscribeBarsSymbol(t *testing.T) {
	s := NewTCPServer(nil)
	s.SetBarsSubscribeSymbol("MNQ")
	s.AddBarsSubscribeSymbols("ES")

	// Primary refused.
	if err := s.UnsubscribeBarsSymbol("mnq"); err == nil {
		t.Fatal("unsubscribing the PRIMARY symbol must be refused")
	}
	if got := s.barsSubscribePayloads(); len(got) != 2 {
		t.Fatalf("refused unsubscribe must not mutate state; got %d frames", len(got))
	}

	// Extra removed from the list even though not connected (send fails).
	if err := s.UnsubscribeBarsSymbol("ES"); err == nil {
		t.Fatal("expected a not-connected send error (state still updated)")
	}
	got := s.barsSubscribePayloads()
	if len(got) != 1 || got[0].Symbol != "MNQ" {
		t.Fatalf("ES must be removed from the auto-subscribe list; got %+v", got)
	}
}
