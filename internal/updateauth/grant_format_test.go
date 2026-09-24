package updateauth

// W-ONE-BUTTON M3 red-team fold (red-3 #6): a Grant never prints its MAC
// through fmt — %v, %+v, %#v, %s, %q, %x, %X, %d, as a value, as a pointer,
// and nested in a slice, a map or a struct's exported field. A logged grant
// is otherwise a usable code (single-use, but live for up to 5 minutes).
// json.Marshal still carries it: the attended CLI's output and the install
// body ARE the grant (positive control).

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
)

func TestGrantNeverFormatsItsMAC(t *testing.T) {
	g := Grant{ReleaseID: "v1.2.3", JobID: "fmt-job-000001", ExpiresAt: 1800000300, HMAC: strings.Repeat("c", 64)}
	type holder struct {
		G Grant
		P *Grant
	}
	operands := map[string]any{
		"value":   g,
		"pointer": &g,
		"slice":   []Grant{g},
		"map":     map[string]Grant{"k": g},
		"struct":  holder{G: g, P: &g},
	}
	for name, op := range operands {
		for _, verb := range []string{"%v", "%+v", "%#v", "%s", "%q", "%x", "%X", "%d", "%10.3v"} {
			s := fmt.Sprintf(verb, op)
			if strings.Contains(s, g.HMAC) || strings.Contains(s, strings.ToUpper(g.HMAC)) || strings.Contains(s, fmt.Sprintf("%x", g.HMAC)) {
				t.Errorf("fmt %s of a Grant %s prints the MAC: %s", verb, name, s)
			}
		}
	}
	if s := fmt.Errorf("hand-off refused for %v: %w", g, errors.New("boom")).Error(); strings.Contains(s, g.HMAC) {
		t.Errorf("an error wrapping a Grant prints the MAC: %s", s)
	}
	// the redacted form still identifies the grant
	if s := fmt.Sprint(g); !strings.Contains(s, g.JobID) || !strings.Contains(s, g.ReleaseID) || !strings.Contains(s, "redacted") {
		t.Errorf("redacted form lost the grant's identity: %s", s)
	}
	// positive control: JSON is the grant's wire form and carries the MAC
	b, err := json.Marshal(g)
	if err != nil || !strings.Contains(string(b), `"hmac":"`+g.HMAC+`"`) {
		t.Fatalf("json.Marshal(g) = %s (%v), want the hmac field", b, err)
	}
}
