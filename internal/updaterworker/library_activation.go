package updaterworker

import (
	"nofx/internal/activation"
)

// activationLibrary is the PRODUCTION Library: every method is ONE line that
// delegates to the same-named function of nofx/internal/activation (Claude-103's
// library, #201) with its arguments converted in, and its results converted out.
//
// It holds no state and adds no behaviour — no retry, no default, no check.
// Every refusal, receipt and identity the worker persists is the library's own;
// the worker's state machine (runner.go / steps.go) decides what to do with it.
// TestAdapterDelegatesToActivation drives each method against the library on
// temp fixtures (and pins CurrentIdentity, which only a live systemd unit can
// answer, at the source level).
//
// This is the ONE file in the worker that imports internal/activation, and the
// four conversions below are the ONE place each mirror type crosses to the
// library's: deps.go's Release/Identity/Receipt are updaterjob's (field for
// field activation's, plus the job file's JSON tags — a Go struct conversion
// ignores tags), and WatchOpts is deps.go's own. A field drift on either side
// is a compile error here; a tag drift is TestAdapterReceiptParity.
type activationLibrary struct{}

// NewActivationLibrary is cmd/nofx-updater's newLibrary: the activation
// library adapter. It needs nothing and touches nothing until a method is
// called — and every method is a step the worker has ALREADY persisted as
// started (TestEveryTransitionPersistsBeforeItsSideEffect).
func NewActivationLibrary() (Library, error) { return activationLibrary{}, nil }

var _ Library = activationLibrary{}

// ── the ONE conversion per type, each direction the adapter needs ───────────

func toRelease(r Release) activation.Release       { return activation.Release(r) }
func toIdentity(id Identity) activation.Identity   { return activation.Identity(id) }
func toWatchOpts(o WatchOpts) activation.WatchOpts { return activation.WatchOpts(o) }
func fromRelease(r activation.Release) Release     { return Release(r) }
func fromIdentity(id activation.Identity) Identity { return Identity(id) }
func fromReceipt(rc activation.Receipt) Receipt    { return Receipt(rc) }

// The result shapes, so each delegation stays one line (Go passes a
// multi-value call straight into a function whose parameters match it).

func releaseOut(r activation.Release, err error) (Release, error)  { return fromRelease(r), err }
func receiptOut(rc activation.Receipt, err error) (Receipt, error) { return fromReceipt(rc), err }
func identityOut(id activation.Identity, err error) (Identity, error) {
	return fromIdentity(id), err
}
func stepOut(id activation.Identity, rc activation.Receipt, err error) (Identity, Receipt, error) {
	return fromIdentity(id), fromReceipt(rc), err
}

// ── the eight delegations (Library, deps.go) ────────────────────────────────

func (activationLibrary) Resolve(dir string) (Release, error) {
	return releaseOut(activation.Resolve(dir))
}

func (activationLibrary) Stage(rel Release) (Receipt, error) {
	return receiptOut(activation.Stage(toRelease(rel)))
}

func (activationLibrary) Backup(dbPath, dest string) (Receipt, error) {
	return receiptOut(activation.Backup(dbPath, dest))
}

func (activationLibrary) Snapshot(install Release, dest string) (Receipt, error) {
	return receiptOut(activation.Snapshot(toRelease(install), dest))
}

func (activationLibrary) Activate(rel, prev Release, id Identity) (Identity, Receipt, error) {
	return stepOut(activation.Activate(toRelease(rel), toRelease(prev), toIdentity(id)))
}

func (activationLibrary) Watch(rel Release, id Identity, opts WatchOpts) (Receipt, error) {
	return receiptOut(activation.Watch(toRelease(rel), toIdentity(id), toWatchOpts(opts)))
}

func (activationLibrary) RollbackTo(prev, install Release, id Identity) (Identity, Receipt, error) {
	return stepOut(activation.RollbackTo(toRelease(prev), toRelease(install), toIdentity(id)))
}

func (activationLibrary) CurrentIdentity() (Identity, error) {
	return identityOut(activation.CurrentIdentity())
}
