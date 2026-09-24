// Package updateauth is the W-ONE-BUTTON M3 update-authorization core: the
// enrollment files (admin.json + device.key), the HMAC over
// nofx-update-install/v1|release_id|job_id|expires_at (Message), the
// single-use job-id store and the manifest verifier seam.
//
// It is a leaf package (stdlib only) so the attended CLI
// (cmd/updater-bootstrap) and the API gate (api/handler_updates.go) share ONE
// implementation of every rule and cannot drift.
//
// Layout (under the installation's data dir, resolved by internal/installpath
// — the same resolver the maintenance hold uses):
//
//	<dataDir>/updater/            0700, owned by the bot's uid
//	  admin.json       0600       {"user_id","email","enrolled_at","password_binding"} — written ONLY by `updater-bootstrap enroll`
//	  device.key       0600       32 random bytes                   — written ONLY by `updater-bootstrap enroll`
//	  seen_job_ids.json 0600      consumed job ids (single-use)     — written ONLY by Consume
//	  .enroll.lock / .seen.lock   flock files
//
// This file is the ONLY place the file names are spelled (a census test
// enforces it), exactly as "hold.json" is confined to store/maintenance_hold.go.
package updateauth

import (
	"errors"
	"path/filepath"
)

const (
	updaterDirName = "updater"
	adminFileName  = "admin.json"
	deviceKeyName  = "device.key"
	seenFileName   = "seen_job_ids.json"
	enrollLockName = ".enroll.lock"
	seenLockName   = ".seen.lock"
)

// ErrNoDataDir is returned for an empty or relative data dir: a relative dir
// would resolve against whatever the caller's cwd happens to be, which is a
// different installation (fail closed).
var ErrNoDataDir = errors.New("updateauth: data dir unset or not absolute")

func checkDataDir(dataDir string) error {
	if dataDir == "" || !filepath.IsAbs(dataDir) {
		return ErrNoDataDir
	}
	return nil
}

// Dir is <dataDir>/updater.
func Dir(dataDir string) string { return filepath.Join(dataDir, updaterDirName) }

// AdminPath is <dataDir>/updater/admin.json.
func AdminPath(dataDir string) string { return filepath.Join(Dir(dataDir), adminFileName) }

// DeviceKeyPath is <dataDir>/updater/device.key.
func DeviceKeyPath(dataDir string) string { return filepath.Join(Dir(dataDir), deviceKeyName) }

// SeenPath is <dataDir>/updater/seen_job_ids.json.
func SeenPath(dataDir string) string { return filepath.Join(Dir(dataDir), seenFileName) }

func enrollLockPath(dataDir string) string { return filepath.Join(Dir(dataDir), enrollLockName) }
func seenLockPath(dataDir string) string   { return filepath.Join(Dir(dataDir), seenLockName) }
