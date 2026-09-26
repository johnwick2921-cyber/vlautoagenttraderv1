package updaterwire

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"reflect"
	"sort"
	"strings"
	"testing"
)

// ── M3 worker channel codec — every forged frame is refused, and the same
// frame with the one defect removed decodes (positive control), so each
// refusal is caused by its defect and not by a broken harness. ─────────────

const (
	okRelease = "v1.4.2"
	okJob     = "job-0001abcd"
)

func mustDecode(t *testing.T, frame string) Request {
	t.Helper()
	req, err := DecodeRequest([]byte(frame))
	if err != nil {
		t.Fatalf("positive control %q must decode, got %v", frame, err)
	}
	return req
}

func mustReject(t *testing.T, frame string, want error) {
	t.Helper()
	_, err := DecodeRequest([]byte(frame))
	if err == nil {
		t.Fatalf("forged frame %q decoded — it must be rejected", frame)
	}
	if !errors.Is(err, want) {
		t.Fatalf("forged frame %q: got %v, want %v", frame, err, want)
	}
	if strings.Contains(err.Error(), frame) && len(frame) > 8 {
		t.Fatalf("error echoes the raw frame: %v", err)
	}
}

// L4 CHANGE OF AN EXISTING PIN (was TestWireVerbSetIsExactlyThree, M3). The
// M4 worker's attended `nofx-updater resume <job>` (3b-B dispatch §0/§3,
// brief C14) adds exactly ONE verb, so the count this pin asserts had to
// change; nothing else did. start_install / cancel are NOT added: install and
// cancel-before-boundary already are those verbs, and a second spelling of a
// verb is the parser differential this package refuses. The three M3 verbs
// keep their exact bytes (TestM3VerbFramesAreByteIdentical, green at the base
// before this change); every M3 forgery below is kept, and the resume
// forgeries join it.
func TestWireVerbSetIsExactlyFour(t *testing.T) {
	got := []string{}
	for _, v := range Verbs() {
		got = append(got, string(v))
		if !v.Known() {
			t.Fatalf("%q listed but not Known()", v)
		}
	}
	sort.Strings(got)
	want := []string{"cancel-before-boundary", "install", "resume", "status"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("verb set = %v, want exactly %v", got, want)
	}
	for _, v := range []Verb{"exec", "STATUS", "Status", "install ", "", "cancel_before_boundary", "shell",
		"Resume", "RESUME", "resume ", " resume", "resume_job", "start_install", "start-install", "cancel"} {
		if v.Known() {
			t.Fatalf("%q must not be a known verb", v)
		}
	}
}

// The resume payload is the job id and nothing else: no release, path, URL,
// MAC or requester (the worker re-proves everything from its own job file).
func TestResumePayloadCarriesOnlyTheJob(t *testing.T) {
	typ := reflect.TypeOf(ResumePayload{})
	var names []string
	for i := 0; i < typ.NumField(); i++ {
		names = append(names, typ.Field(i).Tag.Get("json"))
	}
	if !reflect.DeepEqual(names, []string{"job_id"}) {
		t.Fatalf("resume payload fields = %v, want exactly [job_id]", names)
	}
}

// PIN (M4 3b-B U2, brief C14): resume is decoded with the same strictness as
// the M3 verbs — exact bytes for the verb, exact keys for its payload, one
// verb per frame. Every forgery has the positive control: the same frame with
// the one defect removed decodes to NewResume(job).
func TestDecodeRejectsResumeForgeries(t *testing.T) {
	control := `{"v":1,"verb":"resume","payload":{"job_id":"job-0001abcd"}}`
	if got := mustDecode(t, control); !reflect.DeepEqual(got, NewResume(okJob)) {
		t.Fatalf("positive control decoded to %+v, want %+v", got, NewResume(okJob))
	}
	// the verb: exact bytes only — re-cased, padded, snake/kebab variants, and
	// the start_install / cancel duplicates C14 did not add
	for _, verb := range []string{"Resume", "RESUME", "resume ", " resume", "resume_job", "resume-job",
		"start_install", "start-install", "cancel", `resume\u0000`} {
		t.Run("verb "+verb, func(t *testing.T) {
			mustReject(t, strings.Replace(control, `"resume"`, `"`+verb+`"`, 1), ErrUnknownVerb)
		})
	}
	mustReject(t, `{"v":1,"verb":["resume"],"payload":{"job_id":"job-0001abcd"}}`, ErrUnknownVerb)
	// one verb per frame
	mustReject(t, `{"v":1,"verb":"resume","verb":"status","payload":{"job_id":"job-0001abcd"}}`, ErrDuplicateField)
	mustReject(t, control+control, ErrTrailingData)
	// the payload: job_id and nothing else, exact case, once
	for name, frame := range map[string]string{
		"release beside job": `{"v":1,"verb":"resume","payload":{"job_id":"job-0001abcd","release_id":"v1.4.2"}}`,
		"hmac":               `{"v":1,"verb":"resume","payload":{"job_id":"job-0001abcd","hmac":"00"}}`,
		"path":               `{"v":1,"verb":"resume","payload":{"job_id":"job-0001abcd","path":"/tmp/x"}}`,
		"requested_by":       `{"v":1,"verb":"resume","payload":{"job_id":"job-0001abcd","requested_by":"1"}}`,
		"re-cased key":       `{"v":1,"verb":"resume","payload":{"Job_ID":"job-0001abcd"}}`,
		"top-level extra":    `{"v":1,"verb":"resume","payload":{"job_id":"job-0001abcd"},"force":true}`,
	} {
		t.Run(name, func(t *testing.T) { mustReject(t, frame, ErrUnknownField) })
	}
	mustReject(t, `{"v":1,"verb":"resume","payload":{"job_id":"job-0001abcd","job_id":"job-0002abcd"}}`, ErrDuplicateField)
	for name, frame := range map[string]string{
		"no job":         `{"v":1,"verb":"resume","payload":{}}`,
		"empty job":      `{"v":1,"verb":"resume","payload":{"job_id":""}}`,
		"upper job":      `{"v":1,"verb":"resume","payload":{"job_id":"JOB-0001ABCD"}}`,
		"path job":       `{"v":1,"verb":"resume","payload":{"job_id":"../job-0001abcd"}}`,
		"number job":     `{"v":1,"verb":"resume","payload":{"job_id":12345678}}`,
		"null payload":   `{"v":1,"verb":"resume","payload":null}`,
		"string payload": `{"v":1,"verb":"resume","payload":"job-0001abcd"}`,
	} {
		t.Run(name, func(t *testing.T) { mustReject(t, frame, ErrBadPayload) })
	}
	mustReject(t, `{"v":2,"verb":"resume","payload":{"job_id":"job-0001abcd"}}`, ErrBadVersion)
}

// A resume request that is not exactly {verb resume, one valid ResumePayload}
// never reaches the wire, and a resume payload cannot ride beside another
// verb's.
func TestEncodeRefusesAnInvalidResume(t *testing.T) {
	bad := map[string]Request{
		"no payload":             {Verb: VerbResume},
		"cancel payload":         {Verb: VerbResume, Cancel: &CancelPayload{JobID: okJob}},
		"resume beside status":   {Verb: VerbResume, Resume: &ResumePayload{JobID: okJob}, Status: &StatusPayload{}},
		"status smuggles resume": {Verb: VerbStatus, Status: &StatusPayload{}, Resume: &ResumePayload{JobID: okJob}},
		"cancel verb, resume":    {Verb: VerbCancelBeforeBoundary, Resume: &ResumePayload{JobID: okJob}},
		"empty job":              NewResume(""),
		"upper job":              NewResume("JOB-0001ABCD"),
		"path job":               NewResume("../job-0001abcd"),
	}
	for name, r := range bad {
		if _, err := EncodeRequest(r); err == nil {
			t.Fatalf("%s: EncodeRequest(%+v) must refuse", name, r)
		}
	}
	if _, err := EncodeRequest(NewResume(okJob)); err != nil {
		t.Fatalf("positive control: %v", err)
	}
}

func TestInstallPayloadCarriesOnlyReleaseAndJob(t *testing.T) {
	typ := reflect.TypeOf(InstallPayload{})
	var names []string
	for i := 0; i < typ.NumField(); i++ {
		names = append(names, typ.Field(i).Tag.Get("json"))
	}
	if !reflect.DeepEqual(names, []string{"release_id", "job_id"}) {
		t.Fatalf("install payload fields = %v, want exactly [release_id job_id] (no URL, path, command, MAC)", names)
	}
}

func TestCodecRoundTripsEveryVerb(t *testing.T) {
	for _, req := range []Request{
		NewStatus(""),
		NewStatus(okJob),
		NewInstall(okRelease, okJob),
		NewCancelBeforeBoundary(okJob),
		NewResume(okJob),
	} {
		frame, err := EncodeRequest(req)
		if err != nil {
			t.Fatalf("encode %+v: %v", req, err)
		}
		if !bytes.HasSuffix(frame, []byte("\n")) || bytes.Count(frame, []byte("\n")) != 1 {
			t.Fatalf("frame must be exactly one newline-terminated line: %q", frame)
		}
		got, err := ReadFrame(bufio.NewReader(bytes.NewReader(frame)))
		if err != nil {
			t.Fatalf("ReadFrame: %v", err)
		}
		back, err := DecodeRequest(got)
		if err != nil {
			t.Fatalf("decode %q: %v", got, err)
		}
		if !reflect.DeepEqual(back, req) {
			t.Fatalf("round trip %+v → %+v", req, back)
		}
	}
	// the exact install frame on the wire
	frame, _ := EncodeRequest(NewInstall(okRelease, okJob))
	want := `{"v":1,"verb":"install","payload":{"release_id":"v1.4.2","job_id":"job-0001abcd"}}` + "\n"
	if string(frame) != want {
		t.Fatalf("install frame = %q, want %q", frame, want)
	}
	// the exact resume frame on the wire (M4 3b-B U2)
	frame, _ = EncodeRequest(NewResume(okJob))
	want = `{"v":1,"verb":"resume","payload":{"job_id":"job-0001abcd"}}` + "\n"
	if string(frame) != want {
		t.Fatalf("resume frame = %q, want %q", frame, want)
	}
}

func TestDecodeRejectsUnknownVerb(t *testing.T) {
	control := `{"v":1,"verb":"status","payload":{}}`
	mustDecode(t, control)
	for _, verb := range []string{"exec", "STATUS", "Status", "install ", " status", "", "cancel_before_boundary", "rm -rf /"} {
		frame := strings.Replace(control, `"status"`, `"`+verb+`"`, 1)
		mustReject(t, frame, ErrUnknownVerb)
	}
	// a verb that is not a JSON string at all
	mustReject(t, `{"v":1,"verb":1,"payload":{}}`, ErrUnknownVerb)
	mustReject(t, `{"v":1,"verb":null,"payload":{}}`, ErrUnknownVerb)
}

func TestDecodeRejectsExtraFields(t *testing.T) {
	control := `{"v":1,"verb":"install","payload":{"release_id":"v1.4.2","job_id":"job-0001abcd"}}`
	mustDecode(t, control)
	cases := map[string]string{
		"top-level extra":     `{"v":1,"verb":"install","payload":{"release_id":"v1.4.2","job_id":"job-0001abcd"},"cmd":"sh"}`,
		"payload url":         `{"v":1,"verb":"install","payload":{"release_id":"v1.4.2","job_id":"job-0001abcd","url":"https://x/y"}}`,
		"payload path":        `{"v":1,"verb":"install","payload":{"release_id":"v1.4.2","job_id":"job-0001abcd","path":"/tmp/x"}}`,
		"payload hmac":        `{"v":1,"verb":"install","payload":{"release_id":"v1.4.2","job_id":"job-0001abcd","hmac":"00"}}`,
		"payload expires_at":  `{"v":1,"verb":"install","payload":{"release_id":"v1.4.2","job_id":"job-0001abcd","expires_at":1}}`,
		"re-cased top key":    `{"v":1,"Verb":"install","payload":{"release_id":"v1.4.2","job_id":"job-0001abcd"}}`,
		"re-cased payload":    `{"v":1,"verb":"install","payload":{"Release_ID":"v1.4.2","job_id":"job-0001abcd"}}`,
		"status carries rel":  `{"v":1,"verb":"status","payload":{"release_id":"v1.4.2"}}`,
		"cancel carries rel":  `{"v":1,"verb":"cancel-before-boundary","payload":{"job_id":"job-0001abcd","release_id":"v1.4.2"}}`,
		"unicode lookalike":   `{"v":1,"verb":"install","payload":{"release_ıd":"v1.4.2","job_id":"job-0001abcd"}}`,
		"escaped key variant": `{"v":1,"verb":"install","payload":{"release_\u0049D":"v1.4.2","job_id":"job-0001abcd"}}`,
	}
	for name, frame := range cases {
		t.Run(name, func(t *testing.T) { mustReject(t, frame, ErrUnknownField) })
	}
	// duplicate keys: the last-wins parser differential is closed
	mustReject(t, `{"v":1,"verb":"status","verb":"install","payload":{}}`, ErrDuplicateField)
	mustReject(t, `{"v":1,"verb":"install","payload":{"release_id":"v1.4.2","job_id":"job-0001abcd","job_id":"job-0002abcd"}}`, ErrDuplicateField)
}

func TestDecodeRejectsMalformed(t *testing.T) {
	mustDecode(t, `{"v":1,"verb":"status","payload":{}}`)
	cases := map[string]struct {
		frame string
		want  error
	}{
		"not json":           {`status please`, ErrMalformed},
		"array":              {`[{"v":1,"verb":"status","payload":{}}]`, ErrMalformed},
		"empty":              {``, ErrMalformed},
		"two objects":        {`{"v":1,"verb":"status","payload":{}}{"v":1,"verb":"status","payload":{}}`, ErrTrailingData},
		"trailing garbage":   {`{"v":1,"verb":"status","payload":{}} rm`, ErrTrailingData},
		"missing payload":    {`{"v":1,"verb":"status"}`, ErrMalformed},
		"missing v":          {`{"verb":"status","payload":{}}`, ErrMalformed},
		"null payload":       {`{"v":1,"verb":"status","payload":null}`, ErrBadPayload},
		"string payload":     {`{"v":1,"verb":"status","payload":"{}"}`, ErrBadPayload},
		"v2":                 {`{"v":2,"verb":"status","payload":{}}`, ErrBadVersion},
		"v string":           {`{"v":"1","verb":"status","payload":{}}`, ErrBadVersion},
		"v float":            {`{"v":1.0,"verb":"status","payload":{}}`, ErrBadVersion},
		"job_id number":      {`{"v":1,"verb":"cancel-before-boundary","payload":{"job_id":12345678}}`, ErrBadPayload},
		"install no job":     {`{"v":1,"verb":"install","payload":{"release_id":"v1.4.2"}}`, ErrBadPayload},
		"install empty rel":  {`{"v":1,"verb":"install","payload":{"release_id":"","job_id":"job-0001abcd"}}`, ErrBadPayload},
		"cancel no job":      {`{"v":1,"verb":"cancel-before-boundary","payload":{}}`, ErrBadPayload},
		"invalid utf8":       {"{\"v\":1,\"verb\":\"status\",\"payload\":{\"job_id\":\"job-\xff\xfe0001\"}}", ErrMalformed},
		"unterminated":       {`{"v":1,"verb":"status","payload":{}`, ErrMalformed},
		"path release":       {`{"v":1,"verb":"install","payload":{"release_id":"../../etc/passwd","job_id":"job-0001abcd"}}`, ErrBadPayload},
		"pipe release":       {`{"v":1,"verb":"install","payload":{"release_id":"a|b","job_id":"job-0001abcd"}}`, ErrBadPayload},
		"upper job":          {`{"v":1,"verb":"cancel-before-boundary","payload":{"job_id":"JOB-0001ABCD"}}`, ErrBadPayload},
		"status bad job":     {`{"v":1,"verb":"status","payload":{"job_id":"x"}}`, ErrBadPayload},
		"status empty jobid": {`{"v":1,"verb":"status","payload":{"job_id":""}}`, ErrBadPayload},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) { mustReject(t, c.frame, c.want) })
	}
}

func TestReadFrameCapsAt16KiB(t *testing.T) {
	// positive control: a frame of exactly MaxFrameBytes is read whole
	exact := bytes.Repeat([]byte("a"), MaxFrameBytes)
	got, err := ReadFrame(bufio.NewReader(bytes.NewReader(append(append([]byte{}, exact...), '\n'))))
	if err != nil || len(got) != MaxFrameBytes {
		t.Fatalf("a %d-byte frame must be read whole, got len=%d err=%v", MaxFrameBytes, len(got), err)
	}
	// one byte more ⇒ oversize, and the reader stops without draining the stream
	over := bytes.Repeat([]byte("a"), MaxFrameBytes+1)
	cr := &countingReader{r: io.MultiReader(bytes.NewReader(over), bytes.NewReader([]byte("\n")), endlessA())}
	_, err = ReadFrame(bufio.NewReaderSize(cr, 4096))
	if !errors.Is(err, ErrOversize) {
		t.Fatalf("a %d-byte frame must be ErrOversize, got %v", MaxFrameBytes+1, err)
	}
	if cr.n > MaxFrameBytes+2*4096 {
		t.Fatalf("ReadFrame read %d bytes of an endless stream — the cap is not bounding memory", cr.n)
	}
	// an endless line with no newline ⇒ oversize, bounded (the stand-in is
	// 8 MiB, not truly endless, so a regressed cap fails as ErrTruncated
	// instead of eating the box's memory)
	cr2 := &countingReader{r: endlessA()}
	if _, err := ReadFrame(bufio.NewReaderSize(cr2, 4096)); !errors.Is(err, ErrOversize) {
		t.Fatalf("endless line: got %v", err)
	}
	if cr2.n > MaxFrameBytes+2*4096 {
		t.Fatalf("endless line read %d bytes", cr2.n)
	}
	// EOF mid-frame ⇒ truncated; clean EOF ⇒ io.EOF
	if _, err := ReadFrame(bufio.NewReader(strings.NewReader(`{"v":1`))); !errors.Is(err, ErrTruncated) {
		t.Fatalf("EOF mid-frame: got %v", err)
	}
	if _, err := ReadFrame(bufio.NewReader(strings.NewReader(``))); err != io.EOF {
		t.Fatalf("clean EOF: got %v", err)
	}
	// DecodeRequest applies the same cap to a frame handed to it directly
	big := `{"v":1,"verb":"status","payload":{}}` + strings.Repeat(" ", MaxFrameBytes)
	mustReject(t, big, ErrOversize)
}

func TestEncodeRefusesAnInvalidRequest(t *testing.T) {
	bad := []Request{
		{Verb: "exec"},
		{Verb: VerbInstall}, // no payload
		{Verb: VerbInstall, Status: &StatusPayload{}},                                      // wrong payload
		{Verb: VerbStatus, Status: &StatusPayload{}, Cancel: &CancelPayload{JobID: okJob}}, // two payloads
		NewInstall("../x", okJob),
		NewInstall(okRelease, "short"),
		NewCancelBeforeBoundary(""),
	}
	for _, r := range bad {
		if _, err := EncodeRequest(r); err == nil {
			t.Fatalf("EncodeRequest(%+v) must refuse — the app never sends an invalid frame", r)
		}
	}
	if _, err := EncodeRequest(NewInstall(okRelease, okJob)); err != nil {
		t.Fatalf("positive control: %v", err)
	}
}

func TestResponseCodecIsStrict(t *testing.T) {
	for _, r := range []Response{{OK: true, State: "idle"}, RejectedResponse, {OK: false, Error: "not cancellable past boundary"}} {
		b, err := EncodeResponse(r)
		if err != nil {
			t.Fatalf("encode %+v: %v", r, err)
		}
		got, err := ReadFrame(bufio.NewReader(bytes.NewReader(b)))
		if err != nil {
			t.Fatal(err)
		}
		back, err := DecodeResponse(got)
		if err != nil || back != r {
			t.Fatalf("round trip %+v → %+v, %v", r, back, err)
		}
	}
	for _, b := range []string{
		`{"v":1,"ok":true,"state":"idle","extra":1}`,
		`{"v":1,"ok":true}`,
		`{"v":1,"ok":true,"state":"idle\nFAKE LOG"}`,
		`{"v":1,"ok":false,"error":"rejected","state":"idle"}`,
		`{"v":2,"ok":true,"state":"idle"}`,
		`{"v":1,"OK":true,"state":"idle"}`,
	} {
		if _, err := DecodeResponse([]byte(b)); err == nil {
			t.Fatalf("response %q must be refused", b)
		}
	}
	if _, err := EncodeResponse(Response{OK: true, State: "idle\n"}); err == nil {
		t.Fatal("a handler response with a newline in it must not reach the wire")
	}
}

func TestRejectReasonsAreFixedStrings(t *testing.T) {
	for err, want := range map[error]string{
		ErrOversize: "oversize", ErrMalformed: "malformed", ErrTrailingData: "trailing_data",
		ErrUnknownField: "unknown_field", ErrDuplicateField: "duplicate_field", ErrBadVersion: "bad_version",
		ErrUnknownVerb: "unknown_verb", ErrBadPayload: "bad_payload", ErrTruncated: "truncated",
	} {
		if got := RejectReason(err); got != want {
			t.Fatalf("RejectReason(%v) = %q, want %q", err, got, want)
		}
	}
	if got := RejectReason(errors.New("attacker text")); got != "error" {
		t.Fatalf("an unknown error must map to the fixed reason \"error\", got %q", got)
	}
}

func TestFrameDigestNeverEchoesThePayload(t *testing.T) {
	payload := []byte(`{"v":1,"verb":"exec","payload":{"cmd":"SECRET-MARKER"}}`)
	d := FrameDigest(payload)
	if strings.Contains(d, "SECRET") || strings.Contains(d, "exec") {
		t.Fatalf("digest echoes the payload: %q", d)
	}
	prefix := fmt.Sprintf("len=%d sha256=", len(payload))
	if !strings.HasPrefix(d, prefix) || len(d) != len(prefix)+16 {
		t.Fatalf("digest shape = %q, want len=<n> sha256=<16 hex>", d)
	}
}

type countingReader struct {
	r io.Reader
	n int
}

func (c *countingReader) Read(p []byte) (int, error) {
	n, err := c.r.Read(p)
	c.n += n
	return n, err
}

type infiniteA struct{}

func (infiniteA) Read(p []byte) (int, error) {
	for i := range p {
		p[i] = 'a'
	}
	return len(p), nil
}

// endlessA is a newline-free stream far past the cap but FINITE (8 MiB): a
// ReadFrame whose cap regressed returns ErrTruncated after reading it all,
// so the test goes red instead of growing without bound.
func endlessA() io.Reader { return io.LimitReader(infiniteA{}, 8<<20) }
