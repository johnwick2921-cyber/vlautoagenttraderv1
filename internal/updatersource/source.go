// Package updatersource holds the ONE build-time release source for this
// repository (plan ONE-BUTTON P-A / audit fold B1).
//
// The release workflow (release.yml RELEASE_REPO) and the bot's release
// check must agree on this value; deploy's contract test pins equality
// between the two so a build can never be pointed at the wrong venue.
//
// PARTNER CARVE-OUT: this repo carries the PARTNER repository. The nofx
// repo carries its own value (johnwick2921-cyber/nofx). Each repo's
// allowed_signers and this constant move together — a partner build can
// never verify a nofx-signed release and vice versa.
package updatersource

// ReleaseRepo is the single release source for THIS repository's builds.
const ReleaseRepo = "johnwick2921-cyber/vlautoagenttraderv1"
