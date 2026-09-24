package ninjatrader

import (
	"strings"
	"testing"
	"time"
)

// ── W1b FOLD-6 — the 🧩 MATERIALIZED line says what the position WAS ─────────
//
// reconcilePositions' untracked branch printed "— manual/NT8-side entry now
// tracked" UNCONDITIONALLY, two lines after 🔗 had tagged the same position as
// this trader's own late AI (or armed) fill. The line now names the origin the
// attribution step actually established: this trader's late AI fill (with its
// signal), its late armed fill, an armed fill matched by price in the window,
// an UNTAGGED entry the ring could not resolve — and "manual/NT8-side" only
// when there was no evidence of this trader's own entry at all. Pinned on the
// captured WARN emitted by the production reconcile call site.

// materializedLine returns the one 🧩 MATERIALIZED line captured, or fails.
func materializedLine(t *testing.T, h *lateWarns) string {
	t.Helper()
	h.mu.Lock()
	defer h.mu.Unlock()
	var got []string
	for _, l := range h.lines {
		if strings.Contains(l, "MATERIALIZED untracked NT8 position") {
			got = append(got, l)
		}
	}
	if len(got) != 1 {
		t.Fatalf("want exactly one 🧩 MATERIALIZED line, got %d: %q", len(got), got)
	}
	return got[0]
}

func TestMaterializedLineNamesTheOriginAttributionEstablished(t *testing.T) {
	for _, tc := range []struct {
		name    string
		setup   func(t *testing.T, w *lateFillWire)
		want    []string
		manual  bool
		tagWant string
	}{
		{
			name: "late AI fill",
			setup: func(t *testing.T, w *lateFillWire) {
				w.aiOrderRow(t, lateFillTrader, "sig-ai-late", "open_long", "LONG", "BUY")
				w.fill(t, "sig-ai-late", "long", 29001)
			},
			want:    []string{"late AI fill", "sig-ai-late"},
			tagWant: "sig-ai-late",
		},
		{
			name: "late armed fill",
			setup: func(t *testing.T, w *lateFillWire) {
				w.filledArm(t, "2026-09-23:NY", "S2", "sig-arm-mine", 29001, 90*time.Second)
				w.fill(t, "sig-arm-mine", "long", 29001)
			},
			want:    []string{"late armed fill", "sig-arm-mine"},
			tagWant: "sig-arm-mine",
		},
		{
			name: "armed fill matched by price in the window",
			setup: func(t *testing.T, w *lateFillWire) {
				w.filledArm(t, "2026-09-23:NY", "S3", "sig-armed", 29001, 90*time.Second)
			},
			want:    []string{"matched by price", "sig-armed"},
			tagWant: "sig-armed",
		},
		{
			name: "ambiguous ring",
			setup: func(t *testing.T, w *lateFillWire) {
				w.aiOrderRow(t, lateFillTrader, "sig-ai-a", "open_long", "LONG", "BUY")
				w.aiOrderRow(t, lateFillTrader, "sig-ai-b", "open_long", "LONG", "BUY")
				w.fill(t, "sig-ai-a", "long", 29001)
				w.fill(t, "sig-ai-b", "long", 29001)
			},
			want: []string{"UNTAGGED", "ambiguous"},
		},
		{
			name: "unclaimable ring signal",
			setup: func(t *testing.T, w *lateFillWire) {
				w.aiOrderRow(t, "some-other-trader", "sig-foreign", "open_long", "LONG", "BUY")
				w.fill(t, "sig-foreign", "long", 29001)
			},
			want: []string{"UNTAGGED", "sig-foreign"},
		},
		{
			name:   "no evidence at all",
			setup:  func(t *testing.T, w *lateFillWire) {},
			want:   []string{"manual/NT8-side entry"},
			manual: true,
		},
		{
			name: "only an arm older than the window",
			setup: func(t *testing.T, w *lateFillWire) {
				w.filledArm(t, "2026-09-22:NY", "S1", "sig-armed-yesterday", 29001, 24*time.Hour)
			},
			want:   []string{"manual/NT8-side entry"},
			manual: true,
		},
	} {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			w := newLateFillWire(t)
			warns := captureLateWarns(t)
			tc.setup(t, w)
			row := w.materialize(t, 29001)
			if row.EntryOrderID != tc.tagWant {
				t.Fatalf("fixture: entry_order_id=%q, want %q", row.EntryOrderID, tc.tagWant)
			}
			line := materializedLine(t, warns)
			for _, sub := range tc.want {
				if !strings.Contains(line, sub) {
					t.Fatalf("the 🧩 line must say %q:\n%s", sub, line)
				}
			}
			if !tc.manual && strings.Contains(line, "manual/NT8-side") {
				t.Fatalf("the 🧩 line calls a %s a manual/NT8-side entry:\n%s", tc.name, line)
			}
			for _, keep := range []string{"MNQ LONG", "@ 29001.00", "(acct=Sim101)", "now tracked; its close will record real P&L"} {
				if !strings.Contains(line, keep) {
					t.Fatalf("the 🧩 line lost %q:\n%s", keep, line)
				}
			}
		})
	}
}
