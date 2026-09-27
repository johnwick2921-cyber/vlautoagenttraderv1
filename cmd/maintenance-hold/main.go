// Command maintenance-hold is the local operator CLI for the W-ONE-BUTTON
// installation-wide maintenance hold. See internal/holdcli for usage.
package main

import (
	"os"

	"nofx/internal/holdcli"
)

func main() { os.Exit(holdcli.Run(os.Args[1:], os.Stdout, os.Stderr)) }
