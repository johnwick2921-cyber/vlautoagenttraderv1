package calendar

import (
	"errors"
	"testing"
	"time"
)

const sampleFF = `[
  {"title":"FOMC Statement","country":"USD","date":"2026-08-13T18:00:00Z","impact":"High"},
  {"title":"CPI m/m","country":"USD","date":"2026-08-13T12:30:00Z","impact":"Medium"},
  {"title":"Minor Speech","country":"USD","date":"2026-08-13T15:00:00Z","impact":"Low"},
  {"title":"BOJ Rate","country":"JPY","date":"2026-08-13T05:00:00Z","impact":"High"},
  {"title":"AUD jobs","country":"AUD","date":"2026-08-13T01:00:00Z","impact":"High"}
]`

func allEvents(r Result) []Event {
	var out []Event
	for _, evs := range r.Days {
		out = append(out, evs...)
	}
	return out
}

func TestFetchWeekLive(t *testing.T) {
	res := FetchWeek(func() ([]byte, error) { return []byte(sampleFF), nil }, nil)
	if res.Source != SourceLive || res.Warning != "" {
		t.Fatalf("live fetch source/warn wrong: %s / %q", res.Source, res.Warning)
	}
	ev := allEvents(res)
	// Low (Minor Speech) dropped; AUD (off-list) dropped → 3 kept.
	if len(ev) != 3 {
		t.Fatalf("want 3 filtered events, got %d: %v", len(ev), ev)
	}
	var sawFOMC, sawCPI bool
	for _, e := range ev {
		if e.Title == "FOMC Statement" {
			sawFOMC = true
			if e.Impact != T1 {
				t.Fatalf("FOMC (High) should be T1, got %s", e.Impact)
			}
		}
		if e.Title == "CPI m/m" {
			sawCPI = true
			if e.Impact != T2 {
				t.Fatalf("CPI (Medium) should be T2, got %s", e.Impact)
			}
		}
		if e.Currency == "AUD" {
			t.Fatalf("off-list AUD must be filtered out")
		}
	}
	if !sawFOMC || !sawCPI {
		t.Fatalf("missing expected USD events")
	}
}

func TestFetchWeekFallbackOnOutage(t *testing.T) {
	static := func() []Event {
		return []Event{{Time: time.Date(2026, 8, 13, 12, 30, 0, 0, time.UTC), Currency: "USD", Title: "NFP (static)", Impact: T1}}
	}
	res := FetchWeek(func() ([]byte, error) { return nil, errors.New("network down") }, static)
	if res.Source != SourceStatic {
		t.Fatalf("outage should fall back to static, got %s", res.Source)
	}
	if res.Warning == "" {
		t.Fatalf("outage MUST set a Warning (blackouts never silently vanish)")
	}
	if len(allEvents(res)) != 1 {
		t.Fatalf("static fallback event missing: %v", res.Days)
	}
}

func TestFetchWeekNoStaticNoCrash(t *testing.T) {
	res := FetchWeek(func() ([]byte, error) { return nil, errors.New("down") }, nil)
	if res.Source != SourceNone || res.Warning == "" {
		t.Fatalf("no-feed no-static should be SourceNone + Warning, got %s / %q", res.Source, res.Warning)
	}
	if len(allEvents(res)) != 0 {
		t.Fatalf("expected no events")
	}
}

func TestFetchWeekBadJSONFallsBack(t *testing.T) {
	res := FetchWeek(func() ([]byte, error) { return []byte("{not json"), nil }, nil)
	if res.Source != SourceNone || res.Warning == "" {
		t.Fatalf("unparseable feed should fall back with a Warning, got %s", res.Source)
	}
}

func TestSessionCurrencyFilter(t *testing.T) {
	day := []Event{{Currency: "USD"}, {Currency: "JPY"}, {Currency: "EUR"}, {Currency: "GBP"}}
	if got := len(EventsForSession(day, "ASIA")); got != 2 { // USD + JPY
		t.Fatalf("ASIA filter = %d want 2 (USD+JPY)", got)
	}
	if got := len(EventsForSession(day, "LONDON")); got != 3 { // USD + EUR + GBP
		t.Fatalf("LONDON filter = %d want 3 (USD+EUR+GBP)", got)
	}
	if got := len(EventsForSession(day, "NY")); got != 1 { // USD only
		t.Fatalf("NY filter = %d want 1 (USD)", got)
	}
}
