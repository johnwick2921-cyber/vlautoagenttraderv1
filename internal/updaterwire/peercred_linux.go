//go:build linux

package updaterwire

import (
	"errors"
	"net"
	"syscall"
)

// PeerUID returns the uid of the process on the other end of c, from the
// kernel (SO_PEERCRED) — not from anything the peer says.
func PeerUID(c *net.UnixConn) (uint32, error) {
	if c == nil {
		return 0, errors.New("updaterwire: nil conn")
	}
	raw, err := c.SyscallConn()
	if err != nil {
		return 0, err
	}
	var cred *syscall.Ucred
	var serr error
	if err := raw.Control(func(fd uintptr) {
		cred, serr = syscall.GetsockoptUcred(int(fd), syscall.SOL_SOCKET, syscall.SO_PEERCRED)
	}); err != nil {
		return 0, err
	}
	if serr != nil {
		return 0, serr
	}
	if cred == nil {
		return 0, errors.New("updaterwire: no peer credentials")
	}
	return cred.Uid, nil
}
