package ninjatrader

import (
	"net"
	"strings"
	"testing"
	"time"
)

func TestSendSignalRefusesPayloadAgeRegardlessOfQueueAge(t *testing.T) {
	for _, stamp := range []string{time.Now().Add(-30 * time.Minute).UTC().Format(time.RFC3339Nano), "", "not-a-clock", time.Now().Add(time.Hour).UTC().Format(time.RFC3339Nano)} {
		s := NewTCPServer(nil)
		err := s.SendSignal(SignalPayload{SignalID: "old", Timestamp: stamp})
		if err == nil || !strings.Contains(err.Error(), "payload age") {
			t.Fatalf("stamp=%q: %v", stamp, err)
		}
		if len(s.pending) != 0 {
			t.Fatal("invalid payload queued")
		}
	}
}

func TestFlushRefusesExpiredPayloadAfterWriterWait(t *testing.T) {
	s := NewTCPServer(nil)
	srv, cli := net.Pipe()
	defer srv.Close()
	defer cli.Close()
	s.conn = srv
	// Fresh queue metadata must not launder an old payload.
	s.pending = []timedSignal{{payload: SignalPayload{SignalID: "old", Timestamp: time.Now().Add(-61 * time.Second).UTC().Format(time.RFC3339Nano)}, timestamp: time.Now()}}
	if err := s.flushPending(); err != nil {
		t.Fatal(err)
	}
	cli.SetReadDeadline(time.Now().Add(20 * time.Millisecond))
	if _, err := ReadFrame(cli); err == nil {
		t.Fatal("expired payload sent")
	}
	if len(s.pending) != 0 {
		t.Fatal("expired payload retained for retry")
	}
}

func TestRetryDoesNotRenewPayloadClock(t *testing.T) {
	s := NewTCPServer(nil)
	s.SetStaleSignalAgeForTest(50 * time.Millisecond)
	stamp := time.Now().UTC().Format(time.RFC3339Nano)
	srv, cli := net.Pipe()
	cli.Close()
	s.conn = srv
	if err := s.SendSignal(SignalPayload{SignalID: "retry", Timestamp: stamp}); err == nil {
		t.Fatal("dead socket succeeded")
	}
	if len(s.pending) != 1 || s.pending[0].payload.Timestamp != stamp {
		t.Fatal("retry changed payload")
	}
	time.Sleep(60 * time.Millisecond)
	srv, cli = net.Pipe()
	defer srv.Close()
	defer cli.Close()
	s.conn = srv
	if err := s.flushPending(); err != nil {
		t.Fatal(err)
	}
	cli.SetReadDeadline(time.Now().Add(20 * time.Millisecond))
	if _, err := ReadFrame(cli); err == nil {
		t.Fatal("expired retry sent")
	}
}
