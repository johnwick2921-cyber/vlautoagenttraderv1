package updaterwire

import (
	"errors"
	"fmt"
	"path/filepath"

	"nofx/internal/installpath"
)

// The worker socket's name. This file is the ONLY place the literal may
// appear (TestWorkerSocketLiteralIsConfined), as "hold.json" is confined to
// store/maintenance_hold.go.
const (
	// UpdaterDirName is the updater's directory under the data dir — the one
	// store.MaintenanceHoldPath already uses for hold.json.
	UpdaterDirName = "updater"
	// SocketFileName is the worker socket inside UpdaterDirName.
	SocketFileName = "worker.sock"
	// maxSocketPath is sun_path (108) minus the NUL. A longer path would be
	// truncated by the kernel into a DIFFERENT path, so it is refused.
	maxSocketPath = 107
)

// ErrBadDataDir means the data dir is empty or relative. The socket is never
// resolved against the process's cwd by accident.
var ErrBadDataDir = errors.New("updaterwire: data dir must be an absolute path")

// ErrPathTooLong means the resolved socket path does not fit sun_path.
var ErrPathTooLong = errors.New("updaterwire: socket path longer than 107 bytes")

// SocketPath is <dataDir>/updater/worker.sock. dataDir is the bot's data dir
// (trader.MaintenanceDataDir() in the app; SocketPathFor in a CLI/worker).
func SocketPath(dataDir string) (string, error) {
	if dataDir == "" || !filepath.IsAbs(dataDir) {
		return "", ErrBadDataDir
	}
	p := filepath.Join(filepath.Clean(dataDir), UpdaterDirName, SocketFileName)
	if len(p) > maxSocketPath {
		return "", fmt.Errorf("%w (%d bytes)", ErrPathTooLong, len(p))
	}
	return p, nil
}

// SocketPathFor resolves the socket for an installation directory through
// internal/installpath — the ONE resolver the bot and the hold CLI use
// (DB_PATH from the process env, else <installDir>/.env, else data/data.db,
// anchored on installDir) — so the worker listens where the app dials.
func SocketPathFor(installDir string) (string, error) {
	abs, err := filepath.Abs(installDir)
	if err != nil {
		return "", err
	}
	return SocketPath(installpath.DataDir(abs, installpath.DBPath(installpath.DotEnvGetenv(abs))))
}
