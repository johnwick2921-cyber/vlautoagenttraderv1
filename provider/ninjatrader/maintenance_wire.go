package ninjatrader

import (
	"net"
	"sync"
	"sync/atomic"
	"time"
)

// ── W-ONE-BUTTON M2 site 7 — maintenance frames + the per-connection record ─
//
// The installation hold lives in a file the provider package never reads; the
// trader package hands the server a source (SetMaintenanceSource) exactly as it
// hands it the entry-hold check. The server pushes the hold to the AddOn and
// records what the AddOn answers on the CURRENT connection's record, which a
// reconnect replaces: an ack, a hello, an epoch from connection N is never
// reported for connection N+1 (CTO ruling Q3 — the verifier binds to this
// record, never to FarSideBuildID()).

// Package vars so tests can compress them.
var (
	maintenanceTick   = time.Second     // how often the hold source is polled
	maintenanceResend = 5 * time.Second // re-send cadence while held (fresh census per ack)
)

// wireMonoBase anchors every *_mono_ms field on the process's monotonic clock
// (M1 §7: a WSL wall-clock step moved start times by 151 s; the monotonic
// reading of time.Now() survives it).
var wireMonoBase = time.Now()

func monoMs(t time.Time) int64 { return t.Sub(wireMonoBase).Milliseconds() }

// ConnectionRecord is what the server knows about ONE accepted AddOn
// connection. Hello and Ack are nil until received on this connection.
type ConnectionRecord struct {
	AcceptSeq    uint64                 `json:"accept_seq"`     // 1, 2, 3 … per accept, per process
	AcceptMonoMs int64                  `json:"accept_mono_ms"` // monotonic ms since process start
	RemotePort   int                    `json:"remote_port"`
	Hello        *HelloPayload          `json:"hello"`
	HelloMonoMs  int64                  `json:"hello_mono_ms,omitempty"`
	Ack          *MaintenanceAckPayload `json:"maintenance_ack"`
	AckMonoMs    int64                  `json:"ack_mono_ms,omitempty"`
	// What this connection was last TOLD (so a release goes only to a
	// connection that was held, and a resend keeps the census fresh).
	SentHeld   bool   `json:"sent_held"`
	SentJobID  string `json:"sent_job_id,omitempty"`
	SentMonoMs int64  `json:"sent_mono_ms,omitempty"`
}

type maintenanceWire struct {
	src atomic.Value // func() (held bool, jobID string)
	// tick / resend are copied from the package vars at Start, so a test that
	// compresses them never races a previous server's loop.
	tick, resend time.Duration
	mu           sync.Mutex
	seq          uint64
	conn         net.Conn // the connection rec describes
	rec          ConnectionRecord
}

// SetMaintenanceSource wires the installation hold (trader.maintenanceWireState).
// Unwired = never held = no maintenance frame, ever (byte-identical wire).
func (s *TCPServer) SetMaintenanceSource(fn func() (held bool, jobID string)) {
	if fn != nil {
		s.maint.src.Store(fn)
	}
}

// ConnectionRecord returns a copy of the current connection's record and
// whether that connection is still the connected client. (rec, false) after a
// disconnect: the record is history, not a live AddOn.
func (s *TCPServer) ConnectionRecord() (ConnectionRecord, bool) {
	s.connMu.Lock()
	cur := s.conn
	s.connMu.Unlock()
	s.maint.mu.Lock()
	defer s.maint.mu.Unlock()
	rec := s.maint.rec
	if rec.Hello != nil {
		h := *rec.Hello
		rec.Hello = &h
	}
	if rec.Ack != nil {
		a := *rec.Ack
		a.Connections = append([]CensusConnection(nil), rec.Ack.Connections...)
		a.Accounts = append([]CensusAccount(nil), rec.Ack.Accounts...)
		if rec.Ack.Connections == nil {
			a.Connections = nil
		}
		if rec.Ack.Accounts == nil {
			a.Accounts = nil
		}
		rec.Ack = &a
	}
	return rec, cur != nil && cur == s.maint.conn
}

// beginConnectionRecord starts a fresh record for an accepted connection.
func (s *TCPServer) beginConnectionRecord(c net.Conn, now time.Time) {
	port := 0
	if ta, ok := c.RemoteAddr().(*net.TCPAddr); ok {
		port = ta.Port
	}
	s.maint.mu.Lock()
	s.maint.seq++
	s.maint.conn = c
	s.maint.rec = ConnectionRecord{AcceptSeq: s.maint.seq, AcceptMonoMs: monoMs(now), RemotePort: port}
	s.maint.mu.Unlock()
}

func (s *TCPServer) recordHello(c net.Conn, p HelloPayload, now time.Time) {
	s.maint.mu.Lock()
	defer s.maint.mu.Unlock()
	if s.maint.conn != c {
		return
	}
	s.maint.rec.Hello = &p
	s.maint.rec.HelloMonoMs = monoMs(now)
}

func (s *TCPServer) recordMaintenanceAck(c net.Conn, p MaintenanceAckPayload, now time.Time) bool {
	s.maint.mu.Lock()
	defer s.maint.mu.Unlock()
	if s.maint.conn != c {
		return false
	}
	s.maint.rec.Ack = &p
	s.maint.rec.AckMonoMs = monoMs(now)
	return true
}

// pushMaintenance tells connection c the hold state when it must: held and
// (not yet told, a different job, or the resend interval elapsed); or released
// after having been told held. Anything else sends nothing.
func (s *TCPServer) pushMaintenance(c net.Conn, now time.Time) {
	fn, _ := s.maint.src.Load().(func() (bool, string))
	if fn == nil || c == nil {
		return
	}
	held, job := fn()
	if !held {
		job = ""
	}
	s.maint.mu.Lock()
	if s.maint.conn != c {
		s.maint.mu.Unlock()
		return
	}
	r := s.maint.rec
	changed := held != r.SentHeld || job != r.SentJobID
	resend := s.maint.resend
	if resend <= 0 {
		resend = 5 * time.Second
	}
	due := held && (changed || monoMs(now)-r.SentMonoMs >= resend.Milliseconds())
	if !due && !(changed && !held) {
		s.maint.mu.Unlock()
		return
	}
	s.maint.mu.Unlock()

	s.writeMu.Lock()
	_ = c.SetWriteDeadline(time.Now().Add(5 * time.Second))
	err := WriteFrame(c, FrameMaintenance, MaintenancePayload{Held: held, JobID: job})
	s.writeMu.Unlock()
	if err != nil {
		s.logger.Warn("tcp_server: write maintenance frame", "err", err)
		return
	}
	s.maint.mu.Lock()
	if s.maint.conn == c {
		s.maint.rec.SentHeld, s.maint.rec.SentJobID, s.maint.rec.SentMonoMs = held, job, monoMs(now)
	}
	s.maint.mu.Unlock()
	if changed {
		s.logger.Info("tcp_server: 🔒 maintenance frame sent", "held", held, "job_id", job)
	}
}

// maintenanceLoop polls the hold source and pushes changes/resends to the
// connected AddOn. Exits with ctx (not wg-tracked, like livenessReporter).
func (s *TCPServer) maintenanceLoop(ctxDone <-chan struct{}, tick time.Duration) {
	t := time.NewTicker(tick)
	defer t.Stop()
	for {
		select {
		case <-ctxDone:
			return
		case now := <-t.C:
			s.connMu.Lock()
			c := s.conn
			s.connMu.Unlock()
			s.pushMaintenance(c, now)
		}
	}
}

// AckAge is how long ago this record's maintenance_ack arrived, on the
// monotonic clock; ok=false when this connection has not acked.
func (r ConnectionRecord) AckAge() (time.Duration, bool) {
	if r.Ack == nil {
		return 0, false
	}
	return time.Duration(monoMs(time.Now())-r.AckMonoMs) * time.Millisecond, true
}

// MaintenanceAckMaxAge is the oldest ack the installation gate accepts: three
// resend intervals (the AddOn re-acks every resend while held).
func MaintenanceAckMaxAge() time.Duration { return 3 * maintenanceResend }
