package wireserver

import (
	"bufio"
	"bytes"
	"net"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"nofx/internal/updaterwire"
)

// PIN (M4 3b-B U2): the attended resume verb crosses the PRODUCTION path end
// to end — the CLI's updaterwire.Dial + Client.Do (EncodeRequest) → the
// worker's Listen/Serve (peer uid, ReadFrame, DecodeRequest) → the Handler —
// and the worker's answer comes back through DecodeResponse. Its forgeries
// are refused on the same path, logged as reason + digest, and never reach
// the handler; the valid exchange logs nothing.
func TestServeResumeRoundTripViaDial(t *testing.T) {
	const job = "job-0001abcd"
	const marker = "RESUME-FORGED-MARKER-4c1e"
	_, sock := dataDir(t)
	sink := &logSink{}
	l, err := Listen(sock, sink.logf)
	if err != nil {
		t.Fatalf("Listen: %v", err)
	}
	var calls atomic.Int32
	var last atomic.Value // updaterwire.Request
	done := make(chan error, 1)
	go func() {
		done <- l.Serve(func(r updaterwire.Request) updaterwire.Response {
			calls.Add(1)
			last.Store(r)
			if r.Resume == nil {
				return updaterwire.Response{OK: true, State: "idle"}
			}
			return updaterwire.Response{OK: true, State: "resuming"}
		})
	}()
	t.Cleanup(func() { l.Close(); <-done })

	c, err := updaterwire.Dial(sock)
	if err != nil {
		t.Fatalf("the CLI's Dial must reach the worker's Listen: %v", err)
	}
	defer c.Close()
	resp, err := c.Do(updaterwire.NewResume(job))
	if err != nil || resp != (updaterwire.Response{OK: true, State: "resuming"}) {
		t.Fatalf("resume over the socket: resp=%+v err=%v, want {OK:true State:resuming}", resp, err)
	}
	got, _ := last.Load().(updaterwire.Request)
	if !reflect.DeepEqual(got, updaterwire.NewResume(job)) {
		t.Fatalf("the handler saw %+v, want exactly %+v (verb resume, the job id, no other payload)", got, updaterwire.NewResume(job))
	}
	// the M3 verbs still answer on the same connection after a resume
	if resp, err := c.Do(updaterwire.NewStatus(job)); err != nil || resp.State != "idle" {
		t.Fatalf("status after resume: %+v %v", resp, err)
	}
	if n := calls.Load(); n != 2 {
		t.Fatalf("handler calls = %d, want 2", n)
	}
	if l := sink.all(); l != "" {
		t.Fatalf("a valid resume exchange must log nothing, got %q", l)
	}

	rejected, _ := updaterwire.EncodeResponse(updaterwire.RejectedResponse)
	for _, f := range []struct{ name, reason, raw string }{
		{"re-cased verb", "unknown_verb", `{"v":1,"verb":"RESUME","payload":{"job_id":"job-0001abcd","x":"` + marker + `"}}`},
		{"snake verb", "unknown_verb", `{"v":1,"verb":"resume_job","payload":{"job_id":"job-0001abcd","x":"` + marker + `"}}`},
		{"start_install", "unknown_verb", `{"v":1,"verb":"start_install","payload":{"job_id":"job-0001abcd","x":"` + marker + `"}}`},
		{"release beside job", "unknown_field", `{"v":1,"verb":"resume","payload":{"job_id":"job-0001abcd","release_id":"` + marker + `"}}`},
		{"no job", "bad_payload", `{"v":1,"verb":"resume","payload":{}}`},
		{"forged job", "bad_payload", `{"v":1,"verb":"resume","payload":{"job_id":"../` + marker + `"}}`},
	} {
		t.Run(f.name, func(t *testing.T) {
			before, beforeLog := calls.Load(), len(sink.all())
			out := rawExchange(t, sock, []byte(f.raw+"\n"))
			if !bytes.Equal(out, rejected) {
				t.Fatalf("server answered %q, want exactly the rejected frame %q then close", out, rejected)
			}
			if calls.Load() != before {
				t.Fatal("the handler was called for a forged resume frame")
			}
			logged := sink.all()[beforeLog:]
			if !strings.Contains(logged, "updater-wire: rejected reason="+f.reason+" len=") {
				t.Fatalf("log = %q, want a rejected reason=%s line with a length+hash", logged, f.reason)
			}
			if strings.Contains(logged, marker) {
				t.Fatalf("the log echoes the forged frame: %q", logged)
			}
		})
	}
	// positive control on a fresh connection after the refusals: the valid
	// resume frame, spelled by hand, reaches the handler
	before := calls.Load()
	ok, _ := updaterwire.EncodeResponse(updaterwire.Response{OK: true, State: "resuming"})
	pc, err := net.DialUnix("unix", nil, &net.UnixAddr{Name: sock, Net: "unix"})
	if err != nil {
		t.Fatal(err)
	}
	defer pc.Close()
	pc.SetDeadline(time.Now().Add(5 * time.Second))
	pc.Write([]byte(`{"v":1,"verb":"resume","payload":{"job_id":"job-0001abcd"}}` + "\n"))
	if line, err := bufio.NewReader(pc).ReadBytes('\n'); err != nil || !bytes.Equal(line, ok) {
		t.Fatalf("positive control: got %q %v, want %q", line, err, ok)
	}
	if calls.Load() != before+1 {
		t.Fatal("positive control: the valid resume frame did not reach the handler")
	}
}
