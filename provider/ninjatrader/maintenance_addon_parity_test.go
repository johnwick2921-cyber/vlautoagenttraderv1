package ninjatrader

import (
	"os"
	"reflect"
	"strings"
	"testing"
)

// ── W-ONE-BUTTON M2 site 7 — Go ⟷ AddOn lockstep for the maintenance wire ──
//
// The TCP schema is a high-cascade contract: Go structs and the C# AddOn must
// change together. The AddOn cannot run here, so this pins its SOURCE against
// the Go structs' json tags (read by reflection, so a rename on either side
// fails): every ack / census / hello-epoch key the Go decoder reads is one the
// AddOn writes; the ack builder never touches a name; and HandleSignal's hold
// refusal runs before any order is created.

func csSource(t *testing.T) string {
	t.Helper()
	b, err := os.ReadFile("../../ninjascript/VLTraderTCPClient.cs")
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// csMethod returns the body of a C# method, from its signature to the first
// line that closes it at the method's indentation.
func csMethod(t *testing.T, src, signature string) string {
	t.Helper()
	i := strings.Index(src, signature)
	if i < 0 {
		t.Fatalf("AddOn method not found: %s", signature)
	}
	j := strings.Index(src[i:], "\n        }\n")
	if j < 0 {
		t.Fatalf("AddOn method not closed: %s", signature)
	}
	return src[i : i+j]
}

func jsonTags(v any) []string {
	var out []string
	rt := reflect.TypeOf(v)
	for i := 0; i < rt.NumField(); i++ {
		tag := strings.Split(rt.Field(i).Tag.Get("json"), ",")[0]
		if tag != "" && tag != "-" {
			out = append(out, tag)
		}
	}
	return out
}

func TestAddOnMaintenanceAckWritesEveryKeyGoReads(t *testing.T) {
	src := csSource(t)
	ack := csMethod(t, src, "private Dictionary<string, object> BuildMaintenanceAck(")
	for _, v := range []any{MaintenanceAckPayload{}, CensusConnection{}, CensusAccount{}} {
		for _, tag := range jsonTags(v) {
			if !strings.Contains(ack, `["`+tag+`"]`) {
				t.Errorf("%T.%s: the AddOn's BuildMaintenanceAck never writes [\"%s\"]", v, tag, tag)
			}
		}
	}
	if strings.Contains(strings.ReplaceAll(ack, "ex.GetType().Name", ""), ".Name") {
		t.Error("BuildMaintenanceAck reads a .Name — the census must never carry an account or connection name")
	}
	if !strings.Contains(ack, "ex.GetType().Name") || strings.Contains(ack, "ex.Message") {
		t.Error("census_error must carry the exception TYPE only (a message can carry an account name)")
	}
	handle := csMethod(t, src, "private void HandleMaintenance(")
	if !strings.Contains(handle, `WriteEnvelope("maintenance_ack"`) {
		t.Error("HandleMaintenance must answer every notice with a maintenance_ack")
	}
	if !strings.Contains(src, `type == "`+string(FrameMaintenance)+`"`) {
		t.Errorf("the AddOn's HandleFrame does not dispatch %q", FrameMaintenance)
	}
	if !strings.Contains(src, `"`+string(FrameMaintenanceAck)+`"`) {
		t.Errorf("the AddOn never writes %q", FrameMaintenanceAck)
	}
}

func TestAddOnHelloWritesTheEpochKeysGoReads(t *testing.T) {
	src := csSource(t)
	hello := csMethod(t, src, "private void SendHello(")
	for _, tag := range jsonTags(HelloPayload{}) {
		if !strings.Contains(hello, `["`+tag+`"]`) {
			t.Errorf("HelloPayload.%s: the AddOn's SendHello never writes [\"%s\"]", tag, tag)
		}
	}
}

func TestAddOnRefusesEntriesWhileHeldBeforeCreatingAnOrder(t *testing.T) {
	src := csSource(t)
	sig := csMethod(t, src, "private void HandleSignal(")
	held := strings.Index(sig, "if (maintenanceHeld)")
	create := strings.Index(sig, "CreateOrder(")
	if held < 0 {
		t.Fatal("HandleSignal has no maintenance-hold refusal")
	}
	if create >= 0 && held > create {
		t.Fatal("HandleSignal's hold refusal must come before CreateOrder")
	}
	if !strings.Contains(sig[held:], `"rejected"`) {
		t.Fatal("the hold refusal must answer with the existing rejected fill frame")
	}
}

// M2.1 (review 3 F14) — the AddOn resets the hold on EVERY connect, before
// its hello: Go re-sends the hold at accept while held, and never sends
// held:false to a fresh connection, so a hold that survived a reconnect would
// refuse entries forever.
func TestAddOnResetsTheHoldOnEveryConnectBeforeItsHello(t *testing.T) {
	loop := csMethod(t, csSource(t), "private void RunConnectionLoop(")
	conn := strings.Index(loop, "stream = client.GetStream();")
	reset := strings.Index(loop, "maintenanceHeld = false;")
	hello := strings.Index(loop, "SendHello();")
	if conn < 0 || reset < 0 || hello < 0 || !(conn < reset && reset < hello) {
		t.Fatalf("RunConnectionLoop must reset maintenanceHeld after connecting and before SendHello (connect=%d reset=%d hello=%d)", conn, reset, hello)
	}
}

// M2.1 (review d) — source_hash is taken when the AddOn ACTIVATES, not at the
// first hello: a copy-before-F5 must never carry the new file's hash under the
// old compile.
func TestAddOnHashesItsSourceAtActivation(t *testing.T) {
	src := csSource(t)
	act := strings.Index(src, "sourceHashCache = ComputeSourceHash();")
	reader := strings.Index(src, "readerThread = new Thread(")
	if act < 0 || reader < 0 || act > reader {
		t.Fatalf("source_hash must be computed in the activation block, before the reader thread starts (hash=%d reader=%d)", act, reader)
	}
}

// M2.1 (review d, [C] deadlock risk) — the census never takes an NT8 lock
// while holding another (it runs on the read thread that also carries the
// close / protective frames).
func TestAddOnCensusNeverNestsLocks(t *testing.T) {
	body := csMethod(t, csSource(t), "private Dictionary<string, object> BuildMaintenanceAck(")
	depth, lockDepths := 0, []int{}
	for _, line := range strings.Split(body, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "lock (") {
			if len(lockDepths) > 0 {
				t.Fatalf("nested lock in BuildMaintenanceAck: %q", trimmed)
			}
			lockDepths = append(lockDepths, depth) // closes when depth returns here
		}
		depth += strings.Count(line, "{") - strings.Count(line, "}")
		for len(lockDepths) > 0 && depth <= lockDepths[len(lockDepths)-1] && !strings.HasPrefix(trimmed, "lock (") {
			lockDepths = lockDepths[:len(lockDepths)-1]
		}
	}
}
