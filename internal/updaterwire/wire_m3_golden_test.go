package updaterwire

import "testing"

// L4 PIN (M4 3b-B U2): adding the resume verb is additive — every frame the
// M3 app already sends is byte-identical. These goldens were run GREEN at the
// base (66dde017, dev 47726d84) BEFORE the verb landed, so they are M3's
// bytes, not the new code's own output restated.
func TestM3VerbFramesAreByteIdentical(t *testing.T) {
	for _, c := range []struct {
		req  Request
		want string
	}{
		{NewStatus(""), `{"v":1,"verb":"status","payload":{}}`},
		{NewStatus("job-0001abcd"), `{"v":1,"verb":"status","payload":{"job_id":"job-0001abcd"}}`},
		{NewInstall("v1.4.2", "job-0001abcd"), `{"v":1,"verb":"install","payload":{"release_id":"v1.4.2","job_id":"job-0001abcd"}}`},
		{NewCancelBeforeBoundary("job-0001abcd"), `{"v":1,"verb":"cancel-before-boundary","payload":{"job_id":"job-0001abcd"}}`},
	} {
		got, err := EncodeRequest(c.req)
		if err != nil {
			t.Fatalf("encode %+v: %v", c.req, err)
		}
		if string(got) != c.want+"\n" {
			t.Fatalf("%s frame = %q, want M3's %q", c.req.Verb, got, c.want+"\n")
		}
		back, err := DecodeRequest([]byte(c.want))
		if err != nil {
			t.Fatalf("M3's %s frame no longer decodes: %v", c.req.Verb, err)
		}
		if back.Verb != c.req.Verb {
			t.Fatalf("M3's %s frame decoded as %q", c.req.Verb, back.Verb)
		}
	}
}
