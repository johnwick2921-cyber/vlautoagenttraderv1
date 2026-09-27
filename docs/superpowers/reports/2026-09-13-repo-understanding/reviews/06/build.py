import json,csv,hashlib,pathlib
ROOT=pathlib.Path('/tmp/nofx-understanding-execution-20260913'); OUT=pathlib.Path('/tmp/nofx-review-06'); CAT=pathlib.Path('/tmp/nofx-repo-understanding-20260913/docs/superpowers/reports/2026-09-13-repo-understanding')
a=next(a for a in json.load(open(CAT/'review-plan.json'))['assignments'] if a['id']==6)
notes={}
def add(path,body):
 for line in body.strip().splitlines():
  name,purpose,risk=line.split('|',2);notes[(path,name)]=(purpose,risk)
add('trader/aster/trader.go','''NewAsterTrader|Parse ECDSA signing key and construct 30-second HTTP client, optionally replaced by hook.|Main wallet and signer strings are not validated as addresses here; no broker request in constructor.
genNonce|Use current Unix microseconds as nonce.|No monotonic counter or concurrency uniqueness guarantee.
getPrecision|Read cached precision or fetch exchangeInfo and cache every symbol.|Final map lookup is outside lock; concurrent misses can write while this read runs. HTTP status/read errors not checked.
roundToTickSize|Round value to nearest step multiple.|Nearest rounding can increase requested quantity; nonpositive step leaves value unchanged.
formatPrice|Resolve tick size then precision fallback.|Metadata errors propagate; no positive/finite guard.
formatQuantity|Resolve lot step then decimal fallback.|Nearest rounding can exceed intended size.
formatFloatWithPrecision|Format fixed decimals then trim trailing zeros and decimal point.|Precision zero corrupts integers ending in zero: 100 becomes 1; zero becomes empty.
normalizeAndStringify|Normalize signing parameters and marshal deterministic JSON.|Normalization stringifies primitive values.
normalize|Recursively turn maps/slices and primitive values into signing representation.|Unknown types stringify rather than reject; JSON marshaler provides map ordering.
sign|Add time/window, ABI pack parameters and addresses, EIP-191 hash and sign.|Mutates supplied map; recvWindow 50000ms; address conversion permissive.
request|Copy/sign each attempt and retry selected network errors up to three times.|Retries POST order mutations after ambiguous timeout with fresh nonce and no stable order key.
doRequest|Send form POST or query GET/DELETE and require HTTP 200.|ReadAll errors ignored; no request context/cancellation; unsupported verbs rejected.''')
add('trader/aster/trader_account.go','''GetBalance|Combine USDT balance with position-derived margin and unrealized PnL.|Fallback to raw fields on position error; leverage zero divides by zero; missing asset yields success zeros.
GetMarketPrice|Fetch public ticker and parse string price.|HTTP status and parse checked, positivity not checked.
GetClosedPnL|Convert nonzero-realized-PnL fills to inferred closed-position records.|Breakeven closes disappear; entry time equals exit and entry price inferred, not observed.
GetTrades|Fetch signed userTrades and normalize fills.|HTTP/signing and JSON failures return empty successful list; default limit500, no pagination.
GetOrderBook|Fetch public depth and convert string levels.|Malformed levels can leave nil entries; numeric parse failures become zero.''')
add('trader/aster/trader_orders.go','''OpenLong|Cancel old orders, set leverage and submit BUY GTC limit at ticker+1%.|Cancellation failure tolerated; quantity rounds nearest; limit acceptance does not prove fill.
OpenShort|Cancel old orders, set leverage and submit SELL GTC limit at ticker-1%.|Cancellation failure tolerated; position-exists leverage error accepted.
CloseLong|Resolve zero quantity from long position and submit SELL GTC limit at ticker-1%, then cancel all symbol orders.|No reduceOnly; cancel occurs without fill confirmation and can cancel pending close/protection.
CloseShort|Resolve zero quantity from positive short size and submit BUY GTC limit at ticker+1%, then cancel all.|No reduceOnly; acknowledgement logged as closed without settled execution.
SetStopLoss|Submit opposite-side STOP_MARKET using formatted price/size.|BOTH mode without reduceOnly; only uppercase SHORT selects BUY.
SetTakeProfit|Submit opposite-side TAKE_PROFIT_MARKET using formatted price/size.|BOTH mode without reduceOnly; no paired OCO relationship specified.
CancelStopLossOrders|Fetch v3 open orders and cancel STOP/STOP_MARKET using v1 order endpoint.|Partial failure returns nil; endpoint differs from generic cancellation; float order IDs may lose precision.
CancelTakeProfitOrders|Fetch v3 open orders and cancel TP types via v1 endpoint.|Returns error only if every attempted cancellation failed.
CancelAllOrders|DELETE allOpenOrders for symbol.|Scope includes protection and ordinary entries.
CancelStopOrders|Enumerate and delete SL/TP types via v3 endpoint.|Individual failures logged but returned success even if all fail.
FormatQuantity|Expose internal quantity rounding as string.|Uses generic formatting after metadata rounding.
GetOrderStatus|Read order and normalize price/fill size fields.|Commission hard-coded zero; missing numeric fields may remain absent.
GetOpenOrders|Read v3 open orders into common OpenOrder records.|No pagination, numeric parse errors ignored; empty list nil.
PlaceLimitOrder|Format and submit GTC grid limit; optionally reduce-only.|PostOnly, ClientID and leverage request fields ignored; status NEW synthesized; malformed success can have empty ID.
CancelOrder|Delete a specific exchange order ID.|Transport/API errors propagated.''')
add('trader/aster/trader_sync.go','''SyncOrdersFromAster|Fetch last24h/500 fills, time-sort, create order/fill and call PositionBuilder.|Nontransactional; existing order skips missing fill/position repair; GetTrades errors may be masked.
deriveAsterOrderAction|Infer open versus close from nonzero PnL and direction.|Zero-PnL closes misclassified; invalid side defaults SELL branch.
StartOrderSync|Launch interval ticker running Aster sync.|No stop handle/context/ticker cleanup; nonpositive interval panics.''')
add('trader/binance/futures.go','''getBrOrderID|Generate broker-tagged timestamp plus random hex client ID.|Random error ignored; uniqueness probability in prose not established.
NewFuturesTrader|Build SDK client with hook, sync server time, request hedge mode.|Constructor performs remote account mutation and tolerates refusal.
setDualSidePosition|Ask exchange for dual-side account mode.|Only specific already-set text becomes success.
syncBinanceServerTime|Set SDK time offset from exchange clock.|Single sample, no RTT correction; failure logs and continues.
contains|Length-check then delegate substring scan.|Equivalent local string helper, not validation.
stringContains|Scan byte substrings.|Empty substring matches; case-sensitive.
calculatePrecision|Count decimal places after trimming trailing zeros.|Decimal count alone does not ensure arbitrary tick multiples.
trimTrailingZeros|Trim fractional zeros and terminal decimal.|Unlike Aster formatter, integers without decimal stay intact.''')
add('trader/binance/order_sync.go','''SyncOrdersFromBinance|Recover cursor, discover symbols via commission/positions/recent fills/PnL, fetch500 per symbol, sort and persist.|DB recovery adds1000ms; pagination absent; cursor can advance despite DB write failures; order-exists skip prevents repair.
getPositionSymbols|Collect nonempty symbols from positions.|Position-read error becomes nil list, not explicit uncertainty.
determineOrderAction|Infer opens/closes using side, positionSide and realized PnL.|BOTH falls to open regardless PnL; zero-PnL hedge closes also misclassified.
StartOrderSync|Run initial sync and independent periodic ticker.|Initial and first periodic sync may overlap; no lifetime cancellation.''')
add('trader/bitget/order_sync.go','''GetTrades|Read max100 fills, parse wrapped/nonwrapped formats, derive action and fee.|Empty wrapped fillList falls into array parser and errors; one-way SELL always close_long; no pagination.
SyncOrdersFromBitget|Sort last24h fills and create order/fill/position records.|Existing order short-circuits repair; order exchange ID uses trade ID whereas fill uses real order ID.
StartOrderSync|Launch recurring Bitget sync.|No stop handle; invalid intervals panic.''')
add('trader/bitget/trader_account.go','''GetBalance|Read USDT futures account and cache equity-minus-UPL wallet balance.|Cache reference returned after unlock; numeric failures/missing account become zero.
SetMarginMode|Convert symbol and request isolated/crossed mode.|Position-related refusal treated as nil despite unchanged mode.
SetLeverage|Request leverage and tolerate same-value response.|Other failures returned.
GetMarketPrice|Read converted-symbol ticker last price.|Reject empty ticker and bad number; no finite/positive validation.
GetOrderBook|Read public-style futures depth through request helper.|No depth default here; numeric parse failures become zero.''')
add('trader/bybit/trader_account.go','''GetBalance|Read unified account and cache common wallet/equity fields.|Zero wallet treated as absent and replaced with equity, potentially double-counting UPL downstream.
GetClosedPnL|Delegate to signed direct HTTP closed-PnL reader.|No pagination orchestration.
getClosedPnLViaHTTP|Sign v5 closed-pnl GET with API key, timestamp and5000ms receive window.|Hard-coded mainnet and http.DefaultClient without timeout; HTTP status not checked separately.
parseClosedPnLResult|Normalize closed-PnL rows and infer fees from entry/exit values.|Maps Sell to short and reverses expected-PnL sign; requires venue-side semantics confirmation; parse errors become zero.''')
add('trader/bybit/trader_orders.go','''OpenLong|Cancel regular/conditional orders, set leverage, submit one-way market Buy.|Leverage/cancel errors tolerated; quantity formatting error ignored.
OpenShort|Cancel orders and submit one-way market Sell.|Same nonfatal configuration/refusal behavior as long.
CloseLong|Resolve zero size from positions and submit reduce-only market Sell.|Nonpositive amount refused; cache may be15s old; formatting error ignored.
CloseShort|Negate signed short position amount and submit reduce-only market Buy.|Matches GetPositions negative short convention; nonpositive refused.
SetLeverage|Set buy and sell leverage together.|Handles110043 already set; propagates other RetCode errors.
SetMarginMode|Request cross/isolated position mode.|Handles110026 already set; does not silently swallow other codes.
GetMarketPrice|Read linear market tickers and parse first lastPrice.|Explicit no-data and malformed-price errors.
SetStopLoss|Create reduce-only trigger order with direction derived from ticker versus stop.|Only uppercase SHORT switches action; format error ignored; price not tick-normalized.
SetTakeProfit|Create reduce-only trigger order with ticker-derived direction.|Same formatting/side sensitivity; independent order from SL.
CancelStopLossOrders|Delegate conditional filter for StopLoss.|Generic Stop also treated as stop-loss.
CancelTakeProfitOrders|Delegate conditional filter for TakeProfit.|PartialTakeProfit included.
CancelAllOrders|Call SDK symbol cancel-all.|Only Go error checked; business RetCode discarded.
CancelStopOrders|Attempt both conditional cancellation groups.|Always returns nil after logging errors.
cancelConditionalOrders|List StopOrder orders and cancel matching stopOrderType.|List business error treated as no orders; cancellation errors entirely discarded.
GetOrderStatus|Fetch order history and normalize status/price/qty/fee.|Rejected collapsed into CANCELED; unknown status passes through.
GetOpenOrders|Return StopOrder records only.|Interface promises regular limits too; business errors become empty success; Side keeps Buy/Sell casing.
PlaceLimitOrder|Format qty and submit GTC one-way limit.|Leverage refusal tolerated; fixed8decimal price; no ClientID/PostOnly sent; malformed success may return empty ID.
CancelOrder|Submit specific cancellation and check RetCode.|Checks both transport and business errors.
GetOrderBook|Direct mainnet public HTTP depth request.|http.Get default client has no timeout; parse errors become zero.''')
add('trader/bybit/trader_positions.go','''GetPositions|Fetch USDT-settled linear positions and cache normalized maps.|Short positionAmt negative, side lowercase; numeric failures ignored; no pagination.''')
add('trader/gate/trader.go','''NewGateTrader|Build authenticated gateapi context/client with15s caches.|No remote constructor call; authentication persists in context.
convertSymbol|Convert BTCUSDT to BTC_USDT.|Case-sensitive; any underscore is assumed preformatted.
revertSymbol|Remove every underscore from exchange symbol.|No uppercase normalization.
getContract|Fetch/cache contract metadata by converted symbol.|No TTL; cache pointer returned directly.
clearCache|Clear balance and position caches under mutexes.|Contract metadata retained.''')
add('trader/gate/trader_account.go','''GetBalance|Fetch USDT futures balance and cache wallet/available/UPL.|Numeric errors ignored; shares returned cache map.
GetPositions|Convert signed contract counts to positive base quantities and lowercase side.|Failed metadata lookup silently uses multiplier1; returns exchange underscore symbol and int leverage.
GetClosedPnL|Read max100 close records using Unix-second start time.|Only symbol/side/PnL/time populated; price/qty/fee/IDs absent as zero fields.''')
add('trader/hyperliquid/trader.go','''isXyzDexAsset|Strip suffix/prefix and check static XYZ asset allowlist.|Suffix USD checked before -USD, leaving trailing hyphen for dashed USD names.
convertSymbolToHyperliquid|Uppercase and remove quote then add xyz prefix for known assets.|Static allowlist; same suffix ordering issue.
absFloat|Return absolute magnitude.|NaN remains NaN.
NewHyperliquidTrader|Parse key, require main wallet, select network, build SDK and fetch metadata.|Agent>100USDC refuses only when separate key; main-key match warns; agent balance read failure allowed.
FormatQuantity|Format size using main metadata decimals.|XYZ precision uses main meta fallback4 here, unlike XYZ order helper.
getSzDecimals|Read main metadata under RWMutex, fallback4.|Unknown symbol is accepted with guessed precision.
roundToSzDecimals|Nearest positive-oriented size rounding.|Can round above request; int conversion has no finite/range checks.
roundPriceToSigfigs|Round price to5significant figures.|Positive infinity causes nonterminating magnitude loop; negative rounding biased; no finite check.''')
add('trader/hyperliquid/trader_sync.go','''refreshMetaIfNeeded|Fetch metadata when SDK NameToAsset returns0 and recheck.|No source callers found; zero treated invalid, own cache update does not explicitly update SDK name map.
fetchXyzMeta|Fetch xyz universe and store with lock.|Mainnet URL hard-coded even for testnet trader; no background loop here despite historical graph summary.
getXyzSzDecimals|Read xyz size precision, adding prefix if missing.|Missing meta/asset fallback2.
getXyzAssetIndex|Return xyz universe index for prefixed name.|Missing asset returns-1, permitting caller refusal.''')
add('trader/hyperliquid/trader_orders.go','''OpenLong|Cancel old orders and submit BUY IOC at+1%, via SDK or XYZ wire.|Returns fabricated FILLED/orderId0 and discards actual SDK response.
OpenShort|Cancel old orders and submit SELL IOC at-1%, via SDK or XYZ wire.|Same fabricated settlement status; XYZ leverage skipped.
CloseLong|Resolve zero size and submit reduce-only SELL IOC then cancel all.|Unconditionally claims FILLED and removes protection without confirming full fill.
CloseShort|Resolve short size and submit reduce-only BUY IOC then cancel all.|Same acknowledgement/settlement mismatch.
CancelStopLossOrders|Delegate to broad stop-order cancellation.|Cancels every symbol order, violating selective interface contract.
CancelTakeProfitOrders|Delegate to broad stop-order cancellation.|Also cancels stop-loss and regular entries.
CancelAllOrders|Enumerate symbol open orders and cancel, or delegate XYZ.|Individual failures only logged; nil may mean incomplete cancellation.
CancelStopOrders|Enumerate and cancel all symbol orders.|Cannot distinguish protections, contradicts selective-method contract; errors swallowed.
cancelXyzOrders|Fetch xyz open orders and cancel matching coin.|Reads mainnet regardless testnet; individual failures swallowed.
cancelXyzOrder|Sign direct cancel action and POST to selected network.|Puts order ID into asset field a; only top-level status checked, inner errors ignored.
floatToWireStr|Fixed8decimal string with fractional zeros trimmed.|Finite/positive validation absent.
placeXyzOrder|Resolve XYZ metadata/index, round, sign and send IOC order.|Hardcodes dex index1; missing/empty status array accepted; actual fill not returned to caller.
placeXyzTriggerOrder|Build signed reduce-only XYZ market trigger.|Only first inner error examined; empty statuses accepted; no receipt returned.
SetStopLoss|Create reduce-only SL trigger via standard or XYZ path.|Main SDK result discarded; side requires uppercase SHORT.
SetTakeProfit|Create reduce-only TP trigger via standard or XYZ path.|No OCO grouping; SDK acknowledgement body discarded.
PlaceLimitOrder|Create SDK GTC grid order and return rounded size/price.|Synthesizes UnixNano order ID, unusable as verified venue ID; XYZ special path not used; ClientID/PostOnly ignored.
CancelOrder|Parse supplied order ID and submit SDK cancellation.|Grid-generated timestamp ID sent as if venue ID; response body discarded.''')
add('trader/indodax/trader.go','''NewIndodaxTrader|Initialize REST client, nonce and caches.|30s timeout, spot-only adapter.
getNonce|Increment process-local nonce under mutex.|Uniqueness within instance, not across account-sharing instances.
sign|HMAC-SHA512 form body.|Caller owns nonce insertion and header use.
doPublicRequest|GET public endpoint and require200.|Read/HTTP errors propagated.
doPrivateRequest|Insert nonce, sign form POST and return successful API payload.|429 explicit; other HTTP codes rely on JSON success flag.
convertSymbol|Lowercase and split IDR/BTC/USDT quote suffix.|Unknown names pass through.
convertSymbolBack|Uppercase and remove underscores.|Canonical generic symbol output.
getCoinFromSymbol|Return first underscore-delimited coin.|Unknown no-underscore value retained.
loadPairs|Fetch and cache pairs for5minutes indexed by ticker and ID.|RWMutex protects replacement; no request coalescing.
getPair|Refresh pairs then lookup underscored or compact ID.|Missing metadata is an error here.
clearCache|Invalidate account and positions under mutex.|Executed before placement in order methods, not after acknowledgement.
parseFloat|Convert common primitive numeric representations.|Malformed/unknown/nil collapse to zero.''')
add('trader/indodax/trader_orders.go','''OpenLong|Submit spot BUY limit at current ticker.|Price always integer and qty8decimals, ignoring pair metadata; leverage ignored; status NEW honestly returned.
OpenShort|Refuse unsupported short selling.|Explicit error, no remote request.
CloseLong|Submit spot SELL limit, fetching all available coin if quantity<=0.|Negative quantity also means close-all; price rounded integer; caches invalidated before request.
CloseShort|Refuse unsupported short close.|Explicit error.
SetLeverage|Log no-op for spot-only.|Returns success without leverage support.
SetMarginMode|Log no-op for spot-only.|Returns success without margin support.
GetMarketPrice|Fetch compact-symbol ticker and parse last.|Bad number error propagated.
SetStopLoss|Refuse unsupported stop-loss order.|Explicit error; no protection installed.
SetTakeProfit|Refuse unsupported take-profit order.|Explicit error.
CancelStopLossOrders|No-op because stop orders unsupported.|Nil does not certify external order state.
CancelTakeProfitOrders|No-op because take-profit unsupported.|No remote request.
CancelAllOrders|Enumerate and cancel each spot order by pair/type/ID.|Individual cancellation errors logged and swallowed.
CancelStopOrders|No-op for unsupported stop orders.|No remote request.
FormatQuantity|Round down using pair PriceRound or8decimal fallback.|Uses price-round field rather than VolumePrecision; metadata failure silently accepted.
GetOrderStatus|Map spot statuses and return price/order ID.|Executed qty and fee fabricated zero; unknown statuses default NEW.
GetOpenOrders|Parse spot open orders into common records.|No quantity parsed; all orders report LONG/LIMIT/NEW; no-pair heterogeneous response unresolved.''')
add('trader/kucoin/trader_orders.go','''OpenLong|Cancel old orders, set leverage, convert base quantity to lots and market BUY cross.|Cancel/leverage failures tolerated; result marked FILLED without status confirmation.
OpenShort|Submit cross-margin market SELL in converted lots.|Same synthetic fill status.
queryOrderFillPrice|Sleep500ms then read dealAvgPrice.|Errors return0; status/dealSize parsed but never checked.
CloseLong|Invalidate cache, cap quantity to live long and submit reduce-only closeOrder.|No-position returns nil with NO_POSITION; order acknowledgement marked FILLED then protection canceled.
CloseShort|Invalidate cache, cap size to short and submit reduce-only closeOrder.|Negative requested size not explicitly rejected here; closeOrder semantics require venue confirmation for partial closes.
GetMarketPrice|Fetch converted-symbol ticker price.|Parse failure returns zero nil.
SetStopLoss|Submit mark-price-triggered reduce-only closeOrder with lot size.|Case-insensitive SHORT; fixed8decimal price, no tick alignment.
SetTakeProfit|Submit opposite mark-price trigger for TP.|Same size/closeOrder caveat.
CancelStopLossOrders|Delegate client-ID-based stop filtering.|Only orders with sl substring selected.
CancelTakeProfitOrders|Delegate client-ID-based TP filtering.|Only orders with tp substring selected.
cancelStopOrdersByType|Read stop list and delete IDs matching clientOid substrings.|Substring not strict namespace; individual failures swallowed.
CancelStopOrders|Bulk delete symbol stops.|Not-found or400100 errors treated nil.
CancelAllOrders|Attempt regular and stop cancellation.|Both failures can yield nil.
SetMarginMode|Log per-position handling without mutation.|New opens nevertheless hard-code CROSS.
SetLeverage|Request symbol leverage; tolerate same/already text.|Other errors propagated.
FormatQuantity|Divide by contract multiplier and nearest integer lots.|Returns lot count string, not base quantity; zero multiplier unguarded here.
GetOrderStatus|Normalize venue order fields.|ExecutedQty remains int64 contracts; done treated FILLED; unknown defaults NEW.
GetClosedPnL|Read max100 historical closed positions.|HasMore ignored; quantity contracts not converted; fee absent.
GetOpenOrders|Read regular orders plus optional stop orders.|Stop-fetch/parse failure silently partial; quantities contracts; sell inferred SHORT even for long exits.''')
add('trader/lighter/account.go','''getFullAccountInfo|Fetch wallet accounts, match stored index or select first.|Missing target silently switches read account; index0 treated unset.
GetBalance|Return equity-minus-UPL wallet and detailed margin fields.|Derived rather than independently sourced wallet balance.
GetAccountBalance|Parse account equity/collateral and estimate margins.|Zero equity replaced by collateral; maintenance margin estimated as half initial fraction; parse errors zero.
GetPositions|Map raw positions to common map fields.|Size positive with side separate; no additional normalization.
GetPositionsRaw|Parse positions, derive mark price/leverage/margin and optionally filter.|Assumes position size magnitude plus sign; numeric failures zero and empty sizes skipped.
GetPosition|Return first matching positive-size position or nil.|Relies on positive-size convention for shorts.
GetMarketPrice|Resolve market ID and read positive last_trade_price.|Checks HTTP, business code and symbol match; NaN not explicitly checked.
FormatQuantity|Format every market to4decimals.|TODO ignores actual symbol precision.
GetOrderBook|Fetch orderbook, parse string/numeric levels and truncate by depth.|Filters nonpositive levels; callback grouped here; parse failures collapse zero.''')
add('trader/lighter/order_sync.go','''SyncOrdersFromLighter|Fetch100 fills, sort, preserve derived order action and write order/fill/position.|Nontransactional order-first dedup; absent action defaults opening; fees labelled USDT.
StartOrderSync|Launch ticker sync and suppress404 log messages.|No lifetime cancellation or cleanup; API errors may already be swallowed.''')
add('trader/lighter/trader.go','''NewLighterTraderV2|Checksum wallet, select network, choose account, create TxClient and verify key.|apiKeyIndex int narrowed uint8 without bounds; invalid key check permits read-only initialization.
initializeAccount|Select account and save accountIndex under mutex.|First account chosen; index0 remains sentinel in GetTrades.
getAccountByL1Address|Read wallet main/subaccounts and return first normalized index.|No explicit account selection or matching binding validation.
getApiKeyFromServer|Read account/key-index public keys.|Returns first result without rechecking index identity.
checkClient|Compare SDK public key with server string.|Case/prefix-sensitive comparison; error exposes public keys only.
GenerateAndRegisterAPIKey|Return unimplemented error.|Does not register or generate despite name/historical graph wording.
refreshAuthToken|Generate7hour SDK auth token and cache under lock.|Reads tokenExpiry for log after unlock; callers read authToken without lock.
ensureAuthToken|Refresh within30minutes of expiration.|Concurrent callers can refresh simultaneously; no singleflight.
GetExchangeType|Return lighter identifier.|Constant pure method.
Cleanup|Log completion and return nil.|Does not stop sync goroutine or close client resources.
GetClosedPnL|Filter GetTrades for nonzero PnL and construct inferred records.|Its GetTrades always sets PnL0, so successful path always empty.
GetTrades|Fetch newest fills, map market IDs, choose account side and split position flips.|startTime unused; no pagination; wrong fee role may chosen before isTaker; errors masked empty; all PnL0; log includes auth URL prefix.''')
add('trader/okx/order_sync.go','''GetTrades|Read100 SWAP fills, convert contract count to base qty and infer hedge action.|Net-mode sells/buys always opens; metadata failure treats lots as base qty; no pagination.
SyncOrdersFromOKX|Sort last24h fills then persist order/fill/PositionBuilder.|Existing-order skip prevents downstream repair; no PnL from fills so zero passed to builder.
StartOrderSync|Launch recurring OKX sync.|No context, stop handle or ticker cleanup.''')
add('trader/okx/trader.go','''genOkxClOrdID|Generate tagged timestamp/random ID and truncate32characters.|Tag16+timestamp13 leaves about3random hex chars after truncation, unlike apparent8hex entropy; random error ignored.
NewOKXTrader|Construct30s HTTP client, detect mode and attempt hedge switch.|Remote account mutation; successful switch does not update positionMode field; detect failure assumes hedge.
marginMode|Resolve bool to cross/isolated string.|No lock, intended local mode value.
detectPositionMode|Fetch account config and record first posMode.|Empty list returns nil leaving empty mode.
setPositionMode|Request long_short_mode account setting.|Does not set local positionMode after success or already-set response.
sign|HMAC-SHA256 timestamp+verb+path+body with Base64.|Caller must pass identical wire path/body.
doRequest|Sign JSON HTTP request and unwrap OKX data.|Always simulated-trading0; accepts code1 partial success for callers to inspect; HTTP status not independently checked.
convertSymbol|Append -USDT-SWAP after removing USDT.|Not idempotent for already-formatted symbol; case-sensitive.
convertSymbolBack|Join first two hyphen-separated instrument segments.|Ignores expiry/type remainder.
FormatQuantity|Divide base quantity by ctVal then format lots.|Metadata error falls back to unconverted base units with nil error.
formatSize|Choose decimal places from lotSz and round size.|Does not enforce non-power-of10 lot multiples; %f loses precision below1e-6.''')
files=[]
for f in a['files']:
 p=f['path'];sem=[v[0] for k,v in notes.items() if k[0]==p]
 files.append(dict(path=p,mode='full/manual',ranges=[[1,f['lines']]],sha256=hashlib.sha256((ROOT/p).read_bytes()).hexdigest(),notes='Assigned source, fully manually read. '+ '; '.join(sem)))
extras=[('trader/types/interface.go',[[1,253]],'Full shared19method interface and grid adapter; exact length normalized below.'),('trader/auto_trader.go',[[620,740],[850,943]],'Constructor exchange switch and exchange-specific sync startup; crypto lanes distinct from ninjatrader.'),('trader/aster/trader_positions.go',[[1,70]],'Position value types, positive short size and float leverage feeding assigned balance/close logic.'),('store/order.go',[[150,235]],'Order/fill idempotency and account-scoped ID lookup.'),('store/position_builder.go',[[1,140]],'ProcessTrade dispatch and non-idempotent position averaging; closed-without-open behavior excerpt.'),('trader/aster/trader_test.go',None,'Full HTTP mock suite; mock parses JSON body while production uses form.'),('trader/binance/order_sync_test.go',None,'Full live-gated API diagnostics; not executed.'),('trader/hyperliquid/trader_race_test.go',None,'Full metadata locking stress tests; not executed.'),('trader/okx/trader_margin_mode_test.go',None,'Full recording transport tests for configured margin mode; fixtures preseed hedge mode and bypass constructor.'),('trader/okx/trader_orders.go',[[230,250],[345,360]],'Close request uses cached positionMode to decide posSide.'),('docs/superpowers/AUDIT-CHECKLIST.md',[[1,105],[2281,2310]],'Pre-audit R1-R10, identity/time/binding/refusal evidence; no DB or runtime audit.'),('docs/superpowers/SYSTEM-MAP.md',[[250,269]],'NT8 execution and SIM-only description; treated as source hypothesis.'),('docs/superpowers/VL-TRADING-RULEBOOK-v1.md',[[215,238]],'Owner scope and daily-loss policy, not runtime certification.'),('docs/superpowers/CLAUDE-canon.md',None,'Full tracked operating rules; corrected keeper and worktree checks.')]
for p,r,n in extras:
 b=(ROOT/p).read_bytes();length=len(b.splitlines());r=r or [[1,length]]
 if p=='trader/types/interface.go':r=[[1,length]]
 files.append(dict(path=p,mode='full/manual' if r==[[1,length]] else 'excerpts/manual',ranges=r,sha256=hashlib.sha256(b).hexdigest(),notes=n,additional_dependency=True))
json.dump(dict(assignment=6,base=a.get('base','63968be62e44db2fb07a92883e02127b9064b0be'),files=files,unread=[]),open(OUT/'reads.json','w'),indent=2)
rows=[r for r in csv.DictReader(open(CAT/'go-functions.tsv'),delimiter='\t') if r['path'] in {f['path'] for f in a['files']}]
calls={}
for r in csv.DictReader(open(CAT/'go-calls.tsv'),delimiter='\t'):
 calls.setdefault(r['caller_id'],[]).append(r['callee_expression'])
functions=[]
for r in rows:
 key=(r['path'],r['name'])
 if r['name']=='<anonymous>':
  parents=[v for v in rows if v['path']==r['path'] and v['name']!='<anonymous>' and int(v['start_line'])<int(r['start_line']) and int(v['end_line'])>=int(r['end_line'])]
  parent=parents[-1]['name'] if parents else 'okxTag package initializer'
  purpose='Callback/local closure of '+parent+'; '+('sort ascending by fill time' if int(r['end_line'])-int(r['start_line'])==2 and parent.startswith('Sync') else 'periodic/initial sync task' if parent=='StartOrderSync' else 'decode package order tag' if not parents else 'parse mixed numeric orderbook fields')
  risk='Inherits parent lifecycle and error semantics; callback inventoried separately.'
 else:
  if key not in notes:raise Exception(key)
  purpose,risk=notes[key]
 funcs=list(dict.fromkeys(calls.get(r['id'],[])))
 functions.append(dict(path=r['path'],name=r['name'],receiver=r['receiver'],start_line=int(r['start_line']),end_line=int(r['end_line']),purpose=purpose,connections=['Direct call expression '+x+' (syntax-only; receiver target not inferred)' for x in funcs],invariants_or_risks=risk,evidence='A: manually read source; static behavior, not a runtime incident'))
json.dump(functions,open(OUT/'functions.json','w'),indent=2)
print('saved',len(a['files']),'assigned files',sum(f['lines'] for f in a['files']),'lines',len(functions),'functions/closures; named',sum(x['name']!='<anonymous>' for x in functions))
