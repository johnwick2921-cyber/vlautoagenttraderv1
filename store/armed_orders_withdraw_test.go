package store

import (
	"path/filepath"
	"testing"
)

// ── W-EXEC-TRUTH W0 (f) — a withdrawn row keeps its withdraw head, and the
// withdraw view finds a job's rows by the exact head ────────────────────────

func withdrawTestLedger(t *testing.T) *ArmedOrderStore {
	t.Helper()
	st, err := New(filepath.Join(t.TempDir(), "wd.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	return st.ArmedOrders()
}

func withdrawTestRow(t *testing.T, led *ArmedOrderStore, scenario, sid string) *ArmedOrderDB {
	t.Helper()
	r := &ArmedOrderDB{TraderID: "t-wd", PlanID: "p", Scenario: scenario, Version: 1, State: StateArmed, Side: "long", EntryPx: 100, StopPx: 95, TargetPx: 110}
	if err := led.UpsertArm(r); err != nil {
		t.Fatal(err)
	}
	if err := led.BeginPlacement(r.ID, sid); err != nil {
		t.Fatal(err)
	}
	return r
}

func reasonOf(t *testing.T, led *ArmedOrderStore, id int64) (string, string) {
	t.Helper()
	var r ArmedOrderDB
	if err := led.DB().First(&r, id).Error; err != nil {
		t.Fatal(err)
	}
	return r.State, r.StateReason
}

func TestWithdrawHeadSurvivesEveryLifecycleWrite(t *testing.T) {
	led := withdrawTestLedger(t)
	w := withdrawTestRow(t, led, "S1", "sig-w")
	plain := withdrawTestRow(t, led, "S2", "sig-p")
	head := WithdrawReasonPrefix + "maintenance job j1"
	if err := led.RequestCancel(w.ID, head, 1000); err != nil {
		t.Fatal(err)
	}
	if _, got := reasonOf(t, led, w.ID); got != head {
		t.Fatalf("the withdraw writes its head: %q", got)
	}
	if err := led.RequestCancel(w.ID, "re-requested after 5s unconfirmed", 2000); err != nil {
		t.Fatal(err)
	}
	if err := led.SetState(w.ID, StateCancelled, "cancelled in NT8"); err != nil {
		t.Fatal(err)
	}
	state, got := reasonOf(t, led, w.ID)
	want := head + WithdrawReasonSep + "re-requested after 5s unconfirmed" + WithdrawReasonSep + "cancelled in NT8"
	if state != StateCancelled || got != want {
		t.Fatalf("the head must survive, later reasons appended: got %s %q, want %q", state, got, want)
	}
	// A row that was never withdrawn is written exactly as before.
	if err := led.RequestCancel(plain.ID, "session end", 1000); err != nil {
		t.Fatal(err)
	}
	if err := led.ConfirmCancel(plain.ID, 7, "gone from snapshot 7"); err != nil {
		t.Fatal(err)
	}
	if _, got := reasonOf(t, led, plain.ID); got != "gone from snapshot 7" {
		t.Fatalf("a non-withdraw row's reason is replaced as before: %q", got)
	}
}

func TestListWithdrawnMatchesTheExactHeadNotAPrefix(t *testing.T) {
	led := withdrawTestLedger(t)
	for _, c := range []struct{ scen, sid, job string }{
		{"S1", "s1", "job1"}, {"S2", "s2", "job10"}, {"S3", "s3", "job_1"}, {"S4", "s4", "jobX1"},
	} {
		r := withdrawTestRow(t, led, c.scen, c.sid)
		if err := led.RequestCancel(r.ID, WithdrawReasonPrefix+"maintenance job "+c.job, 1000); err != nil {
			t.Fatal(err)
		}
		if c.job == "job1" {
			if err := led.RequestCancel(r.ID, "re-requested", 2000); err != nil {
				t.Fatal(err)
			}
		}
	}
	for job, want := range map[string]string{"job1": "S1", "job10": "S2", "job_1": "S3", "jobX1": "S4"} {
		rows, err := led.ListWithdrawn(WithdrawReasonPrefix + "maintenance job " + job)
		if err != nil {
			t.Fatal(err)
		}
		if len(rows) != 1 || rows[0].Scenario != want {
			got := []string{}
			for _, r := range rows {
				got = append(got, r.Scenario)
			}
			t.Errorf("job %q must list exactly %s, got %v", job, want, got)
		}
	}
}
