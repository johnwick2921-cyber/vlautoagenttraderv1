package updaterworker

// reverifier.go — W-ONE-BUTTON M4 (3b-B, unit U4N item A): the production
// Reverifier, the release re-proof the job's downloaded and verified states
// (and the nt8 rule, the resume and the boot check, through facts) call. It is
// a thin adapter over U3's own functions and nothing else:
//
//	Verdict(id)  updaterjob.ReadVerdict(<data dir>, id)          (the app-side reader),
//	             refused unless its release_dir is <ReleaseRoot(install)>/<source_sha> NOW
//	Rehash(v)    RehashRelease(<v as the verdict file>)
//	Reverify(v)  ReverifyRelease(<v as the verdict file>, <install>/deploy/release_allowed_signers)
//
// Reverify is bound to the verdict it is GIVEN (deps.go: the signed manifest's
// sha256, release_id and source_sha and the signing key must be the ones that
// verdict records — U3's manifestIsTheVerdicts refuses otherwise); it never
// re-reads a verdict of its own. The allowed-signers file is read NOW, at
// every re-proof, from the installation tree (C6/C7): absent ⇒ refused.
//
// The mirror ⇄ verdict-file mapping lives HERE and nowhere else (mirrorVerdict,
// Verdict.file); TestVerdictMirrorIsTheVerdictFile pins its field parity.

import (
	"fmt"
	"os"
	"path/filepath"

	"nofx/internal/updaterjob"
)

// NewReleaseReverifier is the production Reverifier for the installation t
// (cmd/nofx-updater's newReverifier). It touches no file: the verdict and the
// allowed-signers file are read at each call.
func NewReleaseReverifier(t Target) (Reverifier, error) {
	if !filepath.IsAbs(t.DataDir) || !filepath.IsAbs(t.InstallDir) {
		return nil, fmt.Errorf("updaterworker: re-proof: data dir %q and install dir %q must be absolute", t.DataDir, t.InstallDir)
	}
	return releaseReverifier{installDir: t.InstallDir, dataDir: t.DataDir, allowedSigners: ReleaseAllowedSignersPath(t.InstallDir)}, nil
}

// releaseReverifier is the production Reverifier (see the file comment).
type releaseReverifier struct {
	installDir     string // the installation (the release root must be outside it)
	dataDir        string // the installation's data dir (the verdicts live under it)
	allowedSigners string // <install>/deploy/release_allowed_signers
}

// Verdict is updaterjob.ReadVerdict's file for releaseID, as the mirror —
// and only while it is a verdict for the CURRENT release root (U4F defect 6,
// a fail-closed default named for the CTO): the release root is re-checked
// NOW (ReleaseRoot: set, absolute, its own resolved path, outside the
// install) and the verdict's release_dir must be exactly <that root>/<its
// source sha>. A release fetched under a root the operator no longer names
// is refused, so no step resolves it (brief row 3: Resolve(<RELEASE_DIR>/<sha>)).
func (r releaseReverifier) Verdict(releaseID string) (Verdict, error) {
	v, err := updaterjob.ReadVerdict(r.dataDir, releaseID)
	if err != nil {
		return Verdict{}, err
	}
	root, err := ReleaseRoot(r.installDir)
	if err != nil {
		return Verdict{}, fmt.Errorf("release %s: %w", releaseID, err)
	}
	if want := filepath.Join(root, v.SourceSHA); v.ReleaseDir != want {
		// BOTH steps (U4F verify note 2): a re-fetch alone refuses — the
		// verdict is written once — so the old verdict goes first, by hand.
		vpath, perr := updaterjob.VerdictPath(r.dataDir, releaseID)
		if perr != nil {
			return Verdict{}, fmt.Errorf("%w: the verdict for %s names the release dir %s, not %s under the current NOFX_RELEASE_DIR (and its path: %w)", ErrReleaseRoot, releaseID, v.ReleaseDir, want, perr)
		}
		// d1 (U4G defect 1): when the moved-TO directory ALREADY EXISTS (the
		// operator moved the directory, not just the knob), the re-fetch step
		// refuses "release directory already exists" — ordering the rm first
		// would leave the operator with no verdict AND an unusable release.
		// The text must therefore name the move-aside case BEFORE the rm, or
		// point out that not moving the directory at all keeps the verdict
		// valid.
		if _, werr := os.Lstat(want); werr == nil {
			return Verdict{}, fmt.Errorf("%w: the verdict for %s names the release dir %s, not %s under the current NOFX_RELEASE_DIR, and %s ALREADY EXISTS — "+
				"to use this release there: (1) move %s aside first (mv it elsewhere), or do not move the release directory at all (this verdict is still valid where it is); (2) then remove the old verdict by hand: rm %s (3) and re-fetch it: nofx-updater --install-dir %s fetch %s",
				ErrReleaseRoot, releaseID, v.ReleaseDir, want, want, want, vpath, r.installDir, releaseID)
		}
		return Verdict{}, fmt.Errorf("%w: the verdict for %s names the release dir %s, not %s under the current NOFX_RELEASE_DIR — "+
			"to use this release there: (1) remove the old verdict by hand: rm %s (2) then re-fetch it: nofx-updater --install-dir %s fetch %s",
			ErrReleaseRoot, releaseID, v.ReleaseDir, want, vpath, r.installDir, releaseID)
	}
	return mirrorVerdict(v), nil
}

// Rehash is RehashRelease over the verdict it is given.
func (r releaseReverifier) Rehash(v Verdict) (int, error) {
	return RehashRelease(v.file())
}

// Reverify is ReverifyRelease over the verdict it is given, against the
// installation's allowed-signers file read NOW, and the facts the signed
// manifest proves.
func (r releaseReverifier) Reverify(v Verdict) (ReleaseFacts, error) {
	m, sv, err := ReverifyRelease(v.file(), r.allowedSigners)
	if err != nil {
		return ReleaseFacts{}, err
	}
	arts := make(map[string]string, len(m.Artifacts))
	for _, a := range m.Artifacts {
		arts[a.Path] = a.SHA256
	}
	return ReleaseFacts{
		ReleaseID:         m.ReleaseID,
		SourceSHA:         m.SourceSHA,
		ManifestSHA256:    m.SHA256,
		SignerFingerprint: sv.Fingerprint,
		AddonBuildID:      m.AddonBuildID,
		Artifacts:         arts,
	}, nil
}

// mirrorVerdict is the verdict file as the worker's mirror (the ONE mapping).
func mirrorVerdict(v updaterjob.Verdict) Verdict { return Verdict(v) }

// file is the mirror as the verdict file U3's re-proofs take (the ONE mapping).
func (v Verdict) file() updaterjob.Verdict { return updaterjob.Verdict(v) }
