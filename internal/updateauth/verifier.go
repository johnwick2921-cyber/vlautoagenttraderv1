package updateauth

import "errors"

// ErrNoVerifiedManifest: no manifest for this release id has been verified
// on disk. In M3 it is the ONLY answer (StubVerifier).
var ErrNoVerifiedManifest = errors.New("updateauth: release not verified")

// Manifest is a release manifest that has passed signature + digest
// verification on disk. M4 (release packaging) extends it additively.
type Manifest struct {
	ReleaseID string
}

// Verifier resolves a release id (an IDENTIFIER — never a URL, path or
// command) to a manifest already verified on disk.
type Verifier interface {
	VerifiedManifest(releaseID string) (Manifest, error)
}

// StubVerifier is M3's verifier: it refuses every release id and never
// touches the filesystem, the network or any process — so no input, however
// shaped, reaches disk through it. M4 replaces it.
type StubVerifier struct{}

// VerifiedManifest always returns ErrNoVerifiedManifest.
func (StubVerifier) VerifiedManifest(string) (Manifest, error) {
	return Manifest{}, ErrNoVerifiedManifest
}
