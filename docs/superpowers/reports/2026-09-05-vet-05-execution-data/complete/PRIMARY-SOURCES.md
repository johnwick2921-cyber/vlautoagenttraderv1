# Primary sources consulted 2026-09-05

These sources support contract/API/simulation semantics only. No named market study or empirical live MNQ edge is asserted. No other instrument's parameters or example depth is transferred to MNQ.

1. NinjaTrader, [Account.CreateOrder](https://docs.ninjatrader.com/ninjascript/createorder): ordered arguments distinguish limitPrice from stopPrice; the stop example uses zero limit price and a nonzero stop trigger. Supports the C# parameter diagnosis, not the economic merit of a stop entry.
2. NinjaTrader, [NT8 Simulator](https://ninjatrader-devel.ninjatrader.com/support/helpguides/nt8/simulation.htm): simulation models bid/ask volume, trade volume, queue time and random state delays. It is not a simple every-touch fill model. Simulated queues do not measure this trader's live queue priority; no live transfer coefficient is estimated.
3. CME Group, [MNQ contract specification](https://www.cmegroup.com/markets/equities/nasdaq/micro-e-mini-nasdaq-100.contractSpecs.html): $2 per index point; minimum price increment 0.25 points. Four ticks therefore mean one point/$2 for one contract. This arithmetic does not calibrate an optimal cap.
4. CME Group, [iLink Order Types](https://cmegroupclientsite.atlassian.net/wiki/spaces/EPICSANDBOX/pages/457227032/): exchange limit, stop-limit, stop-with-protection and residual-order behavior. The examples use ES and illustrative protection values; none is copied as an MNQ risk threshold. The actual NT8 broker route and protection configuration remain unmeasured.

All proposed caps, mechanical invalidation policies and evidence gates are first-person analytical judgments [I], untested here. No fictional personal trading history is offered. Code comments citing unidentified round-7 “research” are not accepted as independently verified market evidence.
