# CR-A disposition table — brokers + payments + providers (crypto removal, part CR-A)

base sha (branch point): db412e61c
integrator tip generated against: 133226793
generated: 2026-10-01T19:41:28-05:00 (branch crypto-removal-integration @ 133226793 — provenance: branch + sha only, never a filesystem path)
sweep regex (exported programmatically from branding/no_crypto.go SweepRegexLiteral — the guard's amended literal, never hand-typed):
  `binance|bybit|okx|bitget|kucoin|gate\.io|gateio|indodax|hyperliquid|\baster\b|asterdex|\blighter\b|coinank|usdc|usdt|x402|claw402|blockrun|wallet|ai500|hyper_all|hyper_main|oi_top|oi_low|netflow|quant\b|price ranking|"mixed"|币安|欧易|火币|U本位|永续|btc|ethusdt|\beth\b|altcoin|ethereum`

| path | line | token | disposition | owner | reason |
|---|---|---|---|---|---|
| .audit/f_weekly_reader_belief_census_f1_f5_disp.md | 165 | `Binance` | KEEP | CR-A | historical doc/config file — not shipped code, KEEP byte-identical |
| .audit/guardrails_sizing_dispatch_d9_monte_carl.md | 48 | `btc` | KEEP | CR-A | historical doc/config file — not shipped code, KEEP byte-identical |
| .audit/guardrails_sizing_dispatch_d9_monte_carl.md | 49 | `btc` | KEEP | CR-A | historical doc/config file — not shipped code, KEEP byte-identical |
| .audit/guardrails_sizing_dispatch_d9_monte_carl.md | 93 | `btc` | KEEP | CR-A | historical doc/config file — not shipped code, KEEP byte-identical |
| CHANGELOG.zh-CN.md | 45 | `Binance` | KEEP | CR-A | historical doc/config file — not shipped code, KEEP byte-identical |
| CHANGELOG.zh-CN.md | 90 | `USDT` | KEEP | CR-A | historical doc/config file — not shipped code, KEEP byte-identical |
| CHANGELOG.zh-CN.md | 91 | `USDT` | KEEP | CR-A | historical doc/config file — not shipped code, KEEP byte-identical |
| CHANGELOG.zh-CN.md | 92 | `USDT` | KEEP | CR-A | historical doc/config file — not shipped code, KEEP byte-identical |
| CHANGELOG.zh-CN.md | 103 | `BTC` | KEEP | CR-A | historical doc/config file — not shipped code, KEEP byte-identical |
| CHANGELOG.zh-CN.md | 110 | `USDT` | KEEP | CR-A | historical doc/config file — not shipped code, KEEP byte-identical |
| CHANGELOG.zh-CN.md | 119 | `Aster` | KEEP | CR-A | historical doc/config file — not shipped code, KEEP byte-identical |
| CHANGELOG.zh-CN.md | 121 | `Aster` | KEEP | CR-A | historical doc/config file — not shipped code, KEEP byte-identical |
| CHANGELOG.zh-CN.md | 151 | `币安` | KEEP | CR-A | historical doc/config file — not shipped code, KEEP byte-identical |
| CHANGELOG.zh-CN.md | 170 | `币安` | KEEP | CR-A | historical doc/config file — not shipped code, KEEP byte-identical |
| README.ja.md | 365 | `btc` | KEEP | CR-A | historical doc/config file — not shipped code, KEEP byte-identical |
| README.ja.md | 366 | `altcoin` | KEEP | CR-A | historical doc/config file — not shipped code, KEEP byte-identical |
| README.ja.md | 446 | `btc` | KEEP | CR-A | historical doc/config file — not shipped code, KEEP byte-identical |
| README.ja.md | 447 | `altcoin` | KEEP | CR-A | historical doc/config file — not shipped code, KEEP byte-identical |
| README.ja.md | 468 | `btc` | KEEP | CR-A | historical doc/config file — not shipped code, KEEP byte-identical |
| README.ja.md | 469 | `altcoin` | KEEP | CR-A | historical doc/config file — not shipped code, KEEP byte-identical |
| README.ja.md | 492 | `btc` | KEEP | CR-A | historical doc/config file — not shipped code, KEEP byte-identical |
| README.ja.md | 493 | `altcoin` | KEEP | CR-A | historical doc/config file — not shipped code, KEEP byte-identical |
| README.ja.md | 500 | `btc` | KEEP | CR-A | historical doc/config file — not shipped code, KEEP byte-identical |
| README.ja.md | 501 | `altcoin` | KEEP | CR-A | historical doc/config file — not shipped code, KEEP byte-identical |
| README.ja.md | 508 | `altcoin` | KEEP | CR-A | historical doc/config file — not shipped code, KEEP byte-identical |
| README.ja.md | 1013 | `Aster` | KEEP | CR-A | historical doc/config file — not shipped code, KEEP byte-identical |
| agent/tools.go | 826 | `Wallet` | KEEP | CR-A | skill/tool surface — cut deferred until DS-107 skill-handler consumer cuts land |
| agent/tools.go | 827 | `USDC` | KEEP | CR-A | skill/tool surface — cut deferred until DS-107 skill-handler consumer cuts land |
| agent/tools.go | 941 | `Wallet` | KEEP | CR-A | NT8 account-snapshot shape — balance key family (CTO ruling, byte-identical) |
| agent/tools.go | 2141 | `btc` | KEEP | CR-A | risk-cap knob family (union-gate risk-cap canary) |
| agent/tools.go | 2142 | `altcoin` | KEEP | CR-A | risk-cap knob family (union-gate risk-cap canary) |
| agent/tools.go | 2144 | `AI500` | KEEP | CR-A | skill/tool surface — cut deferred until DS-107 skill-handler consumer cuts land |
| agent/tools.go | 2164 | `BTC` | KEEP | CR-A | risk-cap knob family (union-gate risk-cap canary) |
| agent/tools.go | 2165 | `Altcoin` | KEEP | CR-A | risk-cap knob family (union-gate risk-cap canary) |
| agent/tools.go | 2167 | `AI500` | KEEP | CR-A | skill/tool surface — cut deferred until DS-107 skill-handler consumer cuts land |
| agent/tools.go | 2236 | `BTC` | KEEP | CR-A | risk-cap knob family (union-gate risk-cap canary) |
| agent/tools.go | 2237 | `Altcoin` | KEEP | CR-A | risk-cap knob family (union-gate risk-cap canary) |
| agent/tools.go | 2239 | `AI500` | KEEP | CR-A | skill/tool surface — cut deferred until DS-107 skill-handler consumer cuts land |
| agents.md | 219 | `BTC` | KEEP | CR-A | historical doc/config file — not shipped code, KEEP byte-identical |
| api/exchange_account_state.go | 200 | `Wallet` | KEEP | CR-A | NT8 account-snapshot shape — balance key family (CTO ruling, byte-identical) |
| api/exchange_account_state.go | 250 | `Wallet` | KEEP | CR-A | NT8 account-snapshot shape — balance key family (CTO ruling, byte-identical) |
| api/handler_trader.go | 584 | `BTC` | KEEP | CR-A | risk-cap knob family (union-gate risk-cap canary) |
| api/handler_trader.go | 585 | `Altcoin` | KEEP | CR-A | risk-cap knob family (union-gate risk-cap canary) |
| api/handler_user.go | 548 | `BTC` | KEEP | CR-A | risk-cap knob family (union-gate risk-cap canary) |
| api/handler_user.go | 549 | `Altcoin` | KEEP | CR-A | risk-cap knob family (union-gate risk-cap canary) |
| api/handler_user.go | 550 | `BTC` | KEEP | CR-A | risk-cap knob family (union-gate risk-cap canary) |
| api/handler_user.go | 551 | `Altcoin` | KEEP | CR-A | risk-cap knob family (union-gate risk-cap canary) |
| api/handler_user.go | 563 | `BTC` | KEEP | CR-A | risk-cap knob family (union-gate risk-cap canary) |
| api/handler_user.go | 564 | `Altcoin` | KEEP | CR-A | risk-cap knob family (union-gate risk-cap canary) |
| api/handler_user.go | 566 | `Altcoin` | KEEP | CR-A | risk-cap knob family (union-gate risk-cap canary) |
| clock-seams.list | 35 | `binance` | KEEP | CR-A | historical doc/config file — not shipped code, KEEP byte-identical |
| clock-seams.list | 36 | `BINANCE` | KEEP | CR-A | historical doc/config file — not shipped code, KEEP byte-identical |
| main.go | 252 | `BINANCE` | KEEP | CR-A | wave-name comment (W-NO-BINANCE) — documents the OI-absent-on-MNQ behavior |
| main.go | 371 | `BINANCE` | KEEP | CR-A | wave-name comment (W-NO-BINANCE) — documents the OI-absent-on-MNQ behavior |
| mcp/intro/BUILDER_EXAMPLES.md | 74 | `BTC` | KEEP | CR-A | historical doc/config file — not shipped code, KEEP byte-identical |
| mcp/intro/BUILDER_EXAMPLES.md | 75 | `BTC` | KEEP | CR-A | historical doc/config file — not shipped code, KEEP byte-identical |
| mcp/intro/BUILDER_EXAMPLES.md | 144 | `BTC` | KEEP | CR-A | historical doc/config file — not shipped code, KEEP byte-identical |
| mcp/intro/BUILDER_EXAMPLES.md | 295 | `BTC` | KEEP | CR-A | historical doc/config file — not shipped code, KEEP byte-identical |
| mcp/intro/BUILDER_EXAMPLES.md | 310 | `BTC` | KEEP | CR-A | historical doc/config file — not shipped code, KEEP byte-identical |
| mcp/intro/BUILDER_EXAMPLES.md | 515 | `BTC` | KEEP | CR-A | historical doc/config file — not shipped code, KEEP byte-identical |
| mcp/intro/BUILDER_EXAMPLES.md | 521 | `ETH` | KEEP | CR-A | historical doc/config file — not shipped code, KEEP byte-identical |
| mcp/intro/BUILDER_PATTERN_BENEFITS.md | 280 | `BTC` | KEEP | CR-A | historical doc/config file — not shipped code, KEEP byte-identical |
| mcp/intro/BUILDER_PATTERN_BENEFITS.md | 281 | `BTC` | KEEP | CR-A | historical doc/config file — not shipped code, KEEP byte-identical |
| mcp/intro/LOGRUS_INTEGRATION.md | 217 | `BTC` | KEEP | CR-A | historical doc/config file — not shipped code, KEEP byte-identical |
| mcp/intro/README.md | 78 | `BTC` | KEEP | CR-A | historical doc/config file — not shipped code, KEEP byte-identical |
| mcp/intro/README.md | 79 | `BTC` | KEEP | CR-A | historical doc/config file — not shipped code, KEEP byte-identical |
| store/ai_charge.go | 152 | `USDC` | KEEP | CR-A | EstimateRunway stays per CTO ruling — daily-cost/runway estimate; usdcBalance naming byte-identical (exported API; zero callers after the dead claw402 pre-launch check was cut) |
| store/ai_charge.go | 153 | `usdc` | KEEP | CR-A | EstimateRunway stays per CTO ruling — daily-cost/runway estimate; usdcBalance naming byte-identical (exported API; zero callers after the dead claw402 pre-launch check was cut) |
| store/ai_charge.go | 160 | `usdc` | KEEP | CR-A | EstimateRunway stays per CTO ruling — daily-cost/runway estimate; usdcBalance naming byte-identical (exported API; zero callers after the dead claw402 pre-launch check was cut) |
| store/ai_charge.go | 161 | `usdc` | KEEP | CR-A | EstimateRunway stays per CTO ruling — daily-cost/runway estimate; usdcBalance naming byte-identical (exported API; zero callers after the dead claw402 pre-launch check was cut) |
| store/exchange.go | 133 | `binance` | KEEP | CR-A | legacy crypto columns/params — stored rows must load (C1); column drop pending CTO ruling |
| store/exchange.go | 147 | `binance` | KEEP | CR-A | legacy crypto columns/params — stored rows must load (C1); column drop pending CTO ruling |
| store/exchange.go | 157 | `binance` | KEEP | CR-A | legacy crypto columns/params — stored rows must load (C1); column drop pending CTO ruling |
| trader/auto_trader.go | 256 | `Binance` | KEEP | CR-A | AutoTraderConfig per-broker credential fields — sole populator manager/trader_manager.go (DS-103 deferred) |
| trader/auto_trader.go | 257 | `Binance` | KEEP | CR-A | AutoTraderConfig per-broker credential fields — sole populator manager/trader_manager.go (DS-103 deferred) |
| trader/auto_trader.go | 258 | `Binance` | KEEP | CR-A | AutoTraderConfig per-broker credential fields — sole populator manager/trader_manager.go (DS-103 deferred) |
| trader/auto_trader.go | 260 | `Bybit` | KEEP | CR-A | AutoTraderConfig per-broker credential fields — sole populator manager/trader_manager.go (DS-103 deferred) |
| trader/auto_trader.go | 261 | `Bybit` | KEEP | CR-A | AutoTraderConfig per-broker credential fields — sole populator manager/trader_manager.go (DS-103 deferred) |
| trader/auto_trader.go | 262 | `Bybit` | KEEP | CR-A | AutoTraderConfig per-broker credential fields — sole populator manager/trader_manager.go (DS-103 deferred) |
| trader/auto_trader.go | 264 | `OKX` | KEEP | CR-A | AutoTraderConfig per-broker credential fields — sole populator manager/trader_manager.go (DS-103 deferred) |
| trader/auto_trader.go | 265 | `OKX` | KEEP | CR-A | AutoTraderConfig per-broker credential fields — sole populator manager/trader_manager.go (DS-103 deferred) |
| trader/auto_trader.go | 266 | `OKX` | KEEP | CR-A | AutoTraderConfig per-broker credential fields — sole populator manager/trader_manager.go (DS-103 deferred) |
| trader/auto_trader.go | 267 | `OKX` | KEEP | CR-A | AutoTraderConfig per-broker credential fields — sole populator manager/trader_manager.go (DS-103 deferred) |
| trader/auto_trader.go | 269 | `Bitget` | KEEP | CR-A | AutoTraderConfig per-broker credential fields — sole populator manager/trader_manager.go (DS-103 deferred) |
| trader/auto_trader.go | 270 | `Bitget` | KEEP | CR-A | AutoTraderConfig per-broker credential fields — sole populator manager/trader_manager.go (DS-103 deferred) |
| trader/auto_trader.go | 271 | `Bitget` | KEEP | CR-A | AutoTraderConfig per-broker credential fields — sole populator manager/trader_manager.go (DS-103 deferred) |
| trader/auto_trader.go | 272 | `Bitget` | KEEP | CR-A | AutoTraderConfig per-broker credential fields — sole populator manager/trader_manager.go (DS-103 deferred) |
| trader/auto_trader.go | 278 | `KuCoin` | KEEP | CR-A | AutoTraderConfig per-broker credential fields — sole populator manager/trader_manager.go (DS-103 deferred) |
| trader/auto_trader.go | 279 | `KuCoin` | KEEP | CR-A | AutoTraderConfig per-broker credential fields — sole populator manager/trader_manager.go (DS-103 deferred) |
| trader/auto_trader.go | 280 | `KuCoin` | KEEP | CR-A | AutoTraderConfig per-broker credential fields — sole populator manager/trader_manager.go (DS-103 deferred) |
| trader/auto_trader.go | 281 | `KuCoin` | KEEP | CR-A | AutoTraderConfig per-broker credential fields — sole populator manager/trader_manager.go (DS-103 deferred) |
| trader/auto_trader.go | 283 | `Indodax` | KEEP | CR-A | AutoTraderConfig per-broker credential fields — sole populator manager/trader_manager.go (DS-103 deferred) |
| trader/auto_trader.go | 284 | `Indodax` | KEEP | CR-A | AutoTraderConfig per-broker credential fields — sole populator manager/trader_manager.go (DS-103 deferred) |
| trader/auto_trader.go | 285 | `Indodax` | KEEP | CR-A | AutoTraderConfig per-broker credential fields — sole populator manager/trader_manager.go (DS-103 deferred) |
| trader/auto_trader.go | 292 | `Hyperliquid` | KEEP | CR-A | AutoTraderConfig per-broker credential fields — sole populator manager/trader_manager.go (DS-103 deferred) |
| trader/auto_trader.go | 293 | `Hyperliquid` | KEEP | CR-A | AutoTraderConfig per-broker credential fields — sole populator manager/trader_manager.go (DS-103 deferred) |
| trader/auto_trader.go | 294 | `Hyperliquid` | KEEP | CR-A | NT8 account-snapshot shape — balance key family (CTO ruling, byte-identical) |
| trader/auto_trader.go | 295 | `Hyperliquid` | KEEP | CR-A | AutoTraderConfig per-broker credential fields — sole populator manager/trader_manager.go (DS-103 deferred) |
| trader/auto_trader.go | 296 | `Hyperliquid` | KEEP | CR-A | AutoTraderConfig per-broker credential fields — sole populator manager/trader_manager.go (DS-103 deferred) |
| trader/auto_trader.go | 298 | `Aster` | KEEP | CR-A | AutoTraderConfig per-broker credential fields — sole populator manager/trader_manager.go (DS-103 deferred) |
| trader/auto_trader.go | 299 | `Aster` | KEEP | CR-A | NT8 account-snapshot shape — balance key family (CTO ruling, byte-identical) |
| trader/auto_trader.go | 300 | `Aster` | KEEP | CR-A | NT8 account-snapshot shape — balance key family (CTO ruling, byte-identical) |
| trader/auto_trader.go | 301 | `Aster` | KEEP | CR-A | NT8 account-snapshot shape — balance key family (CTO ruling, byte-identical) |
| trader/auto_trader.go | 303 | `LIGHTER` | KEEP | CR-A | AutoTraderConfig per-broker credential fields — sole populator manager/trader_manager.go (DS-103 deferred) |
| trader/auto_trader.go | 304 | `Wallet` | KEEP | CR-A | NT8 account-snapshot shape — balance key family (CTO ruling, byte-identical) |
| trader/auto_trader.go | 305 | `LIGHTER` | KEEP | CR-A | AutoTraderConfig per-broker credential fields — sole populator manager/trader_manager.go (DS-103 deferred) |
| trader/auto_trader.go | 306 | `LIGHTER` | KEEP | CR-A | AutoTraderConfig per-broker credential fields — sole populator manager/trader_manager.go (DS-103 deferred) |
| trader/auto_trader.go | 307 | `LIGHTER` | KEEP | CR-A | AutoTraderConfig per-broker credential fields — sole populator manager/trader_manager.go (DS-103 deferred) |
| trader/auto_trader.go | 319 | `Claw402` | KEEP | CR-A | NT8 account-snapshot shape — balance key family (CTO ruling, byte-identical) |
| trader/auto_trader.go | 771 | `Wallet` | KEEP | CR-A | NT8 account-snapshot shape — balance key family (CTO ruling, byte-identical) |
| trader/auto_trader_grid.go | 148 | `Wallet` | KEEP | CR-A | NT8 account-snapshot shape — balance key family (CTO ruling, byte-identical) |
| trader/auto_trader_loop.go | 993 | `Wallet` | KEEP | CR-A | NT8 account-snapshot shape — balance key family (CTO ruling, byte-identical) |
| trader/auto_trader_loop.go | 998 | `wallet` | KEEP | CR-A | NT8 account-snapshot shape — balance key family (CTO ruling, byte-identical) |
| trader/auto_trader_loop.go | 999 | `Wallet` | KEEP | CR-A | NT8 account-snapshot shape — balance key family (CTO ruling, byte-identical) |
| trader/auto_trader_loop.go | 1012 | `Wallet` | KEEP | CR-A | NT8 account-snapshot shape — balance key family (CTO ruling, byte-identical) |
| trader/auto_trader_loop.go | 1013 | `Wallet` | KEEP | CR-A | NT8 account-snapshot shape — balance key family (CTO ruling, byte-identical) |
| trader/auto_trader_loop.go | 1147 | `btc` | KEEP | CR-A | risk-cap knob family (union-gate risk-cap canary) |
| trader/auto_trader_loop.go | 1148 | `altcoin` | KEEP | CR-A | risk-cap knob family (union-gate risk-cap canary) |
| trader/auto_trader_loop.go | 1157 | `BTC` | KEEP | CR-A | risk-cap knob family (union-gate risk-cap canary) |
| trader/auto_trader_loop.go | 1158 | `Altcoin` | KEEP | CR-A | risk-cap knob family (union-gate risk-cap canary) |
| trader/auto_trader_orders.go | 496 | `Wallet` | KEEP | CR-A | NT8 account-snapshot shape — balance key family (CTO ruling, byte-identical) |
| trader/auto_trader_risk.go | 272 | `Altcoin` | KEEP | CR-A | risk-cap knob family (union-gate risk-cap canary) |
| trader/auto_trader_risk.go | 274 | `altcoin` | KEEP | CR-A | risk-cap knob family (union-gate risk-cap canary) |
| trader/market_data_boot.go | 10 | `BINANCE` | KEEP | CR-A | wave-name comment (W-NO-BINANCE) — documents the OI-absent-on-MNQ behavior |
