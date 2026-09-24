package updaterwire

import (
	"bufio"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// ErrPeerUID means the process behind the socket is not our uid (or its
// credentials could not be read).
var ErrPeerUID = errors.New("updaterwire: peer is not this uid")

// Seams: the effective uid and the kernel peer-credential lookup.
var (
	dialGeteuid = os.Geteuid
	dialPeerUID = PeerUID
)

const (
	dialTimeout    = 2 * time.Second
	requestTimeout = 10 * time.Second
)

// Client is the app's end of one connection to the worker.
type Client struct {
	mu   sync.Mutex
	conn *net.UnixConn
	br   *bufio.Reader
}

// Dial connects to the worker socket at path (from SocketPath). It refuses —
// before connecting — a path that is relative or not in clean form, a dir that is not a private
// 0700 dir we own, and a socket that is a symlink, not a socket, not ours or
// looser than 0600; after connecting, a listener whose SO_PEERCRED uid is not
// ours. An absent socket (worker not running) wraps fs.ErrNotExist.
func Dial(path string) (*Client, error) {
	if !filepath.IsAbs(path) || filepath.Clean(path) != path {
		return nil, ErrBadPath
	}
	euid := dialGeteuid()
	if err := CheckPrivateDir(filepath.Dir(path), euid); err != nil {
		return nil, err
	}
	if err := CheckSocketFile(path, euid); err != nil {
		return nil, err
	}
	d := net.Dialer{Timeout: dialTimeout}
	c, err := d.Dial("unix", path)
	if err != nil {
		return nil, fmt.Errorf("updaterwire: dial: %w", err)
	}
	uc, ok := c.(*net.UnixConn)
	if !ok {
		c.Close()
		return nil, ErrNotSocket
	}
	uid, err := dialPeerUID(uc)
	if err != nil || euid < 0 || uid != uint32(euid) {
		uc.Close()
		return nil, ErrPeerUID
	}
	return &Client{conn: uc, br: bufio.NewReader(uc)}, nil
}

// Do sends one request and reads one response. The request is validated
// before it is written (an invalid request never reaches the wire); the
// response is decoded strictly.
func (c *Client) Do(req Request) (Response, error) {
	frame, err := EncodeRequest(req)
	if err != nil {
		return Response{}, err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if err := c.conn.SetDeadline(time.Now().Add(requestTimeout)); err != nil {
		return Response{}, err
	}
	if _, err := c.conn.Write(frame); err != nil {
		return Response{}, err
	}
	out, err := ReadFrame(c.br)
	if err != nil {
		return Response{}, err
	}
	return DecodeResponse(out)
}

// Close closes the connection.
func (c *Client) Close() error {
	return c.conn.Close()
}
