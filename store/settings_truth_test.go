package store

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ── W1 SETTINGS TRUTH — the resolvers, the conversion report, the record ─────

func envMap(kv map[string]string) func(string) string {
	return func(k string) string { return kv[k] }
}

// The breaker's whole precedence in one table: saved (incl. 0) → env → 8.
func TestResolveBreakerHaltPrecedence(t *testing.T) {
	withHalt := func(v *int) *StrategyConfig {
		c := &StrategyConfig{}
		c.RiskControl.ConsecutiveLossHalt = v
		return c
	}
	cases := []struct {
		name   string
		cfg    *StrategyConfig
		env    map[string]string
		want   int
		origin string
		srcHas string
	}{
		{"nil config, no env → shipped 8", nil, nil, 8, "[I]", SourceShippedDefault},
		{"absent, no env → shipped 8 (inherit is ON)", withHalt(nil), nil, 8, "[I]", SourceShippedDefault},
		{"saved 3", withHalt(IntPtr(3)), nil, 3, "[O]", SourceSaved},
		{"saved 0 is OFF", withHalt(IntPtr(0)), nil, 0, "[O]", SourceSaved},
		{"absent + env 3 → 3 [E]", withHalt(nil), map[string]string{"BREAKER_HALT_N": "3"}, 3, "[E]", "env BREAKER_HALT_N"},
		{"absent + env 0 → off [E]", withHalt(nil), map[string]string{"BREAKER_HALT_N": "0"}, 0, "[E]", "env BREAKER_HALT_N"},
		{"saved 5 beats env 3", withHalt(IntPtr(5)), map[string]string{"BREAKER_HALT_N": "3"}, 5, "[O]", SourceSaved},
		{"saved 0 beats env 3", withHalt(IntPtr(0)), map[string]string{"BREAKER_HALT_N": "3"}, 0, "[O]", SourceSaved},
		{"absent + env invalid → 8, and the source SAYS invalid", withHalt(nil), map[string]string{"BREAKER_HALT_N": "abc"}, 8, "[I]", "invalid"},
		{"absent + env negative → 8, invalid", withHalt(nil), map[string]string{"BREAKER_HALT_N": "-2"}, 8, "[I]", "invalid"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			n, src := ResolveBreakerHaltEnv(c.cfg, envMap(c.env))
			if n != c.want {
				t.Fatalf("N=%d want %d (source %q)", n, c.want, src)
			}
			if got := OriginLetter(src); got != c.origin {
				t.Fatalf("origin %s want %s (source %q)", got, c.origin, src)
			}
			if !strings.Contains(src, c.srcHas) {
				t.Fatalf("source %q does not name %q", src, c.srcHas)
			}
		})
	}
}

// The replan cap: session → strategy → 2, and 0 is honoured at BOTH levels.
func TestResolveReplanCapPrecedence(t *testing.T) {
	cases := []struct {
		name    string
		dp      *DayPlanConfig
		session string
		want    int
		origin  string
	}{
		{"nil block → 2 [I]", nil, "NY", 2, "[I]"},
		{"absent strategy → 2 [I]", &DayPlanConfig{}, "NY", 2, "[I]"},
		{"strategy 4", &DayPlanConfig{ReplanCap: IntPtr(4)}, "NY", 4, "[O]"},
		{"strategy 0 is 0 (was read as 2)", &DayPlanConfig{ReplanCap: IntPtr(0)}, "NY", 0, "[O]"},
		{"strategy 0 at the strategy level", &DayPlanConfig{ReplanCap: IntPtr(0)}, "", 0, "[O]"},
		{"session 3 beats strategy 0", &DayPlanConfig{ReplanCap: IntPtr(0), Sessions: []DayPlanSessionOverride{{Session: "NY", ReplanCap: IntPtr(3)}}}, "NY", 3, "[O]"},
		{"session 0 beats strategy 4", &DayPlanConfig{ReplanCap: IntPtr(4), Sessions: []DayPlanSessionOverride{{Session: "ASIA", ReplanCap: IntPtr(0)}}}, "ASIA", 0, "[O]"},
		{"session case-folds (class 28)", &DayPlanConfig{Sessions: []DayPlanSessionOverride{{Session: "london", ReplanCap: IntPtr(1)}}}, "LONDON", 1, "[O]"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			n, src := ResolveReplanCap(c.dp, c.session)
			if n != c.want || OriginLetter(src) != c.origin {
				t.Fatalf("got %d%s (%q), want %d%s", n, OriginLetter(src), src, c.want, c.origin)
			}
			// ReplanCapFor is the production entry point every gate calls.
			if got := c.dp.ReplanCapFor(c.session); got != c.want {
				t.Fatalf("ReplanCapFor=%d, resolver=%d — two rules", got, c.want)
			}
		})
	}
}

func TestOriginLetterIsOneMapping(t *testing.T) {
	for src, want := range map[string]string{
		SourceSaved:                     "[O]",
		SourceStrategyValue:             "[O]",
		SourceSessionOverride:           "[O]",
		"env BREAKER_HALT_N":            "[E]",
		SourceShippedDefault:            "[I]",
		SourceSchemaDefault:             "[I]",
		SourceShippedDefault + " (env)": "[I]",
		"something new":                 "[?]",
	} {
		if got := OriginLetter(src); got != want {
			t.Errorf("OriginLetter(%q)=%s want %s", src, got, want)
		}
	}
}

// ── THE REPORT ──────────────────────────────────────────────────────────────

type truthFixtureRow struct {
	ID        string          `json:"id"`
	Bound     int             `json:"bound"`
	Confirmed []string        `json:"confirmed"`
	Config    json.RawMessage `json:"config"`
}

func loadTruthFixture(t *testing.T) []truthFixtureRow {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", "settings_truth_rows.json"))
	if err != nil {
		t.Fatal(err)
	}
	var f struct {
		Rows []truthFixtureRow `json:"rows"`
	}
	if err := json.Unmarshal(b, &f); err != nil {
		t.Fatal(err)
	}
	return f.Rows
}

func truthInputs(rows []truthFixtureRow) []SettingsTruthInput {
	in := make([]SettingsTruthInput, 0, len(rows))
	for _, r := range rows {
		conf := map[string]bool{}
		for _, k := range r.Confirmed {
			conf[k] = true
		}
		in = append(in, SettingsTruthInput{ID: r.ID, Config: string(r.Config), Bound: r.Bound, Confirmed: conf})
	}
	return in
}

const truthGolden = "testdata/settings_truth_report.golden"

func checkTruthGolden(t *testing.T, got string) {
	t.Helper()
	if os.Getenv("UPDATE_GOLDEN") == "1" {
		if err := os.WriteFile(truthGolden, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(truthGolden)
	if err != nil {
		t.Fatalf("read golden (capture with UPDATE_GOLDEN=1): %v", err)
	}
	if got != string(want) {
		t.Fatalf("settings-truth report drifted from the golden.\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}

// The migration report IS a fixture: the nine research-copy shapes plus the
// synthetic zeros, env unset. Pinned byte-for-byte.
func TestSettingsTruthReportGolden(t *testing.T) {
	res := SettingsTruthReport(truthInputs(loadTruthFixture(t)), envMap(nil))
	checkTruthGolden(t, strings.Join(res.Lines(), "\n")+"\n")
}

// The expected live result: the nine research-copy rows change NOTHING, in
// every environment the bot could boot with. Sample ids: the nine below.
func TestSettingsTruthResearchRowsChangeNothing(t *testing.T) {
	var research []SettingsTruthInput
	for _, in := range truthInputs(loadTruthFixture(t)) {
		if !strings.HasPrefix(in.ID, "syn-") {
			research = append(research, in)
		}
	}
	if len(research) != 9 {
		t.Fatalf("fixture carries %d research rows, want the 9 of the 09-16 copy", len(research))
	}
	for _, env := range []map[string]string{nil, {"BREAKER_HALT_N": "3"}, {"BREAKER_HALT_N": "0"}, {"BREAKER_HALT_N": "junk"}} {
		res := SettingsTruthReport(research, envMap(env))
		if res.ChangedRows != 0 || res.RefusedRows != 0 {
			t.Fatalf("env %v: %d research rows change and %d are refused — the conversion must change none:\n%s",
				env, res.ChangedRows, res.RefusedRows, strings.Join(res.Lines(), "\n"))
		}
	}
}

// The refusal: a stored 0 that no Studio save confirmed refuses its bound
// trader(s); a confirmed one loads; BREAKER_HALT_N=0 makes the breaker-0 rows
// UNCHANGED (off then, off now) so nothing is refused for the breaker.
func TestSettingsTruthRefusesOnlyUnconfirmedChanges(t *testing.T) {
	byID := func(res SettingsTruthResult) map[string]SettingsTruthRow {
		m := map[string]SettingsTruthRow{}
		for _, r := range res.Rows {
			m[r.ID] = r
		}
		return m
	}
	rows := byID(SettingsTruthReport(truthInputs(loadTruthFixture(t)), envMap(nil)))
	for id, wantRefuse := range map[string]bool{
		"syn-breaker-0-flat":                   true,
		"syn-breaker-0-nested":                 true,
		"syn-breaker-0-nested-confirmed":       false,
		"syn-breaker-null":                     false,
		"syn-replan-0-strategy":                true,
		"syn-replan-0-session-only":            false, // a session 0 always meant 0
		"a5b7662e-7bf7-49bb-9f09-7efa48f95ac8": false,
	} {
		r, ok := rows[id]
		if !ok {
			t.Fatalf("row %s missing", id)
		}
		if (r.Refuse != "") != wantRefuse {
			t.Errorf("%s: refuse=%q, want refuse=%v", id, r.Refuse, wantRefuse)
		}
	}
	if r := rows["syn-breaker-0-nested"]; !strings.Contains(r.Refuse, "meant 'inherit' then (breaker 8)") || r.Breaker.After != "off[O]" {
		t.Errorf("the refusal must name what 0 meant then and now: %q / after=%s", r.Refuse, r.Breaker.After)
	}
	withEnvOff := byID(SettingsTruthReport(truthInputs(loadTruthFixture(t)), envMap(map[string]string{"BREAKER_HALT_N": "0"})))
	if r := withEnvOff["syn-breaker-0-nested"]; r.Breaker.Changed || r.Refuse != "" {
		t.Errorf("BREAKER_HALT_N=0: a stored 0 was off before and is off now — nothing changed, nothing refused; got %+v", r)
	}
}

// ── THE STORE SEAT — the same rows through the real tables ──────────────────

func seedTruthStore(t *testing.T) *Store {
	t.Helper()
	st, err := New(filepath.Join(t.TempDir(), "truth.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	g := st.GormDB()
	// store.New seeds its own default strategy rows; the report must speak
	// for exactly the fixture's rows, so start from an empty table.
	if err := g.Exec(`DELETE FROM strategies`).Error; err != nil {
		t.Fatal(err)
	}
	for _, r := range loadTruthFixture(t) {
		if err := g.Exec(`INSERT INTO strategies (id, user_id, name, config) VALUES (?,?,?,?)`,
			r.ID, "u1", "s", string(r.Config)).Error; err != nil {
			t.Fatal(err)
		}
		for b := 0; b < r.Bound; b++ {
			if err := g.Exec(`INSERT INTO traders (id, user_id, name, strategy_id, ai_model_id, exchange_id, initial_balance) VALUES (?,?,?,?,?,?,?)`,
				fmt.Sprintf("t-%s-%d", r.ID, b), "u1", "trader", r.ID, "m1", "e1", 50000.0).Error; err != nil {
				t.Fatal(err)
			}
		}
		if len(r.Confirmed) > 0 {
			// The confirmation is written by the record the Studio's save
			// calls — here from the row's own parsed config.
			var cfg StrategyConfig
			if err := json.Unmarshal(r.Config, &cfg); err != nil {
				t.Fatal(err)
			}
			if err := st.Strategy().RecordExplicitZeros(r.ID, &cfg); err != nil {
				t.Fatal(err)
			}
		}
	}
	return st
}

// The boot seat reads the tables and prints EXACTLY the pure report's golden.
func TestSettingsTruthBootReportReadsTheStore(t *testing.T) {
	st := seedTruthStore(t)
	res, err := st.Strategy().SettingsTruthBootReport(envMap(nil))
	if err != nil {
		t.Fatal(err)
	}
	checkTruthGolden(t, strings.Join(res.Lines(), "\n")+"\n")
}

// A Studio save REPLACES the record: turning the breaker back to inherit
// drops it, so a later raw 0 is not covered by a stale confirmation.
func TestRecordExplicitZerosIsReplacedPerSave(t *testing.T) {
	st := seedTruthStore(t)
	s := st.Strategy()
	id := "syn-breaker-0-nested-confirmed"
	if !s.ExplicitZerosConfirmed(id)[KnobBreaker] {
		t.Fatal("seeded confirmation missing")
	}
	var inherit StrategyConfig // breaker nil = inherit
	if err := s.RecordExplicitZeros(id, &inherit); err != nil {
		t.Fatal(err)
	}
	if s.ExplicitZerosConfirmed(id)[KnobBreaker] {
		t.Fatal("a save that holds no explicit 0 must drop the confirmation")
	}
	both := StrategyConfig{DayPlan: &DayPlanConfig{ReplanCap: IntPtr(0)}}
	both.RiskControl.ConsecutiveLossHalt = IntPtr(0)
	if err := s.RecordExplicitZeros(id, &both); err != nil {
		t.Fatal(err)
	}
	got := s.ExplicitZerosConfirmed(id)
	if !got[KnobBreaker] || !got[KnobReplanStrategy] {
		t.Fatalf("both explicit zeros must be recorded, got %v", got)
	}
}

// Duplicate copies the bytes, so it carries the source's confirmation.
func TestDuplicateCarriesTheExplicitZeroRecord(t *testing.T) {
	st := seedTruthStore(t)
	if err := st.Strategy().Duplicate("u1", "syn-breaker-0-nested-confirmed", "dup-1", "copy"); err != nil {
		t.Fatal(err)
	}
	dup, err := st.Strategy().Get("u1", "dup-1")
	if err != nil {
		t.Fatal(err)
	}
	if r := st.Strategy().SettingsTruthRefusal(dup, envMap(nil)); r != "" {
		t.Fatalf("a copy of a confirmed OFF breaker must load, got refusal %q", r)
	}
	if err := st.Strategy().Duplicate("u1", "syn-breaker-0-nested", "dup-2", "copy"); err != nil {
		t.Fatal(err)
	}
	dup2, _ := st.Strategy().Get("u1", "dup-2")
	if r := st.Strategy().SettingsTruthRefusal(dup2, envMap(nil)); r == "" {
		t.Fatal("a copy of an UNCONFIRMED legacy 0 must stay refused")
	}
}
