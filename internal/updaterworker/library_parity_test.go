package updaterworker

import (
	"bytes"
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"time"

	"nofx/internal/activation"
)

// Compile-time half of the parity: every mirror converts to activation's type
// and back. Go allows a struct conversion only between identical field
// sequences (names, types, order; tags ignored), so a renamed, retyped,
// reordered, added or dropped field on EITHER side fails the build here —
// and in library_activation.go's converters, which are the same conversions.
var (
	_ = activation.Receipt(Receipt{})
	_ = Receipt(activation.Receipt{})
	_ = activation.Release(Release{})
	_ = Release(activation.Release{})
	_ = activation.Identity(Identity{})
	_ = Identity(activation.Identity{})
	_ = activation.WatchOpts(WatchOpts{})
	_ = WatchOpts(activation.WatchOpts{})
)

// PIN (the adapter's mirror types ARE activation's): field names, types, order
// and struct tags, read from the REAL internal/activation now that it links —
// the golden in TestLibrarySeamMirrorsTheLandedActivationShape was copied from
// afd60391 by hand; this compares against the library itself.
//
// The conversion ignores tags, so tags are what the compiler cannot see:
//
//   - Receipt and WatchOpts: the tag of every field must be IDENTICAL on both
//     sides. A receipt is persisted in the job file as the worker's Receipt and
//     is the library's evidence verbatim; a tag drift would rename a key of the
//     job file / receipt route silently.
//   - Release and Identity: activation's are UNTAGGED as landed (it never
//     marshals them); the job file's tags are the mirror's own, pinned here
//     verbatim. A tag appearing on activation's side is RED — whoever adds it
//     decides which spelling the job file carries, on purpose.
//
// Plus the behavioural half: a fully-populated activation.Receipt marshals to
// the SAME bytes as the adapter's fromReceipt of it.
func TestAdapterReceiptParity(t *testing.T) {
	type field struct{ name, typ, tag string }
	fieldsOf := func(v any) []field {
		rt := reflect.TypeOf(v)
		out := make([]field, rt.NumField())
		for i := range out {
			f := rt.Field(i)
			out[i] = field{f.Name, f.Type.String(), string(f.Tag)}
		}
		return out
	}
	for _, c := range []struct {
		name        string
		mirror, lib any
		// libUntagged: activation's side carries no tags (as landed); the
		// mirror's are the job file's own, pinned by mirrorTags.
		libUntagged bool
		mirrorTags  []string
	}{
		{name: "Receipt", mirror: Receipt{}, lib: activation.Receipt{}},
		{name: "WatchOpts", mirror: WatchOpts{}, lib: activation.WatchOpts{}},
		{name: "Release", mirror: Release{}, lib: activation.Release{}, libUntagged: true, mirrorTags: []string{
			`json:"dir"`, `json:"sha"`, `json:"binary"`, `json:"dist"`, `json:"release_file"`, `json:"manifest_path,omitempty"`}},
		{name: "Identity", mirror: Identity{}, lib: activation.Identity{}, libUntagged: true, mirrorTags: []string{
			`json:"pid"`, `json:"start_ticks"`}},
	} {
		t.Run(c.name, func(t *testing.T) {
			m, l := fieldsOf(c.mirror), fieldsOf(c.lib)
			if len(m) != len(l) {
				t.Fatalf("%s: the mirror has %d fields, activation's has %d:\nmirror     %v\nactivation %v", c.name, len(m), len(l), m, l)
			}
			for i := range m {
				if m[i].name != l[i].name || m[i].typ != l[i].typ {
					t.Errorf("%s field %d: mirror %s %s, activation %s %s (names, types and ORDER must agree)", c.name, i, m[i].name, m[i].typ, l[i].name, l[i].typ)
				}
				switch {
				case !c.libUntagged && m[i].tag != l[i].tag:
					t.Errorf("%s.%s: tag drift — mirror `%s`, activation `%s`", c.name, m[i].name, m[i].tag, l[i].tag)
				case c.libUntagged && l[i].tag != "":
					t.Errorf("%s.%s: activation's %s now carries a tag `%s` (landed untagged); decide the job file's spelling on purpose and update this pin", c.name, l[i].name, c.name, l[i].tag)
				case c.libUntagged && (i >= len(c.mirrorTags) || m[i].tag != c.mirrorTags[i]):
					want := "(none pinned)"
					if i < len(c.mirrorTags) {
						want = c.mirrorTags[i]
					}
					t.Errorf("%s.%s: the job file's tag drifted — mirror `%s`, pinned `%s`", c.name, m[i].name, m[i].tag, want)
				}
			}
		})
	}

	// behavioural: the job file's receipt bytes are the library's receipt bytes
	t0 := time.Date(2026, 9, 24, 21, 5, 1, 123456789, time.UTC)
	for _, rc := range []activation.Receipt{
		{Step: "watch", StartedAt: t0, EndedAt: t0.Add(3 * time.Second), OK: false,
			Evidence: map[string]string{"expect_sha": strings.Repeat("a", 40), "pid": "4242", "since": t0.Format(time.RFC3339)}, Err: "not proven within 1.2s"},
		{Step: "backup", StartedAt: t0, EndedAt: t0, OK: true, Evidence: map[string]string{}}, // computed-empty evidence
		{Step: "stage", StartedAt: t0, EndedAt: t0, OK: true},                                 // uncomputed evidence
	} {
		lib, err := json.Marshal(rc)
		if err != nil {
			t.Fatal(err)
		}
		mir, err := json.Marshal(fromReceipt(rc))
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(lib, mir) {
			t.Errorf("receipt bytes differ:\nactivation %s\nmirror     %s", lib, mir)
		}
		var back activation.Receipt
		if err := json.Unmarshal(mir, &back); err != nil || !reflect.DeepEqual(back.Evidence, rc.Evidence) && len(rc.Evidence) > 0 || back.Step != rc.Step || back.Err != rc.Err || !back.StartedAt.Equal(rc.StartedAt) {
			t.Errorf("the mirror's bytes do not decode back into activation's receipt: %+v, %v (want %+v)", back, err, rc)
		}
	}
}
