package trader

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"nofx/store"
)

// ── W1 SETTINGS TRUTH — the breaker and the replan cap at their production
// call sites (canon 53): the decision path (consecutiveLossHaltedAt), the arm
// path (sessionRiskGateAt), the boot line on a real store, and the replan
// readers the gates and the prompt use.

// w1BreakerTrader is a trader with a real store and N losing closes seeded
// in THIS CME session-day. No day plan: the no-trade band is open, so the
// arm-path verdict is the breaker's alone.
func w1BreakerTrader(t *testing.T, id string, halt *int, losses int, now time.Time) *AutoTrader {
	t.Helper()
	st, err := store.New(filepath.Join(t.TempDir(), "w1.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	cfg := store.StrategyConfig{}
	cfg.RiskControl.ConsecutiveLossHalt = halt
	at := &AutoTrader{id: id, store: st}
	at.config.StrategyConfig = &cfg
	for i := 0; i < losses; i++ {
		exit := now.Add(-time.Duration(losses-i) * time.Minute)
		pnl := -12.5
		p := &store.TraderPosition{TraderID: id, Account: "Sim101", Symbol: "MNQ", Side: "LONG",
			Quantity: 1, EntryPrice: 100, ExitPrice: 93.75, RealizedPnL: pnl, PnlCorrected: &pnl,
			Status: "CLOSED", CloseReason: "sync", EntryTime: exit.Add(-30 * time.Second).UnixMilli(),
			ExitTime: exit.UnixMilli(), CreatedAt: exit.UnixMilli(), UpdatedAt: exit.UnixMilli()}
		if err := st.GormDB().Create(p).Error; err != nil {
			t.Fatal(err)
		}
	}
	return at
}

// OFF / zero / missing / env / precedence — BOTH entry paths agree, on real
// closes in a real store.
func TestW1BreakerBothPathsPresenceAware(t *testing.T) {
	// Tuesday 10:00 CT — well inside one CME session-day (roll 17:00 CT).
	now := time.Date(2026, 9, 22, 10, 0, 0, 0, chicagoLoc())
	cases := []struct {
		name       string
		halt       *int
		env        string // "" = unset
		losses     int
		wantRefuse bool
		wantN      int
		wantField  string
	}{
		{"explicit OFF + 20 losses → allowed", store.IntPtr(0), "", 20, false, 0, "off[O]"},
		{"missing, no env, 7 losses → allowed", nil, "", 7, false, 8, "8[I]"},
		{"missing, no env, 8 losses → refused (inherit is ON)", nil, "", 8, true, 8, "8[I]"},
		{"missing + env 3, 3 losses → refused", nil, "3", 3, true, 3, "3[E]"},
		{"missing + env 0, 20 losses → allowed", nil, "0", 20, false, 0, "off[E]"},
		{"explicit 5 beats env 3, 3 losses → allowed", store.IntPtr(5), "3", 3, false, 5, "5[O]"},
		{"explicit 5 beats env 3, 5 losses → refused", store.IntPtr(5), "3", 5, true, 5, "5[O]"},
		{"explicit 0 beats env 3, 3 losses → allowed", store.IntPtr(0), "3", 3, false, 0, "off[O]"},
		{"missing + invalid env, 8 losses → refused at the default", nil, "banana", 8, true, 8, "8[I]"},
	}
	for i, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Setenv("BREAKER_HALT_N", c.env)
			at := w1BreakerTrader(t, fmt.Sprintf("w1-br-%d", i), c.halt, c.losses, now)

			reason, halted := at.consecutiveLossHaltedAt(now) // decision + agent path
			if halted != c.wantRefuse {
				t.Fatalf("decision path halted=%v want %v (%q)", halted, c.wantRefuse, reason)
			}
			v := at.sessionRiskGateAt(now) // arm + picture path
			if v.Refuse != c.wantRefuse {
				t.Fatalf("arm path refuse=%v want %v (%q)", v.Refuse, c.wantRefuse, v.Reason)
			}
			if c.wantRefuse && v.Class != "consecutive_loss" {
				t.Fatalf("arm path refused for %q, want consecutive_loss", v.Class)
			}
			if v.N != c.wantN {
				t.Fatalf("arm path resolved N=%d want %d", v.N, c.wantN)
			}
			if got, _ := breakerBootField(at.config.StrategyConfig); got != c.wantField {
				t.Fatalf("boot field %q want %q — the line must say what the gates run", got, c.wantField)
			}
		})
	}
}

// The process 🛑 line on a REAL store, through SessionRiskBootLineForBoot.
func TestW1SessionRiskBootLineReadsTheBreaker(t *testing.T) {
	type bound struct {
		strategyID, config string
	}
	cases := []struct {
		name    string
		env     string
		rows    []bound
		want    []string
		mustNot []string
	}{
		{"explicit OFF", "", []bound{{"s1", `{"ai_config":{"risk_control":{"consecutive_loss_halt":0}}}`}},
			[]string{"breaker=off[O]", "OFF — never refuses"}, []string{"never fires on the retained tape", "breaker=8"}},
		{"explicit 5", "", []bound{{"s1", `{"ai_config":{"risk_control":{"consecutive_loss_halt":5}}}`}},
			[]string{"breaker=5[O]"}, []string{"never fires on the retained tape"}},
		{"absent → 8[I], the tape clause is true of 8", "", []bound{{"s1", `{"ai_config":{"risk_control":{}}}`}},
			[]string{"breaker=8[I]", "never fires on the retained tape, max run 7, ids 585-591"}, nil},
		{"absent + env 3 → 3[E]", "3", []bound{{"s1", `{"ai_config":{"risk_control":{}}}`}},
			[]string{"breaker=3[E]", "UNREACHABLE (halt=3 fires first)"}, []string{"never fires on the retained tape"}},
		{"absent + env 0 → off[E]", "0", []bound{{"s1", `{"ai_config":{"risk_control":{}}}`}},
			[]string{"breaker=off[E]"}, nil},
		{"0 bound strategies → n/a", "", nil,
			[]string{"breaker=n/a (0 bound strategies)"}, []string{"breaker=8"}},
		{"2 bound strategies → n/a", "", []bound{{"s1", `{"ai_config":{"risk_control":{"consecutive_loss_halt":0}}}`}, {"s2", `{"ai_config":{"risk_control":{}}}`}},
			[]string{"breaker=n/a (2 bound strategies"}, []string{"breaker=8", "breaker=off"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Setenv("BREAKER_HALT_N", c.env)
			st, err := store.New(filepath.Join(t.TempDir(), "boot.db"))
			if err != nil {
				t.Fatal(err)
			}
			defer st.Close()
			for i, r := range c.rows {
				if err := st.GormDB().Exec(`INSERT INTO strategies (id, user_id, name, config) VALUES (?,?,?,?)`,
					r.strategyID, "u1", "s", r.config).Error; err != nil {
					t.Fatal(err)
				}
				if err := st.GormDB().Exec(`INSERT INTO traders (id, user_id, name, strategy_id, ai_model_id, exchange_id, initial_balance) VALUES (?,?,?,?,?,?,?)`,
					fmt.Sprintf("t%d", i), "u1", "hoang", r.strategyID, "m1", "e1", 50000.0).Error; err != nil {
					t.Fatal(err)
				}
			}
			line := SessionRiskBootLineForBoot(st)
			for _, w := range c.want {
				if !strings.Contains(line, w) {
					t.Errorf("line lacks %q:\n  %s", w, line)
				}
			}
			for _, w := range c.mustNot {
				if strings.Contains(line, w) {
					t.Errorf("line must not say %q:\n  %s", w, line)
				}
			}
		})
	}
}

// The per-trader breaker line prints the same field the gates resolve.
func TestW1BreakerBootLineForStrategy(t *testing.T) {
	t.Setenv("BREAKER_HALT_N", "")
	cfg := &store.StrategyConfig{}
	cfg.RiskControl.ConsecutiveLossHalt = store.IntPtr(0)
	if got := BreakerBootLineForStrategy(cfg); !strings.HasPrefix(got, "breaker=off[O]") {
		t.Fatalf("got %q", got)
	}
	if got := BreakerBootLineForStrategy(&store.StrategyConfig{}); !strings.HasPrefix(got, "breaker=8[I]") {
		t.Fatalf("got %q", got)
	}
}

// Replan cap 0 at the STRATEGY level is 0 through every production reader:
// the stored-config reader the executor prompt uses, the trader's cached
// resolver the death gate / spend / reread / reset use, and the boot line.
func TestW1StrategyReplanCapZeroIsZeroEverywhere(t *testing.T) {
	st, err := store.New(filepath.Join(t.TempDir(), "t.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	three := 3
	cfg := store.StrategyConfig{
		StrategyType: "ai_trading",
		DayPlan: &store.DayPlanConfig{
			PlanEnabled: true,
			ReplanCap:   store.IntPtr(0),
			Sessions:    []store.DayPlanSessionOverride{{Session: "NY", ReplanCap: &three}},
		},
	}
	seedTraderWithDayPlan(t, st, "trader-w1", "strat-w1", cfg)

	if got := storedReplanCap(st, "trader-w1", "ASIA"); got != 0 {
		t.Errorf("storedReplanCap ASIA = %d, want the saved strategy 0 (it used to read 2)", got)
	}
	if got := storedReplanCap(st, "trader-w1", "NY"); got != 3 {
		t.Errorf("storedReplanCap NY = %d, want the session override 3", got)
	}
	at := &AutoTrader{id: "trader-w1", store: st}
	at.config.StrategyConfig = &cfg
	if got := at.replanCapFor("LONDON"); got != 0 {
		t.Errorf("at.replanCapFor LONDON = %d, want 0", got)
	}
	line := ReplanCapBootLine(cfg.DayPlan)
	if want := "🧮 replan cap: strategy=0[O] · NY=3[O] · ASIA=0[O] · LONDON=0[O]"; line != want {
		t.Errorf("boot line:\n got %s\nwant %s", line, want)
	}
	// Absent reads the shipped default and SAYS so.
	if got := ReplanCapBootLine(&store.DayPlanConfig{PlanEnabled: true}); got != "🧮 replan cap: strategy=2[I] · NY=2[I] · ASIA=2[I] · LONDON=2[I]" {
		t.Errorf("absent: %s", got)
	}
	if got := ReplanCapBootLine(nil); !strings.HasSuffix(got, "(day plan off — not consulted)") {
		t.Errorf("a nil day plan must say the cap is not consulted: %s", got)
	}
}
