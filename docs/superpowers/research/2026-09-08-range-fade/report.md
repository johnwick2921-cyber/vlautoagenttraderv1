# Range fades and level rejection on index futures

## What the style requires to work

**The evidence supports investigating conditional reversal opportunities. It does not validate a standing instruction to fade every marked level.** The strongest directly relevant research establishes intraday momentum in Nasdaq futures and adverse selection in futures limit orders. Most evidence about support and resistance comes from FX. Most rules for recognizing a range day, prioritizing profile references, and managing discretionary fades remain practitioner doctrine.

For the described MNQ system, the immediate research question is whether **executable, selected fades earn positive net expectancy under its actual entry, target, stop, and deadline rules**. A 48.8% first-touch hold rate does not answer that question. Neither does the visual quality of twelve marked levels or three to five scenarios.

The seven answers below distinguish four verdicts: **TESTED-SUPPORTED** means a stated proposition has empirical support in the cited setting; **TESTED-CONTRADICTED** means a specified claim failed a test; **MIXED** means support is conditional, indirect, or conflicting; **UNTESTED** means no adequate direct test was located. None means proven for this MNQ implementation. **[A]** marks directly inspected source findings or arithmetic; **[B]** marks inference; **[C]** marks a proposed hypothesis. Practitioner statements are explicitly identified. Samples count different things—sessions, bars, orders, traders, or rule variants—and are not interchangeable. “Not verified” identifies a source-access limitation rather than an invented sample count.

### The system's existing measurement

**[A: user-supplied, not independently audited]** The reported measurement is 48.8% holds over 423 first-touch episodes, with a stated Wilson interval of [42.5%, 55.0%]. The episode definition, observation dates, raw counts, independence assumptions, and target rule were not supplied.

Three distinctions are essential:

1. **A reaction is not a completed winning trade.** Price can reject briefly, fail to reach the target, and then hit the stop. Conversely, an episode called a failed level can still produce a profitable trade under another exit rule.
2. **A touch is not a limit fill.** Some favorable touches do not execute an order behind the queue. A move through the level is more likely to fill it.
3. **Fifty percent is not automatically the correct null.** The width of the zone, the observation horizon, and the distances defining “hold” and “break” determine the benchmark. Osler's random-level benchmark, discussed below, was itself above 50%.

For completed trades, write net expectancy in consistent units as:

**E = p × W − (1 − p) × L − C**, where *p* is the trade win probability, *W* and *L* are average gross win and loss, and *C* is average round-trip cost. Include forced exits in those averages, or model them as a separate outcome. If—and only if—48.8% were the trade win probability, the gross break-even average win/loss ratio would be **1.049**. Costs raise that requirement. A 20-point target against a 30-point stop requires 60% wins before costs. These are arithmetic illustrations, not estimates of this book.

**[A: calculation]** Using p = 0.488, n = 423, and the standard independent-binomial 95% Wilson formula gives approximately **[44.1%, 53.6%]**, not the supplied interval. The supplied wider interval may reflect another confidence level, denominator, or dependence adjustment; it needs reconciliation. Multiple episodes from the same session also warrant uncertainty estimates clustered by session. Neither interval establishes profitable trading.

## 1. Day-type selection

**MIXED.** Intraday continuation and conditional differences are tested. A dependable, early, binary “range versus trend” classifier with an established out-of-sample hit rate for MNQ is not established by the literature reviewed. Standing aside when the expected fade return is negative is sound decision logic; recognizing those conditions prospectively is the unresolved empirical task.

### Opening-range and early-session evidence

**[A] Gert Baltussen, Zhi Da, Steven Lammers, and Martin Martens, “Hedging Demand and Market Intraday Momentum” (2021), Journal of Financial Economics.** The Nasdaq futures series runs **12 April 1996–1 May 2020, 6,017 daily observations**; the early history predates E-mini NQ. For Nasdaq, overnight-plus-first-half-hour return predicts the final half-hour return, with **1.83% out-of-sample R²** in Appendix Table B.1. The corresponding rest-of-day predictor has 3.76% out-of-sample R², but becomes available much later. Neither number is classification accuracy. **Transfers to intraday Nasdaq futures: YES for evidence that continuation exists; NO for a range-day hit rate, a fade veto threshold, or MNQ execution profitability.**[^1]

**[A] Lei Gao, Yufeng Han, Sophia Zhengzi Li, and Guofu Zhou, “Market Intraday Momentum” (2018), Journal of Financial Economics.** **SPY, 1993–2013**, with extensions to ten other ETFs; exact retained session count was **not verified in the accessible primary material**. The first-half-hour predictor includes the move from the previous close, and predicts the last half-hour. The relationship strengthens in several high-volatility, high-volume, recession, and macro-news conditions. **Transfer: INDIRECT.** This is equity-index ETF evidence about a particular forecast horizon, not proof that the first hour predicts the entire day's path or that a large opening move should always be faded or followed.[^2]

**[A] Yi-Cheng Tsai, Mu-En Wu, Jia-Hao Syu, Chin-Laung Lei, Chung-Shu Wu, Jan-Ming Ho, and Chuan-Ju Wang, “Assessing the Profitability of Timely Opening Range Breakout on Index Futures Markets” (2019), IEEE Access.** Five index futures are tested. The NQ methods/table sample is **2 January 2001–2 December 2013, 3,181 daily observations**; the reported best one-minute opening-range specification makes **3,099 trades**. The abstract's broad 2003–2013 description should not replace the NQ table dates. Results support breakout profitability under the paper's assumptions, but the best probing interval is selected from alternatives in the study. **Transfer: DIRECT instrument relevance; LIMITED strategy inference.** A selected breakout backtest is neither a prospective trend/range classifier nor evidence that its opposite must lose at every level.[^3]

**[A] Ulf Holmberg, Carl Lönnbark, and Christian Lundström, “Assessing the Profitability of Intraday Opening Range Breakout Strategies” (2013), Finance Research Letters.** **US crude-oil futures, 30 March 1983–26 January 2011**; exact daily sample count was **not verified** from accessible primary text. Its thresholds are percentage distances from the open, reconstructed using daily OHLC—not necessarily a completed 30- or 60-minute opening range. Full-sample profitability is not stable across subperiods and is substantially associated with the latest, most volatile period. **Transfer: NO direct Nasdaq validation; YES as a warning that opening-range results depend on definition and regime.**[^4]

### Market Profile: what is actually tested?

**[A] Jan Firich, “Futures Trading Based on Market Profile Day Timeframe Structures” (2012), WSEAS conference proceedings, Advances in Finance and Accounting.** **ES futures, 7 September 2005–8 June 2012; 1,688 trading days.** His operational definitions classify only **71.4%** of days using the initial taxonomy; modifying the normal-variation threshold increases coverage. He reports **9.54% trend days** and little dependence of today's type on yesterday's. These are historical label frequencies and transition tables, **not a 71.4% prediction hit rate**. Definitions use completed-day range extensions, with some subjective thresholds. The paper's suggested trading advantage is not validated by a costed, out-of-sample strategy test. **Transfer: adjacent ES evidence that classification definitions need testing; NO reliable early Nasdaq classifier.**[^30]

**[A] Chiu-Chin Chen, Yi-Chun Kuo, Chien-Hua Huang, and An-Pin Chen, “Applying Market Profile Theory to Forecast Taiwan Index Futures Market” (2014), Expert Systems with Applications.** **TAIEX futures; 48,080 five-minute records**. The paper prints the period ambiguously as “August 10, 2009 to 2010,” without a clear ending date. It uses an 80/20 training/test division and repeated neural-network experiments. A quantitative-profile model reports **83.38% accuracy at a 30-minute horizon**, versus 72.82% for its technical-indicator control. Accuracy means profitable transactions among forecasts meeting a trading threshold; it is **not** correct classification of all sessions as trend or range. The experiment tests particular profile-derived features, not the full Steidlmayer/Dalton taxonomy or a value-area traversal rule. **Transfer: INDIRECT and weak for deployment; no verified Nasdaq replication or equivalent classifier hit rate.**[^5]

**Practitioner doctrine:** J. Peter Steidlmayer and Steven B. Hawkins, *Steidlmayer on Markets: Trading with Market Profile*, second edition (2003, Wiley), and James F. Dalton, Eric T. Jones, and Robert B. Dalton, *Mind Over Markets* (1990; updated 2013) and *Markets in Profile* (2007), describe balance, initiative versus responsive activity, initial balance, range extension, and evolving value. **Sample/period: illustrative market examples, no single controlled validation sample for the doctrine. Transfer: a framework to operationalize and test, not validated NQ probabilities.** The completed day's profile is descriptive information unavailable at the open.[^6][^7]

The conventional first-hour initial balance can provide inputs after that hour. It cannot supply an ex ante classification simply because the finished profile later looks like a trend day. Likewise, a value area containing approximately 70% of historical activity is not a 70% probability interval for future price. No reproducible primary test located here establishes a universal “80% value-area traversal” probability on MNQ. TPO profiles and volume-at-price profiles also require separate definitions.[^6][^7]

### A published MNQ classification claim that should not be adopted

**[A] Mathias Mesfin, “A Validated Volatility-Volume-Gap Classifier for Regime Identification in MNQ Intraday Data” (2026), arXiv preprint 2605.11423, July revision.** It claims **947 MNQ sessions, December 2021–August 2025**. Its first-30-minute feature makes it an early-session, not premarket, classifier. Crucially, Table 2 gives **40** positive days, its annual positive counts sum to **55**, and Table 5 uses **125**, reporting 97 reversals, or **77.6%**. Those denominators are unreconciled. Peak-to-close reversal is also not synonymous with correctly predicting a range day. **Transfer: DIRECT instrument, but INSUFFICIENTLY RELIABLE evidence.** The advertised percentage should not be treated as a validated classifier hit rate. The author also reports that none of the tested strategies meets all of the paper's deployment criteria.[^8]

### What this means for a fade trader

**[A: supplementary practitioner research] Astreka, “Does the First 30 Minutes Predict the Rest of the NQ Trading Day?” (2026), publisher's own research article.** **NQ, 12 December 2024–6 August 2026; 408 sessions**, with 259 development and 149 chronological validation sessions. The simple opening-direction continuation rule reports **53.7% validation accuracy, interval [45.6%, 61.7%]**. Opening-range size correlates with the subsequent range at **0.43** in validation. **Transfer: DIRECT instrument/horizon, but not peer-reviewed or independently replicated.** It tests direction and range magnitude, not trend/range path classification or a trading strategy. It usefully illustrates why an early volatility signal need not be a reliable direction signal.[^31]

**[B]** A full-day label is less useful than the probability that price will continue through the proposed entry **during the intended holding period**. A range day can be too narrow to cover the stop and costs; a trend day can contain a tradable local reversal. “Range” is therefore neither necessary nor sufficient for a profitable individual fade.

**[C: proposed test]** Freeze a decision time and predict only the subsequent path. Compare a simple baseline with candidate inputs such as opening displacement, directional persistence, range expansion, value migration, and scheduled events. Report the confusion matrix, trend recall, precision of the permission-to-fade signal, and net P&L of accepted and rejected opportunities. Compare with always allowing fades and with simple time/news exclusions. Do not substitute a high overall accuracy driven by the majority class for lower drawdown or better net expectancy.

## 2. Level selection

**MIXED.** Some pre-announced levels predict reactions. Prior bounces and elapsed time have empirical support in particular definitions. There is no established hierarchy proving that more confluence, a higher timeframe, an untouched level, or the nearest level is universally strongest on Nasdaq futures.

**[A] Carol L. Osler, “Support for Resistance: Technical Analysis and Intraday Exchange Rates” (2000), Federal Reserve Bank of New York Economic Policy Review.** **USD/DEM, USD/JPY, GBP/USD; January 1996–March 1998**, minute quotes and approximately **64,200 published support/resistance values from six firms**. Published levels produced **60.8% bounces versus 56.2% at randomized levels**. These are reaction statistics, not net trading returns. Predictive content persisted at least five business days, but varied across firms/currencies. The strength categories supplied by three firms did **not consistently rank subsequent bounce probabilities**. **Transfer: INDIRECT.** This supports testing predeclared references and a matched random-level benchmark; it does not transfer the percentages or publishers' ratings to MNQ.[^9]

**[A] Osler, “Currency Orders and Exchange Rate Dynamics: An Explanation for the Predictive Success of Technical Analysis” (2003), Journal of Finance.** The inspected **2001 FRBNY working version** describes approximately **9,700 conditional orders at one FX dealer, 1 September 1999–11 April 2000**, focusing on USD/JPY, GBP/USD, and EUR/USD. Take-profit orders cluster around round numbers; stop orders show different clustering, including just beyond them. This explains why a reference can both repel price and accelerate a move once crossed. These dealer instructions are not identical to an exchange's passive limit queue. **Transfer: INDIRECT mechanism; no MNQ estimate of which round-number spacing, level family, or confluence count is best.**[^10]

**[A] Ken Chung and Anthony Bellotti, “Evidence and Behaviour of Support and Resistance Levels in Financial Time Series” (2021), arXiv 2101.07410.** **2018 minute data: EUR/USD 372,607 observations; Lloyds shares 127,606; Brent 307,678.** With algorithmic support/resistance zones, more previous bounces tend to predict another bounce, while influence decays with elapsed time. Results differ by asset and 60-/240-minute construction window; they do not establish that longer windows always produce stronger levels. Their bounce is a zone-entry/exit event, not reaching a trader's target before a stop. The study is not a complete, executable out-of-sample MNQ strategy; one zone-width input uses the full-series average increment. **Transfer: INDIRECT hypotheses about bounce history and age.** It does not justify calling every fresh first touch strongest.[^11]

### Has “more confluence = stronger” been tested directly?

**[A] Md. Chhali Uddin, “Analyzing the Effectiveness of Confluence of Price Action Disciplines in Forex Market” (2021), Journal of Stock & Forex Trading.** This reports **267 demo-account trades, 1 July–23 December 2020**, across **eight currency pairs plus gold and silver quoted against USD**; the headline is **236 wins/31 losses, 88.39%**. It bundles discretionary price-action factors. It does not compare otherwise similar levels with one, two, and three independent confluences, or isolate incremental effects using a held-out test. **Transfer: NO defensible MNQ effect size.** It is a published confluence case study, not evidence for a monotonic confluence-strength law.[^12]

**TESTED-CONTRADICTED, narrowly:** Osler's publisher-assigned strength labels failed to rank reliably in that sample. **UNTESTED, more broadly:** a count of confluences has not been validated here as an incremental NQ fade predictor. These are different claims; failure of ratings does not prove all forms of confluence useless.

| Candidate discriminator | What the evidence permits | What remains unestablished for MNQ |
|---|---|---|
| Prior bounces | A measurable feature with support under Chung–Bellotti's zone definition | Whether it improves executable first-touch or later-touch trades |
| Recency | Age can matter; “fresh” needs a precise definition | A universal first-touch premium or expiration time |
| Timeframe | A useful conditioning variable and practitioner priority | “Higher timeframe always wins” |
| Confluence | A hypothesis requiring incremental testing | A probability boost for each additional label |
| Proximity | Controls reachability, fill opportunity, and reward/stop geometry | That the closest reachable level has the best conditional rejection probability |

**[B]** A prior high, a swing high, a supply zone, and a profile edge can all describe essentially the same historical event. Counting four labels can double-count information. Conversely, marking a distant level reduces its touch frequency without proving its quality when touched.

**[C: proposed test]** Merge overlapping references into predeclared zones while preserving their component labels. Compare each family against controls matched on distance from price, volatility, time of day, side, and target/stop geometry. Estimate incremental value of bounce count, age, timeframe, and confluence on held-out sessions. Evaluate **net return per opportunity and per fill**, not just which line price visited. No primary source reviewed validates the entire proposed menu—PDH/PDL/close, overnight extremes, VWAP bands, profiles, supply/demand, swings, and round numbers—as equally tradable.

## 3. How many trades a day?

**UNTESTED for an optimal quota; MIXED for frequency and quality.** There is no adequate direct evidence here that one daily trade maximizes net returns for an intraday Nasdaq fade system. Nor does the literature justify taking every loosely described setup. The quality of the marginal opportunity, transaction costs, and shared session risk are the relevant quantities.

**[A] Wei-Yu Kuo and Tse-Chun Lin, “Overconfident Individual Day Traders: Evidence from the Taiwan Futures Market” (2013), Journal of Banking & Finance.** **Taiwan index futures, 8 October 2007–30 September 2008; 3,470 individual day traders**, identified prospectively by designated day-trade orders. More trading is harmful among losing traders, but not uniformly among winners. This is observational trader-level evidence, not randomized assignment of daily trade quotas or a comparison of identical setups. **Transfer: INDIRECT.** It contradicts a universal inference that more activity always improves results, while also failing to establish “less is always better.”[^13]

**[A] Ben R. Marshall, Rochester H. Cahan, and Jared M. Cahan, “Does Intraday Technical Analysis in the U.S. Equity Market Have Value?” (2008), Journal of Empirical Finance.** **SPY five-minute data, January 2002–December 2003; 7,846 rule specifications across five families**; exact retained bar count was not verified. None produces statistically significant profitability after the paper's data-snooping adjustment. The 7,846 count is a search universe, not independent market observations. **Transfer: INDIRECT.** This supports skepticism about selecting an attractive rule after inspecting many variants; it does not show that all discretionary filtering or all modern NQ rules fail.[^14]

### Discretion is part of the strategy being tested

**[B]** A trader who uses additional information to reject setups is running a different policy from an algorithm that executes every occurrence of the named pattern. “The professional trades this setup” is insufficient to specify the professional's decision rule. But a retrospective set of the professional's best examples is also not evidence that the filter added value.

**[C: proposed comparison]** Record every qualifying opportunity before its outcome, including skipped ones, the available information, and the stated rejection reason. Evaluate all-signals, a fixed filter, and a predeclared daily cap on the same chronological holdout. Compare net dollars per session, per-trade expectancy, opportunity coverage, drawdown, and loss clustering. Keep rejected winners in the opportunity set. Match stop/target rules and account for the fact that one open position can prevent another entry.

If additional trades have positive conditional expectancy and acceptable joint risk, a one-trade cap can discard value. If repeated signals are exposures to the same unfavorable directional flow, more trades can compound losses. Either is possible; the aggregate 48.8% hold statistic cannot distinguish them.

**Practitioner clarification:** Mike Bellafiore's *One Good Trade* refers to sound preparation and execution repeated across opportunities. It is not a one-trade-per-day rule. The primary desk account is detailed in answer 7.[^24]

## 4. Entry at the level

**MIXED.** Adverse selection in passive futures execution is measured. No reliable controlled NQ study located here holds the opportunity set and risk rules constant while comparing a resting limit, a rejection-candle entry, and a failed-break entry. Therefore none can be declared universally superior.

**[A] Luca Lalor and Anatoliy Swishchuk, “Market Simulation under Adverse Selection” (2024; revised 2026), arXiv 2409.12721.** In **TT simulated execution driven by real market data**, the June NQ contract on **25 April 2024** has **1,929 fills, 1,269 classified adverse: 65.79%**. The June ES contract on **24 April 2024** has **941 fills, 767 adverse: 81.51%**. These are one-day-per-contract experiments with specified posting rules, not audited live level-fade trades. “Adverse” concerns the next relevant quote movement, not eventual loss of the trade. **Transfer: DIRECT NQ microstructure relevance; NO universal adverse-fill rate or cost in MNQ points.**[^15]

**[A] J. Henning Fock, Christian Klein, and Bernhard Zwergel, “Performance of Candlestick Analysis on Intraday Futures Data” (2005), Journal of Derivatives.** The primary abstract reports **DAX and Bund futures** and no predictive ability for the tested candlestick patterns alone or combined with indicators such as momentum. **Period and observation count were not verified because the full primary paper was unavailable.** This is limited supporting evidence, not a central quantitative anchor. **Transfer: INDIRECT.** It does not test a specifically defined rejection candle at a preselected NQ level and cannot disprove that conditional setup.[^16]

| Entry policy | Economic tradeoff [B] | Required measurement [C] |
|---|---|---|
| Resting limit at the level | Attractive entry price when filled; exposed to changing information while waiting; favorable touches may miss the queue | Queue-aware fill probability, post-fill markouts, canceled orders, net return per posted opportunity |
| Wait for a rejection candle | Observes some reversal before commitment; entry can be farther from the level and closer to the target | Exact candle rule and close time, executable next price, missed reversals, revised reward/stop distance |
| Enter after a failed break | Conditions on penetration followed by return; may distinguish acceptance from failure | Predefined penetration/reclaim thresholds, maximum waiting time, reclaim fill price, renewed-break losses |

**[B]** The passive order's cost is not just a commission or bid–ask spread. Its fills are selected by the incoming flow. The economically relevant quantity is the subsequent price distribution **conditional on getting filled**, relative to alternative execution. Saving spread at entry can be outweighed by unfavorable post-fill movement. A historical candle merely touching a limit price does not establish that the order traded.

A rejection candle and a failed break also overlap unless defined carefully. A candle with a wick through the level followed by a close back inside can qualify as both. Any horse race needs mutually specified policies, including whether the target stays fixed and whether stop placement changes.

**[C]** Compare these policies at two levels: per original opportunity, counting no-fill/no-entry outcomes, and per executed trade. Use actual fees and realistic fills; do not double-count spread if it is already reflected in execution prices. Confirmation is valuable only if the improved conditional outcome exceeds the price concession, missed trades, and remaining risk. The literature supplies that tradeoff, not a universal waiting rule.

## 5. Where the stop belongs on a fade

**MIXED for stop design; UNTESTED for a universal 1.5×ATR floor.** A stop must be considered jointly with entry, target, holding time, and risk budget. There is no direct evidence located that “beyond the nearest level,” “beyond the wick,” or 1.5 times five-minute ATR is optimal for intraday Nasdaq mean-reversion entries.

**[A] Tim Leung and Xin Li, “Optimal Mean Reversion Trading with Transaction Costs and Stop-Loss Exit” (2015), International Journal of Theoretical and Applied Finance.** This is an optimal-stopping model for an **Ornstein–Uhlenbeck mean-reverting spread**. Its illustrative calibration uses **200 daily observations, August 2011–May 2012**, for GLD–GDX and GLD–SLV ETF pairs. It is not an intraday trading-performance sample. The imposed stop changes both optimal entry and take-profit regions; tightening the stop can lower the optimal profit-taking level and restrict where entry is worthwhile. **Transfer: MATHEMATICAL design principle only.** A Nasdaq index price is not automatically a stationary spread, and the paper supplies no optimal ATR multiple for MNQ.[^17]

**[A] Kathryn M. Kaminski and Andrew W. Lo, “When Do Stop-Loss Rules Stop Losses?” (2014), Journal of Financial Markets.** Theory plus **daily S&P and ten-year Treasury-note futures, 5 January 1993–7 November 2011**; exact retained observation count is not stated in the inspected summary table. Its rules switch portfolio exposure after cumulative losses. Results depend on return dynamics: stopping can discard expected recovery in mean-reverting settings and help under momentum or regime switching. **Transfer: INDIRECT and limited.** Portfolio switching across daily/longer horizons is not the same as a protective intraday bracket. This paper does not justify removing a hard loss limit from a leveraged fade.[^18]

### Comparing the proposed locations

**Beyond the level:** defensible as a falsification rule only if “acceptance beyond the zone” invalidates the precise setup. The nearest other marked line is not necessarily its invalidation boundary. Dense lines can produce an arbitrary stop unrelated to the entry thesis.

**Beyond the wick:** uses information from a completed rejection or failed-break event. For an order already resting before the touch, the eventual rejection wick is future information. Comparing that stop with a pre-touch structural stop also changes the information set and often the entry price.

**Volatility multiple:** can standardize distance relative to recent noise, but ATR is an average of true ranges, not a standard deviation or a confidence interval. “1.5×ATR” has no general coverage probability. The ATR lookback, smoothing, session treatment, and timestamp must be specified; “ATR5m” alone does not fully define it.

**Time stop:** appropriate to test when the expected reversal has a limited horizon. Estimate what happens after an unresolved trade has aged, rather than assuming every valid fade must work immediately. A slow but favorable trade and a failed auction are different states. No universal NQ fade time limit was established here.

These four comparisons are **[B: design reasoning], not empirically established rankings**.

### The important consequence of the existing floor

For MNQ, **one index point is $2 and one tick is 0.25 point/$0.50**.[^19] If ATR is hypothetically 20 points, a 1.5×ATR floor is 30 points, or $60 per contract before costs/slippage. If ATR is 60 points, the floor is 90 points, or $180. **One contract does not mean constant risk.**

**[B]** Taking the maximum of a structural distance and an ATR floor can prevent very tight stops, while simultaneously making a range's available profit too small relative to loss. If the required stop exceeds the risk budget or makes the trade uneconomic, the coherent option is to skip that opportunity. A wider stop does not create a larger attainable target. Whether the present floor helps requires a joint entry/target/stop test, not a comparison of stop-out rates alone.

## 6. What kills a fade book

**TESTED-SUPPORTED for several mechanisms; MIXED for an exact equity-curve shape.** Information shocks, persistent flow, and adverse selection are documented. A fixed-stop, one-contract fade book does not necessarily have the same loss distribution as an inventory-building market maker, but repeated intraday losses and adverse stop execution remain possible.

**[A] Muchen Zhao and Vadim Linetsky, “High Frequency Automated Market Making Algorithms with Adverse Selection Risk Control via Reinforcement Learning” (2021), ACM ICAIF.** **ES: 14 September 2017–13 September 2020; Treasury futures: 3 March 2017–2 March 2020.** The paper describes approximately **500,000 sampled 30-second observations over a three-year estimation span**, using market-depth data and 09:00–14:30 Central observations. Its ES baseline market-making backtest in **2018** shows a day's loss erasing several months' gains. Their model addresses adverse selection from liquidity consumption. **Transfer: DIRECT adjacent ES evidence of the failure mechanism; LIMITED to MNQ level fades.** The baseline manages inventory differently from this one-contract fixed-stop book, so its drawdown magnitude is not transferable.[^20]

**[A] Torben G. Andersen, Tim Bollerslev, Francis X. Diebold, and Clara Vega, “Real-Time Price Discovery in Global Stock, Bond and Foreign Exchange Markets” (2007), Journal of International Economics.** The common international futures sample is **1 July 1998–31 December 2002**, including S&P 500, FTSE 100, and DAX futures. The announcement analysis reports **15,764 five-minute return observations**, not 15,764 independent news events. Macroeconomic surprises produce price responses whose equity effects vary with economic conditions. **Transfer: STRONG adjacent mechanism, no NQ fade expectancy estimate.** Scheduled releases can change the relevant price level; the study does not prescribe a universal five-, fifteen-, or thirty-minute blackout.[^21]

**[A] Osler, “Stop-Loss Orders and Price Cascades in Currency Markets” (2005), Journal of International Money and Finance; inspected FRBNY Staff Report 150 (2002).** The study combines **9,655 dealer orders, August 1999–April 2000**, with a separate **January 1996–April 1998 minute-quote sample** for USD/DEM, USD/JPY, and GBP/USD; event counts vary by test. Trends accelerate around stop clusters, and stop-order effects persist longer than take-profit effects. **Transfer: INDIRECT.** This is evidence for cascades in FX, not proof of a specific NQ “stop hunt” or intentional targeting of an individual trader.[^22]

**[A] CFTC and SEC staffs, “Findings Regarding the Market Events of May 6, 2010” (2010), joint official report.** **One crisis event**, involving E-mini S&P futures and linked equity markets. It documents interacting order flow and depleted liquidity during the sharp price dislocation. **Transfer: DIRECT adjacent index-futures liquidity risk; no estimate of how often an MNQ fade will suffer it.** A price stop does not guarantee an execution price during a discontinuity.[^23]

### Documented mean-reversion drawdown beyond market making

**[A] Johannes Stübinger and Lucas Schneider, “Statistical Arbitrage with Mean-Reverting Overnight Price Gaps on High-Frequency Data of the S&P 500” (2019), Journal of Risk and Financial Management.** **984 historical S&P 500 constituents, January 1998–December 2015, 4,527 trading days**, with minute data. Their selected-stock gap strategy exits after 120 minutes. Despite attractive average results, Table 3 reports **68.17% maximum drawdown after modeled transaction costs** for the jump-diffusion strategy. **Transfer: INDIRECT.** This is a portfolio of individual-stock gap trades with its own sizing and capital conventions, not an index-level fade. It demonstrates that profitable intraday mean reversion need not imply small drawdowns; it does not forecast MNQ's drawdown or establish universal negative skew.[^25]

### Failure paths for this particular design

| Exposure | Why the book can lose [B] | What must be measured [C] |
|---|---|---|
| Persistent trend or repricing | Successive levels fail against the same flow | Losses per session, direction, and level cluster; later trades after earlier failures |
| News | Yesterday's reference need not remain a sensible equilibrium after new information | Scheduled-event windows; post-release acceptance versus reclaim; stop slippage |
| Cash open and session transitions | Participation and price discovery change; early ranges are incomplete | Separate overnight, opening, midday, and closing samples; proper local session clocks |
| Thin or rapidly depleted liquidity | Queue fills and exits deteriorate; ATR can lag a jump | Depth, spread, fill rate, post-fill movement, and tail execution loss |
| Narrow or noisy range | Reactions occur but cannot reach a profitable target, or both sides are repeatedly stopped | Excursion paths, target-before-stop frequency, and net return after costs |
| Model/selection decay | Historical level rankings or regime filters stop discriminating | Chronological holdout and ongoing, unchanged-rule SIM results |

**The characteristic failure can be a staircase of gains followed by clustered losses**, rather than a single unlimited losing position. That shape is a mechanism-based inference for this book, not a fitted result. A sequence of several full stops on different levels can surrender many small target wins. “No re-entry” does not prevent that sequence unless it also restricts related opportunities across the session. No scaling avoids an inventory-accumulation mechanism, but does not remove shared exposure to the day's direction.

Flat at **14:45 CT** removes subsequent exposure if positions and working entries are actually closed/canceled. It does not protect earlier trades from an open, a midday announcement, or a liquidity shock. On a normal US cash session it also leaves exposure to the **14:30–14:45 portion of the final half-hour**, relevant to the late momentum literature. A clock rule is an exposure boundary, not demonstrated alpha.

## 7. What practitioners running related styles actually do

**UNTESTED as a package. The following is practitioner doctrine or a documented account of practice, not causal evidence of profitability.** These traders do not all run the exact same strategy. There is no verified common professional template requiring twelve levels, one trade daily, one contract, no stop movement, and a 14:45 CT cutoff.

### Dalton: context, references, and permission to fade

**James F. Dalton, “Jim Reveals His Daily Trading Routine” (2018), author website**, alongside the books cited above. He describes reviewing monthly/weekly/daily structure, removing obsolete references, prioritizing longer-term references reachable during the day, preparing up/in-range/down scenarios, then updating for overnight activity. **Instrument/period/sample:** general auction-market practice illustrated with the March 2018 market; no controlled sample or audited outcome count. **Transfer: practitioner process only.**[^26]

**Dalton, “Top Ten Resolutions for Traders” (undated), author website**, explicitly advises **not fading trend days**, waiting for information when opening in balance, marking references and destinations, and distinguishing breakout from reversion opportunities. **Instrument/period/sample:** pit/session-oriented market guidance, no specified test sample. **Transfer: directly relevant doctrine, unvalidated forecast accuracy.** A list of scenarios is useful only if observed conditions can deactivate a fade scenario and its working orders.[^27]

### Brooks: range tactics include active management

**Al Brooks, *Trading Price Action Trading Ranges* (2012, Wiley), and “Trading Range Days—Two Legs or Swing” (2016), author teaching site.** The documented ES-oriented discussion favors scalping in a perceived range, differentiates that from a trend, and discusses small size, wide stops, optional scaling, exits when the premise deteriorates, and another entry after a loss. **Instrument/period/sample:** an August 2016 trading-room example; no controlled performance sample. **Transfer: doctrine only.** His numerical “80%” inertia heuristic is not a validated MNQ frequency. His discretionary exits and position management mean that copying only a limit entry does not copy the strategy.[^28]

### Raschke: tendencies guide decisions; they are not complete systems

**Mark Etzkorn, interview with Linda Bradford Raschke, Active Trader (March 2004), follow-up to “Linda Raschke Keeps Up the Pace.”** The first-person account distinguishes notebooks of tested market tendencies from fully specified entry/exit/stop systems. It describes trading with increased momentum and using other markets and timeframes for context. **Instrument/period/sample:** S&P and other markets, examples around December 2003–January 2004; one interview, no reproducible trade sample. **Transfer: doctrine about conditional selection and management.** The distinction is particularly relevant when automating a discretionary pattern name.[^29]

### Connors and Raschke: a defined failed-break setup

**Laurence A. Connors and Linda Bradford Raschke, *Street Smarts: High Probability Short-Term Trading Strategies* (1995, M. Gordon Publishing), “Turtle Soup”; rules also published by Connors's TradingMarkets (2004).** This is an explicit failed-break method: a new 20-day extreme followed by entry back through the previous extreme, with initial protection beyond the new extreme. **Instrument/period/sample:** futures and stock examples, no reproducible intraday NQ validation sample. **Transfer: doctrine defining a testable alternative to touching a limit, not evidence of superiority.** Its daily-reference and holding-period conventions cannot simply be assumed optimal for MNQ.[^32]

### Bellafiore: “one good trade” is a process standard

**Mike Bellafiore, *One Good Trade: Inside the Highly Competitive World of Proprietary Trading* (2010, Wiley); excerpts published by SMB Training (2011).** The account emphasizes preparation, patience, planned risk/reward, repeated good executions, and exiting when the reason for a trade fails. It also describes different activity and sizing at the open, midday, and close. **Instrument/period/sample:** US equities prop-desk experience; no controlled futures sample or audited denominator. **Transfer: process doctrine only.** It is explicitly compatible with multiple good trades in a day and does not establish a universal midday ban or closing cutoff.[^24]

### What can fairly be inferred from these accounts?

**[B]** The recurring practice is to connect context, location, entry, management, and the opportunity's horizon. Their rules differ on scaling, re-entry, and early exits. A one-contract system can legitimately exclude those actions, but then its own fixed-management payoff must work. There is no evidential basis to add scaling or stop movement merely to resemble a professional, or to assume removing those actions leaves the professional's expectancy intact.

For session cutoffs and daily trade limits, the reviewed primary accounts support **individualized rules tied to the trader's method and experience**, not a universal clock or quota. A fixed cutoff can be a sensible operational constraint without being the return-maximizing one.

## What a level-fade book on MNQ needs in order to work, in the order it needs it

The ordering below is **[B: synthesis]**, with **[C]** where a new test is proposed. It is not a published seven-factor recipe. The gate to the next step is evidence, not a more elaborate narrative.

### 1. A measurable economic proposition

Define the episode and the complete trade: level creation time, zone boundaries, first-touch rule, permissible entry, target, invalidation, expiry, and forced exit. The missing target/payoff specification is material. Reconcile the 423-episode denominator and interval. Separate reaction probability, fill probability, completed-trade win probability, and net return. A setup exists economically only if realistic wins compensate for losses and costs. When entry, target, and stop fall inside one five-minute bar, use finer event ordering or a conservative ambiguity rule; OHLC alone cannot reveal the sequence.

### 2. Permission to supply liquidity in the current state

Before ranking levels, determine when the book is allowed to fade. That need not require correctly naming the entire day. It requires evidence that the future price path over the trade horizon is favorable enough, given news, participation, direction, and uncertainty. A no-trade state must be possible. Permission should be checked while an order is waiting, because the state can change between placement and touch.

The initial research benchmark should be simple: all qualifying fades versus a few predeclared regime/session exclusions. Add a complex classifier only if it improves held-out economic results beyond those baselines.

### 3. A short, ranked set of distinct and reachable opportunities

Treat levels as candidates, not entitlements to trade. Merge overlapping references; avoid calling correlated labels independent confirmation. Require sufficient distance to a plausible target after entry costs and the required stop. Test level family, previous bounces, age, timeframe, and proximity incrementally. A shortlist earns its place through conditional performance rather than an arbitrary preferred count.

### 4. An entry policy that survives actual fill selection

Evaluate resting, rejection, and reclaim entries on the same opportunity set. Passive fills need queue-aware assumptions or recorded execution. Confirmed entries need the price available after confirmation, with missed trades retained in the analysis. Report how much of any theoretical reversal survives execution. NQ price behavior can inform the hypothesis; MNQ's own order book must establish the fills and costs.

### 5. A jointly feasible stop, target, horizon, and dollar risk

Use a coherent invalidation rule and test volatility/time adjustments jointly. Keep the 1.5×ATR floor as an unvalidated parameter until demonstrated otherwise. Measure favorable/adverse excursions and their order in time, rather than optimizing for fewer stop-outs. With one indivisible contract, some valid-looking trades may be too risky or offer too little reward and must be omitted. No-management rules are part of the strategy specification, not neutral implementation details.

### 6. Limits on shared session risk

Measure the contribution of the worst sessions, runs of losses, time under water, and slippage beyond nominal stops. Establish how orders and new fade permissions respond after several levels fail under the same flow. A per-trade stop, a no-re-entry rule, and a time cutoff address different risks; none alone limits the total cost of repeatedly fading a directional session.

### 7. Prospective validation of the complete policy

Freeze a modest number of hypotheses and evaluate chronologically on unseen sessions, preserving the same day's correlated episodes in the same split. Use session-level uncertainty estimates and stress realistic cost/fill assumptions. Record rejected opportunities as well as fills. Then collect forward SIM results with the exact policy and execution path. A simulator's optimistic fill convention must not become the evidence for the strategy.

**Minimum decision dashboard:** net expectancy per opportunity and per fill; average win/loss and actual costs; permission coverage; target-before-stop outcomes; adverse post-fill movement; maximum session loss; drawdown duration; and results by a few predeclared regimes. With only 423 episodes, subdividing twelve level families across many regimes, entry variants, and ATR multiples would rapidly exhaust the effective sample. A large number of attractive small cells is not replication.

## What the evidence says not to do

- **Do not equate a hold rate with a win rate, or a win rate with expectancy.** A statistically detectable reaction can be too small, too late, or inaccessible to the queue.
- **Do not import FX bounce percentages, Taiwanese forecast accuracy, or practitioner “80%” rules as MNQ probabilities.** Instrument, horizon, denominator, and outcome definitions differ.
- **Do not label the day using its completed profile or full-day extrema and claim the label was available at the open.** Evaluate only information available at the decision time.
- **Do not interpret a significant opening-momentum regression as a high-accuracy day-type classifier.** R², directional hit rate, and trading profitability answer different questions.
- **Do not grant more confidence merely because several names describe the same price zone.** Confluence's incremental value requires a controlled comparison.
- **Do not assume “freshest,” “highest timeframe,” or “nearest” is a proven universal ranking.** Nor should the next marked level automatically authorize another fade after a break.
- **Do not impose one trade per day as a research-established optimum—or take every signal simply to avoid discretion.** A fixed filter is a systematic decision rule; its marginal value can be measured.
- **Do not treat candle confirmation as free information or a resting limit as guaranteed spread capture.** Both policies change which trades execute and at what price.
- **Do not widen the stop to satisfy an ATR rule without rechecking attainable reward and dollar risk.** A lower stop-out rate can coexist with lower expectancy.
- **Do not remove the hard stop by citing mean-reversion mathematics, or add scaling because a discretionary trader uses it.** Both change the loss distribution and the strategy being validated.
- **Do not assume a 14:45 CT exit prevents trend-day damage, news losses, or bad fills before then.** Canceling remaining entries is part of being flat.
- **Do not select the best-looking variants from the same 423 episodes and call that validation.** Preserve unseen sessions and report failures, skipped winners, and dependence between trades.

## Sources and verification notes

Primary research, author material, exchange specifications, and an official event investigation underpin this report. Search coverage included support/resistance, opening-range and intraday momentum, market profile, trade frequency, passive execution, stop-loss models, and mean-reversion drawdowns. Evidence was checked through September 2026 where accessible. This is a targeted research review, not a claim to enumerate every unpublished trading study. Missing sample details, inconsistent preprints, and indirect transfer are identified in the text.

[^1]: Baltussen, G., Da, Z., Lammers, S., and Martens, M. (2021). *Hedging Demand and Market Intraday Momentum*. Journal of Financial Economics 142, 377–403. DOI: [10.1016/j.jfineco.2021.04.029](https://doi.org/10.1016/j.jfineco.2021.04.029). [Author manuscript](https://academicweb.nd.edu/~zda/intramom.pdf), Appendix Tables A.1/B.1.

[^2]: Gao, L., Han, Y., Li, S. Z., and Zhou, G. (2018). *Market Intraday Momentum*. Journal of Financial Economics 129(2), 394–414. DOI: [10.1016/j.jfineco.2018.05.009](https://doi.org/10.1016/j.jfineco.2018.05.009). [Author-university record](https://profiles.wustl.edu/en/publications/market-intraday-momentum/); [SSRN 2440866](https://papers.ssrn.com/sol3/papers.cfm?abstract_id=2440866). Exact sample count not verified.

[^3]: Tsai, Y.-C., Wu, M.-E., Syu, J.-H., Lei, C.-L., Wu, C.-S., Ho, J.-M., and Wang, C.-J. (2019). *Assessing the Profitability of Timely Opening Range Breakout on Index Futures Markets*. IEEE Access 7, 32061–32071. DOI: [10.1109/ACCESS.2019.2899177](https://doi.org/10.1109/ACCESS.2019.2899177). [Author-uploaded manuscript](https://www.researchgate.net/profile/Jia-Hao-Syu/publication/331076454_Assessing_the_Profitability_of_Timely_Opening_Range_Breakout_on_Index_Futures_Markets/links/6285e8a6247e622c2efb5839/Assessing-the-Profitability-of-Timely-Opening-Range-Breakout-on-Index-Futures-Markets.pdf), methods and NQ tables.

[^4]: Holmberg, U., Lönnbark, C., and Lundström, C. (2013). *Assessing the Profitability of Intraday Opening Range Breakout Strategies*. Finance Research Letters 10, 27–33. DOI: [10.1016/j.frl.2012.09.001](https://doi.org/10.1016/j.frl.2012.09.001). [Publisher text](https://www.sciencedirect.com/science/article/pii/S1544612312000438). Exact sample count not verified.

[^5]: Chen, C.-C., Kuo, Y.-C., Huang, C.-H., and Chen, A.-P. (2014). *Applying Market Profile Theory to Forecast Taiwan Index Futures Market*. Expert Systems with Applications 41, 4617–4624. DOI: [10.1016/j.eswa.2014.01.016](https://doi.org/10.1016/j.eswa.2014.01.016). [University-hosted paper](https://ir.lib.nycu.edu.tw/server/api/core/bitstreams/b840e832-cf38-4cda-96c0-3626e6ae5a03/content), §§2.1/2.9 and results tables.

[^6]: Steidlmayer, J. P., and Hawkins, S. B. (2003; hardcover released December 2002). *Steidlmayer on Markets: Trading with Market Profile*, 2nd ed. Wiley. ISBN 9780471215561. [Publisher record](https://uat.store.wiley.com/en-us/steidlmayer-on-markets-trading-with-market-profile-2nd-edition-p-9780471215561). Practitioner exposition, not a classifier validation study.

[^7]: Dalton, J. F., Jones, E. T., and Dalton, R. B. (1990; updated 2013). *Mind Over Markets: Power Trading with Market Generated Information*; and (2007), *Markets in Profile: Profiting from the Auction Process*. Wiley. [Author's book descriptions](https://jimdaltontrading.com/books/). Practitioner exposition.

[^8]: Mesfin, M. (2026). *A Validated Volatility-Volume-Gap Classifier for Regime Identification in MNQ Intraday Data*. [arXiv:2605.11423v2](https://arxiv.org/abs/2605.11423v2), revised 20 July 2026, Tables 2 and 5. Preprint; conflicting counts prevent treating its headline rate as validated.

[^9]: Osler, C. L. (2000). *Support for Resistance: Technical Analysis and Intraday Exchange Rates*. FRBNY Economic Policy Review 6(2), 53–68. [Primary PDF](https://www.newyorkfed.org/medialibrary/media/research/epr/00v06n2/0007osle.pdf), especially randomized benchmark and strength-rating analysis.

[^10]: Osler, C. L. (2003). *Currency Orders and Exchange Rate Dynamics: An Explanation for the Predictive Success of Technical Analysis*. Journal of Finance 58(5), 1791–1819. DOI: [10.1111/1540-6261.00588](https://doi.org/10.1111/1540-6261.00588). [Inspected 2001 FRBNY Staff Report 125](https://www.newyorkfed.org/medialibrary/media/research/staff_reports/sr125.pdf); the sample description in this review is explicitly version-specific.

[^11]: Chung, K., and Bellotti, A. (2021). *Evidence and Behaviour of Support and Resistance Levels in Financial Time Series*. [arXiv:2101.07410](https://arxiv.org/abs/2101.07410). [Full text](https://arxiv.org/html/2101.07410v1). Cited as a preprint, with its own zone/bounce definitions.

[^12]: Uddin, M. C. (2021). *Analyzing the Effectiveness of Confluence of Price Action Disciplines in Forex Market*. Journal of Stock & Forex Trading, cited on the paper's first page as 9:570. [Primary PDF](https://www.longdom.org/open-access-pdfs/analyzing-the-effectiveness-of-confluence-of-price-action-disciplines-in-forex-market.pdf). Uncontrolled demo-account case study; headline counts reported as claims.

[^13]: Kuo, W.-Y., and Lin, T.-C. (2013). *Overconfident Individual Day Traders: Evidence from the Taiwan Futures Market*. Journal of Banking & Finance 37(9), 3548–3561. DOI: [10.1016/j.jbankfin.2013.04.036](https://doi.org/10.1016/j.jbankfin.2013.04.036). [Author manuscript](https://hub.hku.hk/bitstream/10722/184789/1/content.pdf?accept=1); SSRN 1944059.

[^14]: Marshall, B. R., Cahan, R. H., and Cahan, J. M. (2008). *Does Intraday Technical Analysis in the U.S. Equity Market Have Value?* Journal of Empirical Finance 15(2), 199–210. DOI: [10.1016/j.jempfin.2006.05.003](https://doi.org/10.1016/j.jempfin.2006.05.003). [Publisher text](https://www.sciencedirect.com/science/article/pii/S0927539807000588).

[^15]: Lalor, L., and Swishchuk, A. (2024; revised June 2026). *Market Simulation under Adverse Selection*. [arXiv:2409.12721v3](https://arxiv.org/abs/2409.12721v3), revised manuscript Table 2. [PDF](https://arxiv.org/pdf/2409.12721). Simulated execution, real market inputs; not a live trading-account study.

[^16]: Fock, J. H., Klein, C., and Zwergel, B. (2005). *Performance of Candlestick Analysis on Intraday Futures Data*. Journal of Derivatives 13(1), 28–40. DOI: [10.3905/jod.2005.580514](https://doi.org/10.3905/jod.2005.580514). [Primary abstract reproduced with author record](https://www.researchgate.net/publication/236964657_Performance_of_Candlestick_Analysis_on_Intraday_Futures_Data). Full-paper sample details not verified.

[^17]: Leung, T., and Li, X. (2015). *Optimal Mean Reversion Trading with Transaction Costs and Stop-Loss Exit*. International Journal of Theoretical and Applied Finance 18(3), 1550020. DOI: [10.1142/S021902491550020X](https://doi.org/10.1142/S021902491550020X). [arXiv:1411.5062](https://arxiv.org/abs/1411.5062), §2/Table 1 and optimal-stopping results.

[^18]: Kaminski, K. M., and Lo, A. W. (2014). *When Do Stop-Loss Rules Stop Losses?* Journal of Financial Markets 18, 234–254. DOI: [10.1016/j.finmar.2013.07.001](https://doi.org/10.1016/j.finmar.2013.07.001). [Author-linked final paper](https://www.dropbox.com/scl/fi/fei67s711z7nmlh8eolf5/2014_StopLossRules_JFinMarkets.pdf?dl=0&rlkey=c9miha2ehliqwi6ai2w56wjgu), §§4–5.

[^19]: CME Group. *Micro E-mini Nasdaq-100 Index Futures Contract Specifications*, accessed September 2026. [Exchange specifications](https://www.cmegroup.com/markets/equities/nasdaq/micro-e-mini-nasdaq-100.contractSpecs.html). Contract fact, not a strategy study; sample size not applicable.

[^20]: Zhao, M., and Linetsky, V. (2021). *High Frequency Automated Market Making Algorithms with Adverse Selection Risk Control via Reinforcement Learning*. Proceedings of the 2nd ACM International Conference on AI in Finance. DOI: [10.1145/3490354.3494398](https://doi.org/10.1145/3490354.3494398). [NSF-hosted author paper](https://par.nsf.gov/servlets/purl/10344950), Figure 1 and §4.

[^21]: Andersen, T. G., Bollerslev, T., Diebold, F. X., and Vega, C. (2007). *Real-Time Price Discovery in Global Stock, Bond and Foreign Exchange Markets*. Journal of International Economics 73, 251–277. DOI: [10.1016/j.jinteco.2007.02.004](https://doi.org/10.1016/j.jinteco.2007.02.004). [Inspected Federal Reserve working paper, IFDP 871 (2006)](https://www.federalreserve.gov/pubs/ifdp/2006/871/ifdp871.pdf), data and announcement-window sample.

[^22]: Osler, C. L. (2005). *Stop-Loss Orders and Price Cascades in Currency Markets*. Journal of International Money and Finance 24(2), 219–241. DOI: [10.1016/j.jimonfin.2004.12.002](https://doi.org/10.1016/j.jimonfin.2004.12.002). [FRBNY Staff Report 150 (2002)](https://www.newyorkfed.org/research/staff_reports/sr150.html). Order and quote samples are distinct.

[^23]: CFTC and SEC staffs (2010). *Findings Regarding the Market Events of May 6, 2010*. Joint report, 30 September. [Official report](https://www.sec.gov/news/studies/2010/marketevents-report.pdf). Single-event investigation.

[^24]: Bellafiore, M. (2010). *One Good Trade: Inside the Highly Competitive World of Proprietary Trading*. Wiley. [Excerpts published by SMB Training, 13 June 2011](https://www.smbtraining.com/blog/the-best-excerpts-from-one-good-trade). Practitioner/desk account.

[^25]: Stübinger, J., and Schneider, L. (2019). *Statistical Arbitrage with Mean-Reverting Overnight Price Gaps on High-Frequency Data of the S&P 500*. Journal of Risk and Financial Management 12(2), 51. DOI: [10.3390/jrfm12020051](https://doi.org/10.3390/jrfm12020051). [Full publisher text](https://www.mdpi.com/1911-8074/12/2/51), §4.1 and Table 3.

[^26]: Dalton, J. F. (2018). *Jim Reveals His Daily Trading Routine*. [Author website](https://jimdaltontrading.com/routine/); text dates its example 24 March 2018. Practitioner statement.

[^27]: Dalton, J. F. (undated). *Top Ten Resolutions for Traders*. [Author website](https://jimdaltontrading.com/top-ten-resolutions-traders/). Practitioner statement.

[^28]: Brooks, A. (2012). *Trading Price Action Trading Ranges*. Wiley. Related inspected primary account: [*Trading Range Days—Two Legs or Swing*](https://www.brookstradingcourse.com/ask-al/trading-range-day-two-legs-swing/), 28 August 2016, discussing 12 August trading-room Q&A. Practitioner statement.

[^29]: Etzkorn, M., interviewing Raschke, L. B. (2004). Active Trader, March, pp. 76–79; follow-up to *Linda Raschke Keeps Up the Pace* (February 2004). [Primary interview hosted by Raschke](https://lindaraschke.net/wp-content/uploads/2026/03/raschke_pt2_0304.pdf). Practitioner/desk account; the website upload year is not the publication year.

[^30]: Firich, J. (2012). *Futures Trading Based on Market Profile Day Timeframe Structures*. Proceedings of the 1st WSEAS International Conference on Finance, Accounting and Auditing, Zlin; Advances in Finance and Accounting, pp. 80–85. ISBN 9781618041241. [Primary paper](https://www.wseas.us/e-library/conferences/2012/Zlin/FAA/FAA-12.pdf), Tables 1–6. Descriptive classification and proposed tactics, not an out-of-sample forecast evaluation.

[^31]: Astreka (2026). *Does the First 30 Minutes Predict the Rest of the NQ Trading Day?* [Original research article](https://www.astreka.com/market-insights/does-the-first-30-minutes-predict-the-rest-of-the-nq-trading-day). Supplementary, non-peer-reviewed research; reported validation statistics have not been independently reproduced.

[^32]: Connors, L. A., and Raschke, L. B. (1995). *Street Smarts: High Probability Short-Term Trading Strategies*. M. Gordon Publishing. Inspected statement of the rules: TradingMarkets Editors, [*Today's Trading Lesson from TradingMarkets*](https://tradingmarkets.com/recent/todays_trading_lesson_from_tradingmarkets-649736), 9 September 2004. Practitioner system description.
