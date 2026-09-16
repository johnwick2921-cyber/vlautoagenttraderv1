// THE FOUR PINS the owner named for the one-contract invariant, plus the two
// that keep it honest. Each is written against the tape that produced the rule:
// on 2026-09-06 snapshot 7812 carried working_count 2 for ONE plan.
package trader

import (
	"strings"
	"testing"
	"time"

	nt "nofx/provider/ninjatrader"
)

const testMaxAge = 60 * time.Second

func entryOrder(id, name string, px float64, state string) nt.NT8Order {
	return nt.NT8Order{OrderID: id, Name: name, Symbol: "MNQ", Action: "buy",
		Type: "limit", LimitPrice: px, Quantity: 1, State: state}
}

// PIN 1 — one open position + a new arm → REFUSED.
func TestOneContract_OpenPosition_Refuses(t *testing.T) {
	v := adjudicateAccountContract(nil, true, time.Second, testMaxAge, 7812, 1)
	if v.Allowed() {
		t.Fatalf("an open position must refuse a new placement; got %+v", v)
	}
	if v.Action != contractLive {
		t.Fatalf("want contractLive, got %s", v.Action)
	}
	if v.OpenPositions != 1 {
		t.Fatalf("want OpenPositions 1, got %d", v.OpenPositions)
	}
}

// PIN 2 — one working entry order + a new arm → REFUSED.
// This is the exact shape that was live: an EMPTY position list and a book that
// already holds a resting entry. Every pre-existing guard allowed this.
func TestOneContract_WorkingEntry_Refuses(t *testing.T) {
	book := []nt.NT8Order{
		entryOrder("58fa246a", "3278aa8c-1026-4f06-b35d-b962921daf8a", 29541.25, "Working"),
	}
	v := adjudicateAccountContract(book, true, time.Second, testMaxAge, 7812, 0)
	if v.Allowed() {
		t.Fatalf("a working entry must refuse a new placement; got %+v", v)
	}
	if v.WorkingEntries != 1 {
		t.Fatalf("want WorkingEntries 1, got %d", v.WorkingEntries)
	}
	if v.FirstOrderID != "58fa246a" {
		t.Fatalf("the refusal must name the blocking order; got %q", v.FirstOrderID)
	}
}

// PIN 2b — THE LIVE SHAPE. Two working entries from one plan, account flat.
// This is snapshot 7812 exactly, and it must refuse a third.
func TestOneContract_LiveShape_TwoWorkingEntries_Refuses(t *testing.T) {
	book := []nt.NT8Order{
		entryOrder("58fa246a", "3278aa8c-1026-4f06-b35d-b962921daf8a", 29541.25, "Working"),
		entryOrder("cb6c87f5", "c5d8bdde-22a2-494c-8943-e9c6a0999324", 29530.25, "Working"),
	}
	v := adjudicateAccountContract(book, true, time.Second, testMaxAge, 7812, 0)
	if v.Allowed() {
		t.Fatalf("snapshot 7812's shape must refuse; got %+v", v)
	}
	if v.WorkingEntries != 2 {
		t.Fatalf("want WorkingEntries 2 (the live count), got %d", v.WorkingEntries)
	}
}

// PIN 4 — a stale book → REFUSED. An unverifiable book is not an empty book.
func TestOneContract_StaleBook_Refuses(t *testing.T) {
	v := adjudicateAccountContract(nil, true, 10*time.Minute, testMaxAge, 7812, 0)
	if v.Allowed() {
		t.Fatalf("a stale book must refuse; got %+v", v)
	}
	if v.Action != contractUnverifiable {
		t.Fatalf("want contractUnverifiable, got %s", v.Action)
	}
}

// PIN 4b — an ABSENT book → REFUSED, and distinguishable from a stale one.
func TestOneContract_NoBook_Refuses(t *testing.T) {
	v := adjudicateAccountContract(nil, false, 0, testMaxAge, 0, 0)
	if v.Allowed() {
		t.Fatalf("an absent book must refuse; got %+v", v)
	}
	if v.Action != contractUnverifiable {
		t.Fatalf("want contractUnverifiable, got %s", v.Action)
	}
}

// The clear case — a fresh, empty book and a flat account ALLOWS. Without this
// the guard could pass every other pin by refusing everything, which would be a
// bot that never trades.
func TestOneContract_FreshEmptyBook_Allows(t *testing.T) {
	v := adjudicateAccountContract([]nt.NT8Order{}, true, time.Second, testMaxAge, 7812, 0)
	if !v.Allowed() {
		t.Fatalf("a fresh empty book with a flat account must allow; got %+v (%s)", v, v.Refusal())
	}
}

// A protective bracket child is NOT an entry. A filled position's own stop must
// not block the management of that position.
func TestOneContract_BracketChildIsNotAnEntry(t *testing.T) {
	book := []nt.NT8Order{
		{OrderID: "sl1", Name: "c5d8bdde-22a2-494c-8943-e9c6a0999324-sl", Action: "sell", Type: "stop", State: "Working"},
		{OrderID: "tp1", Name: "c5d8bdde-22a2-494c-8943-e9c6a0999324-tp", Action: "sell", Type: "limit", State: "Working"},
	}
	v := adjudicateAccountContract(book, true, time.Second, testMaxAge, 7812, 0)
	if !v.Allowed() {
		t.Fatalf("bracket children must not count as entries; got %+v (%s)", v, v.Refusal())
	}
	if v.WorkingEntries != 0 {
		t.Fatalf("want WorkingEntries 0, got %d", v.WorkingEntries)
	}
}

// A terminal order in the book is history and must not block.
func TestOneContract_TerminalOrderDoesNotBlock(t *testing.T) {
	book := []nt.NT8Order{
		entryOrder("old1", "546fcb41-c072-4d3c-ba78-f993b06a6a84", 29524.75, "Cancelled"),
		entryOrder("old2", "86c9b34d-ac43-46c5-87d4-f277354e7549", 29530.25, "Filled"),
	}
	v := adjudicateAccountContract(book, true, time.Second, testMaxAge, 7812, 0)
	if !v.Allowed() {
		t.Fatalf("terminal orders must not block; got %+v (%s)", v, v.Refusal())
	}
}

// The refusal must NAME what blocked it — A21, so a reader can check the claim
// against the book by hand.
func TestOneContract_RefusalNamesTheBlockingOrder(t *testing.T) {
	book := []nt.NT8Order{
		entryOrder("58fa246a", "3278aa8c-1026-4f06-b35d-b962921daf8a", 29541.25, "Working"),
	}
	got := adjudicateAccountContract(book, true, 3*time.Second, testMaxAge, 7812, 0).Refusal()
	for _, want := range []string{"one contract per account", "7812", "working entry"} {
		if !strings.Contains(got, want) {
			t.Fatalf("refusal %q must mention %q", got, want)
		}
	}
}
