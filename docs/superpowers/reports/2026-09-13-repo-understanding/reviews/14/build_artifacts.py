import json,re,csv,hashlib,pathlib
base=pathlib.Path('/tmp/nofx-understanding-surfaces-20260913'); dst=pathlib.Path('/tmp/nofx-review-14'); root=pathlib.Path('/tmp/nofx-repo-understanding-20260913/docs/superpowers/reports/2026-09-13-repo-understanding')
a=next(a for a in json.load(open(root/'review-plan.json'))['assignments'] if a['id']==14)
notes={
'VLBarsSubscriptionManager.cs':'Native per-root/timeframe BarsRequest lifecycle, DoNotMerge, event range emission, close stamps, platform contract resolution; reconnect preserves cursor and emits no subscribed ACK; watchdog rebuilds 3 fast attempts then75min backstop.',
'VLContractResolver.cs':'Quarterly-root normalization and rolling ##-## naming; old expiry-minus8days calculation retained as explicit fallback, not primary. ResolveFrontMonthContractAt documentation remains stale.',
'VLHistoryPull.cs':'Independent explicit-contract DoNotMerge request; 8000-bar chunks with request id/seq/last; unguarded dictionary writes race callback removals; duplicate-tail timestamps can skip final flush.',
'VLInstrumentLookup.cs':'Rolling instrument -> MasterInstrument.GetNextExpiry(DateTime.Now) -> concrete instrument, logged date fallback. A qualified input still goes through next-expiry resolution, so passthrough comments do not describe Resolve behavior.',
'VLTraderTCPClient.cs':'NT8 AddOn v3/h1: connection, SIM/account guards, entries, deferred partial-fill brackets, standalone stops, account/position/order snapshots and JSON codec. See report for cancel/fill race, close identity, periodic account scope, absent map reattachment and partial outcome risks.',
'bar_cache.go':'2500 bounded open-stamped cache; placeholder rejection; historical union preserving live; append/update tail; older-store extension only after warm seed. Keys retain incoming case despite case-insensitive admission.',
'bar_persist.go':'Global bounded4096 queue,300ms/256 flush batching; close-bearing enqueue retries6s then drops counted. Watchdog runs on same blocked worker as callback and stamps successful after contained panic; cannot independently detect synchronous callback hang.',
'bar_source.go':'Live/historical/mixed provenance and first-live-after-seed percent AND median-body scale test; historical purge on mismatch; async listeners. PurgeSymbol does not reset metadata; empty seed resets verdict despite retaining data.',
'contract_roll.go':'Current contract from subscribed ACK only; changed named contract purges ring and fires listeners. Bar contract field is not decoded, reconnect rebuild emits no ACK. Queued old messages have no epoch and can repopulate after purge.',
'csv_tailer.go':'Legacy full-file poll with row-count cursor; shrink resets; malformed row skipped and cursor advanced. Not live trading path.',
'csv_writer.go':'Legacy validated one-row signal CSV written via temp+rename,3 retries; uppercases direction and rounds2decimal. Does not queue multiple outstanding orders.',
'echo_verify.go':'Seq identity registry bounded4096; known mismatch freezes; seq0/unknown retired accepted; rejected-empty-account tolerated. Compares trader/account, not echoed signal id; seq reused only per server object lifecycle.',
'mock_nt.go':'Offline CSV mock polling100ms, dedup timestamp+direction+entry, delayed synthetic fills. No real execution semantics.',
'order_snapshot.go':'Per-account latest book cache and receipt ages; conservative terminal filtering; missing/null orders coerced empty; cache retains mutable slices and separate latest/age accessor can describe different arrivals.',
'order_state.go':'Normalized6-class liveness; unknown nonterminal; suspended/partfilled live in Go versus Working/Accepted only C#.',
'research_wire.go':'Research fact recording for MNQ bars and selected execution fields; connection-scoped prior observations12000; contract inferred preceding ACK despite contract on bar wire. Outer broker_frame orders raw can retain unknown nested fields; sanitized child facts do not sanitize parent.',
 'tcp_client_mock.go':'Test-only TCP peer with delayed fills and signal channel; no hello/identity/account echo by default; concurrent WriteFrame calls lack shared write mutex; deadline continuation retains old partial-frame desync pattern.',
 'tcp_framing.go':'4-byte BE JSON1MB envelopes; v3 identity, h1 capability, account/order/history payloads; live bar payloads omit emitted contract. WriteFrame assumes full writes without checking n.',
 'tcp_server.go':'Single connection, account/symbol fanout, lifecycle, metadata, queued60s-valid signals and nonqueued management; serialized writes, bar drain, history import channels. See findings: import send/close race, epoch/hello gaps, contract discard, queue and stale cache semantics.',
 'types.go':'Legacy CSV types; direction/geometry validation checks positive values but NaN can bypass comparisons. No live TCP schema role.'}
reads=[]
for f in a['files']:
 p=f['path']; raw=(base/p).read_bytes(); assert hashlib.sha256(raw).hexdigest()==f['sha256']
 reads.append(dict(path=p,mode='full/manual',ranges=[[1,f['lines']]],sha256=f['sha256'],notes=notes[pathlib.Path(p).name]))
extra={'trader/protection_reconciler.go':None,'trader/bracket_cancel_addon_test.go':None,'provider/ninjatrader/order_snapshot_test.go':None,'provider/ninjatrader/contract_roll_test.go':None,'trader/ninjatrader/tcp_trader.go':[[705,851]],'trader/ninjatrader/bar_persist_wire.go':[[70,125]],'trader/auto_trader.go':[[85,119]],'trader/auto_trader_risk.go':[[43,66]],'docs/superpowers/CLAUDE-canon.md':None,'docs/superpowers/AUDIT-CHECKLIST.md':[[2281,2316],[2741,2923]],'docs/superpowers/SYSTEM-MAP.md':[[252,267],[290,300]],'docs/superpowers/VL-TRADING-RULEBOOK-v1.md':[[215,265]]}
for p,r in extra.items():
 raw=(base/p).read_bytes(); n=len(raw.splitlines());reads.append(dict(path=p,mode='full/manual' if r is None else 'excerpts/manual',ranges=r or [[1,n]],sha256=hashlib.sha256(raw).hexdigest(),notes='Supplementary dependency/test/policy read; excluded from assigned coverage.'))
json.dump(dict(assignment=14,base='63968be62e44db2fb07a92883e02127b9064b0be',files=reads,unread=[]),open(dst/'reads.json','w'),indent=2)
# AST declaration census for Go; C# member declaration census includes nested parser methods.
syms=[]
for r in csv.DictReader(open(root/'go-functions.tsv'),delimiter='\t'):
 if r['path'] in {f['path'] for f in a['files']} and r['kind']=='declaration': syms.append(dict(path=r['path'],name=r['name'],start_line=int(r['start_line']),end_line=int(r['end_line'])))
for f in a['files']:
 if not f['path'].endswith('.cs'):continue
 lines=(base/f['path']).read_text().splitlines()
 for i,l in enumerate(lines):
  m=re.match(r'^\s+(?:private|public|internal|protected) (?:static |override )*(?:[\w<>, \[\]]+\s+)?(\w+)\(',l)
  if not m:continue
  # Member body ends at matching braces; no verbatim strings with braces in these member headers.
  j=i;depth=0;entered=False
  while j<len(lines):
   code=re.sub(r'"(?:\\.|[^"\\])*"','""',lines[j].split('//')[0]);code=re.sub(r"'(?:\\.|[^'\\])'","''",code);depth+=code.count('{')-code.count('}')
   entered=entered or '{' in code
   if entered and depth==0:break
   j+=1
  syms.append(dict(path=f['path'],name=m.group(1),start_line=i+1,end_line=min(j+1,len(lines))))
purposes={}
def register(text):
 for line in text.strip().splitlines():
  names,purpose=line.split('|',1)
  for n in names.split(','):purposes[n]=purpose
register('''HandleCancelOrder|Cancel tracked entry then delete deferred bracket intent and ACK; deletion also occurs after caught cancel failure.
SubmitBracketOnEntryFill|On cumulative entry fill amend existing bracket or consume pending intent and submit GTC SL/TP; missing intent silently returns.
AmendBracketQuantity|Stage cumulative filled quantity on surviving bracket legs with Account.Change; no second pair.
HandleClosePosition|Resolve current instrument and root-only position owner, then limit exit or account.Flatten; ignores explicit account and identity stamp.
HandlePlaceProtectiveStop|Validate side/positive qty/stop and SIM-connected allowlisted target; submit standalone GTC stop without bracket-map registration.
HandleModifyBracket|Find signal bracket and sequentially change requested live stop/target; target refusal may follow successful stop change.
HandleMoveStop|Find already tracked live stop and change price in place, leaving OCO intact.
OnOrderUpdate|Emit state and snapshot then branch full/partial/rejected entries/exits; entry fills create/amend bracket, full exit emits close.
HandleSignal|Parse signal, stamp identity, guard active/routed SIM connected allowlist, resolve instrument, register pending intent then submit Day entry.
HandleFrame|Parse JSON envelope and dispatch command/hello/heartbeat to typed handler; exceptions logged.
OnStateChange|Construct and start AddOn managers/threads/subscriptions or dispose them on termination.
RunConnectionLoop|Connect5s retry; hello,accounts,balances,positions before read loop; maps survive TCP reconnect.
RunReadLoop,ReadExact|Read exact length-prefixed payload bytes; terminate on error or oversize.
SendOrderSnapshot|Copy account nonterminal orders into explicit array with build/time metadata and emit broker book.
SendOrderUpdateFrame|Dedup state by order name, stamp original signal identity, emit event and account snapshot.
HandleAccountRegister|Replace nonempty authoritative account allowlist; empty set ignored and undeclared state fails open.
HandleAccountSelect|Switch active account events without clearing global signal maps; old OrderUpdate unsubscribed.
RegisterSessionAccount|Add owner-selected account to session set.
IsSessionAccount|Accept any name before declaration; otherwise enforce set membership.
ResolveAccount|Read preferred account.txt then Sim101 then first Account.All entry; submission performs SIM guard later.
IsSimAccount|Reflect Simulation bool or case-sensitive Sim prefix fallback.
IsRealAccount|Filter named internal account prefixes and require broker+price connection for list visibility.
SubscribeOrderUpdate,UnsubscribeOrderUpdate|Idempotently maintain OrderUpdate hooks under orderSubLock.
SubscribePositionUpdatesAllSim,UnsubscribeAllPositionUpdates|Maintain all-SIM PositionUpdate hooks and teardown registry.
OnVLConnectionStatusUpdate|Track price-feed transitions; debounce30s request rebuild; emit feed status.
CancelBracketsFor|Find signal or account/root bracket after filled limit exit; cancel selected pair then remove matched tracking keys.
CancelAllBracketsFor|On netting-flat collect and cancel every tracked account/root pair, then remove tracking even if cancellation fails.
OnPositionUpdate|Send event-overridden account snapshot and sweep tracked brackets if flat.
SendOpenPositions|Emit complete account nonflat positions; overload substitutes by-value event quantity/average for changing instrument.
RunHeartbeatLoop|Every30s emit heartbeat, active-account order book and all-SIM balance/positions; accounts list every90s.
StampIdentity|Lookup original signal trader/seq and add to fill/close/order event, absent lookup tolerated.
SendAck|Emit only acks string; C# does not send identity-bearing operation ACK.
WriteEnvelope|Encode and write under shared lock; dropped/no-stream/error only logged, no returned delivery result.
EncodeFrame,AppendObject,AppendValue,AppendString|Build invariant-culture compact JSON recursively; escape strings and support enumerable arrays.
JsonParser,Parse,ParseValue,ParseObject,ParseString,ParseNumber,ParseBool,ParseNull,ParseArray,SkipWs|Handwritten JSON recursive-descent parser; Parse does not assert full input consumption and nesting is unbounded.
GetString,GetDouble,GetInt,GetLong,TryGetString,TryGetInt,TryGetLong,TryGetStringList|Read typed dictionary field with absent default; numeric conversion may throw or narrow.
SendHello,SendFillFrame,SendPositionCloseFrame,SendPositionCloseRejectedFrame,SendFeedStatusFrame,SendFrame|Construct corresponding wire payload and delegate to serialized envelope writer.
OnAccountStatusUpdate|Log event and refresh available-account list; no subscription to this handler found in file.
SendAccountsList|Enumerate connected visible accounts and emit names/SIM flags.
SendAccountBalance,SendAccountBalanceFor,SendAllAccountBalances,SendAllOpenPositions,OnAccountItemUpdate|Read/send account state; all-account polls supplement active-only balance change hook.
LogInfo,LogWarn|Best-effort NT8 logging, exceptions contained.
VLBarsSubscriptionManager|Store sender/log delegates and start15s watchdog timer.
HandleBarsSubscribe|Resolve platform contract and specs, subscribe each timeframe, emit symbol ACK even if individual timeframe failed.
HandleBarsUnsubscribe|Dispose selected symbol/timeframe entries and emit removed count.
Subscribe|Replace existing request, map timeframe, create DoNotMerge ETH request, hook update before historical callback.
DisposeEntry,DisposeAll|Dispose owned request resources/timer and clear registry; in-flight callbacks not joined.
EmitHistorical|Send full ascending close-stamped historical bars after cursor filter; mark seeded and update emission cursor.
OnBarsUpdate|After historical seed emit changed-index range at or beyond cursor with contract metadata and live marker.
OnConnectionReconnected|Dispose/re-resolve every live subscription and restore previous time cursor; emits no subscribed ACK.
WatchdogTick|Evaluate seeded-not-live fast failures, max3 rebuilds per degraded period;75min stale backstop.
NowUtcMs,SnapshotNowUtcMs,DateTimeFromUtcMs|Convert UTC wall time to/from epoch milliseconds.
MapTimeframe,BarsPeriodFor|Map supported timeframe strings to native minute/day/week BarsPeriod; unsupported refused.
ToUtcEpochMs|Convert source bar timestamp using TradingHours timezone or local fallback, preserving UTC input.
BuildBarObject|Build six-field OHLCV dictionary.
ResolveFrontMonthContract,RollingContractName|Normalize quarterly root/continuous form to rolling ##-## name; qualified/nonquarterly passthrough.
DateRuleContract,ResolveFrontMonthContractAt,ThirdFridayOf,IsQuarterlyRoot|Fallback quarterly expiry-minus8days resolver and root/date helpers; UTC production fallback with testable clock.
Resolve|Resolve rolling then platform next expiry concrete contract; logged date fallback when unavailable.
ContractName|Render actual instrument expiry root MM-YY, fallback FullName.
VLHistoryPullManager|Initialize independent named-contract pull delegates and registry.
HandleBarsHistoryRequest|Validate correlation and instrument/window; create disposable DoNotMerge request; callback emits chunks/error and removes request.
EmitChunks|Emit ascending deduplicated8000-bar chunks with seq/last; last skipped duplicate can bypass terminal flush.
SendError|Emit history request error with correlation and contract.
ParseOrderSnapshot|Decode and require account; coerce missing/null orders to empty book, contrary to no-fabricated-values boundary.
checkEcho|Verify known seq trader/account; tolerate seq0 or unknown registry entry and rejected empty-account.
verifyInbound|Call echo checker; freeze known owner on mismatch and suppress processing; warn tolerated gap.
assignSeqRegister|Allocate monotonic seq and store op+signal reverse index; drop oldest above4096.
lookupPending,retirePending|Lookup/remove registry by seq or fallback signal; no validation of incoming signal against stored signal.
FarSideProven|Bytewise nonempty build-id minimum comparison; date must advance for safe future floor.
WriteFrame,ReadFrame,marshalEnvelope|Go JSON envelope1MB codec; exact read; writer ignores short-write counts.
readLoop|Read and dispatch peer frames; server processes account/book/events/bars/history and verifies selected identity echoes; mock branch is test-only.
acceptLoop|Accept single peer and start reader/heartbeat; send allowlist,queued signals,bars before hello validation completes.
flushPending|Drain queued signals, recheck original timestamp under writer lock; on write failure requeue unsent tail and close connection.
checkSignalAge|Reject malformed,future,older-than60s original signal timestamps.
closeConn|Close currently stored connection without checking which lifecycle callback requested it.
observeContract|Record subscribed contract, purge ring on changed prior name, invoke roll listeners.
CurrentContract|Read subscribed contract and observed roll metadata; no bar-frame contract support.
drainBarIngest|Apply cache seed/upsert, derive closed cache tail or historical candidates, enqueue persistence.
enqueueBarUpdate|Stamp live receipt and nonblocking enqueue; drop oldest message of shared queue when full.
enqueueBarHistorical|Bounded2s enqueue before historical drop count.
sampleIngestDepth|Update queue peak;17th local-clock hour reset CAS can repeatedly clear peak.
startPersistWorker|Start global single queue consumer; batch256 or300ms, contain callback panic, update flush stamp; watchdog shares same goroutine.
persistWatchdogAlarmAt|Alarm after flush gap only while recent live persistence candidates arrive; zero initial flush suppresses alarm.
ClosedBarsOnly,closedBarsOnly,hasClosedBar,ClosedCacheTail|Identify closed bars by open stamp+duration, take recent closed tail for persistence.
SeedHistorical|Placeholder-filter, open-stamp, historical-mark and merge preserving existing live/mixed on overlap; reset seed verdict.
Upsert|Placeholder-filter,open-stamp,live-mark,scale-test then append/replace latest bar; older updates ignored.
RehydrateOlder|Extend warm ring with strictly older store rows; never seed cold feed; cap preserves live tail.
detectScaleMismatch|Compare first-live close to prior historical scale using pct and median body, purge historical and mark mixed; notify asynchronously.
mergeSeedKeepingLive,mergeBarsByTime|Linear ordered union; ordinary merge incoming wins, historical merge preserves existing live/mixed.
openStampBars,OpenStampBars,timeframeMs|Convert NT close timestamps once into canonical open stamps using timeframe duration table.
PurgeSymbol|Delete every matching symbol cache slice; leaves scale-verdict metadata and queued messages untouched.
barKey,splitBarKey|Encode/decode exact-case symbol|timeframe cache keys.
normalizeOrderState,ClassifyOrderState,Liveness|Canonicalize separators/case then grade order state; unknown defaults conservative nonterminal.
IsWorking,IsCancelInFlight,IsHeldLocally,IsStateReadable|Named predicates over shared Go liveness vocabulary.
IsLiveAtExchange|Go allows accepted/working/suspended/partfilled; C# allows only Working/Accepted.
facts|Create research snapshots from selected raw fields; MNQ contract uses preceding ACK, parent raw orders unfiltered.
observe,observeAt,recordResearchSignal,recordResearchSignalAt|Record bounded research facts with receipt clock and explicit unproven transport semantics.
RequestDeepBarsBackfill|Start global replay capture and send deeper live subscription; receive handler currently never invokes captureBarTruthReplay.
fanOutBarPersist|Enqueue persistence candidates; close-bearing full queue retries3x2s then loud counted drop.
Validate|Legacy CSV positive directional stop/target checks; no finite-number validation.
StartMockNT,processSignals,writeFill|Test-only CSV poll/dedup/delayed synthetic fill pipeline.
scheduleFill,SendFill,sendFrameForTest|Test-only delayed/manual wire emission; lacks synchronization shared with mock read-loop writes.
''')
# For small accessors/config/registrations, the name plus implementation boundaries below describes the operation.
for s in syms:
 lines=(base/s['path']).read_text().splitlines();body='\n'.join(lines[s['start_line']-1:s['end_line']]); name=s['name']; purpose=purposes.get(name)
 if not purpose:
  # Extract the declaration's immediate contract comment, retaining a bounded description; reviewed source, not external graph.
  pre=[];i=s['start_line']-2
  while i>=0 and lines[i].lstrip().startswith('//'):
   pre.insert(0,lines[i].strip().removeprefix('//').strip());i-=1
  purpose=' '.join(pre)[:650] if pre else 'Small '+name+' helper in '+pathlib.Path(s['path']).name+'; '+notes[pathlib.Path(s['path']).name]
 s['purpose']=purpose
 calls=sorted(set(re.findall(r'\b([A-Za-z_]\w*(?:\.[A-Za-z_]\w*)?)\s*\(', re.sub(r'//[^\n]*','',body))))
 s['connections']=['Lexical call/reference (not type-resolved): '+x for x in calls if x not in {name,'if','for','switch','return','func','lock','catch','typeof','sizeof'}][:35]
 if not s['connections']:s['connections']=['Reads/returns or updates fields owned by '+pathlib.Path(s['path']).name]
 s['invariants_or_risks']=notes[pathlib.Path(s['path']).name]
json.dump(syms,open(dst/'functions.json','w'),indent=2)
print('functions',len(syms),'assigned',len(a['files']))
g=json.load(open('/home/hoang/nofx-untracked-stash-20260816/.understand-anything/knowledge-graph.json'))
paths={f['path'] for f in a['files']};ns=[n for n in g['nodes'] if n.get('filePath') in paths];ids={n['id'] for n in ns};es=[e for e in g['edges'] if e['source'] in ids or e['target'] in ids]
json.dump(dict(assignment=14,historical_base='July10@7a8adce0',current_base='63968be62e44db2fb07a92883e02127b9064b0be',matched_nodes=len(ns),matched_edges=len(es),historical_file_nodes=[n for n in ns if n['type']=='file'],corrections=['Old graph AddOn summary says protocol v2; current constants3 on both sides.','Old resolver summary says date-derived primary; rolling->platform next expiry now primary, date fallback only.','Graph lacks9 of20 assigned files including history, instrument lookup, roll, source, persistence, echo, research and book/state additions.','Graph package imports expanded to every package file are not evidence of function-level call edges.','Current fanout owner key is symbol+account, not merely symbol.'],edges=[dict(source='VLTraderTCPClient.HandleFrame',target='VLBarsSubscriptionManager.HandleBarsSubscribe',evidence='ninjascript/VLTraderTCPClient.cs:700'),dict(source='VLTraderTCPClient.OnOrderUpdate',target='VLTraderTCPClient.SubmitBracketOnEntryFill',evidence='ninjascript/VLTraderTCPClient.cs:1445'),dict(source='TCPServer.readLoop/FrameSubscribed',target='TCPServer.observeContract',evidence='provider/ninjatrader/tcp_server.go:1944'),dict(source='TCPServer.drainBarIngest',target='BarCache.SeedHistorical/Upsert',evidence='provider/ninjatrader/tcp_server.go:1600')]),open(dst/'graph.json','w'),indent=2)
