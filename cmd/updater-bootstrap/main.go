// Command updater-bootstrap is the attended, local CLI for W-ONE-BUTTON M3
// update authorization: `enroll <email>` binds the update administrator and
// writes data/updater/admin.json + device.key; `authorize <release_id>`
// prints one 5-minute, single-use install authorization computed from
// device.key. See internal/updaterbootstrap for usage.
package main

import (
	"os"

	"nofx/internal/updaterbootstrap"
)

func main() { os.Exit(updaterbootstrap.Run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr)) }
