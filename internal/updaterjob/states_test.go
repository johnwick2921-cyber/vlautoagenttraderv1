package updaterjob

import (
	"reflect"
	"sort"
	"testing"
)

// wantRow is the test's OWN copy of the state table (brief §3.3 + the C10
// additions ruled by CTO 1790259689740). The validator test derives the legal
// edge set from THIS copy, never from the production table, so a production
// edit that adds or drops an edge is a diff against a reviewed pin.
type wantRow struct {
	eff    Effect
	succ   []State
	fail   State
	cancel bool
	parks  bool
}

var wantOrder = []State{
	StateRequested, StateDownloaded, StateVerified, StatePreflightOK, StateMaintenanceHeld,
	StateDrainedAcked, StateGateOK, StateBackupDone, StateNT8Skipped, StateNT8Updated,
	StateActivated, StateBooted, StateBootVerified, StateComplete, StateRollingBack,
	StateRolledBack, StateRecoveryNeeded, StateCancelled, StateRefused,
}

var wantTable = map[State]wantRow{
	StateRequested:       {EffectNone, []State{StateDownloaded}, "", true, false},
	StateDownloaded:      {EffectDownload, []State{StateVerified}, StateRefused, true, false},
	StateVerified:        {EffectVerify, []State{StatePreflightOK}, StateRefused, true, false},
	StatePreflightOK:     {EffectPreflight, []State{StateMaintenanceHeld}, StateRefused, true, false},
	StateMaintenanceHeld: {EffectHold, []State{StateDrainedAcked}, StateRefused, false, false},
	StateDrainedAcked:    {EffectDrain, []State{StateGateOK}, StateRecoveryNeeded, false, false},
	StateGateOK:          {EffectGate, []State{StateBackupDone}, StateRecoveryNeeded, false, false},
	StateBackupDone:      {EffectBackup, []State{StateNT8Skipped, StateNT8Updated}, StateRecoveryNeeded, false, false},
	StateNT8Skipped:      {EffectNT8, []State{StateActivated}, StateRecoveryNeeded, false, false},
	StateNT8Updated:      {EffectNT8, []State{StateActivated}, StateRecoveryNeeded, false, true},
	StateActivated:       {EffectActivate, []State{StateBooted}, StateRollingBack, false, false},
	StateBooted:          {EffectWatch, []State{StateBootVerified}, StateRollingBack, false, false},
	StateBootVerified:    {EffectBootVerify, []State{StateComplete}, StateRollingBack, false, false},
	StateComplete:        {EffectReleaseHold, nil, StateRecoveryNeeded, false, false},
	StateRollingBack:     {EffectRollback, []State{StateRolledBack}, StateRecoveryNeeded, false, false},
	StateRolledBack:      {EffectReleaseHold, nil, StateRecoveryNeeded, false, false},
	StateRecoveryNeeded:  {EffectNone, nil, "", false, false},
	StateCancelled:       {EffectNone, nil, "", false, false},
	StateRefused:         {EffectNone, nil, "", false, false},
}

var phases = []Phase{PhaseStarted, PhaseDone}

// TestStateTableIsExhaustive: every state has a row with a side-effect name,
// its success and failure successors and its cancellable flag; the row set,
// the terminal set, the cancellable set and the hold effects are exactly the
// ruled ones; the graph is acyclic, every state is reachable, every state can
// finish, and nothing past the hold can end "refused" or "cancelled" (those
// two mean "no hold was ever written").
func TestStateTableIsExhaustive(t *testing.T) {
	if got := AllStates(); !reflect.DeepEqual(got, wantOrder) {
		t.Fatalf("AllStates() = %v\nwant %v", got, wantOrder)
	}
	for _, s := range wantOrder {
		r, ok := Lookup(s)
		if !ok {
			t.Errorf("%s: no row", s)
			continue
		}
		w := wantTable[s]
		if r.State != s {
			t.Errorf("%s: row names state %q", s, r.State)
		}
		if r.Effect == "" {
			t.Errorf("%s: no side-effect name", s)
		}
		if r.Effect != w.eff {
			t.Errorf("%s: effect %q, want %q", s, r.Effect, w.eff)
		}
		if !reflect.DeepEqual(append([]State(nil), r.Success...), append([]State(nil), w.succ...)) {
			t.Errorf("%s: success %v, want %v", s, r.Success, w.succ)
		}
		if r.Failure != w.fail {
			t.Errorf("%s: failure %q, want %q", s, r.Failure, w.fail)
		}
		if r.Cancellable != w.cancel {
			t.Errorf("%s: cancellable %v, want %v", s, r.Cancellable, w.cancel)
		}
		if r.Parks != w.parks {
			t.Errorf("%s: parks %v, want %v", s, r.Parks, w.parks)
		}
		if (r.Effect == EffectNone) != (r.Failure == "") {
			t.Errorf("%s: a failure successor exists iff a side effect runs (effect %q, failure %q)", s, r.Effect, r.Failure)
		}
		if IsTerminal(s) != (len(r.Success) == 0) {
			t.Errorf("%s: IsTerminal=%v but %d success successors", s, IsTerminal(s), len(r.Success))
		}
		for _, n := range append(append([]State(nil), r.Success...), r.Failure) {
			if n == "" {
				continue
			}
			if _, ok := Lookup(n); !ok {
				t.Errorf("%s: successor %q has no row", s, n)
			}
		}
		wantEntry := PhaseStarted
		if r.Effect == EffectNone {
			wantEntry = PhaseDone
		}
		if EntryPhase(s) != wantEntry {
			t.Errorf("%s: entered %q, want %q", s, EntryPhase(s), wantEntry)
		}
	}
	if _, ok := Lookup("verified_again"); ok {
		t.Error("Lookup accepts an unknown state")
	}

	// The sets, exactly.
	var terminal, cancellable, holds, clears []string
	for _, s := range AllStates() {
		r, _ := Lookup(s)
		if IsTerminal(s) {
			terminal = append(terminal, string(s))
		}
		if r.Cancellable {
			cancellable = append(cancellable, string(s))
		}
		if r.Effect == EffectHold {
			holds = append(holds, string(s))
		}
		if r.Effect == EffectReleaseHold {
			clears = append(clears, string(s))
		}
	}
	for name, c := range map[string]struct{ got, want []string }{
		"terminal":    {terminal, []string{"cancelled", "complete", "recovery_needed", "refused", "rolled_back"}},
		"cancellable": {cancellable, []string{"downloaded", "preflight_ok", "requested", "verified"}},
		"hold write":  {holds, []string{"maintenance_held"}},
		"hold clear":  {clears, []string{"complete", "rolled_back"}},
	} {
		sort.Strings(c.got)
		if !reflect.DeepEqual(c.got, c.want) {
			t.Errorf("%s set = %v, want %v", name, c.got, c.want)
		}
	}

	// Finished: exactly the terminal states, once DONE.
	for _, s := range AllStates() {
		for _, p := range phases {
			want := IsTerminal(s) && p == PhaseDone
			if Finished(s, p) != want {
				t.Errorf("Finished(%s,%s) = %v, want %v", s, p, Finished(s, p), want)
			}
		}
	}

	// Graph: table edges only (success + failure; the universal edges to
	// recovery_needed/cancelled are checked by the validator test).
	next := func(s State) []State {
		r, _ := Lookup(s)
		out := append([]State(nil), r.Success...)
		if r.Failure != "" {
			out = append(out, r.Failure)
		}
		return out
	}
	// acyclic (so a legal job enters each state at most once)
	const (
		white = iota
		grey
		black
	)
	color := map[State]int{}
	var visit func(State) bool
	visit = func(s State) bool {
		color[s] = grey
		for _, n := range next(s) {
			if color[n] == grey {
				t.Errorf("cycle through %s → %s", s, n)
				return false
			}
			if color[n] == white && !visit(n) {
				return false
			}
		}
		color[s] = black
		return true
	}
	visit(StateRequested)
	// reachable from requested (the universal edges reach recovery_needed and
	// cancelled; everything else must be reachable through the table)
	for _, s := range AllStates() {
		if color[s] == white && s != StateCancelled {
			t.Errorf("%s is unreachable from requested through the table", s)
		}
	}
	// every state can reach a terminal state
	for _, s := range AllStates() {
		seen := map[State]bool{}
		var reach func(State) bool
		reach = func(x State) bool {
			if IsTerminal(x) {
				return true
			}
			if seen[x] {
				return false
			}
			seen[x] = true
			for _, n := range next(x) {
				if reach(n) {
					return true
				}
			}
			return false
		}
		if !reach(s) {
			t.Errorf("%s can never finish", s)
		}
	}
	// nothing at or past a DONE hold reaches refused or cancelled
	// (maintenance_held's own failure is refused: its write failed, so no hold
	// of ours exists; every state AFTER it may hold)
	post := map[State]bool{}
	var mark func(State)
	mark = func(s State) {
		for _, n := range next(s) {
			if !post[n] {
				post[n] = true
				mark(n)
			}
		}
	}
	for _, s := range []State{StateDrainedAcked} {
		post[s] = true
		mark(s)
	}
	for _, bad := range []State{StateRefused, StateCancelled} {
		if post[bad] {
			t.Errorf("%s is reachable after the hold — it would claim no hold was ever written", bad)
		}
	}
}

// legalMove is the rule, derived from the TEST's pinned table.
func legalMove(from State, p Phase, to State) bool {
	w, ok := wantTable[from]
	if !ok {
		return false
	}
	if _, ok := wantTable[to]; !ok {
		return false
	}
	if len(w.succ) == 0 && p == PhaseDone { // finished
		return false
	}
	if to == StateRecoveryNeeded {
		return true
	}
	if to == StateCancelled && w.cancel {
		return true
	}
	if p == PhaseDone {
		for _, s := range w.succ {
			if s == to {
				return true
			}
		}
		return false
	}
	return to == w.fail
}

// TestTransitionValidatorAllowsOnlyTableEdges: every (from, phase, to) over
// the whole state space, plus unknown states, against the pinned rule; and
// the named forbidden edges the dispatch cares about, each spelled out.
func TestTransitionValidatorAllowsOnlyTableEdges(t *testing.T) {
	all := append(append([]State(nil), wantOrder...), "verified", "", "COMPLETE")
	legal := 0
	for _, from := range all {
		for _, p := range append(append([]Phase(nil), phases...), "", "running") {
			for _, to := range all {
				want := legalMove(from, p, to) && (p == PhaseStarted || p == PhaseDone)
				err := CheckMove(from, p, to)
				if (err == nil) != want {
					t.Errorf("CheckMove(%q,%q,%q) = %v, want legal=%v", from, p, to, err, want)
				}
				if want {
					legal++
				}
			}
		}
	}
	if legal == 0 {
		t.Fatal("positive control: no legal move at all")
	}
	for _, c := range []struct {
		from State
		p    Phase
		to   State
		why  string
	}{
		{StatePreflightOK, PhaseDone, StateDrainedAcked, "skips the hold"},
		{StateRequested, PhaseDone, StateActivated, "skips everything"},
		{StateBooted, PhaseDone, StateComplete, "skips boot_verified"},
		{StateBackupDone, PhaseDone, StateActivated, "skips the AddOn decision"},
		{StateDownloaded, PhaseStarted, StateVerified, "success before the receipt is persisted"},
		{StateActivated, PhaseDone, StateCancelled, "cancel past the boundary"},
		{StateMaintenanceHeld, PhaseStarted, StateCancelled, "cancel while the hold may be on disk"},
		{StateDrainedAcked, PhaseStarted, StateRefused, "refused after a hold"},
		{StateActivated, PhaseStarted, StateRefused, "refused after the swap began"},
		{StateComplete, PhaseDone, StateRecoveryNeeded, "a finished job moves"},
		{StateCancelled, PhaseDone, StateDownloaded, "a finished job restarts"},
		{StateGateOK, PhaseDone, StateGateOK, "self edge"},
		{StateRollingBack, PhaseStarted, StateRollingBack, "self edge"},
	} {
		if err := CheckMove(c.from, c.p, c.to); err == nil {
			t.Errorf("CheckMove(%s,%s,%s) allowed — %s", c.from, c.p, c.to, c.why)
		}
	}
	// CheckFinish: only a STARTED state with a side effect finishes in place.
	for _, s := range all {
		for _, p := range append(append([]Phase(nil), phases...), "") {
			w, ok := wantTable[s]
			want := ok && p == PhaseStarted && w.eff != EffectNone
			if err := CheckFinish(s, p); (err == nil) != want {
				t.Errorf("CheckFinish(%q,%q) = %v, want ok=%v", s, p, err, want)
			}
		}
	}
}
