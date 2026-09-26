package holdcli

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"nofx/store"
)

// W-EXEC-TRUTH W0 (f) — `set --withdraw-entries` writes WithdrawEntries into
// the SAME hold file the bot's withdraw beat reads; without the flag the file
// carries no withdraw_entries key at all (never implied, and a pre-W0 reader
// sees the file it always saw).
func TestCLISetWithdrawEntriesFlag(t *testing.T) {
	inst := botInstall(t)
	dir := DataDirFor(inst)

	var out, errb bytes.Buffer
	if rc := Run([]string{"--install-dir", inst, "set", "--job", "job-wd", "--withdraw-entries"}, &out, &errb); rc != 0 {
		t.Fatalf("set --withdraw-entries rc=%d err=%s", rc, errb.String())
	}
	st := store.ReadMaintenanceHold(dir)
	if !st.Held || st.Hold.JobID != "job-wd" || !st.Hold.WithdrawEntries {
		t.Fatalf("--withdraw-entries must write a held file asking for the withdraw: %+v", st)
	}
	if !strings.Contains(out.String(), "withdraw_entries=true") {
		t.Fatalf("the set line must say the withdraw was asked for: %q", out.String())
	}
	if rc := Run([]string{"--install-dir", inst, "clear", "--job", "job-wd"}, &out, &errb); rc != 0 {
		t.Fatalf("clear rc=%d err=%s", rc, errb.String())
	}

	out.Reset()
	if rc := Run([]string{"--install-dir", inst, "set", "--job", "job-plain"}, &out, &errb); rc != 0 {
		t.Fatalf("set rc=%d err=%s", rc, errb.String())
	}
	st = store.ReadMaintenanceHold(dir)
	if !st.Held || st.Hold.WithdrawEntries {
		t.Fatalf("without the flag the hold must NOT ask for a withdraw: %+v", st)
	}
	if !strings.Contains(out.String(), "withdraw_entries=false") {
		t.Fatalf("the set line must say no withdraw was asked for: %q", out.String())
	}
	raw, err := os.ReadFile(store.MaintenanceHoldPath(dir))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "withdraw_entries") {
		t.Fatalf("a hold without the flag must not carry the key at all: %s", raw)
	}
}
