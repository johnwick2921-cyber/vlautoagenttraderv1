package trader

import "sync/atomic"

// W-ONE-BUTTON M2 (MUST-2) — the installation's data directory, set ONCE by
// main.go from installpath.DataDir(workDir, cfg.DBPath) BEFORE traders load,
// and read by every maintenance-hold gate. Unset (standalone test fixtures)
// means no hold is configured: gates behave exactly as today (L4 OFF state).
var maintenanceDataDir atomic.Value // string

// SetMaintenanceDataDir records the resolved data dir (absolute).
func SetMaintenanceDataDir(dir string) { maintenanceDataDir.Store(dir) }

// MaintenanceDataDir returns the resolved data dir, or "" when unset.
func MaintenanceDataDir() string {
	s, _ := maintenanceDataDir.Load().(string)
	return s
}
