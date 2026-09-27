//go:build !linux

package main

import "io"

// stdinIsTerminal is linux-only (the installation runs on linux/WSL2);
// elsewhere a resume is refused.
func stdinIsTerminal(io.Reader) bool { return false }
