package holdcli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"nofx/store"
)

// The operator CLI drives the SAME store functions the gates read.
func TestCLISetStatusClear(t *testing.T) {
	inst := botInstall(t)
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
	if rc := Run([]string{"--install-dir", botInstall(t), "set"}, &out, &errb); rc == 0 {
		t.Fatal("set without --job must fail")
	}
	if !strings.Contains(errb.String(), "--job") {
		t.Fatalf("error must name the missing flag: %q", errb.String())
	}
}

// M2.1 (review 2 N5): --install-dir defaults to the shell's directory. Run from
// anywhere else, 'set' used to write a hold the bot never reads and print
// "held". It now refuses unless the bot's database exists where the ONE
// resolver says it is, and names that path.
func TestCLIRefusesWithoutABotDatabase(t *testing.T) {
	inst := t.TempDir() // no data/data.db here: not the bot's installation
	t.Setenv("DB_PATH", "")
	var out, errb bytes.Buffer
	if rc := Run([]string{"--install-dir", inst, "set", "--job", "job-x"}, &out, &errb); rc != 2 {
		t.Fatalf("set without the bot's database must refuse with rc=2, got %d (%s)", rc, errb.String())
	}
	if !strings.Contains(errb.String(), filepath.Join(inst, "data", "data.db")) {
		t.Fatalf("the refusal must name the database path it looked for: %s", errb.String())
	}
	if _, err := os.Stat(filepath.Join(inst, "data", "updater", "hold.json")); !os.IsNotExist(err) {
		t.Fatal("a refused set must not write a hold file")
	}
	out.Reset()
	errb.Reset()
	if rc := Run([]string{"--install-dir", inst, "status"}, &out, &errb); rc != 0 || !strings.Contains(out.String(), `"db_present":false`) {
		t.Fatalf("status must still answer, and say the database is absent: rc=%d out=%s", rc, out.String())
	}
}

// M2.1 (review 2 N4): as root, 'set' would create a root-owned data/updater;
// the bot's stat then fails EACCES and every entry is refused as "unreadable"
// until someone repairs the directory. The CLI refuses to run as root.
func TestCLIRefusesToRunAsRoot(t *testing.T) {
	inst := botInstall(t)
	prev := geteuid
	geteuid = func() int { return 0 }
	t.Cleanup(func() { geteuid = prev })
	var out, errb bytes.Buffer
	if rc := Run([]string{"--install-dir", inst, "set", "--job", "job-r"}, &out, &errb); rc != 2 || !strings.Contains(errb.String(), "root") {
		t.Fatalf("set as root must refuse with rc=2 and say why: rc=%d %s", rc, errb.String())
	}
}

// botInstall is an installation directory with the bot's database present.
func botInstall(t *testing.T) string {
	t.Helper()
	inst := t.TempDir()
	t.Setenv("DB_PATH", "")
	if err := os.MkdirAll(filepath.Join(inst, "data"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(inst, "data", "data.db"), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	return inst
}
