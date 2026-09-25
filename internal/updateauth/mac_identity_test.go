package updateauth

// W-ONE-BUTTON M3 red-team fold (red-4 #4, second half): the MAC is bound to
// the enrolled administrator's user_id. A MAC for one identity never
// verifies for another under the same key, and Authorize mints for the
// identity admin.json names.

import "testing"

func TestMACIsBoundToTheAdministratorIdentity(t *testing.T) {
	const rel, job, exp = "v1.2.3", "0123456789abcdef", int64(1800000300)
	key := seqKey(11)
	mac, err := ComputeMAC(key, tUser, rel, job, exp)
	if err != nil || !VerifyMAC(key, tUser, rel, job, exp, mac) {
		t.Fatalf("positive control: %v", err)
	}
	for _, other := range []string{"22222222-2222-3333-4444-555555555555", tUser + "x", "admin"} {
		if VerifyMAC(key, other, rel, job, exp, mac) {
			t.Errorf("a MAC minted for %s verified for %s", tUser, other)
		}
	}
	for _, bad := range []string{"", "a|b", "has space"} {
		if VerifyMAC(key, bad, rel, job, exp, mac) {
			t.Errorf("malformed user id %q verified", bad)
		}
		if _, err := ComputeMAC(key, bad, rel, job, exp); err == nil {
			t.Errorf("ComputeMAC accepted malformed user id %q", bad)
		}
	}
	// Authorize mints for the enrolled identity
	d := enrolled(t)
	g, err := Authorize(d, rel, tNow)
	if err != nil {
		t.Fatal(err)
	}
	k, _ := LoadDeviceKey(d)
	if !VerifyMAC(k, tUser, g.ReleaseID, g.JobID, g.ExpiresAt, g.HMAC) {
		t.Fatal("Authorize's grant does not verify for the enrolled administrator")
	}
	if VerifyMAC(k, "22222222-2222-3333-4444-555555555555", g.ReleaseID, g.JobID, g.ExpiresAt, g.HMAC) {
		t.Fatal("Authorize's grant verifies for an identity that is not enrolled")
	}
}
