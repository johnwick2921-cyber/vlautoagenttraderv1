package updaterbootstrap

// W-ONE-BUTTON M3 red-team fold (red-4 #6): `enroll --replace` over an
// existing enrollment is a different act from a first enroll — it removes
// the current update administrator and rotates the device key — so it has
// its own prompt, naming who is replaced and what dies, and its own typed
// confirmation. The first-enroll line ("ENROLL <email>") no longer replaces
// anyone.

import (
	"os"
	"strings"
	"testing"
	"time"

	"nofx/internal/updateauth"
	"nofx/store"
)

const bOther = "second@example.test"

func addUser(t *testing.T, inst, id, email string) {
	t.Helper()
	st, err := store.New(DBFileFor(inst))
	if err != nil {
		t.Fatal(err)
	}
	past := bNow.Add(-time.Hour)
	if err := st.User().Create(&store.User{ID: id, Email: email, PasswordHash: "$2a$10$not-a-real-hash-but-non-empty", CreatedAt: past, UpdatedAt: past}); err != nil {
		t.Fatal(err)
	}
	st.Plan().Close()
	_ = st.Close()
}

func TestEnrollReplaceNamesTheIncumbentAndRequiresADistinctConfirmation(t *testing.T) {
	attended(t)
	inst := install(t)
	if rc, _, errb := run(inst, enrollLine(bEmail), "enroll", bEmail); rc != 0 {
		t.Fatalf("first enroll rc=%d %s", rc, errb)
	}
	addUser(t, inst, "dddddddd-1111-2222-3333-444444444444", bOther)
	d := DataDirFor(inst)
	a0, k0 := fileSig(t, updateauth.AdminPath(d)), fileSig(t, updateauth.DeviceKeyPath(d))

	// the first-enroll confirmation does not replace anyone
	rc, _, errb := run(inst, enrollLine(bOther), "enroll", "--replace", bOther)
	if rc == 0 {
		t.Fatalf("`enroll --replace` accepted the first-enroll line %q and replaced %s with %s", strings.TrimSpace(enrollLine(bOther)), bEmail, bOther)
	}
	if fileSig(t, updateauth.AdminPath(d)) != a0 || fileSig(t, updateauth.DeviceKeyPath(d)) != k0 {
		t.Fatal("a refused replace changed the enrollment")
	}
	// the prompt names what is replaced and what dies
	for _, want := range []string{"REPLACE", bEmail, bOther, "rotates", "stops working"} {
		if !strings.Contains(errb, want) {
			t.Errorf("replace prompt does not say %q:\n%s", want, errb)
		}
	}
	want := "REPLACE " + bEmail + " WITH " + bOther
	if !strings.Contains(errb, "Type exactly:  "+want) {
		t.Fatalf("replace prompt does not ask for %q:\n%s", want, errb)
	}
	// near misses are refused
	for _, line := range []string{"REPLACE " + bOther + "\n", "REPLACE " + bOther + " WITH " + bEmail + "\n", "replace " + bEmail + " with " + bOther + "\n"} {
		if rc, _, _ := run(inst, line, "enroll", "--replace", bOther); rc == 0 {
			t.Errorf("replace accepted %q", strings.TrimSpace(line))
		}
	}
	// positive control: the exact replace line replaces and rotates the key
	if rc, out, errb := run(inst, want+"\n", "enroll", "--replace", bOther); rc != 0 || !strings.Contains(out, "enrolled:") {
		t.Fatalf("exact replace line: rc=%d out=%q err=%s", rc, out, errb)
	}
	if a, err := updateauth.LoadAdmin(d); err != nil || a.Email != bOther {
		t.Fatalf("after replace: admin = %+v %v", a, err)
	}
	if fileSig(t, updateauth.DeviceKeyPath(d)) == k0 {
		t.Fatal("replace did not rotate the key")
	}
}

// An enrollment whose admin.json cannot be read (a lone key after an
// interrupted first enroll) is replaced with an explicit UNREADABLE line;
// with nothing enrolled at all, --replace replaces nothing and the
// first-enroll line applies.
func TestEnrollReplaceOverAnUnreadableOrAbsentEnrollment(t *testing.T) {
	attended(t)
	inst := install(t)
	d := DataDirFor(inst)
	// nothing enrolled: --replace is a first enroll
	if rc, _, errb := run(inst, enrollLine(bEmail), "enroll", "--replace", bEmail); rc != 0 {
		t.Fatalf("--replace with nothing enrolled, ENROLL line: rc=%d %s", rc, errb)
	}
	// a lone key (admin.json gone)
	if err := os.Remove(updateauth.AdminPath(d)); err != nil {
		t.Fatal(err)
	}
	rc, _, errb := run(inst, enrollLine(bEmail), "enroll", "--replace", bEmail)
	if rc == 0 {
		t.Fatal("replace over an unreadable enrollment accepted the first-enroll line")
	}
	want := "REPLACE UNREADABLE WITH " + bEmail
	if !strings.Contains(errb, "Type exactly:  "+want) {
		t.Fatalf("prompt does not ask for %q:\n%s", want, errb)
	}
	if rc, _, errb := run(inst, want+"\n", "enroll", "--replace", bEmail); rc != 0 {
		t.Fatalf("exact UNREADABLE line: rc=%d %s", rc, errb)
	}
	if _, err := updateauth.LoadAdmin(d); err != nil {
		t.Fatalf("positive control: %v", err)
	}
}
