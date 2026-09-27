//go:build !linux

package updaterwire

import (
	"errors"
	"net"
)

// PeerUID is linux-only (SO_PEERCRED). Elsewhere it fails, and every caller
// treats a failed peer lookup as a refusal — fail closed.
func PeerUID(*net.UnixConn) (uint32, error) {
	return 0, errors.New("updaterwire: SO_PEERCRED unsupported on this OS")
}
