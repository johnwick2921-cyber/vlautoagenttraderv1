package deploy

import (
	"os"
	"strings"
	"testing"
)

// vlReleaseKey is the OWNER repo's (johnwick2921-cyber/vl) release public key.
// A partner box must never trust it: the partner repo publishes releases signed
// with its OWN key (the `release` Environment secret RELEASE_SIGNING_KEY), and
// a partner sync that copies the owner's deploy/release_allowed_signers over
// this file would make every partner release fail verification on a partner
// box. That is exactly what happened before 2026-10-16: the sync carried the
// owner's key into this repo, so the partner repo could not publish at all.
const vlReleaseKey = "AAAAC3NzaC1lZDI1NTE5AAAAIFAnymAm3evz3IP3DQPjgaBIGzMpobd55aLLhWhtwVqF"

// TestPartnerAllowedSignersIsThePartnerKey pins the partner trust anchor: one
// `release` principal, an ed25519 key, never the owner repo's key. A sync PR
// must carve this file out (keep the partner copy).
func TestPartnerAllowedSignersIsThePartnerKey(t *testing.T) {
	b, err := os.ReadFile("release_allowed_signers")
	if err != nil {
		t.Fatalf("read deploy/release_allowed_signers: %v", err)
	}
	var lines []string
	for _, l := range strings.Split(strings.TrimSpace(string(b)), "\n") {
		if l = strings.TrimSpace(l); l != "" && !strings.HasPrefix(l, "#") {
			lines = append(lines, l)
		}
	}
	if len(lines) != 1 {
		t.Fatalf("want exactly one allowed-signers line, got %d: %q", len(lines), lines)
	}
	f := strings.Fields(lines[0])
	if len(f) < 3 || f[0] != "release" || f[1] != "ssh-ed25519" {
		t.Fatalf("want `release ssh-ed25519 <key> …` (principal and namespace `release`, as release.yml signs and verifies), got %q", lines[0])
	}
	if f[2] == vlReleaseKey {
		t.Fatalf("deploy/release_allowed_signers carries the OWNER repo's key — a partner sync overwrote the partner trust anchor; restore the partner key (carve-out)")
	}
}
