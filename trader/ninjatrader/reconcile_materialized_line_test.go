package ninjatrader

import (
	"strings"
	"testing"
	"time"

	"nofx/store"
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

// ── W1b FOLD-6 (finish) — an attribution that could not be ESTABLISHED never
// reads as an answer ──────────────────────────────────────────────────────────
//
// The first cut named every untagged ring outcome "not claimable by this
// trader" and left the price-match fallback's failures on the default
// "manual/NT8-side entry (no … evidence …)" text. Both are claims the process
// did not establish: a store read or write that FAILED is not evidence that the
// signal is someone else's, nor that there was no in-window arm. Each path now
// names what actually happened; "manual/NT8-side" stays reserved for the one
// case where the reads succeeded and found nothing of this trader's.

// breakArmedLedger makes every armed_orders read fail from here on.
func breakArmedLedger(t *testing.T, w *lateFillWire) {
	t.Helper()
	if err := w.st.GormDB().Exec("ALTER TABLE armed_orders RENAME TO armed_orders_injected_gone").Error; err != nil {
		t.Fatal(err)
	}
}

// failPositionUpdates makes every attribution write on trader_positions (the
// entry identity and the plan linkage) fail from here on; the materialization
// itself (INSERT + its MAE/MFE NULL-ing UPDATE) still succeeds.
func failPositionUpdates(t *testing.T, w *lateFillWire) {
	t.Helper()
	if err := w.st.GormDB().Exec("CREATE TRIGGER injected_pos_attr_fail BEFORE UPDATE OF entry_order_id, plan_id ON trader_positions BEGIN SELECT RAISE(ABORT, 'injected position write failure'); END").Error; err != nil {
		t.Fatal(err)
	}
}

func TestMaterializedLineNamesAnAttributionThatFailed(t *testing.T) {
	for _, tc := range []struct {
		name   string
		setup  func(t *testing.T, w *lateFillWire)
		want   []string
		forbid []string
	}{
		{
			name: "price-match fallback: armed read failed",
			setup: func(t *testing.T, w *lateFillWire) {
				w.filledArm(t, "2026-09-23:NY", "S3", "sig-armed", 29001, 90*time.Second)
				breakArmedLedger(t, w)
			},
			want:   []string{"UNTAGGED", "armed-fill read", "failed"},
			forbid: []string{"manual/NT8-side", "not claimable"},
		},
		{
			name: "price-match fallback: matched, lineage stamp failed",
			setup: func(t *testing.T, w *lateFillWire) {
				w.filledArm(t, "2026-09-23:NY", "S3", "sig-armed", 29001, 90*time.Second)
				failPositionUpdates(t, w)
			},
			want:   []string{"UNTAGGED", "matched by price", "lineage stamp failed"},
			forbid: []string{"manual/NT8-side", "not claimable"},
		},
		{
			name: "ring: armed lookup failed",
			setup: func(t *testing.T, w *lateFillWire) {
				w.aiOrderRow(t, lateFillTrader, "sig-ai-late", "open_long", "LONG", "BUY")
				w.fill(t, "sig-ai-late", "long", 29001)
				breakArmedLedger(t, w)
			},
			want:   []string{"UNTAGGED", "sig-ai-late", "armed lookup failed"},
			forbid: []string{"manual/NT8-side", "not claimable"},
		},
		{
			name: "ring: entry-order-id stamp failed",
			setup: func(t *testing.T, w *lateFillWire) {
				w.aiOrderRow(t, lateFillTrader, "sig-ai-late", "open_long", "LONG", "BUY")
				w.fill(t, "sig-ai-late", "long", 29001)
				failPositionUpdates(t, w)
			},
			want:   []string{"UNTAGGED", "sig-ai-late", "entry-order-id stamp failed"},
			forbid: []string{"manual/NT8-side", "not claimable", "late AI fill ("},
		},
		{
			name: "ring: armed lineage stamp failed",
			setup: func(t *testing.T, w *lateFillWire) {
				w.filledArm(t, "2026-09-23:NY", "S2", "sig-arm-mine", 29001, 90*time.Second)
				w.fill(t, "sig-arm-mine", "long", 29001)
				failPositionUpdates(t, w)
			},
			want:   []string{"UNTAGGED", "sig-arm-mine", "lineage stamp failed"},
			forbid: []string{"manual/NT8-side", "not claimable", "late armed fill ("},
		},
		{
			name: "ring: an AI order already settled",
			setup: func(t *testing.T, w *lateFillWire) {
				w.aiOrderRow(t, lateFillTrader, "sig-ai-settled", "open_long", "LONG", "BUY")
				o, err := w.st.Order().GetOrderByExchangeID(lateFillExchange, "sig-ai-settled")
				if err != nil || o == nil {
					t.Fatalf("order row: %v", err)
				}
				if err := w.st.Order().UpdateOrderStatus(o.ID, "FILLED", 1, 28950, 0); err != nil {
					t.Fatal(err)
				}
				w.fill(t, "sig-ai-settled", "long", 29001)
			},
			want:   []string{"UNTAGGED", "sig-ai-settled", "already FILLED"},
			forbid: []string{"manual/NT8-side"},
		},
		{
			name: "ring: an arm of the other side",
			setup: func(t *testing.T, w *lateFillWire) {
				r := &store.ArmedOrderDB{TraderID: lateFillTrader, PlanID: "2026-09-23:NY", Scenario: "S6", Version: 3, State: "armed", Side: "short", EntryPx: 29001, StopPx: 29021, TargetPx: 28951}
				if err := w.st.ArmedOrders().UpsertArm(r); err != nil {
					t.Fatal(err)
				}
				if err := w.st.ArmedOrders().BeginPlacement(r.ID, "sig-arm-short"); err != nil {
					t.Fatal(err)
				}
				w.fill(t, "sig-arm-short", "long", 29001)
			},
			want:   []string{"UNTAGGED", "sig-arm-short", "OTHER side"},
			forbid: []string{"manual/NT8-side"},
		},
	} {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			w := newLateFillWire(t)
			warns := captureLateWarns(t)
			tc.setup(t, w)
			row := w.materialize(t, 29001)
			if row.EntryOrderID != "" {
				t.Fatalf("fixture: a failed attribution must leave the row untagged, entry_order_id=%q", row.EntryOrderID)
			}
			line := materializedLine(t, warns)
			for _, sub := range tc.want {
				if !strings.Contains(line, sub) {
					t.Fatalf("the 🧩 line must say %q:\n%s", sub, line)
				}
			}
			for _, sub := range tc.forbid {
				if strings.Contains(line, sub) {
					t.Fatalf("the 🧩 line claims %q for an attribution that was not established:\n%s", sub, line)
				}
			}
		})
	}
}
