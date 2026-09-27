package updaterworker

import (
	"encoding/json"
	"reflect"
	"sort"
	"strings"
	"testing"

	"nofx/trader"
)

// ── the library seam mirrors the LANDED #201 shape (CTO 1790261377377) ─────
//
// internal/activation is on origin/feat/one-button-m4-activation (#201 head
// afd60391), NOT on dev, so it cannot be imported here. The golden below is
// its exported surface as the worker calls it, copied from afd60391:
//
//	activation.go:31-38  type Release struct{ Dir, SHA, Binary, Dist, ReleaseFile, ManifestPath string }
//	activation.go:44-47  type Identity struct{ PID int; StartTicks uint64 }
//	activation.go:52-59  type Receipt struct{ Step "step"; StartedAt "started_at"; EndedAt "ended_at"; OK "ok"; Evidence "evidence,omitempty"; Err "err,omitempty" }
//	steps.go:214-228     type WatchOpts struct{ LogPath, HealthURL string; Since time.Time; Within time.Duration }
//	activation.go:89     func Resolve(dir string) (Release, error)
//	activation.go:137    func Stage(rel Release) (Receipt, error)
//	steps.go:24          func Backup(dbPath, dest string) (Receipt, error)
//	steps.go:471         func Snapshot(install Release, dest string) (Receipt, error)
//	steps.go:90          func Activate(rel, prev Release, id Identity) (Identity, Receipt, error)
//	steps.go:150         func Watch(rel Release, id Identity, opts WatchOpts) (Receipt, error)
//	steps.go:435         func RollbackTo(prev Release, install Release, id Identity) (Identity, Receipt, error)
//	system.go            func CurrentIdentity() (Identity, error)
//
// Field lists are compared WITH their order: the real adapter
// (library_activation.go, after #201 merges) converts activation.Release(r)
// and updaterjob.Receipt(rc) directly, which Go allows only for identical
// field sequences — this pin fails first, in the worker's own package.
func TestLibrarySeamMirrorsTheLandedActivationShape(t *testing.T) {
	methods := map[string]string{}
	lt := reflect.TypeOf((*Library)(nil)).Elem()
	for i := 0; i < lt.NumMethod(); i++ {
		m := lt.Method(i)
		methods[m.Name] = strings.ReplaceAll(m.Type.String(), "updaterjob.", "")
	}
	want := map[string]string{
		"Resolve":         "func(string) (Release, error)",
		"Stage":           "func(Release) (Receipt, error)",
		"Backup":          "func(string, string) (Receipt, error)",
		"Snapshot":        "func(Release, string) (Receipt, error)",
		"Activate":        "func(Release, Release, Identity) (Identity, Receipt, error)",
		"Watch":           "func(Release, Identity, updaterworker.WatchOpts) (Receipt, error)",
		"RollbackTo":      "func(Release, Release, Identity) (Identity, Receipt, error)",
		"CurrentIdentity": "func() (Identity, error)",
	}
	if !reflect.DeepEqual(methods, want) {
		t.Fatalf("Library seam = %v\nwant the #201 afd60391 shape %v", methods, want)
	}
	fields := func(v any) string {
		rt := reflect.TypeOf(v)
		var out []string
		for i := 0; i < rt.NumField(); i++ {
			f := rt.Field(i)
			s := f.Name + " " + f.Type.String()
			if tag := f.Tag.Get("json"); tag != "" && rt.Name() == "Receipt" {
				s += ` json:"` + tag + `"`
			}
			out = append(out, s)
		}
		return strings.Join(out, "; ")
	}
	for name, c := range map[string]struct {
		v    any
		want string
	}{
		"Release":   {Release{}, "Dir string; SHA string; Binary string; Dist string; ReleaseFile string; ManifestPath string"},
		"Identity":  {Identity{}, "PID int; StartTicks uint64"},
		"Receipt":   {Receipt{}, `Step string json:"step"; StartedAt time.Time json:"started_at"; EndedAt time.Time json:"ended_at"; OK bool json:"ok"; Evidence map[string]string json:"evidence,omitempty"; Err string json:"err,omitempty"`},
		"WatchOpts": {WatchOpts{}, "LogPath string; HealthURL string; Since time.Time; Within time.Duration"},
	} {
		if got := fields(c.v); got != c.want {
			t.Errorf("%s fields = %s\nwant (afd60391) %s", name, got, c.want)
		}
	}
}

// ── the app views mirror the trader JSON the app serves ────────────────────
//
// The worker decodes GET /api/maintenance and /api/installation-gate into its
// own mirrors (the worker must not link trader/). This pin marshals the REAL
// trader types — every field set — and decodes them through the production
// decoders: a renamed or retyped field fails here, not on the first update.
func TestAppViewsMirrorTheTraderJSON(t *testing.T) {
	job, since := "job-u4-0001abcd", "2026-09-24T15:00:00Z"
	mv := trader.MaintenanceStatusView{Held: true, State: "held", JobID: &job, Since: &since, Reason: "updater job", InFlightSends: 2, Drained: true,
		AddonAck: &trader.MaintenanceAckView{Received: "2026-09-24T15:00:05.123456789Z", AgeMs: 1200, Held: true, JobID: job, QueuedCommands: 3, BuildID: "2026-09-23-m21", AcceptSeq: 9}}
	b, err := json.Marshal(mv)
	if err != nil {
		t.Fatal(err)
	}
	got, err := decodeMaintenance(b)
	if err != nil {
		t.Fatalf("decodeMaintenance(the trader's own JSON): %v\n%s", err, b)
	}
	if !got.Held || got.State != "held" || *got.JobID != job || *got.Since != since || got.InFlightSends != 2 || !got.Drained ||
		got.AddonAck == nil || *got.AddonAck != (AckView{Received: mv.AddonAck.Received, AgeMs: 1200, Held: true, JobID: job, QueuedCommands: 3, BuildID: "2026-09-23-m21", AcceptSeq: 9}) {
		t.Fatalf("maintenance view lost a field: %+v (ack %+v)", got, got.AddonAck)
	}
	// the app's "unknown": no hold, no ack → null job_id and addon_ack decode as absent, never as a value
	b, _ = json.Marshal(trader.MaintenanceStatusView{State: "clear"})
	if got, err := decodeMaintenance(b); err != nil || got.JobID != nil || got.AddonAck != nil {
		t.Fatalf("clear view: %+v, %v", got, err)
	}
	gv := trader.InstallationGate{Ready: true, JobID: job, Legs: []trader.InstallationGateLeg{{Name: "hold", Pass: true, Detail: "held", Source: "store"}}, Traders: []string{"t1"}, Note: "n"}
	b, _ = json.Marshal(gv)
	g, err := decodeGate(b)
	if err != nil {
		t.Fatalf("decodeGate(the trader's own JSON): %v", err)
	}
	if !g.Ready || g.JobID != job || len(g.Legs) != 1 || g.Legs[0] != (GateLeg{Name: "hold", Pass: true, Detail: "held", Source: "store"}) || g.Traders[0] != "t1" || g.Note != "n" {
		t.Fatalf("gate view lost a field: %+v", g)
	}
	// every JSON key the trader types carry is a key the mirrors carry
	for name, pair := range map[string][2]any{
		"MaintenanceStatusView": {trader.MaintenanceStatusView{}, MaintenanceView{}},
		"MaintenanceAckView":    {trader.MaintenanceAckView{}, AckView{}},
		"InstallationGate":      {trader.InstallationGate{}, GateView{}},
		"InstallationGateLeg":   {trader.InstallationGateLeg{}, GateLeg{}},
	} {
		src, dst := jsonKeys(pair[0]), jsonKeys(pair[1])
		for _, k := range dst {
			if !contains(src, k) {
				t.Errorf("%s: the worker reads %q, which the trader type does not serve (serves %v)", name, k, src)
			}
		}
	}
}

func jsonKeys(v any) []string {
	rt := reflect.TypeOf(v)
	var out []string
	for i := 0; i < rt.NumField(); i++ {
		tag := strings.Split(rt.Field(i).Tag.Get("json"), ",")[0]
		if tag != "" && tag != "-" {
			out = append(out, tag)
		}
	}
	sort.Strings(out)
	return out
}

func contains(xs []string, x string) bool {
	for _, y := range xs {
		if y == x {
			return true
		}
	}
	return false
}
