// Package wireserver is the WORKER end of the W-ONE-BUTTON M3 updater channel:
// it listens on <data>/updater/worker.sock and answers typed frames from
// nofx/internal/updaterwire.
//
// It is a separate package on purpose. The trading app (api/, trader/,
// kernel/, agent/, telegram/, the root main package) DIALS the worker through
// updaterwire.Dial and must never link the listening side; the import census
// in store/maintenance_hold_writers_test.go fails the build of any such
// import. The M4 updater worker binary is the only intended caller of Listen.
//
// Every refusal fails closed and is logged as a fixed reason token plus
// updaterwire.FrameDigest (length + truncated SHA-256) — never the frame's
// bytes, so nothing a peer sends can reach the log verbatim.
package wireserver

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net"
	"os"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"nofx/internal/updaterwire"
)

// Handler answers one decoded, already-validated request. Its Response is
// validated before it reaches the wire; an invalid one is replaced by
// {ok:false,error:"internal"}.
type Handler func(updaterwire.Request) updaterwire.Response

// Refusals specific to the listening side.
var (
	// ErrInUse means another worker holds the socket: either the worker lock
	// is taken or a live listener answers at the path. Never unlinked.
	ErrInUse = errors.New("wireserver: worker socket already in use")
	// ErrNotSocketPath means Listen was handed a path that is not the
	// canonical <data>/updater/worker.sock shape.
	ErrNotSocketPath = errors.New("wireserver: not the updater socket path")
	// ErrNilHandler means Serve was called without a handler.
	ErrNilHandler = errors.New("wireserver: nil handler")
)

// Seams. geteuid is our uid; peerUID is the kernel SO_PEERCRED lookup;
// fileUID reads a pre-existing path's owner. Tests swap them to play a
// foreign uid without root.
var (
	geteuid = os.Geteuid
	peerUID = updaterwire.PeerUID
	fileUID = func(fi fs.FileInfo) (uint32, bool) {
		st, ok := fi.Sys().(*syscall.Stat_t)
		if !ok {
			return 0, false
		}
		return st.Uid, true
	}
)

// Tunables (vars so tests can shorten them).
var (
	idleTimeout  = 30 * time.Second
	writeTimeout = 5 * time.Second
	maxConns     = 8
	probeTimeout = 500 * time.Millisecond
)

// internalResponse is what the peer gets when the handler panics or returns
// a response that fails validation.
var internalResponse = updaterwire.Response{OK: false, Error: "internal"}

// lockFileName is the single-worker flock beside the socket.
const lockFileName = ".worker.lock"

// Listener is a bound, owner-only worker socket plus the lock that makes it
// the only one.
type Listener struct {
	ln   *net.UnixListener
	lock *os.File
	path string
	logf func(string, ...any)

	mu     sync.Mutex
	conns  map[*net.UnixConn]struct{}
	closed bool

	closeOnce sync.Once
	closeErr  error
}

// Listen binds the worker socket at path, which must be the absolute, clean
// result of updaterwire.SocketPath (base name worker.sock inside a directory
// named updater). logf receives every refusal line (nil = log.Printf).
//
// It refuses, in order:
//   - a path that is relative, unclean or not the socket's canonical shape;
//   - a data dir that does not exist (the worker never invents an install);
//   - an updater dir that is a symlink, not a dir, not ours, or looser than
//     0700 (the dir is created 0700 when absent — the private dir, not a
//     process-wide umask, closes the bind→chmod window);
//   - a second worker (flock on <dir>/.worker.lock, non-blocking);
//   - anything pre-existing at the path that is a symlink, not a socket, not
//     ours, or a socket that still answers. Only a non-answering socket we
//     own inside our private dir — a crashed worker's leftover — is removed.
//
// The bound socket is chmod 0600 and re-checked before Listen returns.
func Listen(path string, logf func(string, ...any)) (*Listener, error) {
	if logf == nil {
		logf = log.Printf
	}
	if !filepath.IsAbs(path) || filepath.Clean(path) != path {
		return nil, updaterwire.ErrBadPath
	}
	dir := filepath.Dir(path)
	if filepath.Base(path) != updaterwire.SocketFileName || filepath.Base(dir) != updaterwire.UpdaterDirName {
		return nil, ErrNotSocketPath
	}
	if len(path) > 107 {
		return nil, updaterwire.ErrPathTooLong
	}
	dataDir := filepath.Dir(dir)
	if fi, err := os.Stat(dataDir); err != nil || !fi.IsDir() {
		return nil, fmt.Errorf("wireserver: data dir: %w", fs.ErrNotExist)
	}
	if err := os.Mkdir(dir, 0o700); err != nil && !errors.Is(err, fs.ErrExist) {
		return nil, fmt.Errorf("wireserver: updater dir: %w", err)
	}
	euid := geteuid()
	if err := updaterwire.CheckPrivateDir(dir, euid); err != nil {
		return nil, err
	}

	lock, err := os.OpenFile(filepath.Join(dir, lockFileName), os.O_CREATE|os.O_RDWR|syscall.O_NOFOLLOW, 0o600)
	if err != nil {
		return nil, fmt.Errorf("wireserver: worker lock: %w", err)
	}
	if err := syscall.Flock(int(lock.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		lock.Close()
		return nil, ErrInUse
	}
	release := func() {
		syscall.Flock(int(lock.Fd()), syscall.LOCK_UN)
		lock.Close()
	}

	if err := clearStale(path, euid); err != nil {
		release()
		return nil, err
	}
	ln, err := net.ListenUnix("unix", &net.UnixAddr{Name: path, Net: "unix"})
	if err != nil {
		release()
		return nil, fmt.Errorf("wireserver: bind: %w", err)
	}
	ln.SetUnlinkOnClose(true)
	if err := os.Chmod(path, 0o600); err != nil {
		ln.Close()
		release()
		return nil, fmt.Errorf("wireserver: chmod socket: %w", err)
	}
	if err := updaterwire.CheckSocketFile(path, euid); err != nil {
		ln.Close()
		release()
		return nil, err
	}
	return &Listener{ln: ln, lock: lock, path: path, logf: logf, conns: map[*net.UnixConn]struct{}{}}, nil
}

// clearStale decides what to do with whatever already sits at path.
func clearStale(path string, euid int) error {
	fi, err := os.Lstat(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("wireserver: socket path: %w", err)
	}
	if fi.Mode()&fs.ModeSymlink != 0 {
		return fmt.Errorf("socket path: %w", updaterwire.ErrSymlink)
	}
	if fi.Mode()&fs.ModeSocket == 0 {
		return updaterwire.ErrNotSocket
	}
	uid, ok := fileUID(fi)
	if !ok || euid < 0 || uid != uint32(euid) {
		return fmt.Errorf("socket path: %w", updaterwire.ErrForeignOwner)
	}
	c, err := net.DialTimeout("unix", path, probeTimeout)
	if err == nil {
		c.Close()
		return ErrInUse
	}
	if !errors.Is(err, syscall.ECONNREFUSED) {
		return fmt.Errorf("wireserver: probe existing socket: %w", err)
	}
	if err := os.Remove(path); err != nil {
		return fmt.Errorf("wireserver: remove stale socket: %w", err)
	}
	return nil
}

// Path is the bound socket path.
func (l *Listener) Path() string { return l.path }

// Close stops accepting, closes every live connection (so Serve returns at
// once instead of waiting out an idle peer), unlinks the socket and releases
// the worker lock.
func (l *Listener) Close() error {
	l.closeOnce.Do(func() {
		l.closeErr = l.ln.Close()
		l.mu.Lock()
		l.closed = true
		for c := range l.conns {
			c.Close()
		}
		l.mu.Unlock()
		syscall.Flock(int(l.lock.Fd()), syscall.LOCK_UN)
		l.lock.Close()
	})
	return l.closeErr
}

// track registers c as live; false means the listener is already closed and
// c must not be served.
func (l *Listener) track(c *net.UnixConn) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.closed {
		return false
	}
	l.conns[c] = struct{}{}
	return true
}

func (l *Listener) untrack(c *net.UnixConn) {
	l.mu.Lock()
	delete(l.conns, c)
	l.mu.Unlock()
}

// Serve accepts connections until the listener is closed (then returns nil).
// Per connection: the peer's SO_PEERCRED uid must equal ours or it is closed
// before a byte is read; then frames are read one per line and each is
// decoded strictly. Any refused frame gets updaterwire.RejectedResponse, is
// logged as reason + digest, and ends the connection.
func (l *Listener) Serve(h Handler) error {
	if h == nil {
		return ErrNilHandler
	}
	sem := make(chan struct{}, maxConns)
	var wg sync.WaitGroup
	defer wg.Wait()
	for {
		c, err := l.ln.AcceptUnix()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				return nil
			}
			var ne net.Error
			if errors.As(err, &ne) && ne.Timeout() {
				continue
			}
			return err
		}
		select {
		case sem <- struct{}{}:
		default:
			l.logf("updater-wire: rejected reason=busy")
			c.Close()
			continue
		}
		if !l.track(c) {
			c.Close()
			<-sem
			continue
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer func() { <-sem }()
			defer l.untrack(c)
			l.serveConn(c, h)
		}()
	}
}

func (l *Listener) reject(c *net.UnixConn, reason string, frame []byte) {
	l.logf("updater-wire: rejected reason=%s %s", reason, updaterwire.FrameDigest(frame))
	out, err := updaterwire.EncodeResponse(updaterwire.RejectedResponse)
	if err == nil {
		c.SetWriteDeadline(time.Now().Add(writeTimeout))
		c.Write(out)
	}
}

func (l *Listener) serveConn(c *net.UnixConn, h Handler) {
	defer c.Close()
	uid, err := peerUID(c)
	own := geteuid()
	if err != nil || own < 0 || uid != uint32(own) {
		if err != nil {
			l.logf("updater-wire: rejected reason=peer_uid uid=unknown")
		} else {
			l.logf("updater-wire: rejected reason=peer_uid uid=%d", uid)
		}
		return
	}
	br := bufio.NewReaderSize(c, 4096)
	for {
		c.SetReadDeadline(time.Now().Add(idleTimeout))
		frame, err := updaterwire.ReadFrame(br)
		if err != nil {
			if errors.Is(err, io.EOF) && len(frame) == 0 {
				return
			}
			if errors.Is(err, net.ErrClosed) {
				return // Close ended the connection: shutdown, not a refusal
			}
			reason := updaterwire.RejectReason(err)
			if errors.Is(err, os.ErrDeadlineExceeded) {
				reason = "timeout"
			}
			l.reject(c, reason, frame)
			return
		}
		req, err := updaterwire.DecodeRequest(frame)
		if err != nil {
			l.reject(c, updaterwire.RejectReason(err), frame)
			return
		}
		resp := l.callHandler(h, req)
		out, err := updaterwire.EncodeResponse(resp)
		if err != nil {
			l.logf("updater-wire: handler response refused verb=%s", req.Verb)
			out, _ = updaterwire.EncodeResponse(internalResponse)
		}
		c.SetWriteDeadline(time.Now().Add(writeTimeout))
		if _, err := c.Write(out); err != nil {
			return
		}
	}
}

// callHandler runs h, turning a panic into the internal response (the
// worker keeps serving; the panic is logged without the request's bytes).
func (l *Listener) callHandler(h Handler, req updaterwire.Request) (resp updaterwire.Response) {
	defer func() {
		if r := recover(); r != nil {
			l.logf("updater-wire: handler panic verb=%s", req.Verb)
			resp = internalResponse
		}
	}()
	return h(req)
}
