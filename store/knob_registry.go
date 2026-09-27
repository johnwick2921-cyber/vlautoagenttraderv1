package store

import (
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"sync"
)

// ── KNOB REGISTRY (settings integrity, 2026-09-03) ───────────────────────────
//
// The Studio audit (2026-09-03-studio-audit.md) found FIFTEEN saved settings
// that cannot take effect: two unreachable clock fields, four parse-only flags,
// a prompt-text-only "limit", a session field with no consumer, two env vars
// with no consumer, a hardcoded crypto override, a dropped per-session
// plan_mode, an env var replacing the Studio R:R floor, an always-on veto, a
// clamped size, a suspended toggle whose wire is untouched, and an
// external-data list with zero engine consumers.
//
// The registry exists so that class cannot recur SILENTLY. The schema is
// enumerated by REFLECTION — it can never drift from the struct — and every
// enumerated field must carry a status. Adding a field and leaving it
// unclassified fails the build. That is the point, not a side effect.
type KnobStatus string

const (
	KnobLive        KnobStatus = "live"         // a consumer reads it and behaviour changes
	KnobSuspended   KnobStatus = "suspended"    // a standing ruling disables it; value preserved
	KnobAdvisory    KnobStatus = "advisory"     // feeds prompt text only — never a gate
	KnobDisplayOnly KnobStatus = "display-only" // rendered, never read by the engine

	// TWO DIFFERENT TESTS, TWO DIFFERENT LABELS (owner ruling 2026-09-03). The
	// 09-03 sweep proved neither subsumes the other, so they must not share a
	// word.
	//
	// KnobIneffective — a consumer DOES read it and the read does not change
	// behaviour. Seven of these: max_margin_usage feeds prompt text only,
	// plan_mode was dropped at the arm seam until R2. The reason is REQUIRED.
	//
	// KnobCandidate — no consumer found by a FIELD grep, which is NOT the same
	// as having no reader: a method-based consumer would not appear. Never
	// "dead", never removed, and rendered "no known reader — pending
	// verification" until a METHOD-level grep is run and its command quoted.
	KnobIneffective KnobStatus = "ineffective"
	KnobCandidate   KnobStatus = "candidate-unverified"
	KnobInfra       KnobStatus = "infra" // ports, paths, keys — not a trading knob

	// KnobFolded (W-KNOB-PRUNE, owner ruling 2026-09-18) — the Studio control is
	// gone and the code path keeps the shipped default as a CONSTANT; the stored
	// JSON field stays readable, a saved non-default is honoured at read and
	// logged once at trader load ("⚙ folded knob <name>=<value> honoured from
	// stored config"). Not dead, not live-with-a-control: a third thing.
	KnobFolded KnobStatus = "folded"
)

// KnobEntry is one schema field's classification.
type KnobEntry struct {
	Path      string     // dotted schema path, e.g. "risk_control.min_risk_reward_ratio"
	Status    KnobStatus //
	Consumers []string   // file:line — REQUIRED when live, or the entry is a claim without a reader
	DualLevel bool       // has a per-session override
	Clamp     string     // "" = none; else what the clamp does to the saved value
	Note      string     // REQUIRED when not live: WHY
}

// EnumerateSchemaKnobs returns every schema leaf as a dotted path — the key
// paths the PRODUCTION StrategyConfig.MarshalJSON actually writes, not the Go
// struct tags. Order is sorted; callers get a copy.
//
// W1 (f), 2026-09-23. The previous walk read struct tags and skipped json:"-".
// StrategyConfig carries five json:"-" compatibility fields (CoinSource,
// Indicators, CustomPrompt, RiskControl, PromptSections) that MarshalJSON
// re-emits under ai_config — so the drift counter skipped exactly the fields a
// custom marshaller re-emits: all of ai_config.risk_control.*, indicators.*,
// coin_source.*, prompt_sections.* and custom_prompt were invisible to schema=
// and to UNCLASSIFIED. A tag walk cannot see what a MarshalJSON method writes,
// and a hand map of "where the '-' fields really go" is a second copy that
// drifts (canon 53: exercise the production call site). So the enumeration
// goes THROUGH the marshaller: populate every field by reflection, json.Marshal
// it once per strategy type (ai_trading emits ai_config, grid_trading emits
// grid_config — MarshalJSON writes one or the other, never both), decode, and
// walk the union of the key paths.
func EnumerateSchemaKnobs() []string {
	schemaKnobsOnce.Do(func() {
		schemaKnobs, schemaKnobsErr = enumerateSchemaKnobsViaMarshal()
	})
	out := make([]string, len(schemaKnobs))
	copy(out, schemaKnobs)
	return out
}

// SchemaEnumerationErr reports why EnumerateSchemaKnobs came back empty, if it
// did. A marshal failure must read as "could not count", never as schema=0.
func SchemaEnumerationErr() error {
	EnumerateSchemaKnobs()
	return schemaKnobsErr
}

var (
	schemaKnobsOnce sync.Once
	schemaKnobs     []string
	schemaKnobsErr  error
)

// schemaSentinelKey is the one key every populated map carries. A decoded
// object whose ONLY key is the sentinel is a map — a leaf, exactly as the
// tag walk treated maps — not a struct to descend into.
const schemaSentinelKey = "__knob_schema_map_key__"

// schemaPopulateDepth bounds the populate recursion so a self-referential type
// can never loop; StrategyConfig's deepest leaf sits well inside it.
const schemaPopulateDepth = 12

// schemaStrategyTypes are the values MarshalJSON branches on (strategy.go).
var schemaStrategyTypes = []string{"ai_trading", "grid_trading"}

func enumerateSchemaKnobsViaMarshal() ([]string, error) {
	paths := map[string]bool{}
	for _, st := range schemaStrategyTypes {
		keys, err := marshalledSchemaPaths(st)
		if err != nil {
			return nil, err
		}
		for _, k := range keys {
			paths[k] = true
		}
	}
	out := make([]string, 0, len(paths))
	for p := range paths {
		out = append(out, p)
	}
	sort.Strings(out)
	return out, nil
}

// populatedStrategyConfig returns a StrategyConfig with every exported field
// set to a non-zero value (so no omitempty drops it) and the given strategy
// type. The tests use it to exercise the same marshal the enumeration does.
func populatedStrategyConfig(strategyType string) StrategyConfig {
	var cfg StrategyConfig
	populateForSchema(reflect.ValueOf(&cfg).Elem(), 0)
	cfg.StrategyType = strategyType
	return cfg
}

// marshalledSchemaPaths marshals a fully populated config through the
// production MarshalJSON and returns the dotted key paths it wrote.
func marshalledSchemaPaths(strategyType string) ([]string, error) {
	cfg := populatedStrategyConfig(strategyType)
	b, err := json.Marshal(cfg)
	if err != nil {
		return nil, fmt.Errorf("marshal populated %s config: %w", strategyType, err)
	}
	return jsonKeyPaths(b)
}

// jsonKeyPaths decodes marshalled JSON and returns every leaf key path.
// Objects descend; arrays read element 0 (the path carries no index, as the
// tag walk's slice strip did); an object holding only the sentinel is a map
// and ends the path; anything else is a leaf.
func jsonKeyPaths(b []byte) ([]string, error) {
	var v any
	if err := json.Unmarshal(b, &v); err != nil {
		return nil, fmt.Errorf("decode marshalled config: %w", err)
	}
	seen := map[string]bool{}
	var walk func(v any, prefix string)
	walk = func(v any, prefix string) {
		switch t := v.(type) {
		case map[string]any:
			if _, isMap := t[schemaSentinelKey]; (isMap && len(t) == 1) || len(t) == 0 {
				if prefix != "" {
					seen[prefix] = true
				}
				return
			}
			for k, child := range t {
				p := k
				if prefix != "" {
					p = prefix + "." + k
				}
				walk(child, p)
			}
		case []any:
			if len(t) == 0 {
				if prefix != "" {
					seen[prefix] = true
				}
				return
			}
			walk(t[0], prefix)
		default:
			if prefix != "" {
				seen[prefix] = true
			}
		}
	}
	walk(v, "")
	out := make([]string, 0, len(seen))
	for p := range seen {
		out = append(out, p)
	}
	sort.Strings(out)
	return out, nil
}

// populateForSchema sets every settable field under v to a non-zero value:
// pointers allocated, slices of length 1, maps with the sentinel key, bools
// true, numbers 1, strings "x". Unexported fields are left alone (time.Time
// marshals through its own method and is a leaf either way).
func populateForSchema(v reflect.Value, depth int) {
	if depth > schemaPopulateDepth || !v.CanSet() {
		return
	}
	switch v.Kind() {
	case reflect.Ptr:
		p := reflect.New(v.Type().Elem())
		populateForSchema(p.Elem(), depth+1)
		v.Set(p)
	case reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			if v.Type().Field(i).PkgPath != "" {
				continue // unexported
			}
			populateForSchema(v.Field(i), depth+1)
		}
	case reflect.Slice:
		if v.Type().Elem().Kind() == reflect.Uint8 {
			// []byte marshals as a base64 string (a leaf); json.RawMessage
			// must hold valid JSON.
			v.SetBytes([]byte(`"x"`))
			return
		}
		s := reflect.MakeSlice(v.Type(), 1, 1)
		populateForSchema(s.Index(0), depth+1)
		v.Set(s)
	case reflect.Array:
		for i := 0; i < v.Len(); i++ {
			populateForSchema(v.Index(i), depth+1)
		}
	case reflect.Map:
		m := reflect.MakeMapWithSize(v.Type(), 1)
		k := reflect.New(v.Type().Key()).Elem()
		if k.Kind() == reflect.String {
			k.SetString(schemaSentinelKey)
		} else {
			populateForSchema(k, depth+1)
		}
		val := reflect.New(v.Type().Elem()).Elem()
		populateForSchema(val, depth+1)
		m.SetMapIndex(k, val)
		v.Set(m)
	case reflect.Interface:
		if v.NumMethod() == 0 {
			v.Set(reflect.ValueOf("x"))
		}
	case reflect.Bool:
		v.SetBool(true)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		v.SetInt(1)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		v.SetUint(1)
	case reflect.Float32, reflect.Float64:
		v.SetFloat(1)
	case reflect.String:
		v.SetString("x")
	}
}

// knobAIConfigPrefix is the envelope MarshalJSON nests the json:"-"
// compatibility fields under. LookupKnob strips it for an exact match so a
// dotted display key (risk_control.min_risk_reward_ratio) can be a table key.
const knobAIConfigPrefix = "ai_config."

// LookupKnob returns the registry entry for a schema path.
//
// Order: exact path → exact path with the ai_config. envelope stripped → the
// json LEAF name. The leaf fallback stays: the table is still keyed mostly by
// leaf, and removing it would leave ~150 enumerated paths UNCLASSIFIED — a
// separate wave. Documented rather than hidden: a leaf name can collide across
// structs, and where it does the leaf's classification wins, which is why a
// leaf that means something different under one parent gets an EXACT entry
// (ai_config.indicators.external_data_sources.* — see the table).
func LookupKnob(path string) (KnobEntry, bool) {
	e, ok, _ := lookupKnob(path)
	return e, ok
}

// knobMatch says HOW a path was classified, so the counts can be told apart.
type knobMatch int

const (
	knobNoMatch knobMatch = iota
	knobExact
	knobExactStripped
	knobLeafFallback
)

func lookupKnob(path string) (KnobEntry, bool, knobMatch) {
	return lookupKnobIn(knobRegistry, path)
}

// lookupKnobIn is LookupKnob's body over a given table (the tests pass a
// local one rather than mutating the shared registry).
func lookupKnobIn(reg map[string]KnobEntry, path string) (KnobEntry, bool, knobMatch) {
	if e, ok := reg[path]; ok {
		return e, true, knobExact
	}
	if strings.HasPrefix(path, knobAIConfigPrefix) {
		if e, ok := reg[strings.TrimPrefix(path, knobAIConfigPrefix)]; ok {
			return e, true, knobExactStripped
		}
	}
	if i := strings.LastIndex(path, "."); i >= 0 {
		if e, ok := reg[path[i+1:]]; ok {
			return e, true, knobLeafFallback
		}
	}
	return KnobEntry{}, false, knobNoMatch
}

// KnobSummary is what the boot line reports — counted, never typed.
type KnobSummary struct {
	Total, Live, Suspended, Advisory, DisplayOnly, Ineffective, Candidate, Infra, Folded int

	// EnvShadows is nil until something COUNTS the env vars that shadow a
	// stored knob. Nothing does yet (W1 (f), 2026-09-23: zero writers), so the
	// boot line prints "env-shadows=n/a (not counted)" and the API omits the
	// key — a counter with no writer printed 0, a fabricated value (L7).
	EnvShadows     *int
	EnvShadowPaths []string
}

// KnobStatusSummary counts the registry by status.
// AllKnobs returns every classified entry, ordered by path so the payload a
// client renders is stable request to request. Callers get a copy of the slice,
// never the registry map.
func AllKnobs() []KnobEntry {
	out := make([]KnobEntry, 0, len(knobRegistry))
	for _, e := range knobRegistry {
		out = append(out, e)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out
}

func KnobStatusSummary() KnobSummary {
	s := KnobSummary{}
	for _, e := range knobRegistry {
		s.Total++
		switch e.Status {
		case KnobLive:
			s.Live++
		case KnobSuspended:
			s.Suspended++
		case KnobAdvisory:
			s.Advisory++
		case KnobDisplayOnly:
			s.DisplayOnly++
		case KnobIneffective:
			s.Ineffective++
		case KnobCandidate:
			s.Candidate++
		case KnobInfra:
			s.Infra++
		case KnobFolded:
			s.Folded++
		}
	}
	return s
}

// KnobRegistryBootLine reports the registry — every field READ from it.
//
// schema= is the number of key paths the production MarshalJSON writes (see
// EnumerateSchemaKnobs) — since W1 (f) that includes the ai_config.* paths of
// the json:"-" compatibility structs. A value the process could not compute
// prints n/a with the reason, never a number (L7).
func KnobRegistryBootLine() string {
	s := KnobStatusSummary()
	paths := EnumerateSchemaKnobs()
	schema := strconv.Itoa(len(paths))
	if err := SchemaEnumerationErr(); err != nil {
		schema = "n/a (enumeration failed: " + err.Error() + ")"
	}
	unclassified := 0
	for _, p := range paths {
		if _, ok := LookupKnob(p); !ok {
			unclassified++
		}
	}
	warn := ""
	if unclassified > 0 {
		warn = fmt.Sprintf(" · ⚠ %d UNCLASSIFIED", unclassified)
	}
	envShadows := "n/a (not counted)"
	if s.EnvShadows != nil {
		envShadows = strconv.Itoa(*s.EnvShadows)
	}
	return fmt.Sprintf("settings: schema=%s classified=%d live=%d ineffective=%d candidate-unverified=%d suspended=%d advisory=%d display-only=%d infra=%d folded=%d · env-shadows=%s%s",
		schema, s.Total, s.Live, s.Ineffective, s.Candidate, s.Suspended, s.Advisory, s.DisplayOnly, s.Infra, s.Folded, envShadows, warn)
}

// AuditDeadKnobs2026_09_03 is the audit's fifteen, by schema path, so the
// registry can be checked against the finding that produced it.
//
// W-KNOB-PRUNE (2026-09-18): day_plan.last_entry_ct and day_plan.eod_flat_ct
// were DELETED from the struct (unreachable since the P2 session-scope
// redesign; nothing read them) — they are no longer schema fields, so they
// leave this list rather than being classified.
//
// W1 (f) (2026-09-23): the paths are the REAL marshalled paths. The list read
// "risk_control.*" and "indicator_config.external_data_sources" — the first is
// shorthand for what MarshalJSON writes under ai_config, the second exists
// nowhere (the struct field is Indicators, tagged "indicators"). Neither was
// ever in the schema walk, which skipped json:"-"; the leaf fallback in
// LookupKnob is the only reason the old spellings resolved at all.
// TestAuditDeadKnobsAreInTheSchemaWalk pins every entry to the enumeration.
var AuditDeadKnobs2026_09_03 = []string{
	"ai_config.risk_control.max_contracts_enabled",
	"ai_config.risk_control.notional_cap_enabled",
	"ai_config.risk_control.max_margin_usage",
	"ai_config.indicators.external_data_sources",
}

// UILabel is what the Studio renders beside a field. The two failure modes get
// DIFFERENT words, because conflating them is how a knob nobody has checked
// gets deleted as "dead".
func (e KnobEntry) UILabel() string {
	switch e.Status {
	case KnobIneffective:
		return "read; does not take effect (" + e.Note + ")"
	case KnobCandidate:
		return "no known reader — pending verification"
	case KnobSuspended:
		return "suspended — " + e.Note
	case KnobAdvisory:
		return "advisory to the model — not a gate"
	case KnobDisplayOnly:
		return "display only"
	case KnobInfra:
		return "infrastructure — not a trading knob"
	case KnobFolded:
		return "folded — no control; stored value honoured (" + e.Note + ")"
	default:
		return "live"
	}
}
