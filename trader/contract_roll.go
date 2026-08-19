package trader

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"nofx/kernel"

	ntTrader "nofx/trader/ninjatrader"
)

// P3 — CONTRACT-ROLL ENTRY BLOCK for the continuous symbol (ledger-close
// dispatch 2026-08-19).
//
// WHY: the existing T19 expiry gate (kernel.ShouldBlockEntryForExpiry →
// databento.DaysUntilExpiry) parses ONLY the dated code form "MNQU6"; the live
// NT8 path trades the bare root "MNQ", which fails that parse → 999 days → the
// gate is structurally dead. Near SEP26 expiry (2026-09-18) entries would ride
// into dying liquidity un-gated.
//
// SOURCE OF TRUTH: the RESOLVED front contract the C# AddOn already ACKs per
// subscription ("MNQ 09-26" — VLContractResolver.ResolveFrontMonthContractAt),
// stored in SymbolSubState.Contract and reachable via the established
// at.trader.(*ntTrader.TCPTrader) assertion (the feed-gate pattern). No second
// resolver is built (multi-instance defect class).
//
// EXPIRY DERIVATION: CME quarterly equity-index futures expire the THIRD FRIDAY
// of the contract month. Verified against CME's own ProductCalendar API for
// product 146 (E-mini Nasdaq-100): NQU26 lastTrade = settlement = 18 Sep 2026
// (archived original: https://web.archive.org/web/20260710162138id_/https://
// www.cmegroup.com/CmeWS/mvc/ProductCalendar/Future/146). Termination 8:30 CT
// per CME Rulebook 35902.G. MNQ tracks NQ's calendar (same rulebook family).
//
// SEMANTICS: within ROLL_BLOCK_DAYS_BEFORE_EXPIRY calendar days of the
// resolved expiry (CT dates, inclusive; default 3 → Sep 15–18 blocked for
// SEP26), NEW entries are refused; closes and position management continue.
// Resolution unavailable (wire down / pre-P5.3 AddOn / unparseable) → WARN +
// PASS (fail-open rule for entry gates in learning mode). Master-INDEPENDENT.
// NOTE: the C# resolver itself rolls the ACK'd contract to the next quarterly
// ~8 days before expiry, so this gate is the BACKSTOP for the final window —
// the ACK is ground truth for what NT8 would actually trade.

// rollBlockDays reads ROLL_BLOCK_DAYS_BEFORE_EXPIRY (calendar days, default 3).
func rollBlockDays() int {
	if v := os.Getenv("ROLL_BLOCK_DAYS_BEFORE_EXPIRY"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			return n
		}
	}
	return 3
}

// parseResolvedContract parses the AddOn's "ROOT MM-YY" form ("MNQ 09-26") →
// root, month, 4-digit year. ok=false on anything else ("pending", "", codes).
func parseResolvedContract(s string) (root string, month, year int, ok bool) {
	fields := strings.Fields(strings.TrimSpace(s))
	if len(fields) != 2 {
		return "", 0, 0, false
	}
	mmyy := strings.Split(fields[1], "-")
	if len(mmyy) != 2 {
		return "", 0, 0, false
	}
	m, err1 := strconv.Atoi(mmyy[0])
	y, err2 := strconv.Atoi(mmyy[1])
	if err1 != nil || err2 != nil || m < 1 || m > 12 || y < 0 || y > 99 {
		return "", 0, 0, false
	}
	return fields[0], m, 2000 + y, true
}

// thirdFriday returns the third Friday of (year, month) as a CT date.
func thirdFriday(year, month int) time.Time {
	ct := kernel.CTLocation()
	first := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, ct)
	// Days until the first Friday (Friday = 5).
	offset := (int(time.Friday) - int(first.Weekday()) + 7) % 7
	return first.AddDate(0, 0, offset+14)
}

var monthCodes = [...]string{"", "JAN", "FEB", "MAR", "APR", "MAY", "JUN",
	"JUL", "AUG", "SEP", "OCT", "NOV", "DEC"}

// rollVerdict is the PURE decision for one resolved contract at one instant —
// the unit under test (no wire, no clock).
// blocked=true only when the contract parses AND now is within the window.
// display is the human contract name ("MNQ SEP26"); daysLeft is CT calendar
// days from now's date to the expiry date (0 = expiry day).
func rollVerdict(resolvedContract string, now time.Time, windowDays int) (display string, expiry time.Time, daysLeft int, blocked, resolved bool) {
	rootSym, m, y, ok := parseResolvedContract(resolvedContract)
	if !ok {
		return "", time.Time{}, 0, false, false
	}
	expiry = thirdFriday(y, m)
	display = rootSym + " " + monthCodes[m] + strconv.Itoa(y%100)
	ct := now.In(kernel.CTLocation())
	nowDate := time.Date(ct.Year(), ct.Month(), ct.Day(), 0, 0, 0, 0, kernel.CTLocation())
	daysLeft = int(expiry.Sub(nowDate).Hours() / 24)
	blocked = daysLeft >= 0 && daysLeft <= windowDays
	return display, expiry, daysLeft, blocked, true
}

// resolvedFrontContract fetches the ACK'd front contract for this trader's
// futures root via the TCP trader (empty string when unavailable).
func (at *AutoTrader) resolvedFrontContract() string {
	ntTCP, ok := at.trader.(*ntTrader.TCPTrader)
	if !ok {
		return ""
	}
	states := ntTCP.BarsSubscriptionStates()
	st, ok := states[strings.ToUpper(at.futuresSymbol())]
	if !ok {
		return ""
	}
	return st.Contract
}

// entryBlockedByRoll is the gate predicate for executeDecisionWithRecord.
// Fail-open: no/unparseable resolution → WARN once per contract-string change
// and PASS (the C# resolver + T19 remain the other layers).
func (at *AutoTrader) entryBlockedByRoll(now time.Time) (string, bool) {
	if at.config.Exchange != "ninjatrader" {
		return "", false
	}
	contract := at.resolvedFrontContract()
	display, expiry, _, blocked, resolved := rollVerdict(contract, now, rollBlockDays())
	if !resolved {
		if at.lastRollWarnContract != contract {
			at.lastRollWarnContract = contract
			at.logWarnf("⚠️ contract-roll gate: front contract unresolved (%q) — gate PASSES (fail-open). The AddOn ACK carries it after (re)subscription.", contract)
		}
		return "", false
	}
	if !blocked {
		return "", false
	}
	return fmt.Sprintf("%s expires %s, entries blocked (%dd window)",
		display, expiry.Format("2006-01-02"), rollBlockDays()), true
}

// RollStatus surfaces the roll picture for the status API / dashboard (3.6).
func (at *AutoTrader) RollStatus(now time.Time) map[string]interface{} {
	contract := at.resolvedFrontContract()
	display, expiry, daysLeft, blocked, resolved := rollVerdict(contract, now, rollBlockDays())
	if !resolved {
		return map[string]interface{}{"resolved_contract": contract, "resolved": false}
	}
	return map[string]interface{}{
		"resolved_contract": display,
		"resolved":          true,
		"contract_expiry":   expiry.Format("2006-01-02"),
		"roll_window_start": expiry.AddDate(0, 0, -rollBlockDays()).Format("2006-01-02"),
		"roll_days_left":    daysLeft,
		"roll_blocked":      blocked,
	}
}
