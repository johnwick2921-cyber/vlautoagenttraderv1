package updateauth

// W-ONE-BUTTON M3 red-team fold (red-3 #7): the MAC message carries a purpose
// and version tag, so a device-key MAC minted for any other action (a future
// rollback authorization, a receipt) or in any other message layout can never
// be replayed as an install authorization. Safe to introduce now: M3 never
// shipped, so no untagged code was ever issued.

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"testing"
)

func TestMACMessageCarriesThePurposeAndVersionTag(t *testing.T) {
	const rel, job, exp = "v1.2.3", "0123456789abcdef", int64(1800000300)
	msg, err := Message(tUser, rel, job, exp)
	if err != nil {
		t.Fatal(err)
	}
	if want := "nofx-update-install/v1|" + tUser + "|v1.2.3|0123456789abcdef|1800000300"; string(msg) != want {
		t.Fatalf("Message = %q, want %q", msg, want)
	}
	key := seqKey(3)
	mac := func(m string) string {
		h := hmac.New(sha256.New, key)
		h.Write([]byte(m))
		return hex.EncodeToString(h.Sum(nil))
	}
	for name, m := range map[string]string{
		"untagged (pre-tag layout)": "v1.2.3|0123456789abcdef|1800000300",
		"another purpose":           "nofx-update-rollback/v1|" + tUser + "|v1.2.3|0123456789abcdef|1800000300",
		"another version":           "nofx-update-install/v2|" + tUser + "|v1.2.3|0123456789abcdef|1800000300",
	} {
		if VerifyMAC(key, tUser, rel, job, exp, mac(m)) {
			t.Errorf("a MAC over the %s message %q verified as an install", name, m)
		}
	}
	// positive control: the MAC over the tagged message verifies, and
	// ComputeMAC produces exactly it
	got, err := ComputeMAC(key, tUser, rel, job, exp)
	if err != nil || got != mac(string(msg)) || !VerifyMAC(key, tUser, rel, job, exp, got) {
		t.Fatalf("positive control: ComputeMAC = %q (%v), want the MAC over %q", got, err, msg)
	}
}
