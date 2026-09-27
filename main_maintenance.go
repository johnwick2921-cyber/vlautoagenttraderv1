package main

import (
	"os"

	"nofx/internal/installpath"
)

// resolveMaintenanceDataDir is main's half of the ONE resolver (MUST-2): the
// directory of the DB path main actually opens, anchored on the process
// working directory (the unit's WorkingDirectory), absolute.
func resolveMaintenanceDataDir(dbPath string) string {
	wd, err := os.Getwd()
	if err != nil {
		wd = "."
	}
	return installpath.DataDir(wd, dbPath)
}
