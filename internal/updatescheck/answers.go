// Package updatescheck builds the two POST /api/updates/check answers the web
// client reads (P0 field-names hotfix, owner 10-02 12:2x CT). The builders are
// the SINGLE source of the wire shape: api/handler_updates.go answers with
// them, the parity test compares them byte-for-byte against the checked-in
// fixture, and cmd/gen-updates-check-fixture regenerates that fixture. A rename
// of any key (e.g. tag -> release_id) fails the parity test in api/.
package updatescheck

// Detail carries the worker check detail fields the answers echo. The handler
// fills it from its own parsed checkDetail; the generator fills it from the
// canonical fixture constants.
type Detail struct {
	Tag             string
	TargetCommitish string
	SourceSHA       string
}

// VerifiedReady is the verified_ready answer: the release exists, its
// signature and files were checked, and the build is ready to install.
func VerifiedReady(d Detail) map[string]any {
	return map[string]any{
		"checked":          true,
		"available":        true,
		"ready":            true,
		"tag":              d.Tag,
		"target_commitish": d.TargetCommitish,
		"source_sha":       d.SourceSHA,
		"reason":           "verified, ready",
	}
}

// UpToDate is the up_to_date answer: nothing to install.
func UpToDate(d Detail) map[string]any {
	return map[string]any{
		"checked":          true,
		"available":        false,
		"ready":            false,
		"tag":              d.Tag,
		"target_commitish": d.TargetCommitish,
		"reason":           "up to date",
	}
}

// Canonical fixture inputs. The generator and the parity test both use these,
// so the fixture content can never drift between the two.
const (
	FixtureTag             = "v2026.10.02.3"
	FixtureTargetCommitish = "dev"
	FixtureSourceSHA       = "0123456789abcdef"
)

// FixturePair returns the canonical fixture the web vitest consumes
// (web/src/lib/api/fixtures/updates-check.json).
func FixturePair() map[string]any {
	return map[string]any{
		"verified_ready": VerifiedReady(Detail{
			Tag:             FixtureTag,
			TargetCommitish: FixtureTargetCommitish,
			SourceSHA:       FixtureSourceSHA,
		}),
		"up_to_date": UpToDate(Detail{
			Tag:             FixtureTag,
			TargetCommitish: FixtureTargetCommitish,
		}),
	}
}
