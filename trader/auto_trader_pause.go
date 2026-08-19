package trader

import (
	"fmt"
	"strconv"
	"time"

	"nofx/kernel"
)

// P2 — stopUntil PRODUCER (ledger-close dispatch 2026-08-19).
//
// The legacy at.stopUntil field (auto_trader.go) had a consumer at the top of
// runCycle but NO producer anywhere — a pause switch the owner believed existed
// but didn't. That consumer also blocks the WHOLE cycle, which contradicts the
// owner contract ("block NEW entries only; stops, targets, EOD-flat, position
// management continue"). Per additive-only, the legacy field/gate stay dormant
// and untouched; the REAL pause lives here:
//
//   - state: pauseUntilMs (atomic — set from API goroutines, read from the
//     trading loop; the legacy field had no mutex, this one needs no lock)
//   - persistence: system_config key "trader_pause_until:<id>" (unix ms),
//     restored in Run() → survives systemd restart
//   - semantics: NEW entries refused while now < pauseUntil; closes, EOD-flat,
//     the 60s monitorTick guards, and NT8 brackets are untouched (the gate
//     lives in executeDecisionWithRecord and matches open_* only)
//   - expiry: first read at/after the deadline auto-resumes (clears state +
//     store) and logs — no timer needed
//   - master-INDEPENDENT (owner control, like size caps)

func pauseConfigKey(traderID string) string { return "trader_pause_until:" + traderID }

// PauseEntriesUntil arms the pause. source names the actor for the log ("owner").
func (at *AutoTrader) PauseEntriesUntil(until time.Time, source string) error {
	if !until.After(time.Now()) {
		return fmt.Errorf("pause deadline %s is not in the future", until.Format(time.RFC3339))
	}
	at.pauseUntilMs.Store(until.UnixMilli())
	if at.store != nil {
		if err := at.store.SetSystemConfig(pauseConfigKey(at.id), strconv.FormatInt(until.UnixMilli(), 10)); err != nil {
			return fmt.Errorf("pause persisted in memory but store write failed: %w", err)
		}
	}
	at.logWarnf("⏸ stop_until ARMED (%s): NEW entries paused until %s CT. Stops, targets, EOD-flat and position management continue.",
		source, kernel.FormatCT(until))
	return nil
}

// ResumeEntries clears the pause immediately.
func (at *AutoTrader) ResumeEntries(source string) {
	at.pauseUntilMs.Store(0)
	if at.store != nil {
		_ = at.store.SetSystemConfig(pauseConfigKey(at.id), "0")
	}
	at.logInfof("▶️ stop_until CLEARED (%s): entries resume.", source)
}

// loadPersistedPause restores a pause across restart (called from Run()).
func (at *AutoTrader) loadPersistedPause() {
	if at.store == nil {
		return
	}
	raw, err := at.store.GetSystemConfig(pauseConfigKey(at.id))
	if err != nil || raw == "" || raw == "0" {
		return
	}
	ms, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || ms <= time.Now().UnixMilli() {
		return // malformed or already expired → boot unpaused (fail-open)
	}
	at.pauseUntilMs.Store(ms)
	at.logWarnf("⏸ stop_until RESTORED after restart: NEW entries paused until %s CT.",
		kernel.FormatCT(time.UnixMilli(ms)))
}

// PauseState reports (until, active) — the API/status surface.
func (at *AutoTrader) PauseState() (time.Time, bool) {
	ms := at.pauseUntilMs.Load()
	if ms == 0 {
		return time.Time{}, false
	}
	return time.UnixMilli(ms), time.Now().UnixMilli() < ms
}

// entryPaused is the gate predicate: reason + true while the pause holds.
// The first read at/after the deadline AUTO-RESUMES (clears state + store).
func (at *AutoTrader) entryPaused() (string, bool) {
	ms := at.pauseUntilMs.Load()
	if ms == 0 {
		return "", false
	}
	if time.Now().UnixMilli() >= ms {
		// Expiry: only the winner of the CAS clears + logs (loop vs API race).
		if at.pauseUntilMs.CompareAndSwap(ms, 0) {
			if at.store != nil {
				_ = at.store.SetSystemConfig(pauseConfigKey(at.id), "0")
			}
			at.logInfof("▶️ stop_until EXPIRED: pause until %s CT elapsed — entries auto-resume.",
				kernel.FormatCT(time.UnixMilli(ms)))
		}
		return "", false
	}
	return fmt.Sprintf("paused until %s CT (owner)", kernel.FormatCT(time.UnixMilli(ms))), true
}

// pauseStatusString renders the status-API "stop_until" value: the pause
// deadline while armed, else the zero time — byte-identical to the legacy
// (never-assigned) field's rendering when unpaused.
func pauseStatusString(at *AutoTrader) string {
	if until, active := at.PauseState(); active {
		return until.Format(time.RFC3339)
	}
	return time.Time{}.Format(time.RFC3339)
}

// SessionEndTime resolves the ACTIVE session's window end as a wall-clock
// instant (wrap-aware: an end HH:MM at/before now belongs to tomorrow CT).
// ok=false when no session is active (overnight/interim — nothing to pause to).
func (at *AutoTrader) SessionEndTime(now time.Time) (time.Time, bool) {
	sess, ok := at.sessionRegistry(now).ActiveSession(now)
	if !ok {
		return time.Time{}, false
	}
	endMin, okMin := hhmmToMin(sess.WindowEndCT)
	if !okMin {
		return time.Time{}, false
	}
	ct := now.In(kernel.CTLocation())
	end := time.Date(ct.Year(), ct.Month(), ct.Day(), endMin/60, endMin%60, 0, 0, kernel.CTLocation())
	if !end.After(now) {
		end = end.Add(24 * time.Hour) // wrapped session (e.g. ASIA → 02:00 next day)
	}
	return end, true
}
