package store

// W1 (f), 2026-09-23 — THE DRIFT COUNTER WALKS WHAT THE MARSHALLER WRITES.
//
// StrategyConfig's json:"-" compatibility fields (CoinSource, Indicators,
// CustomPrompt, RiskControl, PromptSections) are re-emitted by the custom
// MarshalJSON under ai_config. The old tag walk skipped "-", so schema= and
// UNCLASSIFIED never saw any of them. These tests call the production
// marshaller (json.Marshal → StrategyConfig.MarshalJSON, the call every save
// makes) and the production UnmarshalJSON, never a re-built key list.

import (
	"encoding/json"
	"reflect"
	"sort"
	"strings"
	"testing"
)

func enumeratedSet(t *testing.T) map[string]bool {
	t.Helper()
	if err := SchemaEnumerationErr(); err != nil {
		t.Fatalf("schema enumeration failed: %v", err)
	}
	set := map[string]bool{}
	for _, p := range EnumerateSchemaKnobs() {
		set[p] = true
	}
	return set
}

// marshalKeyPaths marshals cfg through the production MarshalJSON and returns
// the key paths it wrote.
func marshalKeyPaths(t *testing.T, cfg StrategyConfig) []string {
	t.Helper()
	b, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	keys, err := jsonKeyPaths(b)
	if err != nil {
		t.Fatalf("key paths: %v", err)
	}
	return keys
}

// pathInJSON reports whether a dotted path resolves in decoded JSON, walking
// arrays through element 0 and stopping at a sentinel map. Independent of
// jsonKeyPaths on purpose: it resolves one path, it does not enumerate.
func pathInJSON(v any, path string) bool {
	for _, seg := range strings.Split(path, ".") {
		for {
			arr, ok := v.([]any)
			if !ok || len(arr) == 0 {
				break
			}
			v = arr[0]
		}
		obj, ok := v.(map[string]any)
		if !ok {
			return false
		}
		child, present := obj[seg]
		if !present {
			return false
		}
		v = child
	}
	return true
}

// THE AUDIT'S OWN KNOBS ARE IN THE WALK. Every path the 2026-09-03 audit named
// must be reachable by the enumeration — a leaf by exact path, a container by
// at least one child. At 853981d2 (tag walk, skips json:"-") this found 0 of 4.
func TestAuditDeadKnobsAreInTheSchemaWalk(t *testing.T) {
	set := enumeratedSet(t)
	var missing []string
	for _, p := range AuditDeadKnobs2026_09_03 {
		if set[p] {
			continue
		}
		child := false
		for q := range set {
			if strings.HasPrefix(q, p+".") {
				child = true
				break
			}
		}
		if !child {
			missing = append(missing, p)
		}
	}
	if len(missing) > 0 {
		t.Fatalf("%d of %d audit-dead knob paths are not in the schema walk — schema= and UNCLASSIFIED cannot see them:\n  %s",
			len(missing), len(AuditDeadKnobs2026_09_03), strings.Join(missing, "\n  "))
	}
}

// EVERY ENUMERATED PATH IS A KEY THE MARSHALLER WRITES. No path may exist only
// in a struct tag: each must resolve in json.Marshal(populated) bytes for at
// least one strategy type (ai_trading writes ai_config, grid_trading writes
// grid_config — never both).
func TestEveryEnumeratedPathIsInTheMarshalledBytes(t *testing.T) {
	var docs []any
	for _, st := range schemaStrategyTypes {
		b, err := json.Marshal(populatedStrategyConfig(st))
		if err != nil {
			t.Fatalf("marshal %s: %v", st, err)
		}
		var v any
		if err := json.Unmarshal(b, &v); err != nil {
			t.Fatalf("decode %s: %v", st, err)
		}
		docs = append(docs, v)
	}
	var orphan []string
	for _, p := range EnumerateSchemaKnobs() {
		found := false
		for _, d := range docs {
			if pathInJSON(d, p) {
				found = true
				break
			}
		}
		if !found {
			orphan = append(orphan, p)
		}
	}
	if len(orphan) > 0 {
		t.Fatalf("%d enumerated path(s) the marshaller never writes:\n  %s", len(orphan), strings.Join(orphan, "\n  "))
	}
}

// EVERY KEY THE MARSHALLER WRITES IS ENUMERATED — the other direction. The
// strategy types are listed HERE, not read from schemaStrategyTypes, so an
// enumeration that forgets a MarshalJSON branch (grid_trading writes
// grid_config and no ai_config) fails instead of agreeing with itself.
func TestEveryMarshalledKeyIsEnumerated(t *testing.T) {
	set := enumeratedSet(t)
	for _, st := range []string{"", "ai_trading", "grid_trading"} {
		var outside []string
		for _, k := range marshalKeyPaths(t, populatedStrategyConfig(st)) {
			if !set[k] {
				outside = append(outside, k)
			}
		}
		if len(outside) > 0 {
			t.Errorf("strategy_type %q: MarshalJSON writes %d key(s) the walk does not enumerate:\n  %s",
				st, len(outside), strings.Join(outside, "\n  "))
		}
	}
}

// dashFieldAllowNotPersisted lists json:"-" StrategyConfig fields that are
// deliberately runtime-only (MarshalJSON never writes them). Empty today: all
// five compatibility fields persist under ai_config. A new "-" field must
// either be written by MarshalJSON (and so land in the walk) or be named here
// with the reason.
var dashFieldAllowNotPersisted = map[string]string{}

// EVERY json:"-" FIELD LANDS INSIDE THE ENUMERATION. Populate ONLY that field,
// marshal, and diff against a zero marshal: every key it adds must be an
// enumerated path. This is the test that catches a future "-" field — and a
// future return to a tag walk, which cannot see any of them.
func TestEveryDashFieldLandsInsideTheEnumeration(t *testing.T) {
	set := enumeratedSet(t)
	typ := reflect.TypeOf(StrategyConfig{})
	dash := 0
	for i := 0; i < typ.NumField(); i++ {
		f := typ.Field(i)
		if strings.Split(f.Tag.Get("json"), ",")[0] != "-" {
			continue
		}
		dash++
		added := map[string]bool{}
		for _, st := range schemaStrategyTypes {
			zero := StrategyConfig{StrategyType: st}
			base := map[string]bool{}
			for _, k := range marshalKeyPaths(t, zero) {
				base[k] = true
			}
			one := StrategyConfig{StrategyType: st}
			populateForSchema(reflect.ValueOf(&one).Elem().Field(i), 0)
			for _, k := range marshalKeyPaths(t, one) {
				if !base[k] {
					added[k] = true
				}
			}
		}
		if len(added) == 0 {
			if _, ok := dashFieldAllowNotPersisted[f.Name]; !ok {
				t.Errorf("json:\"-\" field %s: MarshalJSON writes nothing for it — persist it or name it in dashFieldAllowNotPersisted with the reason", f.Name)
			}
			continue
		}
		var outside []string
		for k := range added {
			if !set[k] {
				outside = append(outside, k)
			}
		}
		sort.Strings(outside)
		if len(outside) > 0 {
			t.Errorf("json:\"-\" field %s: MarshalJSON writes %d key(s) the schema walk does not enumerate:\n  %s",
				f.Name, len(outside), strings.Join(outside, "\n  "))
		}
	}
	if dash < 5 {
		t.Fatalf("found %d json:\"-\" fields on StrategyConfig, want the five compatibility fields — the test is reading the wrong type", dash)
	}
	// Non-vacuous: one known leaf from each compatibility struct is enumerated
	// under the path MarshalJSON really writes.
	for _, p := range []string{
		"ai_config.coin_source.source_type",
		"ai_config.indicators.klines.primary_timeframe",
		"ai_config.custom_prompt",
		"ai_config.risk_control.min_risk_reward_ratio",
		"ai_config.prompt_sections.role_definition",
	} {
		if !set[p] {
			t.Errorf("%s is written by MarshalJSON but not enumerated", p)
		}
	}
}

// A SAVE KEEPS THE KEY SET. UnmarshalJSON → MarshalJSON (the load-then-save
// round trip every config edit makes) writes exactly the keys it read, and a
// LEGACY flat config (risk_control etc. at the root, which UnmarshalJSON still
// accepts) re-emits under ai_config — inside the enumeration, not beside it.
func TestSchemaKeySetSurvivesTheUnmarshalMarshalRoundTrip(t *testing.T) {
	set := enumeratedSet(t)
	for _, st := range schemaStrategyTypes {
		first, err := json.Marshal(populatedStrategyConfig(st))
		if err != nil {
			t.Fatalf("marshal %s: %v", st, err)
		}
		var back StrategyConfig
		if err := json.Unmarshal(first, &back); err != nil {
			t.Fatalf("unmarshal %s: %v", st, err)
		}
		k1, _ := jsonKeyPaths(first)
		k2 := marshalKeyPaths(t, back)
		if !reflect.DeepEqual(k1, k2) {
			t.Errorf("%s: round trip changed the key set\n  before %v\n  after  %v", st, k1, k2)
		}
	}

	// Legacy flat: lift ai_config's children to the root.
	nested, err := json.Marshal(populatedStrategyConfig("ai_trading"))
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]any
	if err := json.Unmarshal(nested, &doc); err != nil {
		t.Fatal(err)
	}
	ai, ok := doc["ai_config"].(map[string]any)
	if !ok {
		t.Fatalf("populated ai_trading config has no ai_config object: %s", nested)
	}
	delete(doc, "ai_config")
	for k, v := range ai {
		doc[k] = v
	}
	flat, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	var legacy StrategyConfig
	if err := json.Unmarshal(flat, &legacy); err != nil {
		t.Fatalf("legacy flat unmarshal: %v", err)
	}
	var outside []string
	for _, k := range marshalKeyPaths(t, legacy) {
		if !set[k] {
			outside = append(outside, k)
		}
	}
	if len(outside) > 0 {
		t.Errorf("a legacy flat config re-saves keys the walk does not enumerate:\n  %s", strings.Join(outside, "\n  "))
	}
	kNested, _ := jsonKeyPaths(nested)
	if kLegacy := marshalKeyPaths(t, legacy); !reflect.DeepEqual(kNested, kLegacy) {
		t.Errorf("legacy flat re-save differs from the nested key set\n  nested %v\n  legacy %v", kNested, kLegacy)
	}
}

// external_data_sources' children are INEFFECTIVE, by an EXACT entry. Through
// the leaf fallback they would borrow the unrelated live "name" / "type" rows
// and the url/method/… rows that cite FetchExternalData, which no production
// code calls.
func TestExternalDataSourcesChildrenClassifyIneffective(t *testing.T) {
	set := enumeratedSet(t)
	for _, leaf := range []string{"name", "type", "url", "method", "headers", "data_path", "refresh_secs"} {
		p := "ai_config.indicators.external_data_sources." + leaf
		if !set[p] {
			t.Errorf("%s is not enumerated — the walk no longer reaches external_data_sources", p)
			continue
		}
		e, ok, how := lookupKnob(p)
		if !ok {
			t.Errorf("%s: unclassified", p)
			continue
		}
		if how != knobExact {
			t.Errorf("%s: classified by match kind %d, want an EXACT entry (the leaf %q means something else elsewhere)", p, how, leaf)
		}
		if e.Status != KnobIneffective {
			t.Errorf("%s: status %q, want ineffective — its parent has zero engine consumers", p, e.Status)
		}
		if e.Note == "" {
			t.Errorf("%s: a non-live knob must carry the reason", p)
		}
	}
}

// ai_config. IS STRIPPED BEFORE THE LEAF FALLBACK. A dotted display key
// (risk_control.max_margin_usage) must be able to become an exact table key
// for the real path ai_config.risk_control.max_margin_usage.
func TestLookupKnobStripsTheAIConfigEnvelopeBeforeTheLeaf(t *testing.T) {
	const display = "risk_control.max_margin_usage"
	reg := map[string]KnobEntry{
		display:            {Path: display, Status: KnobIneffective, Note: "dotted display key"},
		"max_margin_usage": {Path: "max_margin_usage", Status: KnobLive, Consumers: []string{"leaf row"}},
	}
	e, ok, how := lookupKnobIn(reg, "ai_config."+display)
	if !ok || how != knobExactStripped || e.Path != display {
		t.Fatalf("lookup ai_config.%s → %+v ok=%v how=%d; want the stripped exact entry, not the leaf", display, e, ok, how)
	}
	// A path outside the envelope is never stripped into a match.
	if _, _, how := lookupKnobIn(reg, "grid_config."+display); how == knobExactStripped {
		t.Fatalf("grid_config.%s matched by stripping — only ai_config. is the envelope", display)
	}
	// custom_prompt is the one enumerated compatibility path the table already
	// holds by its stripped key.
	if _, ok, how := lookupKnob("ai_config.custom_prompt"); !ok || how != knobExactStripped {
		t.Fatalf("ai_config.custom_prompt: ok=%v how=%d, want the stripped exact match", ok, how)
	}
}

// HOW each enumerated path is classified, counted — the report quotes these.
// The leaf fallback is still how most paths classify; that is recorded, not
// hidden, and every enumerated path must classify SOME way.
func TestSchemaWalkClassificationByMatchKind(t *testing.T) {
	counts := map[knobMatch]int{}
	for _, p := range EnumerateSchemaKnobs() {
		_, _, how := lookupKnob(p)
		counts[how]++
	}
	if counts[knobNoMatch] != 0 {
		t.Errorf("%d enumerated path(s) unclassified", counts[knobNoMatch])
	}
	// A registry key is REACHED when some enumerated path hits it exactly,
	// stripped, by its leaf, or names it as a container segment (klines,
	// picture_htf, external_data_sources). The rest are orphans — rows keyed
	// by a name the StrategyConfig schema does not carry. Counted and listed.
	reached := map[string]bool{}
	for _, p := range EnumerateSchemaKnobs() {
		reached[p] = true
		reached[strings.TrimPrefix(p, knobAIConfigPrefix)] = true
		for _, seg := range strings.Split(p, ".") {
			reached[seg] = true
		}
	}
	var orphans []string
	for k := range knobRegistry {
		if !reached[k] {
			orphans = append(orphans, k)
		}
	}
	sort.Strings(orphans)
	t.Logf("schema=%d exact=%d stripped-exact=%d leaf-fallback=%d unclassified=%d · orphan registry keys (no enumerated path reaches them): %d %v",
		len(EnumerateSchemaKnobs()), counts[knobExact], counts[knobExactStripped], counts[knobLeafFallback], counts[knobNoMatch], len(orphans), orphans)
}
