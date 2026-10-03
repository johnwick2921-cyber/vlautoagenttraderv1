package branding

// C8 + C13 guard (crypto removal, plan v10 FINAL).
//
// This file is the SINGLE home of the sweep/host literals (plan v10 C13:
// "ONE regex, exported as a literal from the guard and used by the tables
// and the gate — never re-typed"). The builders extract these literals
// programmatically from THIS file (never hand-typed), and
// scripts/crypto-union-gate.sh asserts every disposition table's `regex:`
// header equals SweepRegexLiteral byte-for-byte.
//
// GO item 2 was DECLINED 2026-10-01 (CTO ruling): stocks/forex stay. The
// literals below are the BASE form — do NOT append alpaca|twelvedata|sina.

// SweepRegexLiteral is THE assembled C13 sweep regex, byte-for-byte the
// literal in plan v10 line 108 (the `aster\b`/`lighter\b` bounds carry BOTH
// word boundaries; `quant\b` keeps its trailing boundary; `"mixed"` matches
// only the DOUBLE-quoted literal) PLUS the CTO BTC/ETH amendment
// 2026-10-01 (P1 finding: 119 non-test Go lines invisible — isBTCETH(),
// isBTCETHSymbol, ETHUSDT, the BTC/ETH prompt vocabulary).
//
// Amendment tokens, engineered the way \baster\b was:
//
//	btc        — no English word contains it; catches BTC, BTCUSDT and the
//	             identifier forms isBTCETHSymbol / BTCETHMaxLeverage /
//	             btcEthPosValueRatio (a boundary form cannot — there is no
//	             boundary inside an identifier)
//	ethusdt    — `\beth\b` does NOT match ETHUSDT (U is a word char); the
//	             bare pair-token catches it with zero flood
//	\beth\b    — standalone ETH ("BTC/ETH max", quoted "ETH"), while a bare
//	             eth floods on method/together/ethernet/whether/threshold
//	altcoin    — same-class vocabulary (AltcoinMaxLeverage, "Altcoins max") —
//	             included with the amendment, zero flood
//	ethereum   — same class (go-ethereum imports, docs) — zero flood
//
// bitcoin is ABSENT on purpose: zero non-test occurrences at dc630ad8
// (surveyed 2026-10-01).
// Run case-insensitive with Go regexp or GNU grep -E (`\b` is not POSIX ERE).
const SweepRegexLiteral = `binance|bybit|okx|bitget|kucoin|gate\.io|gateio|indodax|hyperliquid|\baster\b|asterdex|\blighter\b|coinank|usdc|usdt|x402|claw402|blockrun|wallet|ai500|hyper_all|hyper_main|oi_top|oi_low|netflow|quant\b|price ranking|"mixed"|币安|欧易|火币|U本位|永续|btc|ethusdt|\beth\b|altcoin|ethereum`

// HostCensusLiteral is the C8 host census list (plan v10 C8), byte-for-byte:
// non-test Go files and web/src hold ZERO of these. GO item 2 declined, so
// no alpaca.markets|twelvedata.com|sinajs… additions.
const HostCensusLiteral = `binance\.com|binance\.vision|bybit\.com|okx\.com|bitget\.com|kucoin\.com|gateio|gate\.io|indodax\.com|hyperliquid\.xyz|asterdex|lighter\.xyz|coinank|claw402\.ai|blockrun`

// DeletedImportPrefixes is DERIVED from the plan C4 directory list (one
// import-prefix entry per deleted top-level dir), relative to the module
// root. The census fails on any Go import whose path contains one of these
// as a path segment prefix.
var DeletedImportPrefixes = []string{
	"trader/aster", "trader/binance", "trader/bitget", "trader/bybit",
	"trader/gate", "trader/hyperliquid", "trader/indodax", "trader/kucoin",
	"trader/lighter", "trader/okx",
	"provider/coinank", "provider/hyperliquid",
	"mcp/payment", "wallet",
}

// DeletedSDKModules is the C4 go.mod drop list. CR-A's go mod tidy diff
// appends every additional module tidy removes (quoted in the PR) before
// the integrated head.
var DeletedSDKModules = []string{
	"go-binance", "bybit.go.api", "gateapi-go", "antihax/optional",
	"lighter-go", "poseidon", "go-hyperliquid", "go-ethereum",
}

// RiskCapAssertSites — P0 ruling 2026-10-01 10:5x: the four LIVE futures risk
// caps are crypto-NAMED but futures-ACTIVE (kernel/engine_analysis.go passes
// them into parseFullDecisionResponse on the futures path; for MNQ the
// Altcoin pair is the binding cap; agent/trade.go reads them as the chat-entry
// caps). The owner's stored strategies carry real values — deleting a field
// makes the loader ignore the key and the cap silently falls back to defaults
// (C1 class). File+needle pairs, each MUST be present at the integrated head:
// the gate extracts this list programmatically and FAILS on any missing pair.
// KEEP byte-identical this wave; renaming to futures names with JSON aliases
// is a separate owner-gated wave. Extend this list only with a new ruling.
var RiskCapAssertSites = []string{
	// the four Go fields and their JSON tags
	"store/strategy.go", "BTCETHMaxLeverage",
	"store/strategy.go", "AltcoinMaxLeverage",
	"store/strategy.go", "BTCETHMaxPositionValueRatio",
	"store/strategy.go", "AltcoinMaxPositionValueRatio",
	"store/strategy.go", "btc_eth_max_leverage",
	"store/strategy.go", "altcoin_max_leverage",
	"store/strategy.go", "btc_eth_max_position_value_ratio",
	"store/strategy.go", "altcoin_max_position_value_ratio",
	// the futures decision-path arguments
	"kernel/engine_analysis.go", "riskConfig.BTCETHMaxLeverage",
	"kernel/engine_analysis.go", "riskConfig.AltcoinMaxLeverage",
	"kernel/engine_analysis.go", "riskConfig.BTCETHMaxPositionValueRatio",
	"kernel/engine_analysis.go", "riskConfig.AltcoinMaxPositionValueRatio",
	// the chat-entry cap reads
	"agent/trade.go", "riskControl.AltcoinMaxLeverage",
	"agent/trade.go", "riskControl.AltcoinMaxPositionValueRatio",
	// the knob registry rows (KnobLive)
	"store/knob_registry_table.go", "altcoin_max_leverage",
	"store/knob_registry_table.go", "altcoin_max_position_value_ratio",
}

// ContentAssertSites are the single-quoted sites the sweep regex CANNOT see
// (a trailing-boundary regex with a double-quoted "mixed" never matches
// 'mixed'). CTO ruling 2026-10-01: assert them by CONTENT in the gate,
// separately from the regex sweep. disposition is what the CR-C table must
// carry for the site.
var ContentAssertSites = []struct{ File, Needle, Disposition string }{
	{"web/src/components/plan/ExecutorVerdict.tsx", `arm.state === 'mixed'`, "KEEP"},
	{"web/src/components/trader/TraderConfigModal.tsx", `source_type === 'mixed'`, "CUT"},
}

// LineLevelOwnershipPaths is the EXACT two-path allowlist (CTO ruling
// 2026-10-01, 06:01 "FOUR RULINGS", ruling 2 — SUPERSEDES the 05:52
// three-path form): ownership follows the FILE everywhere EXCEPT these two
// files, where it follows the LINE (CR-A owns the
// payment/wallet/broker-factory/provider-branch lines; CR-B owns everything
// else). agent/trade.go was REMOVED from the exception by ruling 2 — it is
// WHOLLY CR-A (the six futures-active chat-entry notional caps must never
// meet a CR-B DELETE). The gate asserts FILE-level exclusivity everywhere
// else and LINE-level ownership on exactly these two. A third path here is
// a FAIL — this list is pinned by TestLineLevelOwnershipAllowlistExact,
// never extended without a new ruling.
var LineLevelOwnershipPaths = []string{
	"agent/agent.go",
	"agent/tools.go",
}
