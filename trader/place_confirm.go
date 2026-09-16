// Placement confirmation: a fresh received book can prove a standing entry.
// An unanswered request stays place_pending and holds its slot; age alone is
// neither acceptance nor evidence that an order disappeared.
package trader

import (
	"strings"
	"time"

	nt "nofx/provider/ninjatrader"
	"nofx/store"
	"nofx/telemetry"
)

// placeConfirmMaxWait is the existing broker-book freshness bound. It is not
// a payload freshness clock and cannot prove that an unanswered order is gone.
func placeConfirmMaxWait() time.Duration { return snapshotMaxAge() }

// confirmPendingPlacements accepts only a fresh received book naming the live
// ENTRY. No frame means place_pending with the slot held, regardless of age.
func (at *AutoTrader) confirmPendingPlacements(ledger *store.ArmedOrderStore, now time.Time) (confirmed, stillPending, unconfirmed int) {
	if at == nil || ledger == nil {
		return
	}
	rows, err := ledger.ListPlacePending(at.id)
	if err != nil {
		at.logWarnf("place-confirm: ledger read failed: %v", err)
		return
	}
	book, haveBook, age := at.liveBook(now)
	_, _, _, snapID := at.persistedBook(now)
	for _, r := range rows {
		if haveBook && age <= placeConfirmMaxWait() {
			if o, ok := bookOrderForSignal(book, r.SignalID); ok {
				frame := "order_snapshot " + snapshotLabel(snapID) + " (" + o.State + ", " + o.Type + ")"
				if err := ledger.ConfirmPlacement(r.ID, frame); err != nil {
					at.logWarnf("place-confirm: %v", err)
					continue
				}
				confirmed++
				telemetry.IncGateBlock(at.id, "place_confirmed")
				continue
			}
		}
		stillPending++
		// UpdatedAt was written by BeginPlacement before sending, unlike the arm's
		// potentially much older authoring time. This only controls a warning.
		waited := now.Sub(r.UpdatedAt)
		if waited > placeConfirmMaxWait() && !strings.HasPrefix(r.StateReason, "unconfirmed:no_frame") {
			if err := ledger.ExpirePlacement(r.ID, waited); err != nil {
				at.logWarnf("place-confirm: %v", err)
				continue
			}
			unconfirmed++
			at.logWarnf("📤 armed %s signal=%s awaits a broker receipt after %s — place_pending, slot held", r.Scenario, r.SignalID, waited.Round(time.Second))
			telemetry.IncGateBlock(at.id, "place_unconfirmed_no_frame")
		}
	}
	return
}

// A bracket child shares the signal but is not a standing entry; pending/local
// broker states likewise do not prove the entry is working at the exchange.
func bookOrderForSignal(book []nt.NT8Order, signalID string) (nt.NT8Order, bool) {
	if strings.TrimSpace(signalID) == "" {
		return nt.NT8Order{}, false
	}
	for _, o := range book {
		if o.Name == signalID && nt.ClassifyOrderState(o.State) == nt.LivenessLive {
			return o, true
		}
	}
	return nt.NT8Order{}, false
}

// snapshotLabel renders a snapshot id, or says it has none rather than printing
// a zero that reads like snapshot 0 (A24).
func snapshotLabel(id int64) string {
	if id <= 0 {
		return "(live cache, unpersisted)"
	}
	return "#" + itoa64(id)
}

func itoa64(n int64) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	if neg {
		return "-" + string(b)
	}
	return string(b)
}

// PlaceConfirmBootLine exposes the contract and the deferred AddOn producer.
func PlaceConfirmBootLine(addonReasonLive bool) string {
	reason := store.PlacementReasonUnavailable + "; C# reason producer deferred to next AddOn wave"
	if addonReasonLive {
		reason = "received reason carried verbatim"
	}
	return "place-confirm: pre-send identity + place_pending · working requires a received live ENTRY frame · rejected requires a received rejection · no receipt: place_pending, slot held · broker-reason: " + reason
}
