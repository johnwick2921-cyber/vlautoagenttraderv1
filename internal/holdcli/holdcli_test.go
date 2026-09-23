package holdcli

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"nofx/store"
)

// The operator CLI drives the SAME store functions the gates read.
func TestCLISetStatusClear(t *testing.T) {
	inst := t.TempDir()
	t.Setenv("DB_PATH", "")
	dir := DataDirFor(inst)
	var out, errb bytes.Buffer
	if rc := Run([]string{"--install-dir", inst, "set", "--job", "job-7", "--reason", "test"}, &out, &errb); rc != 0 {
		t.Fatalf("set rc=%d err=%s", rc, errb.String())
	}
	if st := store.ReadMaintenanceHold(dir); !st.Held || st.Hold.JobID != "job-7" || st.Hold.Owner != "cli" {
		t.Fatalf("set must write a held file owned by cli: %+v", st)
	}
	out.Reset()
	if rc := Run([]string{"--install-dir", inst, "status"}, &out, &errb); rc != 0 {
		t.Fatalf("status rc=%d", rc)
	}
	var got map[string]any
	if err := json.Unmarshal(out.Bytes(), &got); err != nil || got["held"] != true || got["job_id"] != "job-7" {
		t.Fatalf("status JSON: %s (%v)", out.String(), err)
	}
	if rc := Run([]string{"--install-dir", inst, "clear", "--job", "other"}, &out, &errb); rc == 0 {
		t.Fatal("clear with the wrong job must fail")
	}
	if rc := Run([]string{"--install-dir", inst, "clear", "--job", "job-7"}, &out, &errb); rc != 0 {
		t.Fatalf("clear rc=%d err=%s", rc, errb.String())
	}
	if st := store.ReadMaintenanceHold(dir); st.Present {
		t.Fatalf("after clear: %+v", st)
	}
}

func TestCLIRefusesSetWithoutJob(t *testing.T) {
	var out, errb bytes.Buffer
	if rc := Run([]string{"--install-dir", t.TempDir(), "set"}, &out, &errb); rc == 0 {
		t.Fatal("set without --job must fail")
	}
	if !strings.Contains(errb.String(), "--job") {
		t.Fatalf("error must name the missing flag: %q", errb.String())
	}
}
