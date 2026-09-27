package updateauth

// W-ONE-BUTTON M3 red-team fold (red-3 #4): Enroll creates the seen-job
// store; a replace never resets an existing one (the key rotates, the
// consumed ids stay consumed); once enrolled, a missing store is
// ErrSeenCorrupt, not empty — and a replace re-creates a missing one.

import (
	"bytes"
	"errors"
	"os"
	"testing"
)

func TestEnrollCreatesTheSeenStoreAndAMissingOneAfterwardsIsCorrupt(t *testing.T) {
	d := enrolled(t)
	b, err := os.ReadFile(SeenPath(d))
	if err != nil {
		t.Fatalf("Enroll did not create the seen store: %v", err)
	}
	if st, err := parseSeen(b); err != nil || len(st.IDs) != 0 {
		t.Fatalf("fresh seen store %q: %+v %v, want a valid empty store", b, st, err)
	}
	if m := mode(t, SeenPath(d)).Perm(); m != 0o600 {
		t.Fatalf("seen store mode %04o", m)
	}
	if err := Consume(d, "0123456789abcdef", tNow.Unix()+60, clockAt(tNow)); err != nil { // positive control
		t.Fatalf("first consume after enroll: %v", err)
	}
	before, _ := os.ReadFile(SeenPath(d))
	if err := Enroll(d, tUser, tEmail, tHash, tNow, true); err != nil {
		t.Fatal(err)
	}
	if after, _ := os.ReadFile(SeenPath(d)); !bytes.Equal(before, after) {
		t.Fatal("a replace rewrote the existing seen store")
	}
	if err := os.Remove(SeenPath(d)); err != nil {
		t.Fatal(err)
	}
	if err := Consume(d, "0123456789abcdee", tNow.Unix()+60, clockAt(tNow)); !errors.Is(err, ErrSeenCorrupt) {
		t.Fatalf("missing store after enrollment: %v, want ErrSeenCorrupt", err)
	}
	if _, err := os.Lstat(SeenPath(d)); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("a refused Consume re-created the missing store")
	}
	// a lone device.key (interrupted first enroll) also counts as enrolled
	d2 := enrolled(t)
	_ = os.Remove(AdminPath(d2))
	_ = os.Remove(SeenPath(d2))
	if err := Consume(d2, "0123456789abcdef", tNow.Unix()+60, clockAt(tNow)); !errors.Is(err, ErrSeenCorrupt) {
		t.Fatalf("lone key, missing store: %v, want ErrSeenCorrupt", err)
	}
	// a replace re-creates a missing store (the key rotates: every earlier
	// code dies with it)
	if err := Enroll(d, tUser, tEmail, tHash, tNow, true); err != nil {
		t.Fatal(err)
	}
	if err := Consume(d, "0123456789abcdee", tNow.Unix()+60, clockAt(tNow)); err != nil {
		t.Fatalf("after replace re-created the store: %v", err)
	}
	// positive control: a directory that was never enrolled still reads an
	// absent store as empty (Consume is reachable only behind the gate)
	if err := Consume(t.TempDir(), "0123456789abcdef", tNow.Unix()+60, clockAt(tNow)); err != nil {
		t.Fatalf("never-enrolled dir: %v", err)
	}
}
