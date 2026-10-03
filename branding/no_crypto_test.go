package branding

import (
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"vl/internal/censuswalk"
)

// TestSweepRegexPinsSentinels proves the exported literal does what plan v10
// line 108 PLUS the CTO BTC/ETH amendment 2026-10-01 says: BOTH-boundary
// aster/lighter kill master/disaster/faster/easter/highlighter; quant\b
// kills "quantity"; "mixed" is DOUBLE-quoted only (the single-quoted
// 'mixed' sites are ContentAssertSites, asserted by content in the gate,
// never by the regex); btc catches the identifier forms a boundary cannot;
// ethusdt catches ETHUSDT; \beth\b catches standalone ETH without flooding
// on method/together/ethernet/whether/threshold.
func TestSweepRegexPinsSentinels(t *testing.T) {
	re, err := regexp.Compile("(?i)" + SweepRegexLiteral)
	if err != nil {
		t.Fatalf("SweepRegexLiteral does not compile: %v", err)
	}
	mustMatch := []string{
		"aster exchange", "**/aster*.go", "bg-vl-neo-bg-lighter",
		`case "mixed":`, `bracketOCO = "MIXED"`, "币安",
		"use_oi_top", "netflow ranking", "a quant run", "price ranking 5",
		// CTO BTC/ETH amendment 2026-10-01 — identifier forms (no boundary
		// inside an identifier, so a \b form cannot see these):
		"isBTCETHSymbol(sym)", "isBTCETH()", "BTCETHMaxLeverage",
		"btcEthPosValueRatio", "BTCETHMaxPositionValueRatio",
		"symbol == \"BTCUSDT\"", "BTCIDR -> btc_idr",
		// standalone / pair forms:
		"symbol == \"ETHUSDT\"", "\"ETH\" and \"ETHUSDT\" formats",
		"BTC/ETH max", "- Trading Leverage: Altcoins max",
		"AltcoinMaxLeverage", "go-ethereum/accounts/abi",
		"ethereum.org",
	}
	for _, s := range mustMatch {
		if !re.MatchString(s) {
			t.Errorf("sweep regex must match %q", s)
		}
	}
	mustNot := []string{
		"master branch", "disaster recovery", "faster code", "easter egg",
		"highlighter pen", "a quantity of 3",
		`arm.state === 'mixed'`, `source_type === 'mixed'`,
		// the flood the CTO named: a bare eth must NOT pull these in —
		// \beth\b keeps them out (no word boundary inside them)
		"method", "together", "ethernet", "whether", "something",
		"threshold", "failureThreshold", "gather", "further", "rather",
		"either", "neither", "other", "ether", "the", "tether",
		// and the amendment's own non-crypto neighbours stay out
		"backtest", "bitwise", "topic",
	}
	for _, s := range mustNot {
		if re.MatchString(s) {
			t.Errorf("sweep regex must NOT match %q", s)
		}
	}
}

// TestHostCensusLiteralByteExact pins the C8 host list byte-for-byte (plan
// v10 C8; GO item 2 declined — no alpaca/sina additions).
func TestHostCensusLiteralByteExact(t *testing.T) {
	want := `binance\.com|binance\.vision|bybit\.com|okx\.com|bitget\.com|kucoin\.com|gateio|gate\.io|indodax\.com|hyperliquid\.xyz|asterdex|lighter\.xyz|coinank|claw402\.ai|blockrun`
	if HostCensusLiteral != want {
		t.Fatalf("HostCensusLiteral drifted from the plan literal\n got: %s\nwant: %s", HostCensusLiteral, want)
	}
}

// TestSweepRegexMatchesThePlanLiteral re-derives the plan literal from the
// local plan checkout the same way DS-104's extraction does and asserts the
// guard's export is byte-identical to the plan literal PLUS the recorded
// CTO BTC/ETH amendment 2026-10-01 — the guard cannot drift from either.
func TestSweepRegexMatchesThePlanLiteral(t *testing.T) {
	planPath := "/home/hoang/crypto-removal-plan/2026-09-30-crypto-removal-plan-v10.md"
	if _, err := os.Stat(planPath); err != nil {
		t.Skipf("plan file not on this box (%v) — literal pin not evaluable here", err)
	}
	b, err := os.ReadFile(planPath)
	if err != nil {
		t.Fatal(err)
	}
	var lit string
	for _, line := range strings.Split(string(b), "\n") {
		if strings.Contains(line, "THE assembled regex") && strings.Contains(line, "R6-102-P1-1") {
			start := strings.Index(line, ": `")
			if start < 0 {
				t.Fatal("plan literal line shape changed")
			}
			rest := line[start+3:]
			end := strings.Index(rest, "`")
			if end < 0 {
				t.Fatal("plan literal line shape changed (no closing backtick)")
			}
			lit = rest[:end]
		}
	}
	if lit == "" {
		t.Fatal("plan literal not found in the v10 line")
	}
	// the CTO BTC/ETH amendment 2026-10-01 (P1 finding, CTO 06:03 mail) —
	// appended to the plan literal; recorded HERE so the amendment is pinned
	// exactly as hard as the plan line (an amended guard can still not drift)
	want := lit + "|btc|ethusdt|\\beth\\b|altcoin|ethereum"
	if SweepRegexLiteral != want {
		t.Fatalf("SweepRegexLiteral != plan v10 literal + CTO BTC/ETH amendment\n guard: %s\n want:  %s", SweepRegexLiteral, want)
	}
}

// TestLineLevelOwnershipAllowlistExact pins the two-path line-level
// ownership allowlist (CTO ruling 2026-10-01, 06:01 — ruling 2 removed
// agent/trade.go): exactly agent/agent.go and agent/tools.go. A third path
// is a FAIL.
func TestLineLevelOwnershipAllowlistExact(t *testing.T) {
	want := []string{"agent/agent.go", "agent/tools.go"}
	if len(LineLevelOwnershipPaths) != len(want) {
		t.Fatalf("line-level allowlist has %d paths, want %d: %v", len(LineLevelOwnershipPaths), len(want), LineLevelOwnershipPaths)
	}
	for i, p := range want {
		if LineLevelOwnershipPaths[i] != p {
			t.Fatalf("line-level allowlist[%d] = %q, want %q (the list is a ruling, never extended silently)", i, LineLevelOwnershipPaths[i], p)
		}
	}
}

// TestRiskCapAssertSitesExact pins the P0 risk-cap KEEP-canary list (CTO
// ruling 2026-10-01 10:5x): exactly the four fields + tags + decision-path
// args + chat-entry reads + knob rows. A site added or dropped without a new
// ruling FAILS — the list is never extended silently.
func TestRiskCapAssertSitesExact(t *testing.T) {
	want := []string{
		"store/strategy.go", "BTCETHMaxLeverage",
		"store/strategy.go", "AltcoinMaxLeverage",
		"store/strategy.go", "BTCETHMaxPositionValueRatio",
		"store/strategy.go", "AltcoinMaxPositionValueRatio",
		"store/strategy.go", "btc_eth_max_leverage",
		"store/strategy.go", "altcoin_max_leverage",
		"store/strategy.go", "btc_eth_max_position_value_ratio",
		"store/strategy.go", "altcoin_max_position_value_ratio",
		"kernel/engine_analysis.go", "riskConfig.BTCETHMaxLeverage",
		"kernel/engine_analysis.go", "riskConfig.AltcoinMaxLeverage",
		"kernel/engine_analysis.go", "riskConfig.BTCETHMaxPositionValueRatio",
		"kernel/engine_analysis.go", "riskConfig.AltcoinMaxPositionValueRatio",
		"agent/trade.go", "riskControl.AltcoinMaxLeverage",
		"agent/trade.go", "riskControl.AltcoinMaxPositionValueRatio",
		"store/knob_registry_table.go", "altcoin_max_leverage",
		"store/knob_registry_table.go", "altcoin_max_position_value_ratio",
	}
	if len(RiskCapAssertSites) != len(want) {
		t.Fatalf("RiskCapAssertSites has %d entries, want %d: %v", len(RiskCapAssertSites), len(want), RiskCapAssertSites)
	}
	for i := range want {
		if RiskCapAssertSites[i] != want[i] {
			t.Fatalf("RiskCapAssertSites[%d] = %q, want %q (the P0 cap list is a ruling, never extended silently)", i, RiskCapAssertSites[i], want[i])
		}
	}
}

func importTargetsCrypto(source []byte) (map[string]bool, error) {
	f, err := parser.ParseFile(token.NewFileSet(), "source.go", source, parser.ImportsOnly)
	if err != nil {
		return nil, err
	}
	targets := map[string]bool{}
	for _, spec := range f.Imports {
		path, err := strconv.Unquote(spec.Path.Value)
		if err != nil {
			return nil, err
		}
		targets[path] = true
	}
	return targets, nil
}

// moduleRoot returns the module path the census sees (vl; the rename made
// it vl — never hardcode a second copy).
func moduleRoot(t *testing.T) string {
	if m, err := censuswalk.ModulePath(".."); err == nil {
		return m
	}
	return "vl"
}

// TestCensusEnumeratesFailsLoud is the C8 enumeration invariant: git
// ls-files -z, t.Fatal on git failure (never a skip), and a 2,000-file
// floor so an empty enumeration cannot pass.
func TestCensusEnumeratesFailsLoud(t *testing.T) {
	out, err := exec.Command("git", "-C", "..", "ls-files", "-z").Output()
	if err != nil {
		t.Fatalf("git ls-files failed (never skip): %v", err)
	}
	files := strings.Split(strings.TrimSuffix(string(out), "\x00"), "\x00")
	if len(files) < 2000 {
		t.Fatalf("census enumerated only %d tracked files (< 2000 floor) — it is going vacuous", len(files))
	}
}

// TestNoCryptoImportsSDKsHosts is the C8 zero-state census: no import of a
// deleted package prefix, no deleted SDK module in go.mod/go.sum, no
// crypto-venue host in non-test Go files or web/src.
//
// BY DESIGN this test is RED at every pre-cut head and goes GREEN only at
// the integrated head (plan v10 C8: "Build gate: the same grep returns 0 at
// the integrated head"). The parts cut the sites it names.
func TestNoCryptoImportsSDKsHosts(t *testing.T) {
	root := ".."
	out, err := exec.Command("git", "-C", root, "ls-files", "-z").Output()
	if err != nil {
		t.Fatalf("git ls-files failed (never skip): %v", err)
	}
	files := strings.Split(strings.TrimSuffix(string(out), "\x00"), "\x00")
	hostRe := regexp.MustCompile("(?i)" + HostCensusLiteral)
	var problems []string

	prefixHit := func(target string) bool {
		target = strings.TrimPrefix(target, moduleRoot(t)+"/")
		for _, p := range DeletedImportPrefixes {
			if target == p || strings.HasPrefix(target, p+"/") {
				return true
			}
		}
		return false
	}
	sdkHit := func(line string) bool {
		for _, m := range DeletedSDKModules {
			if strings.Contains(line, m) {
				return true
			}
		}
		return false
	}

	for _, f := range files {
		// the guard file IS the literal's home — the census sweeps the tree
		// for crypto venues, and without this skip its own HostCensusLiteral
		// would be a permanent violation and 0 would be unreachable
		if f == "branding/no_crypto.go" {
			continue
		}
		b, rerr := os.ReadFile(filepath.Join(root, f))
		if rerr != nil {
			continue
		}
		base := filepath.Base(f)
		if strings.HasSuffix(f, ".go") {
			if targets, perr := importTargetsCrypto(b); perr == nil {
				for tg := range targets {
					if prefixHit(tg) {
						problems = append(problems, f+": imports deleted package prefix "+tg)
					}
				}
			}
			// host census: non-test Go files only
			if !strings.HasSuffix(base, "_test.go") && hostRe.Match(b) {
				problems = append(problems, f+": crypto-venue host in a non-test Go file")
			}
		}
		if strings.HasPrefix(f, "web/src/") && hostRe.Match(b) {
			problems = append(problems, f+": crypto-venue host under web/src")
		}
		if f == "go.mod" || f == "go.sum" {
			for _, line := range strings.Split(string(b), "\n") {
				if sdkHit(line) {
					problems = append(problems, f+": deleted SDK module present ("+line+")")
				}
			}
		}
	}
	if len(problems) > 0 {
		n := len(problems)
		if n > 12 {
			n = 12
		}
		for _, p := range problems[:n] {
			t.Errorf("%s", p)
		}
		t.Fatalf("%d crypto census violations (0 expected at the integrated head)", len(problems))
	}
}
