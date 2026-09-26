package trader

import "time"

// rthInstant is a FIXED instant inside the CME regular session — Wednesday
// 2026-09-23 10:30:00 America/Chicago.
//
// Tests that drive production SEND paths must pin their clock to it. The entry
// admission chain asks kernel.CMEClosedReason(now) on the clock it is GIVEN
// (entry_admission.go:304), so a test that hands it time.Now() inherits the
// real calendar: green before 16:00 CT, and red through the 16:00–17:00 daily
// break with "🌙 cme closed: ... REFUSED — daily break". A suite that does that
// is making a claim about the hour it ran in, not about the code — checklist
// class 110, and it cost two lanes a gate each on 2026-09-23 (Claude-103's
// bridge message 1790198047586, Claude-101's 1790197778185, and the CTO's own
// run at clean dev eb7294c9 at 16:27 CT).
//
// The fix is to pin the SEAM, never to retry and never to touch production:
// the …At seams exist precisely so the clock is an input (class 60).
func rthInstant() time.Time {
	chicago, err := time.LoadLocation("America/Chicago")
	if err != nil {
		// A box without tzdata still gets a deterministic RTH instant:
		// -0500 is Chicago's offset on this date.
		chicago = time.FixedZone("CDT", -5*60*60)
	}
	return time.Date(2026, 9, 23, 10, 30, 0, 0, chicago)
}
