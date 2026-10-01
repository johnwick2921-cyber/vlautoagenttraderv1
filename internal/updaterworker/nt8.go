package updaterworker

import (
	"context"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"vl/internal/updaterjob"
)

// ── the AddOn decision (C12 as ruled: the stricter composite) ──────────────
//
// nt8_skipped ONLY when ALL of these are READ true, else nt8_updated (the job
// parks for the owner's attended F5 and resumes only on `vl-updater resume`):
//
//	(a) /api/maintenance addon_ack is present, held, carries THIS job's id and
//	    is at most 15 s old (the AddOn running now answered this hold);
//	(b) addon_ack.build_id == the SIGNED manifest's addon.build_id, both
//	    non-empty and not "n/a";
//	(c) the release's ninjascript/*.cs sha256 set (signed artifacts[]) equals
//	    the install tree's ninjascript/*.cs hashed NOW by content.
//
// No file date is ever read (TestNT8DecisionReadsTheAckNeverAFileDate touches
// every mtime and the decision does not move). A read that fails is
// "updated" — the park is the safe side.

// ninjascriptGlob is the C# the AddOn compiles from (top level only).
const ninjascriptGlob = "ninjascript/*.cs"

func (w *Worker) decideNT8(ctx context.Context, j updaterjob.Job) updaterjob.NT8Decision {
	// UPDATER-NT8-CLOSED: when the drain passed in the nt8_absent path, the
	// decision is made WITHOUT an ack (NT8 is closed; none can exist). But
	// the verdict is revoked the moment an AddOn connects — if the gate is no
	// longer absent-eligible here, the normal decision (which reads the ack)
	// applies again, never grandfathered.
	if j.DrainPath == drainPathNT8Absent {
		if b, _ := w.absentGate(ctx, j); b == "" {
			return w.decideNT8Absent(j)
		}
	}
	d := updaterjob.NT8Decision{Decision: updaterjob.NT8Updated}
	reasons := []string{}
	why := func(format string, a ...any) { reasons = append(reasons, fmt.Sprintf(format, a...)) }

	f, err := w.facts(j)
	if err != nil {
		why("the release could not be re-verified: %v", err)
	}
	d.ManifestBuildID = f.AddonBuildID
	if err == nil && !validBuildID(f.AddonBuildID) {
		why("the manifest names no addon build_id")
	}
	m, merr := w.app.Maintenance(ctx)
	switch {
	case merr != nil:
		why("maintenance read failed: %v", merr)
	case m.AddonAck == nil:
		why("no AddOn maintenance_ack")
	default:
		a := m.AddonAck
		d.AckedBuildID, d.AckedAt, d.AckAcceptSeq = a.BuildID, a.Received, a.AcceptSeq
		if b := ackFor(a, j.JobID); b != "" {
			why("%s", b)
		} else if err == nil && validBuildID(f.AddonBuildID) && a.BuildID != f.AddonBuildID {
			why("the AddOn runs build %q, the release is %q", a.BuildID, f.AddonBuildID)
		}
	}
	if err == nil {
		rel := csFromArtifacts(f.Artifacts)
		inst, ierr := csOfTree(w.cfg.Target.InstallDir)
		switch {
		case ierr != nil:
			why("the install's %s could not be hashed: %v", ninjascriptGlob, ierr)
		case len(rel) == 0:
			why("the release lists no %s — the AddOn source cannot be proven unchanged", ninjascriptGlob)
		default:
			same := equalSets(rel, inst)
			d.CSUnchanged = &same
			if !same {
				why("the release's %s differs from the install's (%s)", ninjascriptGlob, csDiff(rel, inst))
			}
		}
	}
	if len(reasons) == 0 {
		d.Decision = updaterjob.NT8Skipped
		return d
	}
	d.Reason = clipText(strings.Join(reasons, "; "))
	return d
}

// decideNT8Absent (UPDATER-NT8-CLOSED item 3): NT8 is closed — no ack is
// read, none is waited for. The decision rests on the signed manifest vs the
// install's ninjascript/*.cs hashed by content:
//
//   - C# unchanged → nt8_skipped, as today;
//   - C# differs → STILL nt8_skipped (the Go side completes), with F5Owed
//     recorded — "AddOn F5 owed at next NT8 start". The job does NOT park:
//     NT8 is closed, there is nobody to F5 now.
//   - a read failure → F5Owed too (fail closed: assume the F5 is needed).
func (w *Worker) decideNT8Absent(j updaterjob.Job) updaterjob.NT8Decision {
	d := updaterjob.NT8Decision{Decision: updaterjob.NT8Skipped, Absent: true}
	reasons := []string{}
	why := func(format string, a ...any) { reasons = append(reasons, fmt.Sprintf(format, a...)) }

	f, err := w.facts(j)
	if err != nil {
		why("the release could not be re-verified: %v", err)
	} else {
		d.ManifestBuildID = f.AddonBuildID
		rel := csFromArtifacts(f.Artifacts)
		inst, ierr := csOfTree(w.cfg.Target.InstallDir)
		switch {
		case ierr != nil:
			why("the install's %s could not be hashed: %v", ninjascriptGlob, ierr)
		case len(rel) == 0:
			why("the release lists no %s — the AddOn source cannot be proven unchanged", ninjascriptGlob)
		default:
			same := equalSets(rel, inst)
			d.CSUnchanged = &same
			if !same {
				why("the release's %s differs from the install's (%s)", ninjascriptGlob, csDiff(rel, inst))
			}
		}
	}
	if len(reasons) == 0 {
		return d
	}
	// fail closed: with the C# state unprovable or differing, the F5 is owed
	d.F5Owed = true
	d.Reason = "AddOn F5 owed at next NT8 start: " + clipText(strings.Join(reasons, "; "))
	return d
}

// csFromArtifacts is the signed manifest's ninjascript/*.cs entries, by base
// name → sha256.
func csFromArtifacts(arts map[string]string) map[string]string {
	out := map[string]string{}
	for p, sum := range arts {
		if ok, _ := path.Match(ninjascriptGlob, p); ok {
			out[path.Base(p)] = sum
		}
	}
	return out
}

// csOfTree hashes <dir>/ninjascript/*.cs by CONTENT (regular files only).
func csOfTree(dir string) (map[string]string, error) {
	hits, err := filepath.Glob(filepath.Join(dir, filepath.FromSlash(ninjascriptGlob)))
	if err != nil {
		return nil, err
	}
	out := map[string]string{}
	for _, h := range hits {
		fi, err := os.Lstat(h)
		if err != nil {
			return nil, err
		}
		if !fi.Mode().IsRegular() {
			return nil, fmt.Errorf("%s is not a regular file", h)
		}
		sum, err := sha256File(h)
		if err != nil {
			return nil, err
		}
		out[filepath.Base(h)] = sum
	}
	return out, nil
}

func equalSets(a, b map[string]string) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if b[k] != v {
			return false
		}
	}
	return true
}

// csDiff names the files that differ (names only, sorted).
func csDiff(rel, inst map[string]string) string {
	seen := map[string]bool{}
	var out []string
	for k, v := range rel {
		seen[k] = true
		if inst[k] != v {
			out = append(out, k)
		}
	}
	for k := range inst {
		if !seen[k] {
			out = append(out, k)
		}
	}
	sort.Strings(out)
	return strings.Join(out, ",")
}
