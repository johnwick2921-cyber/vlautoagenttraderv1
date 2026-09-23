package trader

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	ntwire "nofx/provider/ninjatrader"
	"nofx/store"
)

// ── W-ONE-BUTTON M2 — GET /api/maintenance and the 🔒 boot line ────────────
//
// {held, job_id, since, in_flight_sends, drained, addon_ack:{received, held,
// queued_commands, build_id}|null}. A value the process does not know is null
// on the API and n/a on the boot line — never a guess (canon: boot lines are
// READ; an uncomputed value is absent, not empty).

func statusJSON(t *testing.T, v MaintenanceStatusView) map[string]any {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatal(err)
	}
	return m
}

func TestMaintenanceStatusAbsentHold(t *testing.T) {
	withMaintenanceDir(t)
	m := statusJSON(t, MaintenanceStatus(nil))
	for _, k := range []string{"held", "job_id", "since", "in_flight_sends", "drained", "addon_ack"} {
		if _, ok := m[k]; !ok {
			t.Fatalf("key %q missing from %v", k, m)
		}
	}
	if m["held"] != false || m["job_id"] != nil || m["since"] != nil || m["addon_ack"] != nil || m["state"] != "clear" {
		t.Fatalf("absent hold: want held=false, job_id/since/addon_ack null, state clear; got %v", m)
	}
	if line := MaintenanceBootLine(nil); line != "🔒 maintenance: hold=clear job=n/a since=n/a addon_ack=n/a" {
		t.Fatalf("boot line: %q", line)
	}
}

func TestMaintenanceStatusHeldAndUnreadable(t *testing.T) {
	dir := withMaintenanceDir(t)
	setHold(t, dir, "job-s")
	st := store.ReadMaintenanceHold(dir)
	m := statusJSON(t, MaintenanceStatus(nil))
	if m["held"] != true || m["job_id"] != "job-s" || m["since"] != st.Hold.Since || m["state"] != "held" || m["drained"] != true {
		t.Fatalf("held: %v", m)
	}
	want := "🔒 maintenance: hold=held job=job-s since=" + st.Hold.Since + " addon_ack=n/a"
	if line := MaintenanceBootLine(nil); line != want {
		t.Fatalf("boot line:\n got %q\nwant %q", line, want)
	}
	if err := writeRaw(dir, "{broken"); err != nil {
		t.Fatal(err)
	}
	m = statusJSON(t, MaintenanceStatus(nil))
	if m["held"] != true || m["state"] != "unreadable" || m["job_id"] != nil {
		t.Fatalf("unreadable: held (fail-closed), no job: %v", m)
	}
	if line := MaintenanceBootLine(nil); !strings.HasPrefix(line, "🔒 maintenance: hold=unreadable job=n/a since=n/a") {
		t.Fatalf("boot line: %q", line)
	}
}

// addon_ack is the CURRENT connection's ack only; a disconnected record's ack
// is history and reads null.
func TestMaintenanceStatusAddOnAck(t *testing.T) {
	dir := withMaintenanceDir(t)
	setHold(t, dir, "job-s")
	prev := installationWireView
	t.Cleanup(func() { installationWireView = prev })
	wire := installationWire{Connected: true, HasAck: true, AckAge: 1500 * time.Millisecond,
		Rec: ntwire.ConnectionRecord{AcceptSeq: 4, Ack: &ntwire.MaintenanceAckPayload{Held: true, JobID: "job-s", QueuedCommands: 0, BuildID: "2026-09-22-m2"}}}
	installationWireView = func([]*AutoTrader) (installationWire, bool) { return wire, true }
	m := statusJSON(t, MaintenanceStatus(nil))
	ack, ok := m["addon_ack"].(map[string]any)
	if !ok || ack["held"] != true || ack["build_id"] != "2026-09-22-m2" || ack["queued_commands"] != float64(0) || ack["received"] == nil {
		t.Fatalf("addon_ack: %v", m["addon_ack"])
	}
	if line := MaintenanceBootLine(nil); !strings.Contains(line, "addon_ack=held job=job-s build=2026-09-22-m2") {
		t.Fatalf("boot line must READ the ack: %q", line)
	}
	wire.Connected = false
	if m := statusJSON(t, MaintenanceStatus(nil)); m["addon_ack"] != nil {
		t.Fatalf("a disconnected record's ack is history, not a live ack: %v", m["addon_ack"])
	}
}

// main.go prints the line after the data dir is configured (it READS the file).
func TestMainPrintsTheMaintenanceBootLine(t *testing.T) {
	b, err := os.ReadFile("../main.go")
	if err != nil {
		t.Fatal(err)
	}
	src := string(b)
	set := strings.Index(src, "trader.SetMaintenanceDataDir(")
	line := strings.Index(src, "trader.MaintenanceBootLine(")
	if line < 0 {
		t.Fatal("main.go never prints trader.MaintenanceBootLine()")
	}
	if set < 0 || set > line {
		t.Fatal("the boot line must be printed after SetMaintenanceDataDir")
	}
}
