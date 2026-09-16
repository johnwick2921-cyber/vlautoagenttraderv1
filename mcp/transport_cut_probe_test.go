package mcp

import (
	"bufio"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// P1c probe (transport-resets dispatch): what error string does the stream
// reader surface when (1) the idle watchdog fires vs (2) the peer closes the
// body mid-stream? Answers whether the watchdog is a suspect for "unexpected EOF".
func probeClient(t *testing.T, srv *httptest.Server) *Client {
	t.Helper()
	return streamClient(t, srv.URL, 1)
}

func TestProbeWatchdogFireErrorString(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(200)
		fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"a\"}}]}\n\n")
		w.(http.Flusher).Flush()
		time.Sleep(3 * time.Second) // silence longer than the idle deadline
	}))
	defer srv.Close()
	c := probeClient(t, srv)
	_, err := c.CallWithRequestStreamDeadlines(&Request{}, nil, 300*time.Millisecond, 0)
	t.Logf("WATCHDOG READER ERROR: %q class=%s", err, classifyAIError(err))
	// CLASS 46 D4 — the watchdog's reader error must be DISTINCT from a peer
	// EOF, or "0 idle kills" cannot be told from "0 idle kills we can see".
	if !errors.Is(err, ErrWatchdogIdle) {
		t.Fatalf("expected the watchdog sentinel, got %v", err)
	}
	if strings.Contains(err.Error(), "unexpected EOF") {
		t.Fatalf("a watchdog close must not look like a peer EOF: %v", err)
	}
}

func TestProbePeerCloseMidBodyErrorString(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(200)
		fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"a\"}}]}\n\n")
		w.(http.Flusher).Flush()
		// peer closes the TCP connection mid-body (chunked, no terminator) = FIN
		hj, _ := w.(http.Hijacker)
		conn, _, _ := hj.Hijack()
		conn.Close()
	}))
	defer srv.Close()
	c := probeClient(t, srv)
	_, err := c.CallWithRequestStreamDeadlines(&Request{}, nil, 30*time.Second, 0)
	t.Logf("PEER-FIN READER ERROR: %q class=%s", err, classifyAIError(err))
	if err == nil || !strings.Contains(err.Error(), "unexpected EOF") {
		t.Fatalf("expected unexpected EOF, got %v", err)
	}
}

func TestProbePeerRSTMidBodyErrorString(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(200)
		fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"a\"}}]}\n\n")
		w.(http.Flusher).Flush()
		hj, _ := w.(http.Hijacker)
		conn, _, _ := hj.Hijack()
		if tc, ok := conn.(*net.TCPConn); ok {
			tc.SetLinger(0) // RST instead of FIN
		}
		conn.Close()
	}))
	defer srv.Close()
	c := probeClient(t, srv)
	_, err := c.CallWithRequestStreamDeadlines(&Request{}, nil, 30*time.Second, 0)
	t.Logf("PEER-RST READER ERROR: %q class=%s", err, classifyAIError(err))
	_ = bufio.NewReader
}
