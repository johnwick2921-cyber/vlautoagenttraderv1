# CR-B disposition table — engine + tests + config surface (crypto removal, part CR-B, regenerated at the integrated head)

branch-point: dc630ad8fc06c49aebd5e59a8814744c2cbab5e7
integrator-tip: 133226793336a5e0a1ef9a4c8f71cbe7cacfeb23 (regenerated at the integrated head; provenance: branch + sha only)
paths: manager kernel market config store agent branding internal ninjascript screenshots cmd scripts deploy docker nginx .github patches hook telegram provider/ninjatrader provider/databento trader/ninjatrader SECURITY.md Makefile .env.example docs/superpowers/AUDIT-CHECKLIST.md docs/superpowers/SYSTEM-MAP.md trader/auto_trader_decision.go api/handler_debug.go api/strategy_effective.go api/handler_competition.go api/handler_plan_order_truth.go api/handler_order.go api/handler_trader_config.go trader/protection_reconciler.go
regex: binance|bybit|okx|bitget|kucoin|gate\.io|gateio|indodax|hyperliquid|\baster\b|asterdex|\blighter\b|coinank|usdc|usdt|x402|claw402|blockrun|wallet|ai500|hyper_all|hyper_main|oi_top|oi_low|netflow|quant\b|price ranking|"mixed"|币安|欧易|火币|U本位|永续|btc|ethusdt|\beth\b|altcoin|ethereum

| path | line | token | disposition | owner | reason |
|---|---|---|---|---|---|
| .github/ISSUE_TEMPLATE/bug_report.md | 118 | `btc` | KEEP | CR-B | config/doc/script file — not shipped code, KEEP byte-identical |
| .github/ISSUE_TEMPLATE/bug_report.md | 119 | `altcoin` | KEEP | CR-B | config/doc/script file — not shipped code, KEEP byte-identical |
| agent/agent_config_load_test.go | 15 | `BTC` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| agent/backend_logs_test.go | 32 | `okx` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| agent/central_brain.go | 1177 | `btc` | KEEP | CR-B | risk-cap knob family (union-gate risk-cap canary) |
| agent/central_brain.go | 1178 | `altcoin` | KEEP | CR-B | risk-cap knob family (union-gate risk-cap canary) |
| agent/central_brain.go | 1200 | `btc` | KEEP | CR-B | risk-cap knob family (union-gate risk-cap canary) |
| agent/central_brain.go | 1201 | `altcoin` | KEEP | CR-B | risk-cap knob family (union-gate risk-cap canary) |
| agent/clear_memory_test.go | 38 | `claw402` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| agent/clear_memory_test.go | 42 | `claw402` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| agent/clear_memory_test.go | 60 | `claw402` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| agent/config_tools_test.go | 30 | `binance` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| agent/config_tools_test.go | 47 | `binance` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| agent/config_tools_test.go | 227 | `binance` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| agent/config_visibility_test.go | 177 | `AI500` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| agent/config_visibility_test.go | 187 | `btc` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| agent/config_visibility_test.go | 200 | `BTC` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| agent/config_visibility_test.go | 201 | `btc` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| agent/config_visibility_test.go | 263 | `Binance` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| agent/config_visibility_test.go | 339 | `hyperliquid` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| agent/config_visibility_test.go | 372 | `ETHUSDT` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| agent/config_visibility_test.go | 434 | `AI500` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| agent/config_visibility_test.go | 537 | `btc` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| agent/config_visibility_test.go | 618 | `BTC` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| agent/config_visibility_test.go | 663 | `BTC` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| agent/memory_test.go | 27 | `OKX` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| agent/memory_test.go | 52 | `BTC` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| agent/onboard_test.go | 19 | `okx` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| agent/planner_runtime_state_test.go | 50 | `OKX` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| agent/planner_runtime_state_test.go | 52 | `BTC` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| agent/planner_runtime_state_test.go | 122 | `BTC` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| agent/planner_runtime_state_test.go | 171 | `okx` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| agent/planner_runtime_state_test.go | 197 | `okx` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| agent/planner_runtime_state_test.go | 587 | `binance` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| agent/planner_tools_test.go | 27 | `okx` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| agent/planner_tools_test.go | 57 | `BTC` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| agent/skill_dispatcher_test.go | 31 | `okx` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| agent/skill_dispatcher_test.go | 132 | `binance` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| agent/skill_dispatcher_test.go | 187 | `OKX` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| agent/skill_dispatcher_test.go | 282 | `binance` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| agent/skill_dispatcher_test.go | 328 | `okx` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| agent/skill_dispatcher_test.go | 429 | `BTC` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| agent/skill_dispatcher_test.go | 446 | `BTC` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| agent/skill_dispatcher_test.go | 482 | `Binance` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| agent/skill_dispatcher_test.go | 484 | `binance` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| agent/skill_dispatcher_test.go | 485 | `Binance` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| agent/skill_dispatcher_test.go | 488 | `binance` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| agent/skill_dispatcher_test.go | 491 | `Binance` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| agent/skill_dispatcher_test.go | 492 | `binance` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| agent/skill_dispatcher_test.go | 494 | `OKX` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| agent/skill_dispatcher_test.go | 496 | `okx` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| agent/skill_dispatcher_test.go | 497 | `OKX` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| agent/skill_dispatcher_test.go | 500 | `okx` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| agent/skill_dispatcher_test.go | 503 | `OKX` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| agent/skill_dispatcher_test.go | 504 | `okx` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| agent/skill_dispatcher_test.go | 526 | `binance` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| agent/skill_dispatcher_test.go | 536 | `OKX` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| agent/skill_dispatcher_test.go | 545 | `okx` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| agent/skill_dispatcher_test.go | 601 | `okx` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| agent/skill_dispatcher_test.go | 610 | `okx` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| agent/skill_dispatcher_test.go | 619 | `okx` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| agent/skill_dispatcher_test.go | 659 | `okx` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| agent/skill_domain_context.go | 41 | `wallet` | KEEP | CR-B | skill-handler surface — cut deferred until DS-107 consumer cuts land |
| agent/skill_execution_handlers.go | 315 | `Hyperliquid` | KEEP | CR-B | skill-handler surface — cut deferred until DS-107 consumer cuts land |
| agent/skill_execution_handlers.go | 319 | `Wallet` | KEEP | CR-B | skill-handler surface — cut deferred until DS-107 consumer cuts land |
| agent/skill_execution_handlers.go | 326 | `Hyperliquid` | KEEP | CR-B | skill-handler surface — cut deferred until DS-107 consumer cuts land |
| agent/skill_execution_handlers.go | 327 | `Wallet` | KEEP | CR-B | skill-handler surface — cut deferred until DS-107 consumer cuts land |
| agent/skill_execution_handlers.go | 350 | `Hyperliquid` | KEEP | CR-B | skill-handler surface — cut deferred until DS-107 consumer cuts land |
| agent/skill_execution_handlers.go | 351 | `hyperliquid` | KEEP | CR-B | skill-handler surface — cut deferred until DS-107 consumer cuts land |
| agent/skill_execution_handlers.go | 362 | `Wallet` | KEEP | CR-B | skill-handler surface — cut deferred until DS-107 consumer cuts land |
| agent/skill_execution_handlers.go | 363 | `wallet` | KEEP | CR-B | skill-handler surface — cut deferred until DS-107 consumer cuts land |
| agent/skill_execution_handlers.go | 392 | `Hyperliquid` | KEEP | CR-B | skill-handler surface — cut deferred until DS-107 consumer cuts land |
| agent/skill_execution_handlers.go | 393 | `Hyperliquid` | KEEP | CR-B | skill-handler surface — cut deferred until DS-107 consumer cuts land |
| agent/skill_execution_handlers.go | 404 | `Wallet` | KEEP | CR-B | skill-handler surface — cut deferred until DS-107 consumer cuts land |
| agent/skill_execution_handlers.go | 405 | `Wallet` | KEEP | CR-B | skill-handler surface — cut deferred until DS-107 consumer cuts land |
| agent/skill_execution_handlers.go | 430 | `Hyperliquid` | KEEP | CR-B | skill-handler surface — cut deferred until DS-107 consumer cuts land |
| agent/skill_execution_handlers.go | 434 | `Wallet` | KEEP | CR-B | skill-handler surface — cut deferred until DS-107 consumer cuts land |
| agent/skill_execution_handlers.go | 561 | `btc` | KEEP | CR-B | risk-cap knob family (union-gate risk-cap canary) |
| agent/skill_execution_handlers.go | 563 | `BTC` | KEEP | CR-B | risk-cap knob family (union-gate risk-cap canary) |
| agent/skill_execution_handlers.go | 565 | `BTC` | KEEP | CR-B | risk-cap knob family (union-gate risk-cap canary) |
| agent/skill_execution_handlers.go | 566 | `altcoin` | KEEP | CR-B | risk-cap knob family (union-gate risk-cap canary) |
| agent/skill_execution_handlers.go | 570 | `altcoin` | KEEP | CR-B | risk-cap knob family (union-gate risk-cap canary) |
| agent/skill_execution_handlers.go | 571 | `btc` | KEEP | CR-B | risk-cap knob family (union-gate risk-cap canary) |
| agent/skill_execution_handlers.go | 573 | `BTC` | KEEP | CR-B | risk-cap knob family (union-gate risk-cap canary) |
| agent/skill_execution_handlers.go | 575 | `BTC` | KEEP | CR-B | risk-cap knob family (union-gate risk-cap canary) |
| agent/skill_execution_handlers.go | 576 | `altcoin` | KEEP | CR-B | risk-cap knob family (union-gate risk-cap canary) |
| agent/skill_execution_handlers.go | 580 | `altcoin` | KEEP | CR-B | risk-cap knob family (union-gate risk-cap canary) |
| agent/skill_execution_handlers.go | 812 | `BTC` | KEEP | CR-B | risk-cap knob family (union-gate risk-cap canary) |
| agent/skill_execution_handlers.go | 813 | `Altcoin` | KEEP | CR-B | risk-cap knob family (union-gate risk-cap canary) |
| agent/skill_execution_handlers.go | 814 | `btc` | KEEP | CR-B | risk-cap knob family (union-gate risk-cap canary) |
| agent/skill_execution_handlers.go | 817 | `BTC` | KEEP | CR-B | risk-cap knob family (union-gate risk-cap canary) |
| agent/skill_execution_handlers.go | 819 | `BTC` | KEEP | CR-B | risk-cap knob family (union-gate risk-cap canary) |
| agent/skill_execution_handlers.go | 820 | `altcoin` | KEEP | CR-B | risk-cap knob family (union-gate risk-cap canary) |
| agent/skill_execution_handlers.go | 825 | `Altcoin` | KEEP | CR-B | risk-cap knob family (union-gate risk-cap canary) |
| agent/skill_execution_handlers.go | 826 | `btc` | KEEP | CR-B | risk-cap knob family (union-gate risk-cap canary) |
| agent/skill_execution_handlers.go | 828 | `altcoin` | KEEP | CR-B | risk-cap knob family (union-gate risk-cap canary) |
| agent/skill_execution_handlers.go | 1514 | `wallet` | KEEP | CR-B | skill-handler surface — cut deferred until DS-107 consumer cuts land |
| agent/skill_execution_handlers.go | 1524 | `hyperliquid` | KEEP | CR-B | skill-handler surface — cut deferred until DS-107 consumer cuts land |
| agent/skill_execution_handlers.go | 1528 | `wallet` | KEEP | CR-B | skill-handler surface — cut deferred until DS-107 consumer cuts land |
| agent/skill_execution_handlers.go | 2396 | `BTC` | KEEP | CR-B | risk-cap knob family (union-gate risk-cap canary) |
| agent/skill_execution_handlers.go | 2399 | `BTC` | KEEP | CR-B | risk-cap knob family (union-gate risk-cap canary) |
| agent/skill_execution_handlers.go | 2401 | `BTC` | KEEP | CR-B | risk-cap knob family (union-gate risk-cap canary) |
| agent/skill_execution_handlers.go | 2403 | `BTC` | KEEP | CR-B | risk-cap knob family (union-gate risk-cap canary) |
| agent/skill_execution_handlers.go | 2437 | `BTC` | KEEP | CR-B | risk-cap knob family (union-gate risk-cap canary) |
| agent/skill_execution_handlers.go | 2438 | `BTC` | KEEP | CR-B | risk-cap knob family (union-gate risk-cap canary) |
| agent/skill_execution_handlers.go | 2500 | `USDT` | KEEP | CR-B | skill-handler surface — cut deferred until DS-107 consumer cuts land |
| agent/skill_execution_handlers.go | 2508 | `USDT` | KEEP | CR-B | skill-handler surface — cut deferred until DS-107 consumer cuts land |
| agent/skill_management_handlers.go | 513 | `btc` | KEEP | CR-B | risk-cap knob family (union-gate risk-cap canary) |
| agent/skill_management_handlers.go | 514 | `btc` | KEEP | CR-B | risk-cap knob family (union-gate risk-cap canary) |
| agent/skill_management_handlers.go | 515 | `altcoin` | KEEP | CR-B | risk-cap knob family (union-gate risk-cap canary) |
| agent/skill_management_handlers.go | 602 | `btc` | KEEP | CR-B | risk-cap knob family (union-gate risk-cap canary) |
| agent/skill_management_handlers.go | 603 | `altcoin` | KEEP | CR-B | risk-cap knob family (union-gate risk-cap canary) |
| agent/skill_management_handlers.go | 684 | `btc` | KEEP | CR-B | risk-cap knob family (union-gate risk-cap canary) |
| agent/skill_management_handlers.go | 685 | `btc` | KEEP | CR-B | risk-cap knob family (union-gate risk-cap canary) |
| agent/skill_management_handlers.go | 686 | `altcoin` | KEEP | CR-B | risk-cap knob family (union-gate risk-cap canary) |
| agent/skill_management_handlers.go | 687 | `altcoin` | KEEP | CR-B | risk-cap knob family (union-gate risk-cap canary) |
| agent/skill_management_handlers.go | 771 | `btc` | KEEP | CR-B | risk-cap knob family (union-gate risk-cap canary) |
| agent/skill_management_handlers.go | 811 | `btc` | KEEP | CR-B | risk-cap knob family (union-gate risk-cap canary) |
| agent/skill_management_handlers.go | 812 | `BTC` | KEEP | CR-B | risk-cap knob family (union-gate risk-cap canary) |
| agent/skill_management_handlers.go | 813 | `altcoin` | KEEP | CR-B | risk-cap knob family (union-gate risk-cap canary) |
| agent/skill_management_handlers.go | 868 | `BTC` | KEEP | CR-B | skill-handler surface — cut deferred until DS-107 consumer cuts land |
| agent/skill_management_handlers.go | 873 | `btc` | KEEP | CR-B | risk-cap knob family (union-gate risk-cap canary) |
| agent/skill_management_handlers.go | 959 | `btc` | KEEP | CR-B | risk-cap knob family (union-gate risk-cap canary) |
| agent/skill_management_handlers.go | 960 | `BTC` | KEEP | CR-B | risk-cap knob family (union-gate risk-cap canary) |
| agent/skill_management_handlers.go | 961 | `altcoin` | KEEP | CR-B | risk-cap knob family (union-gate risk-cap canary) |
| agent/skill_management_handlers.go | 1106 | `BTC` | KEEP | CR-B | risk-cap knob family (union-gate risk-cap canary) |
| agent/skill_management_handlers.go | 1107 | `Altcoin` | KEEP | CR-B | risk-cap knob family (union-gate risk-cap canary) |
| agent/skill_management_handlers.go | 1111 | `BTC` | KEEP | CR-B | risk-cap knob family (union-gate risk-cap canary) |
| agent/skill_management_handlers.go | 1112 | `Altcoin` | KEEP | CR-B | risk-cap knob family (union-gate risk-cap canary) |
| agent/skill_management_handlers.go | 1553 | `BTC` | KEEP | CR-B | risk-cap knob family (union-gate risk-cap canary) |
| agent/skill_management_handlers.go | 1554 | `BTC` | KEEP | CR-B | risk-cap knob family (union-gate risk-cap canary) |
| agent/skill_management_handlers.go | 1601 | `BTC` | KEEP | CR-B | risk-cap knob family (union-gate risk-cap canary) |
| agent/skill_management_handlers.go | 1602 | `BTC` | KEEP | CR-B | risk-cap knob family (union-gate risk-cap canary) |
| agent/skill_management_handlers.go | 2149 | `BTC` | KEEP | CR-B | risk-cap knob family (union-gate risk-cap canary) |
| agent/skill_management_handlers.go | 2150 | `Altcoin` | KEEP | CR-B | risk-cap knob family (union-gate risk-cap canary) |
| agent/skills/exchange_management.json | 31 | `okx` | KEEP | CR-B | config/doc/script file — not shipped code, KEEP byte-identical |
| agent/skills/strategy_diagnosis.json | 16 | `btc` | KEEP | CR-B | config/doc/script file — not shipped code, KEEP byte-identical |
| agent/skills/strategy_management.json | 68 | `btc` | KEEP | CR-B | config/doc/script file — not shipped code, KEEP byte-identical |
| agent/skills/strategy_management.json | 72 | `BTC` | KEEP | CR-B | config/doc/script file — not shipped code, KEEP byte-identical |
| agent/skills/strategy_management.json | 74 | `altcoin` | KEEP | CR-B | config/doc/script file — not shipped code, KEEP byte-identical |
| agent/skills/trader_diagnosis.json | 21 | `btc` | KEEP | CR-B | config/doc/script file — not shipped code, KEEP byte-identical |
| agent/strategy_field_catalog.go | 43 | `btc` | KEEP | CR-B | risk-cap knob family (union-gate risk-cap canary) |
| agent/strategy_field_catalog.go | 44 | `altcoin` | KEEP | CR-B | risk-cap knob family (union-gate risk-cap canary) |
| agent/strategy_field_catalog.go | 102 | `btc` | KEEP | CR-B | risk-cap knob family (union-gate risk-cap canary) |
| agent/strategy_field_catalog.go | 103 | `altcoin` | KEEP | CR-B | risk-cap knob family (union-gate risk-cap canary) |
| agent/tools_test.go | 32 | `btc` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| agent/tools_test.go | 34 | `BTC` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| agent/trade.go | 21 | `USDT` | KEEP | CR-B | futures-active notional caps (CTO ruling, byte-identical) |
| agent/trade.go | 22 | `USDT` | KEEP | CR-B | futures-active notional caps (CTO ruling, byte-identical) |
| agent/trade.go | 327 | `USDT` | KEEP | CR-B | futures-active notional caps (CTO ruling, byte-identical) |
| agent/trade.go | 328 | `USDT` | KEEP | CR-B | futures-active notional caps (CTO ruling, byte-identical) |
| agent/trade.go | 346 | `USDT` | KEEP | CR-B | futures-active notional caps (CTO ruling, byte-identical) |
| agent/trade.go | 357 | `USDT` | KEEP | CR-B | futures-active notional caps (CTO ruling, byte-identical) |
| agent/trade.go | 379 | `Altcoin` | KEEP | CR-B | risk-cap knob family (union-gate risk-cap canary) |
| agent/trade.go | 380 | `Altcoin` | KEEP | CR-B | risk-cap knob family (union-gate risk-cap canary) |
| agent/trade.go | 396 | `USDT` | KEEP | CR-B | futures-active notional caps (CTO ruling, byte-identical) |
| agent/trade.go | 405 | `USDT` | KEEP | CR-B | futures-active notional caps (CTO ruling, byte-identical) |
| agent/trade_admission_test.go | 167 | `BTC` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| agent/trade_admission_test.go | 175 | `BTC` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| agent/trade_cme_route_test.go | 100 | `binance` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| agent/trade_cme_route_test.go | 105 | `USDT` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| agent/trade_cme_route_test.go | 110 | `USDT` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| agent/trade_cme_route_test.go | 137 | `binance` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| agent/trade_cme_route_test.go | 184 | `btc` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| agent/trader_scope_test.go | 102 | `USDT` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| agent/trader_scope_test.go | 112 | `USDT` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| agent/trader_scope_test.go | 194 | `USDT` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| agent/trader_scope_test.go | 243 | `USDT` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| agent/trader_scope_test.go | 304 | `ai500` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| agent/trader_scope_test.go | 306 | `hyper_all` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| agent/trader_scope_test.go | 510 | `okx` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| agent/trader_scope_test.go | 538 | `BTC` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| agent/trader_scope_test.go | 539 | `AI500` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| agent/trader_scope_test.go | 546 | `BTC` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| agent/trader_scope_test.go | 551 | `ai500` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| agent/trader_scope_test.go | 552 | `ai500` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| agent/trader_scope_test.go | 553 | `ai500` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| agent/trader_scope_test.go | 555 | `AI500` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| agent/trader_scope_test.go | 569 | `AI500` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| agent/trader_scope_test.go | 615 | `BTC` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| agent/trader_scope_test.go | 622 | `BTC` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| agent/trader_scope_test.go | 716 | `okx` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| agent/trader_scope_test.go | 738 | `OKX` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| agent/trader_scope_test.go | 749 | `ai500` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| agent/trader_scope_test.go | 751 | `AI500` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| agent/trader_scope_test.go | 785 | `Binance` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| agent/trader_scope_test.go | 787 | `BTC` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| agent/trader_scope_test.go | 803 | `Binance` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| agent/trader_scope_test.go | 805 | `BTC` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| agent/trader_scope_test.go | 829 | `ai500` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| agent/trader_scope_test.go | 830 | `oi_low` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| agent/trader_scope_test.go | 831 | `oi_low` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| agent/trader_scope_test.go | 835 | `btc` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| agent/trader_scope_test.go | 836 | `altcoin` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| agent/trader_scope_test.go | 848 | `BTC` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| agent/trader_scope_test.go | 850 | `BTC` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| agent/trader_scope_test.go | 857 | `BTC` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| agent/trader_scope_test.go | 873 | `BTC` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| agent/trader_scope_test.go | 887 | `BTC` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| agent/trader_scope_test.go | 892 | `BTC` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| agent/trader_scope_test.go | 910 | `BTC` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| agent/trader_scope_test.go | 935 | `BTC` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| agent/trader_scope_test.go | 937 | `BTC` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| agent/trader_scope_test.go | 940 | `BTC` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| agent/trader_scope_test.go | 942 | `BTC` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| agent/trader_scope_test.go | 946 | `BTC` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| agent/trader_scope_test.go | 1028 | `BTC` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| agent/trader_scope_test.go | 1031 | `BTC` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| agent/trader_scope_test.go | 1086 | `ai500` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| agent/trader_scope_test.go | 1109 | `AI500` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| agent/trader_scope_test.go | 1110 | `BTC` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| agent/trader_scope_test.go | 1133 | `ai500` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| agent/trader_scope_test.go | 1142 | `ETHUSDT` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| agent/trader_scope_test.go | 1143 | `ai500` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| agent/trader_scope_test.go | 1297 | `btc` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| agent/trader_scope_test.go | 1298 | `altcoin` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| agent/trader_scope_test.go | 1313 | `AI500` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| agent/trader_scope_test.go | 1325 | `AI500` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| agent/trader_scope_test.go | 1331 | `AI500` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| agent/trader_scope_test.go | 1362 | `btc` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| agent/trader_scope_test.go | 1363 | `altcoin` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| agent/trader_scope_test.go | 1378 | `AI500` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| agent/trader_scope_test.go | 1388 | `AI500` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| agent/trader_scope_test.go | 1421 | `btc` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| agent/trader_scope_test.go | 1422 | `altcoin` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| agent/trader_scope_test.go | 1436 | `AI500` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| agent/trader_scope_test.go | 1469 | `AI500` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| agent/trader_scope_test.go | 1474 | `AI500` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| agent/trader_scope_test.go | 1572 | `AI500` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| agent/trader_scope_test.go | 1609 | `BTC` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| agent/trader_scope_test.go | 1615 | `btc` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| agent/trader_scope_test.go | 1633 | `BTC` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| agent/trader_scope_test.go | 1640 | `AI500` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| agent/trader_scope_test.go | 1645 | `ai500` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| agent/trader_scope_test.go | 1666 | `BTC` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| agent/trader_scope_test.go | 1678 | `AI500` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| agent/trader_scope_test.go | 1685 | `AI500` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| agent/trader_scope_test.go | 1691 | `ai500` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| agent/trader_scope_test.go | 1699 | `AI500` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| agent/trader_scope_test.go | 1712 | `BTC` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| agent/trader_scope_test.go | 1724 | `btc` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| agent/trader_scope_test.go | 1729 | `AI500` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| agent/trader_scope_test.go | 1773 | `ETHUSDT` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| agent/trader_scope_test.go | 1818 | `ETHUSDT` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| agent/trader_scope_test.go | 1917 | `ETHUSDT` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| agent/trader_scope_test.go | 1986 | `okx` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| agent/unified_turn_router_test.go | 18 | `BTC` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| agent/unified_turn_router_test.go | 195 | `BTC` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| api/handler_competition.go | 154 | `wallet` | KEEP | CR-B | NT8 account-snapshot shape — balance key family (CTO ruling, byte-identical) |
| api/handler_competition.go | 418 | `wallet` | KEEP | CR-B | NT8 account-snapshot shape — balance key family (CTO ruling, byte-identical) |
| api/handler_competition.go | 419 | `wallet` | KEEP | CR-B | NT8 account-snapshot shape — balance key family (CTO ruling, byte-identical) |
| api/handler_competition.go | 420 | `wallet` | KEEP | CR-B | NT8 account-snapshot shape — balance key family (CTO ruling, byte-identical) |
| api/handler_competition.go | 432 | `wallet` | KEEP | CR-B | NT8 account-snapshot shape — balance key family (CTO ruling, byte-identical) |
| api/handler_order.go | 102 | `btc` | KEEP | CR-B | risk-cap knob family (union-gate risk-cap canary) |
| api/handler_order.go | 103 | `altcoin` | KEEP | CR-B | risk-cap knob family (union-gate risk-cap canary) |
| api/handler_plan_order_truth.go | 230 | `"mixed"` | KEEP | CR-B | plan-state string constant — same family as the ExecutorVerdict mixed content-assert (KEEP) |
| branding/census_test.go | 47 | `wallet` | KEEP | CR-B | guard fixture — pins the literal and KEEP surfaces byte-for-byte |
| branding/census_test.go | 197 | `wallet` | KEEP | CR-B | guard fixture — pins the literal and KEEP surfaces byte-for-byte |
| branding/census_test.go | 319 | `lighter` | KEEP | CR-B | guard fixture — pins the literal and KEEP surfaces byte-for-byte |
| branding/census_test.go | 403 | `lighter` | KEEP | CR-B | guard fixture — pins the literal and KEEP surfaces byte-for-byte |
| branding/gate_format_test.go | 29 | `bybit` | KEEP | CR-B | guard fixture — pins the literal and KEEP surfaces byte-for-byte |
| branding/gate_format_test.go | 44 | `bybit` | KEEP | CR-B | guard fixture — pins the literal and KEEP surfaces byte-for-byte |
| branding/gate_format_test.go | 67 | `bybit` | KEEP | CR-B | guard fixture — pins the literal and KEEP surfaces byte-for-byte |
| branding/gate_format_test.go | 72 | `bybit` | KEEP | CR-B | guard fixture — pins the literal and KEEP surfaces byte-for-byte |
| branding/gate_format_test.go | 73 | `bybit` | KEEP | CR-B | guard fixture — pins the literal and KEEP surfaces byte-for-byte |
| branding/gate_format_test.go | 74 | `bybit` | KEEP | CR-B | guard fixture — pins the literal and KEEP surfaces byte-for-byte |
| branding/gate_format_test.go | 109 | `bybit` | KEEP | CR-B | guard fixture — pins the literal and KEEP surfaces byte-for-byte |
| branding/gate_format_test.go | 110 | `bybit` | KEEP | CR-B | guard fixture — pins the literal and KEEP surfaces byte-for-byte |
| branding/gate_format_test.go | 126 | `bybit` | KEEP | CR-B | guard fixture — pins the literal and KEEP surfaces byte-for-byte |
| branding/gate_format_test.go | 144 | `bybit` | KEEP | CR-B | guard fixture — pins the literal and KEEP surfaces byte-for-byte |
| branding/gate_format_test.go | 145 | `bybit` | KEEP | CR-B | guard fixture — pins the literal and KEEP surfaces byte-for-byte |
| branding/gate_format_test.go | 161 | `bybit` | KEEP | CR-B | guard fixture — pins the literal and KEEP surfaces byte-for-byte |
| branding/gate_format_test.go | 172 | `bybit` | KEEP | CR-B | guard fixture — pins the literal and KEEP surfaces byte-for-byte |
| branding/gate_format_test.go | 173 | `bybit` | KEEP | CR-B | guard fixture — pins the literal and KEEP surfaces byte-for-byte |
| branding/gate_format_test.go | 190 | `bybit` | KEEP | CR-B | guard fixture — pins the literal and KEEP surfaces byte-for-byte |
| branding/gate_format_test.go | 210 | `bybit` | KEEP | CR-B | guard fixture — pins the literal and KEEP surfaces byte-for-byte |
| branding/gate_format_test.go | 217 | `bybit` | KEEP | CR-B | guard fixture — pins the literal and KEEP surfaces byte-for-byte |
| branding/gate_format_test.go | 218 | `bybit` | KEEP | CR-B | guard fixture — pins the literal and KEEP surfaces byte-for-byte |
| branding/gate_format_test.go | 219 | `bybit` | KEEP | CR-B | guard fixture — pins the literal and KEEP surfaces byte-for-byte |
| branding/gate_format_test.go | 221 | `bybit` | KEEP | CR-B | guard fixture — pins the literal and KEEP surfaces byte-for-byte |
| branding/gate_format_test.go | 222 | `bybit` | KEEP | CR-B | guard fixture — pins the literal and KEEP surfaces byte-for-byte |
| branding/gate_format_test.go | 238 | `bybit` | KEEP | CR-B | guard fixture — pins the literal and KEEP surfaces byte-for-byte |
| branding/gate_format_test.go | 239 | `bybit` | KEEP | CR-B | guard fixture — pins the literal and KEEP surfaces byte-for-byte |
| branding/gate_format_test.go | 265 | `bybit` | KEEP | CR-B | guard fixture — pins the literal and KEEP surfaces byte-for-byte |
| branding/gate_format_test.go | 268 | `bybit` | KEEP | CR-B | guard fixture — pins the literal and KEEP surfaces byte-for-byte |
| branding/gate_format_test.go | 271 | `bybit` | KEEP | CR-B | guard fixture — pins the literal and KEEP surfaces byte-for-byte |
| branding/gate_format_test.go | 288 | `bybit` | KEEP | CR-B | guard fixture — pins the literal and KEEP surfaces byte-for-byte |
| branding/gate_format_test.go | 292 | `bybit` | KEEP | CR-B | guard fixture — pins the literal and KEEP surfaces byte-for-byte |
| branding/gate_format_test.go | 293 | `bybit` | KEEP | CR-B | guard fixture — pins the literal and KEEP surfaces byte-for-byte |
| branding/gate_format_test.go | 295 | `bybit` | KEEP | CR-B | guard fixture — pins the literal and KEEP surfaces byte-for-byte |
| branding/gate_format_test.go | 296 | `bybit` | KEEP | CR-B | guard fixture — pins the literal and KEEP surfaces byte-for-byte |
| branding/gate_format_test.go | 315 | `bybit` | KEEP | CR-B | guard fixture — pins the literal and KEEP surfaces byte-for-byte |
| branding/gate_format_test.go | 323 | `bybit` | KEEP | CR-B | guard fixture — pins the literal and KEEP surfaces byte-for-byte |
| branding/gate_format_test.go | 324 | `bybit` | KEEP | CR-B | guard fixture — pins the literal and KEEP surfaces byte-for-byte |
| branding/gate_format_test.go | 325 | `bybit` | KEEP | CR-B | guard fixture — pins the literal and KEEP surfaces byte-for-byte |
| branding/gate_format_test.go | 327 | `bybit` | KEEP | CR-B | guard fixture — pins the literal and KEEP surfaces byte-for-byte |
| branding/gate_format_test.go | 328 | `bybit` | KEEP | CR-B | guard fixture — pins the literal and KEEP surfaces byte-for-byte |
| branding/gate_format_test.go | 348 | `bybit` | KEEP | CR-B | guard fixture — pins the literal and KEEP surfaces byte-for-byte |
| branding/gate_format_test.go | 349 | `bybit` | KEEP | CR-B | guard fixture — pins the literal and KEEP surfaces byte-for-byte |
| branding/gate_format_test.go | 350 | `bybit` | KEEP | CR-B | guard fixture — pins the literal and KEEP surfaces byte-for-byte |
| branding/gate_format_test.go | 352 | `bybit` | KEEP | CR-B | guard fixture — pins the literal and KEEP surfaces byte-for-byte |
| branding/gate_format_test.go | 353 | `bybit` | KEEP | CR-B | guard fixture — pins the literal and KEEP surfaces byte-for-byte |
| branding/gate_format_test.go | 354 | `bybit` | KEEP | CR-B | guard fixture — pins the literal and KEEP surfaces byte-for-byte |
| branding/no_crypto.go | 16 | `aster` | KEEP | CR-B | the guard's own file — the literal's home (byte-identical) |
| branding/no_crypto.go | 17 | `quant` | KEEP | CR-B | the guard's own file — the literal's home (byte-identical) |
| branding/no_crypto.go | 18 | `BTC` | KEEP | CR-B | the guard's own file — the literal's home (byte-identical) |
| branding/no_crypto.go | 19 | `BTC` | KEEP | CR-B | the guard's own file — the literal's home (byte-identical) |
| branding/no_crypto.go | 20 | `BTC` | KEEP | CR-B | the guard's own file — the literal's home (byte-identical) |
| branding/no_crypto.go | 24 | `btc` | KEEP | CR-B | the guard's own file — the literal's home (byte-identical) |
| branding/no_crypto.go | 25 | `BTC` | KEEP | CR-B | the guard's own file — the literal's home (byte-identical) |
| branding/no_crypto.go | 26 | `btc` | KEEP | CR-B | the guard's own file — the literal's home (byte-identical) |
| branding/no_crypto.go | 28 | `ethusdt` | KEEP | CR-B | the guard's own file — the literal's home (byte-identical) |
| branding/no_crypto.go | 30 | `ETH` | KEEP | CR-B | the guard's own file — the literal's home (byte-identical) |
| branding/no_crypto.go | 31 | `eth` | KEEP | CR-B | the guard's own file — the literal's home (byte-identical) |
| branding/no_crypto.go | 32 | `altcoin` | KEEP | CR-B | the guard's own file — the literal's home (byte-identical) |
| branding/no_crypto.go | 34 | `ethereum` | KEEP | CR-B | the guard's own file — the literal's home (byte-identical) |
| branding/no_crypto.go | 39 | `binance` | KEEP | CR-B | the guard's own file — the literal's home (byte-identical) |
| branding/no_crypto.go | 44 | `binance` | KEEP | CR-B | the guard's own file — the literal's home (byte-identical) |
| branding/no_crypto.go | 51 | `aster` | KEEP | CR-B | the guard's own file — the literal's home (byte-identical) |
| branding/no_crypto.go | 52 | `hyperliquid` | KEEP | CR-B | the guard's own file — the literal's home (byte-identical) |
| branding/no_crypto.go | 53 | `lighter` | KEEP | CR-B | the guard's own file — the literal's home (byte-identical) |
| branding/no_crypto.go | 54 | `coinank` | KEEP | CR-B | the guard's own file — the literal's home (byte-identical) |
| branding/no_crypto.go | 55 | `wallet` | KEEP | CR-B | the guard's own file — the literal's home (byte-identical) |
| branding/no_crypto.go | 62 | `binance` | KEEP | CR-B | the guard's own file — the literal's home (byte-identical) |
| branding/no_crypto.go | 63 | `lighter` | KEEP | CR-B | the guard's own file — the literal's home (byte-identical) |
| branding/no_crypto.go | 69 | `Altcoin` | KEEP | CR-B | the guard's own file — the literal's home (byte-identical) |
| branding/no_crypto.go | 78 | `BTC` | KEEP | CR-B | the guard's own file — the literal's home (byte-identical) |
| branding/no_crypto.go | 79 | `Altcoin` | KEEP | CR-B | the guard's own file — the literal's home (byte-identical) |
| branding/no_crypto.go | 80 | `BTC` | KEEP | CR-B | the guard's own file — the literal's home (byte-identical) |
| branding/no_crypto.go | 81 | `Altcoin` | KEEP | CR-B | the guard's own file — the literal's home (byte-identical) |
| branding/no_crypto.go | 82 | `btc` | KEEP | CR-B | the guard's own file — the literal's home (byte-identical) |
| branding/no_crypto.go | 83 | `altcoin` | KEEP | CR-B | the guard's own file — the literal's home (byte-identical) |
| branding/no_crypto.go | 84 | `btc` | KEEP | CR-B | the guard's own file — the literal's home (byte-identical) |
| branding/no_crypto.go | 85 | `altcoin` | KEEP | CR-B | the guard's own file — the literal's home (byte-identical) |
| branding/no_crypto.go | 87 | `BTC` | KEEP | CR-B | the guard's own file — the literal's home (byte-identical) |
| branding/no_crypto.go | 88 | `Altcoin` | KEEP | CR-B | the guard's own file — the literal's home (byte-identical) |
| branding/no_crypto.go | 89 | `BTC` | KEEP | CR-B | the guard's own file — the literal's home (byte-identical) |
| branding/no_crypto.go | 90 | `Altcoin` | KEEP | CR-B | the guard's own file — the literal's home (byte-identical) |
| branding/no_crypto.go | 92 | `Altcoin` | KEEP | CR-B | the guard's own file — the literal's home (byte-identical) |
| branding/no_crypto.go | 93 | `Altcoin` | KEEP | CR-B | the guard's own file — the literal's home (byte-identical) |
| branding/no_crypto.go | 95 | `altcoin` | KEEP | CR-B | the guard's own file — the literal's home (byte-identical) |
| branding/no_crypto.go | 96 | `altcoin` | KEEP | CR-B | the guard's own file — the literal's home (byte-identical) |
| branding/no_crypto.go | 100 | `"mixed"` | KEEP | CR-B | the guard's own file — the literal's home (byte-identical) |
| branding/no_crypto.go | 113 | `wallet` | KEEP | CR-B | the guard's own file — the literal's home (byte-identical) |
| branding/no_crypto_test.go | 18 | `BTC` | KEEP | CR-B | guard fixture — pins the literal and KEEP surfaces byte-for-byte |
| branding/no_crypto_test.go | 19 | `aster` | KEEP | CR-B | guard fixture — pins the literal and KEEP surfaces byte-for-byte |
| branding/no_crypto_test.go | 20 | `"mixed"` | KEEP | CR-B | guard fixture — pins the literal and KEEP surfaces byte-for-byte |
| branding/no_crypto_test.go | 22 | `btc` | KEEP | CR-B | guard fixture — pins the literal and KEEP surfaces byte-for-byte |
| branding/no_crypto_test.go | 23 | `ethusdt` | KEEP | CR-B | guard fixture — pins the literal and KEEP surfaces byte-for-byte |
| branding/no_crypto_test.go | 31 | `aster` | KEEP | CR-B | guard fixture — pins the literal and KEEP surfaces byte-for-byte |
| branding/no_crypto_test.go | 32 | `"mixed"` | KEEP | CR-B | guard fixture — pins the literal and KEEP surfaces byte-for-byte |
| branding/no_crypto_test.go | 33 | `oi_top` | KEEP | CR-B | guard fixture — pins the literal and KEEP surfaces byte-for-byte |
| branding/no_crypto_test.go | 34 | `BTC` | KEEP | CR-B | guard fixture — pins the literal and KEEP surfaces byte-for-byte |
| branding/no_crypto_test.go | 36 | `BTC` | KEEP | CR-B | guard fixture — pins the literal and KEEP surfaces byte-for-byte |
| branding/no_crypto_test.go | 37 | `btc` | KEEP | CR-B | guard fixture — pins the literal and KEEP surfaces byte-for-byte |
| branding/no_crypto_test.go | 38 | `BTC` | KEEP | CR-B | guard fixture — pins the literal and KEEP surfaces byte-for-byte |
| branding/no_crypto_test.go | 40 | `ETHUSDT` | KEEP | CR-B | guard fixture — pins the literal and KEEP surfaces byte-for-byte |
| branding/no_crypto_test.go | 41 | `BTC` | KEEP | CR-B | guard fixture — pins the literal and KEEP surfaces byte-for-byte |
| branding/no_crypto_test.go | 42 | `Altcoin` | KEEP | CR-B | guard fixture — pins the literal and KEEP surfaces byte-for-byte |
| branding/no_crypto_test.go | 43 | `ethereum` | KEEP | CR-B | guard fixture — pins the literal and KEEP surfaces byte-for-byte |
| branding/no_crypto_test.go | 54 | `eth` | KEEP | CR-B | guard fixture — pins the literal and KEEP surfaces byte-for-byte |
| branding/no_crypto_test.go | 72 | `binance` | KEEP | CR-B | guard fixture — pins the literal and KEEP surfaces byte-for-byte |
| branding/no_crypto_test.go | 81 | `BTC` | KEEP | CR-B | guard fixture — pins the literal and KEEP surfaces byte-for-byte |
| branding/no_crypto_test.go | 109 | `BTC` | KEEP | CR-B | guard fixture — pins the literal and KEEP surfaces byte-for-byte |
| branding/no_crypto_test.go | 112 | `btc` | KEEP | CR-B | guard fixture — pins the literal and KEEP surfaces byte-for-byte |
| branding/no_crypto_test.go | 114 | `BTC` | KEEP | CR-B | guard fixture — pins the literal and KEEP surfaces byte-for-byte |
| branding/no_crypto_test.go | 140 | `BTC` | KEEP | CR-B | guard fixture — pins the literal and KEEP surfaces byte-for-byte |
| branding/no_crypto_test.go | 141 | `Altcoin` | KEEP | CR-B | guard fixture — pins the literal and KEEP surfaces byte-for-byte |
| branding/no_crypto_test.go | 142 | `BTC` | KEEP | CR-B | guard fixture — pins the literal and KEEP surfaces byte-for-byte |
| branding/no_crypto_test.go | 143 | `Altcoin` | KEEP | CR-B | guard fixture — pins the literal and KEEP surfaces byte-for-byte |
| branding/no_crypto_test.go | 144 | `btc` | KEEP | CR-B | guard fixture — pins the literal and KEEP surfaces byte-for-byte |
| branding/no_crypto_test.go | 145 | `altcoin` | KEEP | CR-B | guard fixture — pins the literal and KEEP surfaces byte-for-byte |
| branding/no_crypto_test.go | 146 | `btc` | KEEP | CR-B | guard fixture — pins the literal and KEEP surfaces byte-for-byte |
| branding/no_crypto_test.go | 147 | `altcoin` | KEEP | CR-B | guard fixture — pins the literal and KEEP surfaces byte-for-byte |
| branding/no_crypto_test.go | 148 | `BTC` | KEEP | CR-B | guard fixture — pins the literal and KEEP surfaces byte-for-byte |
| branding/no_crypto_test.go | 149 | `Altcoin` | KEEP | CR-B | guard fixture — pins the literal and KEEP surfaces byte-for-byte |
| branding/no_crypto_test.go | 150 | `BTC` | KEEP | CR-B | guard fixture — pins the literal and KEEP surfaces byte-for-byte |
| branding/no_crypto_test.go | 151 | `Altcoin` | KEEP | CR-B | guard fixture — pins the literal and KEEP surfaces byte-for-byte |
| branding/no_crypto_test.go | 152 | `Altcoin` | KEEP | CR-B | guard fixture — pins the literal and KEEP surfaces byte-for-byte |
| branding/no_crypto_test.go | 153 | `Altcoin` | KEEP | CR-B | guard fixture — pins the literal and KEEP surfaces byte-for-byte |
| branding/no_crypto_test.go | 154 | `altcoin` | KEEP | CR-B | guard fixture — pins the literal and KEEP surfaces byte-for-byte |
| branding/no_crypto_test.go | 155 | `altcoin` | KEEP | CR-B | guard fixture — pins the literal and KEEP surfaces byte-for-byte |
| branding/scope_test.go | 48 | `wallet` | KEEP | CR-B | guard fixture — pins the literal and KEEP surfaces byte-for-byte |
| branding/scope_test.go | 50 | `claw402` | KEEP | CR-B | guard fixture — pins the literal and KEEP surfaces byte-for-byte |
| branding/scope_test.go | 51 | `claw402` | KEEP | CR-B | guard fixture — pins the literal and KEEP surfaces byte-for-byte |
| branding/scope_test.go | 52 | `wallet` | KEEP | CR-B | guard fixture — pins the literal and KEEP surfaces byte-for-byte |
| branding/scope_test.go | 53 | `binance` | KEEP | CR-B | guard fixture — pins the literal and KEEP surfaces byte-for-byte |
| branding/scope_test.go | 54 | `wallet` | KEEP | CR-B | guard fixture — pins the literal and KEEP surfaces byte-for-byte |
| branding/scope_test.go | 242 | `wallet` | KEEP | CR-B | guard fixture — pins the literal and KEEP surfaces byte-for-byte |
| cmd/picture_htf_replay/main.go | 21 | `ETH` | KEEP | CR-B | CME ETH session acronym (Electronic Trading Hours) — not the venue token |
| cmd/picture_htf_replay/main.go | 109 | `ETH` | KEEP | CR-B | CME ETH session acronym (Electronic Trading Hours) — not the venue token |
| cmd/picture_htf_replay/main.go | 112 | `ETH` | KEEP | CR-B | CME ETH session acronym (Electronic Trading Hours) — not the venue token |
| cmd/picture_htf_replay/main.go | 355 | `ETH` | KEEP | CR-B | CME ETH session acronym (Electronic Trading Hours) — not the venue token |
| cmd/picture_htf_replay/main.go | 356 | `ETH` | KEEP | CR-B | CME ETH session acronym (Electronic Trading Hours) — not the venue token |
| cmd/picture_htf_replay/main.go | 360 | `ETH` | KEEP | CR-B | CME ETH session acronym (Electronic Trading Hours) — not the venue token |
| docs/superpowers/AUDIT-CHECKLIST.md | 196 | `ethereum` | KEEP | CR-B | config/doc/script file — not shipped code, KEEP byte-identical |
| docs/superpowers/AUDIT-CHECKLIST.md | 541 | `USDT` | KEEP | CR-B | config/doc/script file — not shipped code, KEEP byte-identical |
| docs/superpowers/AUDIT-CHECKLIST.md | 545 | `USDT` | KEEP | CR-B | config/doc/script file — not shipped code, KEEP byte-identical |
| docs/superpowers/AUDIT-CHECKLIST.md | 1297 | `binance` | KEEP | CR-B | config/doc/script file — not shipped code, KEEP byte-identical |
| docs/superpowers/AUDIT-CHECKLIST.md | 6315 | `BINANCE` | KEEP | CR-B | config/doc/script file — not shipped code, KEEP byte-identical |
| docs/superpowers/AUDIT-CHECKLIST.md | 6316 | `Binance` | KEEP | CR-B | config/doc/script file — not shipped code, KEEP byte-identical |
| docs/superpowers/AUDIT-CHECKLIST.md | 6318 | `BINANCE` | KEEP | CR-B | config/doc/script file — not shipped code, KEEP byte-identical |
| docs/superpowers/AUDIT-CHECKLIST.md | 6322 | `BINANCE` | KEEP | CR-B | config/doc/script file — not shipped code, KEEP byte-identical |
| docs/superpowers/AUDIT-CHECKLIST.md | 6325 | `BINANCE` | KEEP | CR-B | config/doc/script file — not shipped code, KEEP byte-identical |
| docs/superpowers/AUDIT-CHECKLIST.md | 6336 | `BINANCE` | KEEP | CR-B | config/doc/script file — not shipped code, KEEP byte-identical |
| docs/superpowers/AUDIT-CHECKLIST.md | 6337 | `CoinAnk` | KEEP | CR-B | config/doc/script file — not shipped code, KEEP byte-identical |
| docs/superpowers/AUDIT-CHECKLIST.md | 6339 | `BTC` | KEEP | CR-B | config/doc/script file — not shipped code, KEEP byte-identical |
| docs/superpowers/AUDIT-CHECKLIST.md | 6814 | `USDT` | KEEP | CR-B | config/doc/script file — not shipped code, KEEP byte-identical |
| docs/superpowers/AUDIT-CHECKLIST.md | 6817 | `USDT` | KEEP | CR-B | config/doc/script file — not shipped code, KEEP byte-identical |
| docs/superpowers/AUDIT-CHECKLIST.md | 6900 | `USDT` | KEEP | CR-B | config/doc/script file — not shipped code, KEEP byte-identical |
| docs/superpowers/AUDIT-CHECKLIST.md | 7197 | `wallet` | KEEP | CR-B | config/doc/script file — not shipped code, KEEP byte-identical |
| hook/README.md | 60 | `BINANCE` | KEEP | CR-B | config/doc/script file — not shipped code, KEEP byte-identical |
| hook/README.md | 62 | `binance` | KEEP | CR-B | config/doc/script file — not shipped code, KEEP byte-identical |
| hook/README.md | 66 | `Binance` | KEEP | CR-B | config/doc/script file — not shipped code, KEEP byte-identical |
| hook/README.md | 68 | `Binance` | KEEP | CR-B | config/doc/script file — not shipped code, KEEP byte-identical |
| hook/README.md | 74 | `Binance` | KEEP | CR-B | config/doc/script file — not shipped code, KEEP byte-identical |
| hook/README.md | 78 | `Aster` | KEEP | CR-B | config/doc/script file — not shipped code, KEEP byte-identical |
| hook/README.md | 92 | `Aster` | KEEP | CR-B | config/doc/script file — not shipped code, KEEP byte-identical |
| hook/README.md | 116 | `Binance` | KEEP | CR-B | config/doc/script file — not shipped code, KEEP byte-identical |
| hook/README.md | 117 | `BINANCE` | KEEP | CR-B | config/doc/script file — not shipped code, KEEP byte-identical |
| hook/README.md | 126 | `Binance` | KEEP | CR-B | config/doc/script file — not shipped code, KEEP byte-identical |
| hook/README.md | 211 | `BINANCE` | KEEP | CR-B | config/doc/script file — not shipped code, KEEP byte-identical |
| hook/README.md | 270 | `binance` | KEEP | CR-B | config/doc/script file — not shipped code, KEEP byte-identical |
| internal/updateauth/census_mint_test.go | 65 | `okx` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| internal/updateauth/census_mint_test.go | 291 | `okx` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| kernel/engine.go | 136 | `BTC` | KEEP | CR-B | risk-cap knob family (union-gate risk-cap canary) |
| kernel/engine.go | 137 | `Altcoin` | KEEP | CR-B | risk-cap knob family (union-gate risk-cap canary) |
| kernel/engine_analysis.go | 559 | `BTC` | KEEP | CR-B | risk-cap knob family (union-gate risk-cap canary) |
| kernel/engine_analysis.go | 560 | `Altcoin` | KEEP | CR-B | risk-cap knob family (union-gate risk-cap canary) |
| kernel/engine_analysis.go | 561 | `BTC` | KEEP | CR-B | risk-cap knob family (union-gate risk-cap canary) |
| kernel/engine_analysis.go | 562 | `Altcoin` | KEEP | CR-B | risk-cap knob family (union-gate risk-cap canary) |
| kernel/engine_analysis.go | 836 | `BINANCE` | KEEP | CR-B | wave-name comment (W-NO-BINANCE) — documents the OI-absent-on-MNQ behavior |
| kernel/engine_analysis.go | 862 | `btc` | KEEP | CR-B | risk-cap knob family (union-gate risk-cap canary) |
| kernel/engine_analysis.go | 877 | `btc` | KEEP | CR-B | risk-cap knob family (union-gate risk-cap canary) |
| kernel/engine_position.go | 31 | `btc` | KEEP | CR-B | risk-cap knob family (union-gate risk-cap canary) |
| kernel/engine_position.go | 33 | `btc` | KEEP | CR-B | risk-cap knob family (union-gate risk-cap canary) |
| kernel/engine_position.go | 43 | `btc` | KEEP | CR-B | risk-cap knob family (union-gate risk-cap canary) |
| kernel/engine_position.go | 58 | `altcoin` | KEEP | CR-B | risk-cap knob family (union-gate risk-cap canary) |
| kernel/engine_position.go | 59 | `altcoin` | KEEP | CR-B | risk-cap knob family (union-gate risk-cap canary) |
| kernel/engine_position.go | 73 | `BTC` | KEEP | CR-B | BTC/ETH major-pair classification — same-class vocabulary per the amendment |
| kernel/engine_position.go | 74 | `btc` | KEEP | CR-B | risk-cap knob family (union-gate risk-cap canary) |
| kernel/engine_position.go | 75 | `btc` | KEEP | CR-B | risk-cap knob family (union-gate risk-cap canary) |
| kernel/engine_position.go | 92 | `BTC` | KEEP | CR-B | risk-cap knob family (union-gate risk-cap canary) |
| kernel/engine_position.go | 94 | `BTC` | KEEP | CR-B | BTC/ETH major-pair classification — same-class vocabulary per the amendment |
| kernel/engine_position.go | 95 | `BTC` | KEEP | CR-B | risk-cap knob family (union-gate risk-cap canary) |
| kernel/engine_position.go | 96 | `USDT` | KEEP | CR-B | risk-cap knob family (union-gate risk-cap canary) |
| kernel/engine_position.go | 100 | `USDT` | KEEP | CR-B | legacy crypto-path min-position validation (unreachable in futures mode — futures uses enforceMinPositionSize) |
| kernel/engine_position.go | 109 | `BTC` | KEEP | CR-B | BTC/ETH major-pair classification — same-class vocabulary per the amendment |
| kernel/engine_position.go | 110 | `BTC` | KEEP | CR-B | risk-cap knob family (union-gate risk-cap canary) |
| kernel/engine_position.go | 112 | `altcoin` | KEEP | CR-B | risk-cap knob family (union-gate risk-cap canary) |
| kernel/engine_position_futures_test.go | 6 | `btc` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| kernel/engine_position_futures_test.go | 7 | `btc` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| kernel/engine_position_futures_test.go | 15 | `btc` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| kernel/engine_position_futures_test.go | 24 | `btc` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| kernel/engine_position_futures_test.go | 33 | `btc` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| kernel/engine_position_futures_test.go | 42 | `btc` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| kernel/engine_position_futures_test.go | 51 | `btc` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| kernel/engine_position_futures_test.go | 53 | `USDT` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| kernel/engine_position_futures_test.go | 60 | `btc` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| kernel/engine_position_futures_test.go | 61 | `USDT` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| kernel/engine_position_futures_test.go | 68 | `btc` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| kernel/engine_position_futures_test.go | 70 | `btc` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| kernel/engine_prompt.go | 69 | `btc` | KEEP | CR-B | risk-cap knob family (union-gate risk-cap canary) |
| kernel/engine_prompt.go | 70 | `btc` | KEEP | CR-B | risk-cap knob family (union-gate risk-cap canary) |
| kernel/engine_prompt.go | 71 | `btc` | KEEP | CR-B | risk-cap knob family (union-gate risk-cap canary) |
| kernel/engine_prompt.go | 73 | `altcoin` | KEEP | CR-B | risk-cap knob family (union-gate risk-cap canary) |
| kernel/engine_prompt.go | 74 | `altcoin` | KEEP | CR-B | risk-cap knob family (union-gate risk-cap canary) |
| kernel/engine_prompt.go | 75 | `altcoin` | KEEP | CR-B | risk-cap knob family (union-gate risk-cap canary) |
| kernel/engine_prompt.go | 81 | `Altcoin` | KEEP | CR-B | risk-cap knob family (union-gate risk-cap canary) |
| kernel/engine_prompt.go | 82 | `altcoin` | KEEP | CR-B | risk-cap knob family (union-gate risk-cap canary) |
| kernel/engine_prompt.go | 83 | `BTC` | KEEP | CR-B | risk-cap knob family (union-gate risk-cap canary) |
| kernel/engine_prompt.go | 84 | `btc` | KEEP | CR-B | risk-cap knob family (union-gate risk-cap canary) |
| kernel/engine_prompt.go | 86 | `USDT` | KEEP | CR-B | legacy trade-history prompt formatter (unreachable in futures mode — NoCryptoVocab golden green) |
| kernel/engine_prompt.go | 89 | `Altcoin` | KEEP | CR-B | risk-cap knob family (union-gate risk-cap canary) |
| kernel/engine_prompt.go | 90 | `Altcoin` | KEEP | CR-B | risk-cap knob family (union-gate risk-cap canary) |
| kernel/engine_prompt.go | 100 | `BTC` | KEEP | CR-B | risk-cap knob family (union-gate risk-cap canary) |
| kernel/engine_prompt.go | 101 | `btc` | KEEP | CR-B | risk-cap knob family (union-gate risk-cap canary) |
| kernel/engine_prompt.go | 154 | `BTC` | KEEP | CR-B | risk-cap knob family (union-gate risk-cap canary) |
| kernel/engine_prompt.go | 155 | `btc` | KEEP | CR-B | risk-cap knob family (union-gate risk-cap canary) |
| kernel/engine_prompt.go | 156 | `BTC` | KEEP | CR-B | legacy crypto-path prompt example — interlocked with riskControl.BTCETHMaxLeverage (unreachable in futures mode) |
| kernel/engine_prompt.go | 157 | `BTC` | KEEP | CR-B | risk-cap knob family (union-gate risk-cap canary) |
| kernel/engine_prompt.go | 158 | `ETHUSDT` | KEEP | CR-B | legacy trade-history prompt formatter (unreachable in futures mode — NoCryptoVocab golden green) |
| kernel/engine_prompt.go | 262 | `BINANCE` | KEEP | CR-B | wave-name comment (W-NO-BINANCE) — documents the OI-absent-on-MNQ behavior |
| kernel/engine_prompt.go | 302 | `USDT` | KEEP | CR-B | legacy trade-history prompt formatter (unreachable in futures mode — NoCryptoVocab golden green) |
| kernel/engine_prompt.go | 313 | `USDT` | KEEP | CR-B | legacy trade-history prompt formatter (unreachable in futures mode — NoCryptoVocab golden green) |
| kernel/engine_prompt.go | 402 | `ETH` | KEEP | CR-B | CME ETH session acronym (Electronic Trading Hours) — not the venue token |
| kernel/engine_prompt.go | 464 | `USDT` | KEEP | CR-B | legacy trade-history prompt formatter (unreachable in futures mode — NoCryptoVocab golden green) |
| kernel/engine_prompt.go | 621 | `BINANCE` | KEEP | CR-B | wave-name comment (W-NO-BINANCE) — documents the OI-absent-on-MNQ behavior |
| kernel/engine_prompt.go | 631 | `BINANCE` | KEEP | CR-B | wave-name comment (W-NO-BINANCE) — documents the OI-absent-on-MNQ behavior |
| kernel/engine_prompt_futures_test.go | 81 | `USDT` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| kernel/engine_prompt_futures_test.go | 83 | `usdt` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| kernel/engine_prompt_futures_test.go | 165 | `altcoin` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| kernel/funding_suppress_test.go | 32 | `BTC` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| kernel/funding_suppress_test.go | 33 | `BTC` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| kernel/funding_suppress_test.go | 35 | `BTC` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| kernel/funding_suppress_test.go | 44 | `AI500` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| kernel/funding_suppress_test.go | 45 | `OI_Top` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| kernel/funding_suppress_test.go | 48 | `AI500` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| kernel/funding_suppress_test.go | 57 | `AI500` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| kernel/funding_suppress_test.go | 58 | `AI500` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| kernel/funding_suppress_test.go | 62 | `BTC` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| kernel/funding_suppress_test.go | 63 | `AI500` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| kernel/funding_suppress_test.go | 64 | `AI500` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| kernel/live_network_guard_test.go | 5 | `CoinAnk` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| kernel/live_network_guard_test.go | 9 | `coinank` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| kernel/live_network_guard_test.go | 21 | `CoinAnk` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| kernel/live_network_guard_test.go | 22 | `coinank` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| kernel/live_network_guard_test.go | 24 | `coinank` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| kernel/live_network_guard_test.go | 25 | `coinank` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| kernel/live_network_guard_test.go | 45 | `Coinank` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| kernel/live_network_guard_test.go | 47 | `Coinank` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| kernel/live_network_guard_test.go | 123 | `Coinank` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| kernel/live_network_guard_test.go | 154 | `coinank` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| kernel/no_binance_prompt_test.go | 16 | `BINANCE` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| kernel/no_binance_prompt_test.go | 51 | `BTC` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| kernel/no_binance_prompt_test.go | 52 | `BTC` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| kernel/no_binance_prompt_test.go | 57 | `BTC` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| kernel/no_binance_prompt_test.go | 159 | `BTC` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| kernel/no_binance_prompt_test.go | 161 | `BTC` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| kernel/no_binance_prompt_test.go | 162 | `BTC` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| kernel/pnl_truth_pin_test.go | 15 | `USDT` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| kernel/prompt_builder_test.go | 148 | `BTC` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| kernel/prompt_builder_test.go | 200 | `BTC` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| kernel/prompt_builder_test.go | 215 | `BTC` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| kernel/prompt_builder_test.go | 230 | `BTC` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| kernel/prompt_builder_test.go | 249 | `BTC` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| kernel/prompt_builder_test.go | 270 | `BTC` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| kernel/prompt_builder_test.go | 291 | `BTC` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| kernel/prompt_builder_test.go | 293 | `BTC` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| kernel/prompt_builder_test.go | 296 | `ETHUSDT` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| kernel/prompt_builder_test.go | 300 | `ETH` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| kernel/prompt_builder_test.go | 316 | `BTC` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| kernel/prompt_builder_test.go | 453 | `BTC` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| kernel/prompt_builder_test.go | 454 | `ai500` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| kernel/prompt_builder_test.go | 457 | `ETHUSDT` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| kernel/prompt_builder_test.go | 458 | `oi_top` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| kernel/risk_config_truth_test.go | 41 | `BTC` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| kernel/risk_config_truth_test.go | 42 | `BTC` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| kernel/tz_test.go | 84 | `BTC` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| kernel/validate_test.go | 13 | `btc` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| kernel/validate_test.go | 14 | `altcoin` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| kernel/validate_test.go | 19 | `Altcoin` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| kernel/validate_test.go | 21 | `USDT` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| kernel/validate_test.go | 29 | `btc` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| kernel/validate_test.go | 30 | `altcoin` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| kernel/validate_test.go | 35 | `BTC` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| kernel/validate_test.go | 37 | `BTC` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| kernel/validate_test.go | 45 | `btc` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| kernel/validate_test.go | 46 | `altcoin` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| kernel/validate_test.go | 53 | `ETHUSDT` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| kernel/validate_test.go | 61 | `btc` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| kernel/validate_test.go | 62 | `altcoin` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| kernel/validate_test.go | 69 | `USDT` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| kernel/validate_test.go | 77 | `btc` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| kernel/validate_test.go | 78 | `altcoin` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| kernel/validate_test.go | 86 | `BTC` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| kernel/validate_test.go | 87 | `btc` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| market/data.go | 13 | `BINANCE` | KEEP | CR-B | wave-name comment (W-NO-BINANCE) — documents the OI-absent-on-MNQ behavior |
| market/data.go | 21 | `BINANCE` | KEEP | CR-B | wave-name comment (W-NO-BINANCE) — documents the OI-absent-on-MNQ behavior |
| market/data.go | 87 | `BINANCE` | KEEP | CR-B | wave-name comment (W-NO-BINANCE) — documents the OI-absent-on-MNQ behavior |
| market/data.go | 148 | `BINANCE` | KEEP | CR-B | wave-name comment (W-NO-BINANCE) — documents the OI-absent-on-MNQ behavior |
| market/data.go | 354 | `BINANCE` | KEEP | CR-B | wave-name comment (W-NO-BINANCE) — documents the OI-absent-on-MNQ behavior |
| market/data_freshness.go | 18 | `ETH` | KEEP | CR-B | CME ETH session acronym (Electronic Trading Hours) — not the venue token |
| market/data_freshness.go | 28 | `ETH` | KEEP | CR-B | CME ETH session acronym (Electronic Trading Hours) — not the venue token |
| market/data_freshness.go | 144 | `ETH` | KEEP | CR-B | CME ETH session acronym (Electronic Trading Hours) — not the venue token |
| market/data_freshness.go | 148 | `ETH` | KEEP | CR-B | CME ETH session acronym (Electronic Trading Hours) — not the venue token |
| market/data_test.go | 361 | `BTC` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| market/data_test.go | 378 | `USDT` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| market/data_test.go | 410 | `BTC` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| market/data_test.go | 428 | `USDT` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| market/data_test.go | 442 | `USDT` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| market/data_test.go | 463 | `BTC` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| market/data_test.go | 485 | `USDT` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| market/data_test.go | 497 | `BTC` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| market/futures_symbol_test.go | 31 | `BTC` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| market/futures_symbol_test.go | 32 | `BTC` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| market/futures_symbol_test.go | 33 | `ETH` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| market/futures_symbol_test.go | 34 | `ETHUSDT` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| market/futures_symbol_test.go | 61 | `USDT` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| market/futures_symbol_test.go | 65 | `USDT` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| market/futures_symbol_test.go | 77 | `USDT` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| market/futures_symbol_test.go | 117 | `BTC` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| market/no_binance_source_guard_test.go | 14 | `BINANCE` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| market/no_binance_source_guard_test.go | 17 | `Binance` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| market/no_binance_source_guard_test.go | 18 | `Binance` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| market/no_binance_source_guard_test.go | 21 | `binance` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| market/no_binance_source_guard_test.go | 22 | `binance` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| market/no_binance_source_guard_test.go | 24 | `Binance` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| market/no_binance_source_guard_test.go | 28 | `binance` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| market/no_binance_source_guard_test.go | 30 | `binance` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| market/no_binance_source_guard_test.go | 31 | `binance` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| market/no_binance_source_guard_test.go | 32 | `binance` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| market/no_binance_source_guard_test.go | 34 | `Binance` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| market/no_binance_source_guard_test.go | 35 | `binance` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| market/no_binance_source_guard_test.go | 36 | `binance` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| market/no_binance_source_guard_test.go | 40 | `binance` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| market/no_binance_source_guard_test.go | 41 | `binance` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| market/no_binance_source_guard_test.go | 47 | `Binance` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| market/no_binance_source_guard_test.go | 79 | `binance` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| market/no_binance_source_guard_test.go | 84 | `binance` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| market/no_binance_source_guard_test.go | 85 | `Binance` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| market/no_binance_source_guard_test.go | 95 | `binance` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| market/no_binance_source_guard_test.go | 106 | `BINANCE` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| market/no_binance_source_guard_test.go | 107 | `BINANCE` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| market/no_binance_source_guard_test.go | 108 | `BINANCE` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| market/types.go | 38 | `BINANCE` | KEEP | CR-B | wave-name comment (W-NO-BINANCE) — documents the OI-absent-on-MNQ behavior |
| ninjascript/VLBarsSubscriptionManager.cs | 81 | `ETH` | KEEP | CR-B | config/doc/script file — not shipped code, KEEP byte-identical |
| ninjascript/VLBarsSubscriptionManager.cs | 96 | `ETH` | KEEP | CR-B | config/doc/script file — not shipped code, KEEP byte-identical |
| ninjascript/VLBarsSubscriptionManager.cs | 349 | `ETH` | KEEP | CR-B | config/doc/script file — not shipped code, KEEP byte-identical |
| ninjascript/VLBarsSubscriptionManager.cs | 631 | `ETH` | KEEP | CR-B | config/doc/script file — not shipped code, KEEP byte-identical |
| ninjascript/VLContractResolver.cs | 66 | `BTC` | KEEP | CR-B | config/doc/script file — not shipped code, KEEP byte-identical |
| ninjascript/VLContractResolver_VERIFY.md | 20 | `BTC` | KEEP | CR-B | config/doc/script file — not shipped code, KEEP byte-identical |
| ninjascript/VLContractResolver_VERIFY.md | 123 | `BTC` | KEEP | CR-B | config/doc/script file — not shipped code, KEEP byte-identical |
| ninjascript/VLContractResolver_VERIFY.md | 124 | `ETH` | KEEP | CR-B | config/doc/script file — not shipped code, KEEP byte-identical |
| ninjascript/VLHistoryPull.cs | 132 | `ETH` | KEEP | CR-B | config/doc/script file — not shipped code, KEEP byte-identical |
| ninjascript/vltrader_tcp_PROTOCOL.md | 333 | `ETH` | KEEP | CR-B | config/doc/script file — not shipped code, KEEP byte-identical |
| patches/partner-2026-08-20/MANIFEST.md | 458 | `wallet` | KEEP | CR-B | config/doc/script file — not shipped code, KEEP byte-identical |
| provider/ninjatrader/bar_source.go | 18 | `"mixed"` | KEEP | CR-B | bar-source state constant — same family as the ExecutorVerdict mixed content-assert (KEEP) |
| provider/ninjatrader/multisymbol_subscribe_test.go | 133 | `BTC` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| provider/ninjatrader/tcp_framing.go | 675 | `"mixed"` | KEEP | CR-B | bar-source state constant — same family as the ExecutorVerdict mixed content-assert (KEEP) |
| scripts/replay_write_time_feasibility.py | 162 | `"mixed"` | KEEP | CR-B | config/doc/script file — not shipped code, KEEP byte-identical |
| store/arm_state_source_guard_test.go | 105 | `bybit` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| store/arm_state_source_guard_test.go | 106 | `kucoin` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| store/armed_orders_test.go | 51 | `ETH` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| store/bar_history.go | 151 | `"mixed"` | KEEP | CR-B | bar-source state constant — same family as the ExecutorVerdict mixed content-assert (KEEP) |
| store/bar_history_across_roll.go | 82 | `"mixed"` | KEEP | CR-B | bar-source state constant — same family as the ExecutorVerdict mixed content-assert (KEEP) |
| store/knob_registry_table.go | 14 | `altcoin` | KEEP | CR-B | risk-cap knob family (union-gate risk-cap canary) |
| store/knob_registry_table.go | 15 | `altcoin` | KEEP | CR-B | risk-cap knob family (union-gate risk-cap canary) |
| store/knob_registry_table.go | 26 | `btc` | KEEP | CR-B | risk-cap knob family (union-gate risk-cap canary) |
| store/knob_registry_table.go | 27 | `btc` | KEEP | CR-B | risk-cap knob family (union-gate risk-cap canary) |
| store/legacy_crypto_rows_c1_test.go | 40 | `ai500` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| store/legacy_crypto_rows_c1_test.go | 42 | `ai500` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| store/legacy_crypto_rows_c1_test.go | 43 | `ai500` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| store/legacy_crypto_rows_c1_test.go | 44 | `oi_top` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| store/legacy_crypto_rows_c1_test.go | 45 | `oi_top` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| store/legacy_crypto_rows_c1_test.go | 46 | `hyper_all` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| store/legacy_crypto_rows_c1_test.go | 47 | `hyper_main` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| store/legacy_crypto_rows_c1_test.go | 48 | `netflow` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| store/legacy_crypto_rows_c1_test.go | 51 | `BTC` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| store/legacy_crypto_rows_c1_test.go | 62 | `btc` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| store/legacy_crypto_rows_c1_test.go | 63 | `altcoin` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| store/legacy_crypto_rows_c1_test.go | 102 | `hyperliquid` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| store/legacy_crypto_rows_c1_test.go | 103 | `hyperliquid` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| store/legacy_crypto_rows_c1_test.go | 107 | `wallet` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| store/legacy_crypto_rows_c1_test.go | 120 | `binance` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| store/legacy_crypto_rows_c1_test.go | 122 | `binance` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| store/legacy_crypto_rows_c1_test.go | 132 | `hyperliquid` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| store/legacy_crypto_rows_c1_test.go | 133 | `hyperliquid` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| store/legacy_crypto_rows_c1_test.go | 134 | `aster` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| store/legacy_crypto_rows_c1_test.go | 135 | `aster` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| store/legacy_crypto_rows_c1_test.go | 136 | `aster` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| store/legacy_crypto_rows_c1_test.go | 137 | `wallet` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| store/legacy_crypto_rows_c1_test.go | 138 | `lighter` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| store/legacy_crypto_rows_c1_test.go | 139 | `lighter` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| store/legacy_crypto_rows_c1_test.go | 148 | `binance` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| store/legacy_crypto_rows_c1_test.go | 161 | `Hyperliquid` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| store/legacy_crypto_rows_c1_test.go | 162 | `Hyperliquid` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| store/legacy_crypto_rows_c1_test.go | 163 | `aster` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| store/legacy_crypto_rows_c1_test.go | 164 | `aster` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| store/legacy_crypto_rows_c1_test.go | 165 | `Wallet` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| store/legacy_crypto_rows_c1_test.go | 192 | `Hyperliquid` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| store/legacy_crypto_rows_c1_test.go | 193 | `Hyperliquid` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| store/legacy_crypto_rows_c1_test.go | 196 | `Wallet` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| store/legacy_crypto_rows_c1_test.go | 208 | `claw402` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| store/legacy_crypto_rows_c1_test.go | 209 | `Claw402` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| store/legacy_crypto_rows_c1_test.go | 216 | `claw402` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| store/legacy_crypto_rows_c1_test.go | 245 | `binance` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| store/legacy_crypto_rows_c1_test.go | 251 | `Hyperliquid` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| store/legacy_crypto_rows_c1_test.go | 252 | `Hyperliquid` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| store/legacy_crypto_rows_c1_test.go | 255 | `Wallet` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| store/legacy_crypto_rows_c1_test.go | 270 | `Hyperliquid` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| store/legacy_crypto_rows_c1_test.go | 271 | `Hyperliquid` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| store/legacy_crypto_rows_c1_test.go | 275 | `Wallet` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| store/legacy_crypto_rows_c1_test.go | 284 | `hyperliquid` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| store/legacy_crypto_rows_c1_test.go | 286 | `wallet` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| store/legacy_crypto_rows_c1_test.go | 362 | `btc` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| store/legacy_crypto_rows_c1_test.go | 365 | `BTC` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| store/legacy_crypto_rows_c1_test.go | 366 | `BTC` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| store/strategy.go | 49 | `BTC` | KEEP | CR-B | risk-cap knob family (union-gate risk-cap canary) |
| store/strategy.go | 153 | `BTC` | KEEP | CR-B | risk-cap knob family (union-gate risk-cap canary) |
| store/strategy.go | 154 | `BTC` | KEEP | CR-B | risk-cap knob family (union-gate risk-cap canary) |
| store/strategy.go | 156 | `BTC` | KEEP | CR-B | risk-cap knob family (union-gate risk-cap canary) |
| store/strategy.go | 157 | `BTC` | KEEP | CR-B | risk-cap knob family (union-gate risk-cap canary) |
| store/strategy.go | 159 | `Altcoin` | KEEP | CR-B | risk-cap knob family (union-gate risk-cap canary) |
| store/strategy.go | 160 | `Altcoin` | KEEP | CR-B | risk-cap knob family (union-gate risk-cap canary) |
| store/strategy.go | 162 | `Altcoin` | KEEP | CR-B | risk-cap knob family (union-gate risk-cap canary) |
| store/strategy.go | 163 | `Altcoin` | KEEP | CR-B | risk-cap knob family (union-gate risk-cap canary) |
| store/strategy.go | 167 | `BTC` | KEEP | CR-B | risk-cap knob family (union-gate risk-cap canary) |
| store/strategy.go | 168 | `BTC` | KEEP | CR-B | risk-cap knob family (union-gate risk-cap canary) |
| store/strategy.go | 170 | `BTC` | KEEP | CR-B | risk-cap knob family (union-gate risk-cap canary) |
| store/strategy.go | 171 | `BTC` | KEEP | CR-B | risk-cap knob family (union-gate risk-cap canary) |
| store/strategy.go | 173 | `Altcoin` | KEEP | CR-B | risk-cap knob family (union-gate risk-cap canary) |
| store/strategy.go | 174 | `Altcoin` | KEEP | CR-B | risk-cap knob family (union-gate risk-cap canary) |
| store/strategy.go | 176 | `Altcoin` | KEEP | CR-B | risk-cap knob family (union-gate risk-cap canary) |
| store/strategy.go | 177 | `Altcoin` | KEEP | CR-B | risk-cap knob family (union-gate risk-cap canary) |
| store/strategy.go | 217 | `AI500` | KEEP | CR-B | C1 legacy-row normalization — removed source types collapse to static (stored rows must load) |
| store/strategy.go | 224 | `ai500` | KEEP | CR-B | C1 legacy-row normalization — removed source types collapse to static (stored rows must load) |
| store/strategy.go | 260 | `ai500` | KEEP | CR-B | C1 legacy-row normalization — removed source types collapse to static (stored rows must load) |
| store/strategy.go | 261 | `ai500` | KEEP | CR-B | C1 legacy-row normalization — removed source types collapse to static (stored rows must load) |
| store/strategy.go | 263 | `oi_top` | KEEP | CR-B | C1 legacy-row normalization — removed source types collapse to static (stored rows must load) |
| store/strategy.go | 265 | `oi_low` | KEEP | CR-B | C1 legacy-row normalization — removed source types collapse to static (stored rows must load) |
| store/strategy.go | 627 | `BTC` | KEEP | CR-B | risk-cap knob family (union-gate risk-cap canary) |
| store/strategy.go | 628 | `altcoin` | KEEP | CR-B | risk-cap knob family (union-gate risk-cap canary) |
| store/strategy.go | 629 | `BTC` | KEEP | CR-B | risk-cap knob family (union-gate risk-cap canary) |
| store/strategy.go | 630 | `altcoin` | KEEP | CR-B | risk-cap knob family (union-gate risk-cap canary) |
| store/strategy.go | 1941 | `BTC` | KEEP | CR-B | risk-cap knob family (union-gate risk-cap canary) |
| store/strategy.go | 1942 | `BTC` | KEEP | CR-B | risk-cap knob family (union-gate risk-cap canary) |
| store/strategy.go | 1943 | `Altcoin` | KEEP | CR-B | risk-cap knob family (union-gate risk-cap canary) |
| store/strategy.go | 1944 | `Altcoin` | KEEP | CR-B | risk-cap knob family (union-gate risk-cap canary) |
| store/strategy.go | 1946 | `BTC` | KEEP | CR-B | risk-cap knob family (union-gate risk-cap canary) |
| store/strategy.go | 1947 | `BTC` | KEEP | CR-B | risk-cap knob family (union-gate risk-cap canary) |
| store/strategy.go | 1948 | `Altcoin` | KEEP | CR-B | risk-cap knob family (union-gate risk-cap canary) |
| store/strategy.go | 1949 | `Altcoin` | KEEP | CR-B | risk-cap knob family (union-gate risk-cap canary) |
| store/strategy.go | 2119 | `BTC` | KEEP | CR-B | risk-cap knob family (union-gate risk-cap canary) |
| store/strategy.go | 2120 | `Altcoin` | KEEP | CR-B | risk-cap knob family (union-gate risk-cap canary) |
| store/strategy.go | 2121 | `BTC` | KEEP | CR-B | risk-cap knob family (union-gate risk-cap canary) |
| store/strategy.go | 2122 | `Altcoin` | KEEP | CR-B | risk-cap knob family (union-gate risk-cap canary) |
| store/strategy.go | 2174 | `BINANCE` | KEEP | CR-B | wave-name comment (W-NO-BINANCE) — documents the OI-absent-on-MNQ behavior |
| store/strategy.go | 2186 | `BINANCE` | KEEP | CR-B | wave-name comment (W-NO-BINANCE) — documents the OI-absent-on-MNQ behavior |
| store/strategy.go | 2390 | `ai500` | KEEP | CR-B | C1 legacy-row normalization — removed source types collapse to static (stored rows must load) |
| store/strategy_dayplan_test.go | 162 | `BTC` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| store/strategy_futures_indicators_test.go | 7 | `Binance` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| store/strategy_futures_indicators_test.go | 22 | `Binance` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| store/strategy_futures_indicators_test.go | 23 | `BINANCE` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| store/strategy_futures_symbol_test.go | 2 | `USDT` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| store/strategy_futures_symbol_test.go | 23 | `btc` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| store/strategy_futures_symbol_test.go | 24 | `BTC` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| store/strategy_futures_symbol_test.go | 33 | `btc` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| store/strategy_futures_symbol_test.go | 34 | `BTC` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| store/strategy_legacy_hyper_test.go | 17 | `hyper_all` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| store/strategy_legacy_hyper_test.go | 18 | `hyper_main` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| store/strategy_legacy_hyper_test.go | 19 | `hyper_main` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| store/strategy_legacy_hyper_test.go | 33 | `hyper_all` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| store/strategy_legacy_hyper_test.go | 41 | `hyper_all` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| store/strategy_legacy_hyper_test.go | 42 | `ai500` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| store/strategy_legacy_hyper_test.go | 44 | `"mixed"` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| store/strategy_legacy_hyper_test.go | 46 | `hyper_all` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| store/strategy_schema_test.go | 13 | `BTC` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| store/strategy_schema_test.go | 47 | `ETHUSDT` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| store/strategy_schema_test.go | 51 | `ETH` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| store/strategy_schema_test.go | 58 | `ETHUSDT` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| store/strategy_schema_test.go | 87 | `AI500` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| store/strategy_schema_test.go | 108 | `ai500` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| store/strategy_schema_test.go | 126 | `quant` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| store/strategy_schema_test.go | 127 | `ai500` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| store/strategy_schema_test.go | 135 | `ai500` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| store/strategy_schema_test.go | 141 | `netflow` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| store/strategy_schema_test.go | 145 | `netflow` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| store/strategy_schema_test.go | 146 | `netflow` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| store/strategy_schema_test.go | 147 | `netflow` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| store/strategy_schema_test.go | 152 | `btc` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| store/strategy_schema_test.go | 181 | `ai500` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| store/strategy_token_test.go | 55 | `BTC` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| store/strategy_token_test.go | 100 | `BTC` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| store/trader.go | 63 | `BTC` | KEEP | CR-B | risk-cap knob family (union-gate risk-cap canary) |
| store/trader.go | 64 | `Altcoin` | KEEP | CR-B | risk-cap knob family (union-gate risk-cap canary) |
| store/trader.go | 66 | `AI500` | KEEP | CR-B | C1 legacy crypto columns — stored rows must load (column drop pending CTO ruling) |
| store/trader.go | 67 | `oi_top` | KEEP | CR-B | C1 legacy crypto columns — stored rows must load (column drop pending CTO ruling) |
| store/trader.go | 160 | `btc` | KEEP | CR-B | risk-cap knob family (union-gate risk-cap canary) |
| store/trader.go | 161 | `altcoin` | KEEP | CR-B | risk-cap knob family (union-gate risk-cap canary) |
| store/trader.go | 163 | `AI500` | KEEP | CR-B | C1 legacy crypto columns — stored rows must load (column drop pending CTO ruling) |
| store/trader.go | 164 | `oi_top` | KEEP | CR-B | C1 legacy crypto columns — stored rows must load (column drop pending CTO ruling) |
| telegram/agent/agent_test.go | 131 | `BTC` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| telegram/agent/agent_test.go | 137 | `BTC` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| telegram/agent/agent_test.go | 143 | `BTC` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| telegram/agent/agent_test.go | 185 | `USDT` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| telegram/agent/agent_test.go | 211 | `BTC` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| telegram/agent/agent_test.go | 230 | `BTC` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| telegram/agent/agent_test.go | 238 | `BTC` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| telegram/agent/agent_test.go | 252 | `USDT` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| telegram/agent/agent_test.go | 261 | `USDT` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| telegram/agent/agent_test.go | 269 | `USDT` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| telegram/agent/agent_test.go | 276 | `BTC` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| telegram/agent/agent_test.go | 280 | `BTC` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| telegram/agent/agent_test.go | 281 | `BTC` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| telegram/agent/agent_test.go | 286 | `BTC` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| telegram/agent/agent_test.go | 288 | `BTC` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| telegram/agent/agent_test.go | 291 | `btc` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| telegram/agent/agent_test.go | 310 | `BTC` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| telegram/agent/agent_test.go | 312 | `BTC` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| telegram/agent/agent_test.go | 314 | `BTC` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| telegram/agent/agent_test.go | 326 | `BTC` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| telegram/agent/agent_test.go | 328 | `BTC` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| telegram/agent/agent_test.go | 330 | `BTC` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| telegram/agent/agent_test.go | 333 | `btc` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| telegram/agent/agent_test.go | 360 | `BTC` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| telegram/agent/agent_test.go | 374 | `BTC` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| telegram/agent/agent_test.go | 382 | `BTC` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| telegram/bot_token_blacklist_test.go | 48 | `btC` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| telegram/bot_token_blacklist_test.go | 59 | `btC` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| telegram/bot_token_blacklist_test.go | 80 | `btC` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| telegram/bot_token_same_second_test.go | 76 | `btC` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| telegram/bot_token_same_second_test.go | 214 | `btC` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| telegram/bot_token_same_second_test.go | 217 | `btC` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| telegram/bot_token_test.go | 95 | `btC` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| telegram/bot_token_test.go | 134 | `btC` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| telegram/bot_token_test.go | 139 | `btC` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| telegram/bot_token_test.go | 147 | `btC` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| telegram/bot_token_test.go | 152 | `btC` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| telegram/bot_token_test.go | 169 | `btC` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| telegram/bot_token_test.go | 209 | `btC` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| telegram/bot_token_test.go | 219 | `btC` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| telegram/bot_token_test.go | 227 | `btC` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| telegram/bot_token_test.go | 230 | `btC` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| telegram/bot_token_test.go | 246 | `btC` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| trader/auto_trader_decision.go | 131 | `Wallet` | KEEP | CR-B | NT8 account-snapshot shape — balance key family (CTO ruling, byte-identical) |
| trader/auto_trader_decision.go | 136 | `wallet` | KEEP | CR-B | NT8 account-snapshot shape — balance key family (CTO ruling, byte-identical) |
| trader/auto_trader_decision.go | 137 | `Wallet` | KEEP | CR-B | NT8 account-snapshot shape — balance key family (CTO ruling, byte-identical) |
| trader/auto_trader_decision.go | 150 | `Wallet` | KEEP | CR-B | NT8 snapshot/comma-ok block — byte-identical (CTO F2 ruling family) |
| trader/auto_trader_decision.go | 151 | `Wallet` | KEEP | CR-B | NT8 account-snapshot shape — balance key family (CTO ruling, byte-identical) |
| trader/auto_trader_decision.go | 163 | `Binance` | KEEP | CR-B | NT8 snapshot/comma-ok block — byte-identical (CTO F2 ruling family) |
| trader/auto_trader_decision.go | 182 | `Lighter` | KEEP | CR-B | NT8 snapshot/comma-ok block — byte-identical (CTO F2 ruling family) |
| trader/auto_trader_decision.go | 184 | `USDT` | KEEP | CR-B | NT8 snapshot/comma-ok block — byte-identical (CTO F2 ruling family) |
| trader/auto_trader_decision.go | 185 | `Lighter` | KEEP | CR-B | NT8 snapshot/comma-ok block — byte-identical (CTO F2 ruling family) |
| trader/auto_trader_decision.go | 222 | `wallet` | KEEP | CR-B | NT8 snapshot/comma-ok block — byte-identical (CTO F2 ruling family) |
| trader/auto_trader_decision.go | 223 | `wallet` | KEEP | CR-B | NT8 account-snapshot shape — balance key family (CTO ruling, byte-identical) |
| trader/auto_trader_decision.go | 255 | `Binance` | KEEP | CR-B | NT8 snapshot/comma-ok block — byte-identical (CTO F2 ruling family) |
| trader/auto_trader_decision.go | 363 | `binance` | KEEP | CR-B | NT8 snapshot/comma-ok block — byte-identical (CTO F2 ruling family) |
| trader/auto_trader_decision.go | 382 | `Binance` | KEEP | CR-B | NT8 snapshot/comma-ok block — byte-identical (CTO F2 ruling family) |
| trader/auto_trader_decision.go | 524 | `binance` | KEEP | CR-B | NT8 snapshot/comma-ok block — byte-identical (CTO F2 ruling family) |
| trader/auto_trader_decision.go | 660 | `USDT` | KEEP | CR-B | NT8 snapshot/comma-ok block — byte-identical (CTO F2 ruling family) |
| trader/auto_trader_decision.go | 704 | `USDT` | KEEP | CR-B | NT8 snapshot/comma-ok block — byte-identical (CTO F2 ruling family) |
| trader/ninjatrader/reconcile_late_fill.go | 258 | `USDT` | KEEP | CR-B | NT8 fill commission-asset default — TCP wire schema lockstep (byte-identical) |
| trader/ninjatrader/tcp_trader.go | 1258 | `Wallet` | KEEP | CR-B | NT8 account-snapshot shape — balance key family (CTO ruling, byte-identical) |
| trader/ninjatrader/wire_tick_root_test.go | 35 | `USDT` | KEEP | CR-B | test fixture — pins the guard KEEP surfaces (byte-identical) |
| trader/protection_reconciler.go | 425 | `"MIXED"` | KEEP | CR-B | bracket-state string constant — same family as the ExecutorVerdict mixed content-assert (KEEP) |
