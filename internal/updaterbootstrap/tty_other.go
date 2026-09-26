//go:build !linux

package updaterbootstrap

import "io"

// stdinIsTerminal is linux-only (the installation runs on linux/WSL2);
// elsewhere nothing counts as attended — fail closed.
func stdinIsTerminal(io.Reader) bool { return false }
