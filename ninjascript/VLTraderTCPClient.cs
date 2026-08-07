// VLTraderTCPClient.cs — Plan 1.5 TCP NinjaTrader bridge (alternative to CSV).
//
// NinjaScript AddOn that connects to the VL Trader Go bot via local TCP
// (127.0.0.1:36974) and exchanges length-prefixed JSON frames with the Go-side
// server in provider/ninjatrader/tcp_server.go. This file CANNOT be compiled
// in WSL2 — the operator compiles on Windows via NT8's NinjaScript editor (F5)
// or Visual Studio. See vltrader_tcp_README.md for the install procedure and
// vltrader_tcp_PROTOCOL.md for the wire-format spec.
//
// Spec: docs/superpowers/plans/2026-05-22-nq-databento-ninjatrader.md L4343-4447.
// ADR : docs/adr/ADR-001-csv-bridge-vs-tcp.md.

#region Using declarations
using System;
using System.Collections.Generic;
using System.Globalization;
using System.IO;
using System.Net.Sockets;
using System.Text;
using System.Threading;
using NinjaTrader.Cbi;
using NinjaTrader.Core;
using NinjaTrader.NinjaScript;
#endregion

namespace NinjaTrader.NinjaScript.AddOns
{
    /// <summary>
    /// VLTrader TCP client. Loaded automatically by NT8 when dropped in
    /// Documents\NinjaTrader 8\bin\Custom\AddOns\. Lifecycle is driven by
    /// OnStateChange (SetDefaults / Active / Terminated).
    /// </summary>
    public class VLTraderTCPClient : AddOnBase
    {
        // === Connection constants (must match Go-side tcp_server.go) ===
        private const string GO_SERVER_HOST          = "127.0.0.1";
        private const int    GO_SERVER_PORT          = 36974; // NOT NT's ATI port 36973
        private const int    HEARTBEAT_INTERVAL_MS   = 30000; // spec L4408
        private const int    RECONNECT_INTERVAL_MS   = 5000;  // spec L4415
        private const int    STALE_SIGNAL_AGE_SECONDS = 60;   // spec L4414
        // P5.2 wire-protocol generation. v2 = symbol-tagged fills + the hello
        // handshake. MUST match provider/ninjatrader/tcp_framing.go
        // ProtocolVersion — bump ONLY with a lockstep C#+Go ship.
        private const int    PROTOCOL_VERSION        = 2;
        private const int    MAX_FRAME_BYTES         = 1 << 20; // 1 MB, spec L4376

        // === State ===
        private TcpClient                client;
        private NetworkStream            stream;
        private Thread                   readerThread;
        private Thread                   heartbeatThread;
        private CancellationTokenSource  cts;
        private Account                  account;   // primary trading account
        private readonly object          writeLock = new object();

        // Track signal_id → original entry price for slippage calculation.
        private readonly Dictionary<string, double> signalEntryByOco = new Dictionary<string, double>();
        private readonly Dictionary<string, double> signalTickSizeByOco = new Dictionary<string, double>();
        private readonly object signalMapLock = new object();

        // Pending protective brackets, keyed by signal_id. The SL/TP are placed
        // only AFTER the entry fills (see OnOrderUpdate) — placing them in the
        // entry's OCO group caused NT to cancel them the instant the entry
        // filled (OCO = one-cancels-other), leaving the position unprotected.
        private readonly Dictionary<string, PendingBracket> pendingBrackets = new Dictionary<string, PendingBracket>();

        // PHASE 3 (per-order routing) — the set of accounts whose OrderUpdate we have
        // hooked to OnOrderUpdate. The fill→bracket→fill-frame lifecycle must fire for
        // EVERY account we route an order to, not just the active one. This set is the
        // single source of truth, so we never double-subscribe (which would duplicate
        // fills). Per-account balance/position snapshots remain Phase 4.
        private readonly HashSet<Account> orderSubscribed = new HashSet<Account>();
        private readonly object orderSubLock = new object();

        // PHASE 4 (per-account close) — which account holds each open position, keyed by
        // ROOT symbol (e.g. "MNQ"). Populated on entry-fill (from e.Order.Account) and
        // cleared on exit-fill, so HandleClosePosition flattens the CORRECT account
        // instead of the active one. Back-compat: one trader → resolves to Sim101.
        private readonly Dictionary<string, Account> positionAccountBySymbol = new Dictionary<string, Account>();
        private readonly object posAcctLock = new object();

        private class PendingBracket
        {
            public Instrument  Instrument;
            public OrderAction ExitAction;
            public int         Qty;
            public double      Sl;
            public double      Tp;
            public Account     Account;   // PHASE 3: the routed submit account (SL/TP follow the entry)
        }

        // A bracket whose entry has FILLED and whose SL/TP now rest at the exchange,
        // keyed by signal_id. Populated in SubmitBracketOnEntryFill, removed on exit
        // fill. Lets HandleMoveStop find the live resting stop to MOVE it (auto-
        // breakeven) without closing the position.
        private class PlacedBracket
        {
            public Order       SlOrder;     // the live resting stop order (movable)
            public Account     Account;
            public Instrument  Instrument;
            public OrderAction ExitAction;
            public int         Qty;
            public string      ExitOco;     // the OCO group the SL + TP share
            public double      TickSize;
        }
        private readonly Dictionary<string, PlacedBracket> placedBrackets = new Dictionary<string, PlacedBracket>();

        // Plan 4.4 Stage 1 — multi-timeframe BarsRequest subscriptions. Owns
        // its own state; calls back into SendFrame() which serializes through
        // writeLock so bar frames cannot interleave with signal/fill bytes.
        private VLBarsSubscriptionManager barsManager;

        // Phase 0 — data-feed reconnect recovery state. NT8 does NOT auto-resume
        // a BarsRequest after the price feed drops, so we recreate them on
        // recovery (VLBarsSubscriptionManager.OnConnectionReconnected). Guard so
        // only a genuine was-lost -> Connected transition fires the recreate.
        private readonly object connStatusLock = new object();
        private bool dataFeedWasLost = false;
        // Debounce for the recreate-on-(re)connect path. The recreate is now a
        // full multi-timeframe/multi-symbol BarsRequest refetch with a DEEP
        // (2000-bar / ~33h) lookback so it can backfill an overnight feed-gap
        // from NT8's DB. That depth makes thrashing expensive, so a rapidly
        // flapping feed must coalesce into ONE rebuild per cooldown rather than
        // rebuilding on every micro-blip. A genuine recovery (or startup) still
        // fires once; the 20s FAST_STALL fast-guard backstops a .Update that dies
        // inside the cooldown window.
        private DateTime lastRecreateUtc = DateTime.MinValue;
        private static readonly TimeSpan RecreateDebounce = TimeSpan.FromSeconds(30);

        protected override void OnStateChange()
        {
            if (State == State.SetDefaults)
            {
                Name = "VLTrader TCP Client";
                Description = "Connects to the VL Trader Go bot via local TCP (Plan 1.5). " +
                              "Receives signals over the wire, places OCO bracket orders, " +
                              "and emits fills back. Falls back to CSV bridge if disabled.";
            }
            else if (State == State.Active)
            {
                cts = new CancellationTokenSource();
                ResolveAccount();
                if (account != null)
                {
                    SubscribeOrderUpdate(account);   // PHASE 3: dedup'd OrderUpdate subscription
                    // Plan 4.11 — emit the real account balance on cash/PnL
                    // changes so the dashboard reflects the live SIM account.
                    account.AccountItemUpdate += OnAccountItemUpdate;
                    // Open-position read-back — emit the account's current open
                    // positions on ANY change (incl. MANUAL trades in NT8) so the
                    // Go side / AI always tracks NT8's real position. The initial
                    // snapshot is also sent on connect + on account_select.
                    account.PositionUpdate += OnPositionUpdate;
                }
                // Plan 4.4 Stage 1 — instantiate the bars manager BEFORE the
                // reader thread starts so an early bars_subscribe frame has a
                // destination. SendFrame is wired through this instance so
                // bar frames inherit the same writeLock + encoder used by the
                // proven signal/fill/heartbeat path.
                barsManager = new VLBarsSubscriptionManager(SendFrame, LogInfo, LogWarn);

                // Phase 0 — hook the NT8 data-feed status so BarsRequests get
                // recreated when the NT8<->provider price feed recovers. Without
                // this, NT8 silently stops delivering bars after a feed reconnect
                // and the Go BarCache goes stale with no recovery. Static event
                // across all connections; the handler filters on PriceStatus.
                Connection.ConnectionStatusUpdate += OnVLConnectionStatusUpdate;

                readerThread = new Thread(() => RunConnectionLoop(cts.Token))
                {
                    IsBackground = true,
                    Name = "VLTrader-TCP-Reader"
                };
                heartbeatThread = new Thread(() => RunHeartbeatLoop(cts.Token))
                {
                    IsBackground = true,
                    Name = "VLTrader-TCP-Heartbeat"
                };
                readerThread.Start();
                heartbeatThread.Start();
                LogInfo("VLTraderTCPClient: AddOn Active, connecting to "
                        + GO_SERVER_HOST + ":" + GO_SERVER_PORT);
            }
            else if (State == State.Terminated)
            {
                try { cts?.Cancel(); } catch { }
                // PHASE 3: unsubscribe OrderUpdate from EVERY account we hooked (active + routed)
                lock (orderSubLock)
                {
                    foreach (var a in orderSubscribed) { try { a.OrderUpdate -= OnOrderUpdate; } catch { } }
                    orderSubscribed.Clear();
                }
                if (account != null)
                {
                    try { account.AccountItemUpdate -= OnAccountItemUpdate; } catch { }
                    try { account.PositionUpdate -= OnPositionUpdate; } catch { }
                }
                try { Connection.ConnectionStatusUpdate -= OnVLConnectionStatusUpdate; } catch { }
                try { barsManager?.DisposeAll(); } catch { }
                try { stream?.Close(); } catch { }
                try { client?.Close(); } catch { }
                LogInfo("VLTraderTCPClient: AddOn Terminated");
            }
        }

        // Phase 0 — data-feed reconnect recovery handler. Subscribed to the
        // static Connection.ConnectionStatusUpdate in State.Active and
        // unsubscribed in State.Terminated. NT8 raises this for every
        // connection's status change on a background thread, so we work off a
        // copy and keep the body light. DATA feed = PriceStatus (Status is the
        // order/broker connection). We recreate BarsRequests only on a genuine
        // data-feed recovery — PriceStatus was lost (or Disconnected) and is now
        // Connected — which also covers the ConnectionLost -> Connecting ->
        // Connected path. The existing OnConnectionReconnected() does the
        // dispose + front-month re-resolve + re-subscribe.
        private void OnVLConnectionStatusUpdate(object sender, ConnectionStatusEventArgs e)
        {
            // Multi-threading: NT8 staff guidance — copy the args, they may race
            // ahead of us while we process.
            ConnectionStatusEventArgs eCopy = e;
            try
            {
                string priceStatus = eCopy.PriceStatus.ToString();
                string prevPriceStatus = eCopy.PreviousPriceStatus.ToString();
                bool fireRecreate = false;

                lock (connStatusLock)
                {
                    if (priceStatus == "ConnectionLost" || priceStatus == "Disconnected")
                    {
                        if (!dataFeedWasLost)
                        {
                            dataFeedWasLost = true;
                            LogWarn("VLTraderTCPClient: data feed lost (PriceStatus="
                                    + priceStatus + ") — BarsRequests will be recreated on recovery");
                        }
                    }
                    else if (priceStatus == "Connected")
                    {
                        // Recreate on ANY transition INTO Connected — recovery
                        // (dataFeedWasLost) AND a FRESH STARTUP (PreviousPriceStatus =
                        // Connecting/Disconnected). The old loss-only guard left a clean
                        // startup with no recreate, so .Update sat dead from the restart
                        // until the slow 75-min watchdog. PreviousPriceStatus != Connected
                        // means a real transition, so this fires once per connect (not on
                        // repeated Connected events where the price feed never dropped).
                        bool freshConnect = (prevPriceStatus != "Connected");
                        if (dataFeedWasLost || freshConnect)
                        {
                            dataFeedWasLost = false;  // clear inside the lock -> no double-fire
                            // Flap debounce: skip the (deep, expensive) rebuild if one
                            // fired within RecreateDebounce. The first connect
                            // (lastRecreateUtc=MinValue) always passes; a genuine
                            // resume-after-a-real-gap passes; only rapid flaps coalesce.
                            DateTime nowUtc = DateTime.UtcNow;
                            if (nowUtc - lastRecreateUtc >= RecreateDebounce)
                            {
                                lastRecreateUtc = nowUtc;
                                fireRecreate = true;
                            }
                            else
                            {
                                LogInfo("VLTraderTCPClient: feed Connected but a BarsRequest rebuild "
                                        + "fired <" + (int)RecreateDebounce.TotalSeconds
                                        + "s ago — debounced (feed-flap guard)");
                            }
                        }
                    }
                }

                if (fireRecreate)
                {
                    LogInfo("VLTraderTCPClient: data feed Connected (prev=" + prevPriceStatus
                            + ") — recreating BarsRequests via OnConnectionReconnected");
                    barsManager?.OnConnectionReconnected();
                }

                // TRACK A — forward the price-feed status to Go so the bot can gate
                // opens/closes when the feed is down (the SIM rejects "no market
                // data" — the upstream condition behind the phantom-close mess).
                // Emitted on every connection-status transition.
                SendFeedStatusFrame(priceStatus);
            }
            catch (Exception ex)
            {
                LogWarn("VLTraderTCPClient: OnVLConnectionStatusUpdate failed: " + ex.Message);
            }
        }

        // === Account resolution ===
        // Selection order (first match wins):
        //   1. The account name in %USERPROFILE%\NofxTrader\account.txt
        //      (operator-editable — one line, e.g. "Sim101" or "MyPropAcct").
        //   2. "Sim101" (SIM default).
        //   3. The first available account.
        private void ResolveAccount()
        {
            string preferred = null;
            try
            {
                string path = Path.Combine(
                    Environment.GetEnvironmentVariable("USERPROFILE") ?? "",
                    "NofxTrader", "account.txt");
                if (File.Exists(path))
                {
                    preferred = File.ReadAllText(path).Trim();
                    if (preferred.Length == 0) preferred = null;
                }
            }
            catch (Exception ex)
            {
                LogWarn("VLTraderTCPClient: could not read account.txt: " + ex.Message);
            }

            lock (Account.All)
            {
                if (preferred != null)
                {
                    foreach (var a in Account.All)
                        if (a.Name == preferred) { account = a; break; }
                    if (account == null)
                        LogWarn("VLTraderTCPClient: account.txt requested '" + preferred
                                + "' but no such account — falling back");
                }
                if (account == null)
                    foreach (var a in Account.All)
                        if (a.Name == "Sim101") { account = a; break; }
                if (account == null && Account.All.Count > 0) account = Account.All[0];
            }

            if (account == null)
            {
                LogWarn("VLTraderTCPClient: no account resolved; signals will be rejected");
            }
            else
            {
                LogInfo("VLTraderTCPClient: using account " + account.Name
                        + (preferred != null && account.Name == preferred ? " (from account.txt)" : ""));
            }
        }

        // === SIM detection ===
        // Returns true if the account is a simulation account. Tries Account.Simulation
        // property first (NT8 8.1+), then falls back to name pattern matching (Sim*).
        private bool IsSimAccount(Account a)
        {
            if (a == null) return false;
            try
            {
                // NT8 8.1+ exposes Account.Simulation property
                var simProp = a.GetType().GetProperty("Simulation");
                if (simProp != null && simProp.PropertyType == typeof(bool))
                {
                    return (bool)simProp.GetValue(a, null);
                }
            }
            catch { }
            // Fallback: name-based detection (Sim prefix is NT8 standard for SIM accounts)
            return a.Name != null && a.Name.StartsWith("Sim");
        }

        // === PHASE 3 — dedup'd OrderUpdate (un)subscription ===
        // SubscribeOrderUpdate is idempotent: a no-op if the account is already hooked,
        // so routing to the active account (or re-routing to an already-routed one) never
        // double-hooks OnOrderUpdate (which would duplicate fills). ALL OrderUpdate
        // subscribe/unsubscribe flows through these so `orderSubscribed` stays the single
        // source of truth. (AccountItemUpdate/PositionUpdate stay active-account-only = P4.)
        private void SubscribeOrderUpdate(Account a)
        {
            if (a == null) return;
            bool added = false;
            lock (orderSubLock)
            {
                if (!orderSubscribed.Contains(a))
                {
                    a.OrderUpdate += OnOrderUpdate;
                    orderSubscribed.Add(a);
                    added = true;
                }
            }
            if (added) LogInfo("VLTraderTCPClient: subscribed OrderUpdate on account '" + a.Name + "'");
        }
        private void UnsubscribeOrderUpdate(Account a)
        {
            if (a == null) return;
            lock (orderSubLock)
            {
                if (!orderSubscribed.Contains(a)) return;
                try { a.OrderUpdate -= OnOrderUpdate; } catch { }
                orderSubscribed.Remove(a);
            }
        }

        // === Connection loop with reconnect (spec L4415: every 5s) ===
        private void RunConnectionLoop(CancellationToken ct)
        {
            while (!ct.IsCancellationRequested)
            {
                try
                {
                    client = new TcpClient();
                    client.Connect(GO_SERVER_HOST, GO_SERVER_PORT);
                    stream = client.GetStream();
                    LogInfo("VLTraderTCPClient: CONNECTED");

                    // P5.2 — hello MUST be the FIRST frame so the Go server
                    // can version-check before any data. On a mismatch the
                    // server refuses (closes); we land back in the reconnect
                    // loop with its loud mismatch log explaining why.
                    SendHello();
                    LogInfo("VLTraderTCPClient: About to call SendAccountsList");

                    // Trigger on connect in case accounts are already loaded
                    SendAccountsList();
                    LogInfo("VLTraderTCPClient: SendAccountsList completed");

                    // Plan 4.11 — push the current account balance immediately
                    // on (re)connect so the dashboard shows the real SIM
                    // account without waiting for the next AccountItemUpdate.
                    SendAccountBalance();
                    // Open-position read-back — snapshot the selected account's
                    // current open positions on (re)connect so the Go side
                    // re-syncs the AI's tracked position after a restart/feed
                    // reconnect (companion to the Phase 0 reconnect fix).
                    SendOpenPositions(account);
                    RunReadLoop(ct);
                }
                catch (Exception ex)
                {
                    LogWarn("VLTraderTCPClient: connect failed: " + ex.Message);
                }
                finally
                {
                    try { stream?.Close(); } catch { }
                    try { client?.Close(); } catch { }
                    stream = null;
                    client = null;
                }
                if (!ct.IsCancellationRequested)
                {
                    Thread.Sleep(RECONNECT_INTERVAL_MS);
                }
            }
        }

        // === Read loop: 4-byte BE length prefix + JSON body (spec L4376) ===
        private void RunReadLoop(CancellationToken ct)
        {
            byte[] header = new byte[4];
            while (!ct.IsCancellationRequested && stream != null)
            {
                if (!ReadExact(stream, header, 4)) return;
                uint length =
                    ((uint)header[0] << 24) |
                    ((uint)header[1] << 16) |
                    ((uint)header[2] << 8)  |
                    ((uint)header[3]);
                if (length > MAX_FRAME_BYTES)
                {
                    // Spec L4416: oversized frame is a protocol error; close.
                    LogWarn("VLTraderTCPClient: oversized frame " + length + " bytes — closing");
                    return;
                }
                byte[] body = new byte[length];
                if (!ReadExact(stream, body, (int)length)) return;
                string json = Encoding.UTF8.GetString(body);
                HandleFrame(json);
            }
        }

        private bool ReadExact(NetworkStream s, byte[] buf, int n)
        {
            int got = 0;
            while (got < n)
            {
                int r;
                try
                {
                    r = s.Read(buf, got, n - got);
                }
                catch (Exception ex)
                {
                    LogWarn("VLTraderTCPClient: read err " + ex.Message);
                    return false;
                }
                if (r <= 0) return false;
                got += r;
            }
            return true;
        }

        // === Frame dispatch ===
        private void HandleFrame(string json)
        {
            try
            {
                var root = (Dictionary<string, object>)new JsonParser(json).Parse();
                string type = GetString(root, "type");
                Dictionary<string, object> payload = null;
                if (root.ContainsKey("payload"))
                {
                    payload = root["payload"] as Dictionary<string, object>;
                }
                if (type == "signal")
                {
                    HandleSignal(payload);
                }
                else if (type == "close_position")
                {
                    HandleClosePosition(payload);
                }
                else if (type == "move_stop")
                {
                    HandleMoveStop(payload);
                }
                else if (type == "account_select")
                {
                    HandleAccountSelect(payload);
                }
                else if (type == "heartbeat")
                {
                    // Ack incoming heartbeats per spec L4410.
                    SendAck("heartbeat");
                }
                else if (type == "ack")
                {
                    // Server-side ack of our heartbeat/fill — informational only.
                }
                else if (type == "bars_subscribe")
                {
                    // Plan 4.4 Stage 1 — forward to the bars manager.
                    barsManager?.HandleBarsSubscribe(payload);
                }
                else if (type == "bars_unsubscribe")
                {
                    // Plan 4.4 Stage 1 — forward to the bars manager.
                    barsManager?.HandleBarsUnsubscribe(payload);
                }
                else if (type == "hello")
                {
                    // P5.2 — the Go server's hello reply. A version mismatch is
                    // logged LOUDLY (the server side refuses mismatched clients;
                    // this is the symmetric check so the operator sees it from
                    // the NT8 Output window too).
                    int goVersion = 0;
                    try { goVersion = GetInt(payload, "protocol_version"); } catch { }
                    if (goVersion != PROTOCOL_VERSION)
                    {
                        LogWarn("VLTraderTCPClient: *** PROTOCOL VERSION MISMATCH *** Go server v"
                                + goVersion + " vs AddOn v" + PROTOCOL_VERSION
                                + " — redeploy the C# AddOn + Go binary in LOCKSTEP (cp → F5 → NT8 restart)");
                    }
                    else
                    {
                        LogInfo("VLTraderTCPClient: hello handshake OK (protocol v" + goVersion + ")");
                    }
                }
                else
                {
                    LogWarn("VLTraderTCPClient: unknown frame type " + type);
                }
            }
            catch (Exception ex)
            {
                LogWarn("VLTraderTCPClient: malformed frame: " + ex.Message);
            }
        }

        // === Signal handling: OCO bracket placement (spec L4387-4396) ===
        private void HandleSignal(Dictionary<string, object> p)
        {
            if (p == null)
            {
                LogWarn("VLTraderTCPClient: signal payload missing or not an object");
                return;
            }
            string symbol;
            string side;
            int qty;
            double entry;
            double sl;
            double tp;
            string signalId;
            string ts;
            try
            {
                symbol   = GetString(p, "symbol");
                side     = GetString(p, "side");                 // "long" | "short"
                qty      = GetInt(p, "quantity");
                entry    = GetDouble(p, "entry");
                sl       = GetDouble(p, "stop_loss");
                tp       = GetDouble(p, "take_profit");
                signalId = GetString(p, "signal_id");
                ts       = GetString(p, "timestamp");
            }
            catch (Exception ex)
            {
                LogWarn("VLTraderTCPClient: signal payload missing field: " + ex.Message);
                return;
            }

            // PHASE 2 (parse only, back-compat) — optional target account. GetString
            // returns null when the field is absent (TryGetValue, not a throw), so
            // today's Go frames (which omit it) leave this null → it falls through to
            // the active account below, byte-identical to current behavior. This is NOT
            // a new required field; the un-changed Go side keeps working. The actual
            // per-order ROUTING that submits to this account is Phase 3.
            string targetAccount = GetString(p, "account");

            // Spec L4414: stale signal rejection at 60s.
            if (DateTime.TryParse(ts, null,
                System.Globalization.DateTimeStyles.RoundtripKind, out var sigTime))
            {
                var ageSec = (DateTime.UtcNow - sigTime.ToUniversalTime()).TotalSeconds;
                if (ageSec > STALE_SIGNAL_AGE_SECONDS)
                {
                    LogWarn("VLTraderTCPClient: stale signal " + signalId
                            + " (age " + ageSec.ToString("F1") + "s) — rejecting");
                    SendFillFrame(signalId, 0.0, side, qty, 0.0, "rejected", symbol: symbol);
                    return;
                }
            }

            if (account == null)
            {
                LogWarn("VLTraderTCPClient: no account — rejecting signal " + signalId);
                SendFillFrame(signalId, 0.0, side, qty, 0.0, "rejected", symbol: symbol);
                return;
            }

            // SAFETY RAIL (Stage-2 Phase-1, defense-in-depth) — the LAST line before an
            // order reaches NT8. NEVER submit on a non-SIM (live/funded) account, and
            // never on a disconnected one. Do NOT fall back to another account — reject.
            // The Go side + account_select already guard; this is the C# half of the
            // double-guard, so a live order is structurally impossible even if an
            // upstream check is bypassed.
            if (!IsSimAccount(account))
            {
                LogWarn("VLTraderTCPClient: REFUSING order " + signalId + " — account '"
                        + account.Name + "' is NOT a SIM account; live/funded accounts are never auto-traded");
                SendFillFrame(signalId, 0.0, side, qty, 0.0, "rejected", symbol: symbol);
                return;
            }
            if (account.Connection == null || account.Connection.Status != ConnectionStatus.Connected)
            {
                LogWarn("VLTraderTCPClient: REFUSING order " + signalId + " — account '"
                        + account.Name + "' is not Connected");
                SendFillFrame(signalId, 0.0, side, qty, 0.0, "rejected", symbol: symbol);
                return;
            }

            // PHASE 2 (resolve + guard, NO routing yet) — if the frame named a target
            // account, resolve it against Account.All (same pattern as account_select)
            // and run the SAME double-guard on it (SIM + connected). This PROVES the
            // parse → resolve → guard path end-to-end and LOGS the resolution. It does
            // NOT change which account submits: the active `account` (already guarded
            // above) still places the order — Phase 3 adds the per-order routing that
            // actually submits to the resolved account. A named-but-bad target (unknown
            // / non-SIM / disconnected) is REJECTED here (reuse the P1 reject); we never
            // silently fall back to the active account when a specific one was requested.
            // An ABSENT/empty field skips this block entirely = today's behavior.
            Account resolved = null;   // PHASE 3: the routed submit target (null = active-account fallback)
            if (!string.IsNullOrEmpty(targetAccount))
            {
                lock (Account.All)
                {
                    foreach (var a in Account.All)
                        if (a.Name == targetAccount) { resolved = a; break; }
                }
                if (resolved == null)
                {
                    LogWarn("VLTraderTCPClient: REFUSING order " + signalId + " — target account '"
                            + targetAccount + "' not found in Account.All");
                    SendFillFrame(signalId, 0.0, side, qty, 0.0, "rejected", symbol: symbol);
                    return;
                }
                if (!IsSimAccount(resolved))
                {
                    LogWarn("VLTraderTCPClient: REFUSING order " + signalId + " — target account '"
                            + targetAccount + "' is NOT a SIM account; live/funded accounts are never auto-traded");
                    SendFillFrame(signalId, 0.0, side, qty, 0.0, "rejected", symbol: symbol);
                    return;
                }
                if (resolved.Connection == null || resolved.Connection.Status != ConnectionStatus.Connected)
                {
                    LogWarn("VLTraderTCPClient: REFUSING order " + signalId + " — target account '"
                            + targetAccount + "' is not Connected");
                    SendFillFrame(signalId, 0.0, side, qty, 0.0, "rejected", symbol: symbol);
                    return;
                }
                // Resolved + guarded — PHASE 3 ROUTES the submit to THIS account (below),
                // not the active one. A named-but-bad target was already rejected above.
                LogInfo("VLTraderTCPClient: signal " + signalId + " routed to account '" + targetAccount
                        + "' (resolved, sim, connected)");
            }

            // Resolve the bare root ("MNQ") to the NT8 front-month
            // contract ("MNQ 06-26") via VLContractResolver (date-derived,
            // auto-rolls). NT8's Instrument.GetInstrument requires the
            // qualified contract; a bare root returns null. Canonical
            // symbol on the wire stays "MNQ"; contract form only exists
            // inside this GetInstrument call.
            string contract = VLContractResolver.ResolveFrontMonthContract(symbol);
            LogInfo("VLTraderTCPClient: resolved " + symbol + " -> " + contract);
            var instrument = Instrument.GetInstrument(contract);
            if (instrument == null)
            {
                LogWarn("VLTraderTCPClient: instrument " + symbol
                        + " (resolved to " + contract + ") not found — rejecting");
                SendFillFrame(signalId, 0.0, side, qty, 0.0, "rejected", symbol: symbol);
                return;
            }

            // PHASE 3 (per-order ROUTING) — submit to the RESOLVED account when the frame
            // named one (already guarded above), else the active account (back-compat —
            // every order today, since Go sends no account field until Phase 5). FINAL
            // guard on the EXACT submit target (defense-in-depth): re-assert SIM + connected
            // on the account that will receive CreateOrder, no matter how it was chosen —
            // the LIVE account can never be a submit target. (The Go-side allow-list is
            // enforced at placeEntry; there is no config channel into the AddOn.)
            Account submitAccount = resolved ?? account;
            if (!IsSimAccount(submitAccount)
                || submitAccount.Connection == null
                || submitAccount.Connection.Status != ConnectionStatus.Connected)
            {
                LogWarn("VLTraderTCPClient: REFUSING order " + signalId + " — submit target '"
                        + (submitAccount != null ? submitAccount.Name : "<null>")
                        + "' failed the final SIM/connected guard");
                SendFillFrame(signalId, 0.0, side, qty, 0.0, "rejected", symbol: symbol);
                return;
            }
            // The fill→bracket→fill-frame lifecycle (OnOrderUpdate) must fire for this
            // account. Idempotent — a no-op when it is the already-subscribed active
            // account. (Per-account balance/position snapshots + manual-close Flatten
            // routing remain Phase 4.)
            SubscribeOrderUpdate(submitAccount);

            // Track entry + tick size for slippage computation in fill emission.
            double tickSize = 0.25; // sensible NQ default; overridden if instrument exposes it
            try { tickSize = instrument.MasterInstrument.TickSize; } catch { }
            lock (signalMapLock)
            {
                signalEntryByOco[signalId] = entry;
                signalTickSizeByOco[signalId] = tickSize;
            }

            // Submit the ENTRY only; the protective SL/TP are placed once the
            // entry fills (OnOrderUpdate → SubmitBracketOnEntryFill). Submitting
            // all three in one OCO group cancels the SL/TP the instant the
            // Market entry fills (OCO = one-cancels-other) — the position then
            // has no working stop/target in NT8 and never auto-closes. The
            // entry stands alone (empty OCO); SL+TP get their own OCO pair.
            try
            {
                var entryAction = side == "long" ? OrderAction.Buy : OrderAction.SellShort;
                var exitAction  = side == "long" ? OrderAction.Sell : OrderAction.BuyToCover;

                var entryOrder = submitAccount.CreateOrder(
                    instrument, entryAction, OrderType.Market, OrderEntry.Manual,
                    TimeInForce.Day, qty, 0, 0, string.Empty, signalId,
                    Core.Globals.MaxDate, null);

                lock (signalMapLock)
                {
                    pendingBrackets[signalId] = new PendingBracket
                    {
                        Instrument = instrument,
                        ExitAction = exitAction,
                        Qty        = qty,
                        Sl         = sl,
                        Tp         = tp,
                        Account    = submitAccount,   // PHASE 3: the bracket follows the entry's account
                    };
                }

                submitAccount.Submit(new[] { entryOrder });
                LogInfo("VLTraderTCPClient: submitted entry signal_id=" + signalId
                        + " on account=" + submitAccount.Name
                        + " " + side + " " + qty + " " + symbol
                        + " entry≈" + entry + " (SL=" + sl + " TP=" + tp + " on fill)");
            }
            catch (Exception ex)
            {
                LogWarn("VLTraderTCPClient: order submit failed: " + ex.Message);
                lock (signalMapLock) { pendingBrackets.Remove(signalId); }
                SendFillFrame(signalId, 0.0, side, qty, 0.0, "rejected", symbol: symbol);
            }
        }

        // === Manual close: flatten the position + cancel its working orders ===
        // account.Flatten closes the position at market AND cancels the bracket's
        // SL/TP, so no orphaned exit orders are left to re-open a position. The
        // flatten's market fill arrives in OnOrderUpdate as an exit (Sell /
        // BuyToCover) and is reported as a position_close with reason "manual".
        private void HandleClosePosition(Dictionary<string, object> p)
        {
            if (p == null) { LogWarn("VLTraderTCPClient: close_position empty payload"); return; }
            if (account == null) { LogWarn("VLTraderTCPClient: close_position no account"); return; }
            try
            {
                string symbol = GetString(p, "symbol");
                string contract = VLContractResolver.ResolveFrontMonthContract(symbol);
                var instrument = Instrument.GetInstrument(contract);
                if (instrument == null)
                {
                    LogWarn("VLTraderTCPClient: close_position instrument not found " + symbol);
                    return;
                }
                // PHASE 4 — resolve the account that actually HOLDS this symbol's open
                // position (recorded on entry-fill), and flatten THAT account, not the
                // active one. Back-compat: one trader → the map misses (or returns Sim101)
                // → byte-identical today. GUARD: never flatten a non-SIM (LIVE) account —
                // the resolved target must pass IsSimAccount or the close is refused.
                Account closeTarget = null;
                // Look up by the SAME normalized root the entry-fill populate used
                // (MasterInstrument.Name), not the raw wire symbol — so a non-canonical
                // close symbol can't miss the map and silently fall back to the active
                // account. Defaults to the raw symbol if MasterInstrument is unavailable.
                string posKey = symbol;
                try { posKey = instrument.MasterInstrument.Name; } catch { }
                lock (posAcctLock) { positionAccountBySymbol.TryGetValue(posKey, out closeTarget); }
                if (closeTarget == null) closeTarget = account;
                if (closeTarget == null || !IsSimAccount(closeTarget))
                {
                    LogWarn("VLTraderTCPClient: REFUSING close " + symbol + " — resolved account '"
                            + (closeTarget != null ? closeTarget.Name : "<null>")
                            + "' is not a SIM account; position left OPEN");
                    return;
                }
                closeTarget.Flatten(new[] { instrument });
                // Flatten is async fire-and-forget: the actual fill (or a REJECT, e.g. "no
                // market data" with the feed down) arrives later in OnOrderUpdate. This log
                // means SUBMITTED, not closed — the close is confirmed only by the
                // position_close fill frame (or reported failed by position_close_rejected).
                LogInfo("VLTraderTCPClient: flatten submitted " + symbol + " (" + contract
                        + ") on account=" + closeTarget.Name + " — awaiting NT8 fill/reject (not yet closed)");
            }
            catch (Exception ex)
            {
                LogWarn("VLTraderTCPClient: close_position failed: " + ex.Message);
            }
        }

        // === Account switching (Go-server → C#-AddOn command) ===
        // Unsubscribe from the old account's events, resolve the new account by name,
        // subscribe to its events, clear pending state tied to the old account,
        // and re-emit account_balance for the new account.
        private void HandleAccountSelect(Dictionary<string, object> p)
        {
            if (p == null) { LogWarn("VLTraderTCPClient: account_select empty payload"); return; }
            try
            {
                string requestedAccount = GetString(p, "account");
                if (string.IsNullOrEmpty(requestedAccount))
                {
                    LogWarn("VLTraderTCPClient: account_select missing account name");
                    return;
                }

                // Resolve the requested account from Account.All
                Account newAccount = null;
                lock (Account.All)
                {
                    foreach (var a in Account.All)
                        if (a.Name == requestedAccount) { newAccount = a; break; }
                }

                if (newAccount == null)
                {
                    LogWarn("VLTraderTCPClient: account_select account not found: " + requestedAccount);
                    return;
                }

                // If already on that account, no-op
                if (account == newAccount)
                {
                    LogInfo("VLTraderTCPClient: account_select already on " + requestedAccount);
                    return;
                }

                // Unsubscribe from the old account's events
                if (account != null)
                {
                    UnsubscribeOrderUpdate(account);   // PHASE 3: dedup'd
                    try { account.AccountItemUpdate -= OnAccountItemUpdate; } catch { }
                    try { account.PositionUpdate -= OnPositionUpdate; } catch { }
                }

                // Switch to the new account and subscribe to its events
                account = newAccount;
                SubscribeOrderUpdate(account);   // PHASE 3: dedup'd
                account.AccountItemUpdate += OnAccountItemUpdate;
                account.PositionUpdate += OnPositionUpdate;

                // Clear pending brackets and signal maps (tied to old account context)
                lock (signalMapLock)
                {
                    signalEntryByOco.Clear();
                    signalTickSizeByOco.Clear();
                    pendingBrackets.Clear();
                }

                // Re-emit account_balance immediately for the new account
                SendAccountBalance();
                // Open-position read-back — snapshot the NEWLY-SELECTED account's
                // current open positions so the AI re-tracks the real position on
                // switch-back (fixes the orphaned-position bug). NT8 reports
                // positions on change only, so this on-select re-report is what
                // repopulates the Go-side per-account cache.
                SendOpenPositions(account);
                LogInfo("VLTraderTCPClient: switched to account " + account.Name);
            }
            catch (Exception ex)
            {
                LogWarn("VLTraderTCPClient: account_select failed: " + ex.Message);
            }
        }

        // === Fill subscription (spec L4398-4406) ===
        private void OnOrderUpdate(object sender, OrderEventArgs e)
        {
            // Only act on filled/rejected states; ignore accepted/working.
            if (e.OrderState != OrderState.Filled
                && e.OrderState != OrderState.Rejected
                && e.OrderState != OrderState.PartFilled)
            {
                return;
            }
            if (e.Order == null) return;

            // Classify by ORDER ACTION, not name: Buy/SellShort = entry,
            // Sell/BuyToCover = exit (an SL/TP leg OR a manual flatten — a
            // flatten order may have no -sl/-tp name, so name alone is not
            // enough). The order Name still carries the -sl/-tp suffix when
            // present → the exit reason; otherwise the exit is a "manual" close.
            string orderName = e.Order.Name ?? "";
            string ocoId     = e.Order.Oco ?? "";
            string signalId  = orderName.Length > 0 ? orderName : ocoId;
            string exitReason = null;
            if (signalId.EndsWith("-sl")) { exitReason = "sl"; signalId = signalId.Substring(0, signalId.Length - 3); }
            else if (signalId.EndsWith("-tp")) { exitReason = "tp"; signalId = signalId.Substring(0, signalId.Length - 3); }

            var action = e.Order.OrderAction;
            bool isExit = (action == OrderAction.Sell || action == OrderAction.BuyToCover);

            // Exit fill (SL, TP, or manual flatten) → the position closed. Emit
            // position_close with the real exit price (reason from the leg name,
            // else "manual"). Only on a full Filled — PartFilled is transient (NT
            // sends a final Filled; per-partial would duplicate the close) and a
            // rejection is an error. Held side is opposite the exit action.
            if (isExit)
            {
                if (e.OrderState == OrderState.Filled)
                {
                    string positionSide = (action == OrderAction.BuyToCover) ? "short" : "long";
                    string rootSymbol = "";
                    try { rootSymbol = e.Order.Instrument.MasterInstrument.Name; } catch { }
                    string exitAcct = e.Order.Account != null ? e.Order.Account.Name
                                      : (account != null ? account.Name : "");
                    SendPositionCloseFrame(signalId, rootSymbol, positionSide,
                                           e.AverageFillPrice, e.Filled, exitReason ?? "manual", exitAcct);
                    // PHASE 4: the position closed → drop its account ownership.
                    if (!string.IsNullOrEmpty(rootSymbol))
                        lock (posAcctLock) { positionAccountBySymbol.Remove(rootSymbol); }
                    // Position closed → drop its bracket tracking (auto-breakeven).
                    lock (signalMapLock) { placedBrackets.Remove(signalId); }
                }
                else if (e.OrderState == OrderState.Rejected)
                {
                    // The SIM/broker REJECTED the exit/flatten (e.g. "There is no
                    // market data available to drive the simulation engine" while the
                    // data feed is down). The close did NOT take — the position is
                    // STILL OPEN in NT8. Tell Go so it never records a phantom close
                    // and raises an alarm (the id=45→id=46 net-2 root cause).
                    string positionSide = (action == OrderAction.BuyToCover) ? "short" : "long";
                    string rootSymbol = "";
                    try { rootSymbol = e.Order.Instrument.MasterInstrument.Name; } catch { }
                    string reason = "";
                    try { reason = e.Comment ?? ""; } catch { }
                    if (string.IsNullOrEmpty(reason)) { try { reason = e.Error.ToString(); } catch { } }
                    string rejAcct = e.Order.Account != null ? e.Order.Account.Name
                                     : (account != null ? account.Name : "");
                    SendPositionCloseRejectedFrame(signalId, rootSymbol, positionSide, reason, rejAcct);
                    LogWarn("VLTraderTCPClient: exit/flatten REJECTED signal_id=" + signalId
                            + " " + positionSide + " reason=" + reason
                            + " — position STILL OPEN (not closed)");
                }
                return;
            }

            // Entry leg (Buy = long, SellShort = short) → fill-frame path.
            string status;
            switch (e.OrderState)
            {
                case OrderState.Filled:     status = "filled";   break;
                case OrderState.Rejected:   status = "rejected"; break;
                case OrderState.PartFilled: status = "partial";  break;
                default:                    return;
            }

            // Entry just filled → place the protective SL/TP now that the
            // position exists (deferred from HandleSignal to dodge the OCO
            // cancel-on-entry-fill bug). On rejection, drop the pending bracket.
            if (e.OrderState == OrderState.Filled)
            {
                SubmitBracketOnEntryFill(signalId);
                // PHASE 4: record which account now holds this symbol's open position so a
                // later close flattens the RIGHT account (not the active one). Sourced from
                // the order's own account — account-agnostic, correct per routed account.
                try
                {
                    string entrySym = e.Order.Instrument.MasterInstrument.Name;
                    if (e.Order.Account != null && !string.IsNullOrEmpty(entrySym))
                        lock (posAcctLock) { positionAccountBySymbol[entrySym] = e.Order.Account; }
                }
                catch (Exception ex)
                {
                    LogWarn("VLTraderTCPClient: PHASE 4 — failed to record position account for "
                            + signalId + ": " + ex.Message + " (a later close falls back to the active account)");
                }
            }
            else if (e.OrderState == OrderState.Rejected)
            {
                lock (signalMapLock) { pendingBrackets.Remove(signalId); }
            }

            string sideStr = (action == OrderAction.Buy) ? "long" : "short";

            double slippageTicks = 0.0;
            lock (signalMapLock)
            {
                if (signalEntryByOco.TryGetValue(signalId, out var origEntry)
                    && signalTickSizeByOco.TryGetValue(signalId, out var tick)
                    && tick > 0)
                {
                    slippageTicks = (e.AverageFillPrice - origEntry) / tick;
                }
            }

            // P5.2 — tag the fill with the order's instrument root for
            // multi-symbol attribution (same accessor PHASE 4 uses above).
            string fillSymbol = "";
            try { fillSymbol = e.Order.Instrument.MasterInstrument.Name; } catch { }

            SendFillFrame(signalId, e.AverageFillPrice, sideStr, e.Filled,
                          slippageTicks, status,
                          e.Order.Account != null ? e.Order.Account.Name
                              : (account != null ? account.Name : ""),
                          fillSymbol);
        }

        // === Write helpers ===

        /// <summary>
        /// P5.2 handshake — sent as the FIRST frame on every (re)connect so the
        /// Go server can version-check before any data flows. The server replies
        /// with its own hello (handled in HandleFrame) or refuses on mismatch.
        /// </summary>
        private void SendHello()
        {
            WriteEnvelope("hello", new Dictionary<string, object>
            {
                ["protocol_version"] = PROTOCOL_VERSION,
                ["source"]           = "vltrader-addon"
            });
        }

        private void SendFillFrame(string signalId, double fillPrice, string side,
                                   int qty, double slippageTicks, string status,
                                   string acctName = "", string symbol = "")
        {
            var payload = new Dictionary<string, object>
            {
                ["signal_id"]      = signalId,
                ["fill_price"]     = fillPrice,
                ["fill_time"]      = DateTime.UtcNow.ToString("o"),
                ["side"]           = side,
                ["quantity"]       = qty,
                ["slippage_ticks"] = slippageTicks,
                ["status"]         = status,
                // PHASE 4: which account this fill is on (e.Order.Account). Additive +
                // back-compat: an un-updated Go binary ignores unknown JSON fields.
                ["account"]        = acctName ?? "",
                // P5.2: which INSTRUMENT this fill is for (multi-symbol
                // attribution). Empty = unknown; the Go side treats empty as the
                // primary trading symbol (legacy-compatible).
                ["symbol"]         = symbol ?? ""
            };
            WriteEnvelope("fill", payload);
        }

        // Place the protective SL + TP once the entry has filled. They share
        // their OWN OCO group (signal_id-exit) so one cancels the other when
        // triggered, and — crucially — they are NOT in the entry's OCO group,
        // so the entry fill no longer cancels them. Names keep the -sl/-tp
        // suffix so OnOrderUpdate routes their fills to position_close.
        // Auto-breakeven: move a live bracket's resting stop to a new price WITHOUT
        // closing the position. The new StopMarket is submitted in the SAME OCO group
        // FIRST, then the old one is cancelled — so the position is NEVER momentarily
        // unprotected (and if anything throws, the original stop survives). No-op if
        // the bracket is gone (already exited) or the stop is already at/better than
        // the requested price (restart idempotency). Acks back to Go.
        private void HandleMoveStop(Dictionary<string, object> p)
        {
            try
            {
                if (p == null) { LogWarn("VLTraderTCPClient: move_stop empty payload"); return; }
                string signalId = GetString(p, "signal_id");
                double newStop  = GetDouble(p, "new_stop_loss");
                if (string.IsNullOrEmpty(signalId) || newStop <= 0)
                { LogWarn("VLTraderTCPClient: move_stop bad payload (signal_id/new_stop_loss)"); return; }

                PlacedBracket pb;
                lock (signalMapLock)
                {
                    if (!placedBrackets.TryGetValue(signalId, out pb))
                    { LogWarn("VLTraderTCPClient: move_stop no live bracket for " + signalId + " (already exited?)"); return; }
                }
                if (pb.SlOrder != null && Math.Abs(pb.SlOrder.StopPrice - newStop) < (pb.TickSize / 2.0))
                { LogInfo("VLTraderTCPClient: move_stop no-op (stop already ~" + newStop + ")"); return; }

                Account ba = pb.Account ?? account;

                // GUARD: only a resting stop (Working/Accepted) can be modified in place.
                // A Filled/Cancelled/pending stop means the position is already exiting or
                // the order is gone — do NOT touch it; error-ACK so Go knows the move did
                // not happen (it re-arms and retries on the next cycle).
                OrderState st = pb.SlOrder != null ? pb.SlOrder.OrderState : OrderState.Unknown;
                if (pb.SlOrder == null || (st != OrderState.Working && st != OrderState.Accepted))
                {
                    LogWarn("VLTraderTCPClient: move_stop refused — stop not changeable (state=" + st + ") for " + signalId);
                    SendAck("move_stop_error");
                    return;
                }

                // IN-PLACE modification (fix 2026-08-07): move the SAME resting stop's price
                // via Account.Change — same Order object, same OCO group, NO new order and
                // NO cancel. The target and the OCO group are NEVER disturbed.
                //
                // The previous code created a NEW stop INTO THE SAME OCO group (pb.ExitOco)
                // and then cancelled the old stop. NT8 OCO cancels the WHOLE group when any
                // member is cancelled, so the target AND the freshly-submitted stop both
                // died → NAKED position. Proven live 2026-08-07 11:25:05 (signal b846e082…:
                // -tp Cancelled + both -sl orders Cancelled within ~215ms → naked).
                try
                {
                    pb.SlOrder.StopPriceChanged = newStop;
                    ba.Change(new[] { pb.SlOrder });
                }
                catch (Exception cx)
                {
                    LogWarn("VLTraderTCPClient: move_stop Change failed: " + cx.Message + " for " + signalId);
                    SendAck("move_stop_error");
                    return;
                }
                LogInfo("VLTraderTCPClient: move_stop → " + newStop + " for signal_id=" + signalId + " (auto-breakeven, in-place Change — target + OCO preserved)");
                SendAck("move_stop");
            }
            catch (Exception ex)
            {
                LogWarn("VLTraderTCPClient: move_stop failed: " + ex.Message);
            }
        }

        private void SubmitBracketOnEntryFill(string signalId)
        {
            PendingBracket b;
            lock (signalMapLock)
            {
                if (!pendingBrackets.TryGetValue(signalId, out b)) return;
                pendingBrackets.Remove(signalId); // idempotent: only place once
            }
            try
            {
                // PHASE 3: the protective bracket is placed on the SAME account the entry
                // routed to (stored in PendingBracket), not necessarily the active one.
                Account ba = b.Account ?? account;
                string exitOco = signalId + "-exit";
                var slOrder = ba.CreateOrder(
                    b.Instrument, b.ExitAction, OrderType.StopMarket, OrderEntry.Manual,
                    TimeInForce.Day, b.Qty, 0, b.Sl, exitOco, signalId + "-sl",
                    Core.Globals.MaxDate, null);
                var tpOrder = ba.CreateOrder(
                    b.Instrument, b.ExitAction, OrderType.Limit, OrderEntry.Manual,
                    TimeInForce.Day, b.Qty, b.Tp, 0, exitOco, signalId + "-tp",
                    Core.Globals.MaxDate, null);
                ba.Submit(new[] { slOrder, tpOrder });
                // Track the live SL order so auto-breakeven can move it later.
                double tick = 0.25;
                try { tick = b.Instrument.MasterInstrument.TickSize; } catch { }
                lock (signalMapLock)
                {
                    placedBrackets[signalId] = new PlacedBracket
                    {
                        SlOrder = slOrder, Account = ba, Instrument = b.Instrument,
                        ExitAction = b.ExitAction, Qty = b.Qty, ExitOco = exitOco, TickSize = tick,
                    };
                }
                LogInfo("VLTraderTCPClient: placed protective bracket signal_id=" + signalId
                        + " sl=" + b.Sl + " tp=" + b.Tp);
            }
            catch (Exception ex)
            {
                LogWarn("VLTraderTCPClient: bracket submit failed signal_id=" + signalId
                        + ": " + ex.Message);
            }
        }

        // Position-history fix — emit a position_close frame when an OCO exit
        // leg (SL/TP) fills. position_side is the side that was HELD. The Go
        // side computes realized PnL against the recorded entry × the futures
        // point value, so we send only the raw exit price here.
        private void SendPositionCloseFrame(string signalId, string symbol,
                                            string positionSide, double exitPrice,
                                            int qty, string exitReason,
                                            string acctName = "")
        {
            var payload = new Dictionary<string, object>
            {
                ["signal_id"]     = signalId,
                ["symbol"]        = symbol ?? "",
                ["position_side"] = positionSide,
                ["exit_price"]    = exitPrice,
                ["quantity"]      = qty,
                ["exit_reason"]   = exitReason ?? "",
                // PHASE 4: the account this close is on (was absent; the Go struct already
                // has the field). e.Order.Account from the caller; "" → Go keeps today's
                // in-place-row behavior (the entry row already holds the account).
                ["account"]       = acctName ?? "",
                ["exit_time"]     = DateTime.UtcNow.ToString("o")
            };
            WriteEnvelope("position_close", payload);
        }

        // Emit when an exit/flatten order is REJECTED (e.g. SIM "no market data"
        // while the feed is down). Tells Go the close did NOT take — the position
        // is STILL OPEN in NT8 — so the bot must not record it closed. Mirrors
        // SendPositionCloseFrame; reason carries the NT8 reject comment.
        private void SendPositionCloseRejectedFrame(string signalId, string symbol,
                                                    string positionSide, string reason,
                                                    string acctName = null)
        {
            var payload = new Dictionary<string, object>
            {
                ["signal_id"]     = signalId ?? "",
                ["symbol"]        = symbol ?? "",
                ["position_side"] = positionSide ?? "",
                ["reason"]        = reason ?? "",
                // PHASE 4 fix: stamp the account the rejected EXIT was on (e.Order.Account
                // from the caller), NOT the active field — the active-account stamp was a
                // latent bug once routing is live. Falls back to the active account for
                // back-compat if the caller passes none.
                ["account"]       = !string.IsNullOrEmpty(acctName) ? acctName : (account != null ? account.Name : ""),
                ["reject_time"]   = DateTime.UtcNow.ToString("o")
            };
            WriteEnvelope("position_close_rejected", payload);
        }

        // TRACK A — emit the NT8 price-feed status so Go can gate opens/closes when
        // the feed is down (the SIM rejects orders with "no market data"). Sent on
        // every connection-status transition (OnVLConnectionStatusUpdate).
        private void SendFeedStatusFrame(string priceStatus)
        {
            var payload = new Dictionary<string, object>
            {
                ["price_status"] = priceStatus ?? "",
                ["time"]         = DateTime.UtcNow.ToString("o")
            };
            WriteEnvelope("feed_status", payload);
        }

        // Open-position read-back — emit the selected account's CURRENT open
        // positions as a FULL snapshot. Called on connect, on account_select,
        // and on any Account.PositionUpdate (incl. MANUAL trades in NT8). NT8 is
        // the source of truth; before this the Go side only knew positions it
        // opened itself (via fills), so it lost the position on account
        // switch-back and never saw manual trades. The Go side REPLACES its
        // per-account cache with this set (an empty list = the account is flat).
        // Full snapshot from the (settled) live collection — used by the connect
        // + account_select paths, where positions are already settled. Delegates
        // to the evt-aware overload with no event override.
        private void SendOpenPositions(Account acc)
        {
            SendOpenPositions(acc, null);
        }

        // Emit the account's open positions. When called from OnPositionUpdate
        // (evt != null), the just-updated instrument's AveragePrice in the live
        // acc.Positions collection has NOT settled to the new fill yet — it still
        // holds the PRIOR netted average (the stale-entry bug). The
        // PositionEventArgs carries the SETTLED account-level values at the event
        // (NT staff + help guide: e.MarketPosition/e.Quantity/e.AveragePrice are
        // "the actual account value at that time", unlike e.Position.* which is
        // only the currently-updating position), so for THAT instrument we use
        // evt's by-value fields. Other instruments + the connect/select snapshot
        // (evt == null) read the settled collection directly.
        private void SendOpenPositions(Account acc, PositionEventArgs evt)
        {
            if (acc == null) return;
            try
            {
                // Copy the event values by-value up front (NT8 background-thread
                // guidance — the args can race ahead while we process). qty==0 or
                // Flat means the just-updated instrument went flat (close/remove).
                string evtRoot = null;
                MarketPosition evtMp = MarketPosition.Flat;
                int evtQty = 0;
                double evtAvg = 0;
                if (evt != null && evt.Position != null)
                {
                    try { evtRoot = evt.Position.Instrument.MasterInstrument.Name; } catch { }
                    evtMp = evt.MarketPosition;
                    evtQty = evt.Quantity;
                    evtAvg = evt.AveragePrice;
                }

                var list = new List<object>();
                bool evtSeen = false;
                // Background-thread safe: lock the Positions collection while
                // enumerating (NT8 mutates it from worker threads).
                lock (acc.Positions)
                {
                    foreach (Position pos in acc.Positions)
                    {
                        if (pos == null) continue;
                        string root = "";
                        try { root = pos.Instrument.MasterInstrument.Name; } catch { }

                        MarketPosition mp = pos.MarketPosition;
                        int qty = pos.Quantity;
                        double avg = pos.AveragePrice;
                        // Override the just-updated instrument with the SETTLED
                        // event values (its collection avg is still stale here).
                        if (evtRoot != null && root == evtRoot)
                        {
                            mp = evtMp; qty = evtQty; avg = evtAvg; evtSeen = true;
                        }
                        if (mp == MarketPosition.Flat || qty == 0)
                            continue; // skip flat / closed entries
                        list.Add(new Dictionary<string, object>
                        {
                            ["symbol"]    = root,
                            ["side"]      = (mp == MarketPosition.Long) ? "long" : "short",
                            ["quantity"]  = qty,
                            ["avg_price"] = avg
                        });
                    }
                }
                // Edge: a brand-new position may not be in acc.Positions at the
                // event instant — add it from the settled event values so we
                // never under-report the just-opened position.
                if (evtRoot != null && !evtSeen && evtMp != MarketPosition.Flat && evtQty != 0)
                {
                    list.Add(new Dictionary<string, object>
                    {
                        ["symbol"]    = evtRoot,
                        ["side"]      = (evtMp == MarketPosition.Long) ? "long" : "short",
                        ["quantity"]  = evtQty,
                        ["avg_price"] = evtAvg
                    });
                }

                var payload = new Dictionary<string, object>
                {
                    ["account"]   = acc.Name,
                    ["positions"] = list
                };
                WriteEnvelope("positions", payload);
                LogInfo("VLTraderTCPClient: sent positions snapshot account=" + acc.Name + " count=" + list.Count);
            }
            catch (Exception ex)
            {
                LogWarn("VLTraderTCPClient: SendOpenPositions failed: " + ex.Message);
            }
        }

        // Account.PositionUpdate handler — fires on ANY position change for the
        // subscribed account, INCLUDING a manual open/close in NT8. Raised on a
        // background thread; we pass the event into SendOpenPositions so the
        // just-updated instrument uses the SETTLED event average (the live
        // acc.Positions average lags the fill at this instant — the stale-entry
        // bug). A close/flat arrives as MarketPosition.Flat or Quantity 0.
        private void OnPositionUpdate(object sender, PositionEventArgs e)
        {
            try { SendOpenPositions(account, e); }
            catch (Exception ex) { LogWarn("VLTraderTCPClient: OnPositionUpdate failed: " + ex.Message); }
        }

        // Emit the accounts_list frame reporting all available NT accounts with
        // SIM detection. Fired on connect so the Go server can populate the UI
        // account selector. Also used on account list change (if NT fires an event).
        // WriteEnvelope no-ops if not yet connected.
        // Handler for when accounts become available
        private void OnAccountStatusUpdate(object sender, AccountStatusEventArgs e)
        {
            LogInfo(string.Format("@@@ OnAccountStatusUpdate FIRED: {0}, status={1}", e.Account.Name, e.Status));
            SendAccountsList();
        }

        private bool IsRealAccount(Account a)
        {
            if (a == null) return false;

            // Skip NT8 internal/test/bracket accounts (auto-generated).
            if (a.Name.StartsWith("FTPROPLUSM") || a.Name.StartsWith("FTPROPLUS") ||
                a.Name.StartsWith("TAKEPROFIT") || a.Name.StartsWith("TDFYSL") ||
                a.Name.StartsWith("TDFYG") || a.Name.StartsWith("MFFU"))
                return false;

            // Per NinjaTrader staff: filter to accounts on a FULLY CONNECTED connection.
            // A connection is "green/fully connected" when BOTH Status and PriceStatus
            // are ConnectionStatus.Connected (one can be up without the other).
            try
            {
                if (a.Connection == null) return false;

                // Check both Status and PriceStatus (per NT staff guidance).
                // ConnectionStatus.Connected is the enum value we're matching.
                var connStatus = a.Connection.Status;
                var priceStatus = a.Connection.PriceStatus;

                // Only include if BOTH are Connected (fully green in NT8 Control Center)
                bool isConnected = (connStatus.ToString() == "Connected") &&
                                  (priceStatus.ToString() == "Connected");

                if (!isConnected) return false;

                // If Status && PriceStatus filter isn't enough and Backtest/Playback
                // still show as green, add a provider/name narrowing (not needed if
                // only real accounts are green).
                return true;
            }
            catch (Exception ex)
            {
                LogWarn(string.Format("@@@ IsRealAccount check failed for {0}: {1}", a.Name, ex.Message));
                return false;
            }
        }

        private void SendAccountsList()
        {
            LogInfo("@@@ SendAccountsList START");
            var accountsList = new List<object>();
            try
            {
                lock (Account.All)
                {
                    LogInfo(string.Format("@@@ Account.All.Count={0}", Account.All.Count));
                    foreach (var a in Account.All)
                    {
                        if (!IsRealAccount(a)) continue;
                        if (accountsList.Count < 5)  // Log first 5
                            LogInfo(string.Format("@@@   account[{0}]: {1}", accountsList.Count, a.Name));
                        accountsList.Add(new Dictionary<string, object>
                        {
                            ["name"]   = a.Name,
                            ["is_sim"] = IsSimAccount(a)
                        });
                    }
                }
            }
            catch (Exception ex)
            {
                LogWarn(string.Format("@@@ lock failed: {0}", ex.Message));
                return;
            }

            LogInfo(string.Format("@@@ built list with {0} accounts, calling WriteEnvelope", accountsList.Count));
            LogInfo(string.Format("@@@ stream status: {0}", stream == null ? "NULL" : "CONNECTED"));
            var payload = new Dictionary<string, object> { ["accounts"] = accountsList };
            try
            {
                WriteEnvelope("accounts_list", payload);
                LogInfo(string.Format("@@@ WriteEnvelope succeeded, sent {0} accounts", accountsList.Count));
            }
            catch (Exception ex)
            {
                LogWarn(string.Format("@@@ WriteEnvelope FAILED: {0}", ex.Message));
            }
        }

        // Plan 4.11 — emit the real NT SIM account balance as an account_balance
        // frame (replaces the Go-side $50k mock). Fired on (re)connect and on
        // AccountItemUpdate. WriteEnvelope no-ops if not yet connected.
        //
        // FLAG: NT8 API — account.Get(AccountItem.X, Currency.UsDollar) and the
        // AccountItem enum (CashValue/BuyingPower/RealizedProfitLoss/
        // UnrealizedProfitLoss) match NT8 8.1. If the operator's build differs,
        // adjust the enum names here (compile error will point right at it).
        // NetLiquidation is derived (cash + unrealized) rather than via a
        // possibly-absent AccountItem.NetLiquidation.
        private void SendAccountBalance()
        {
            if (account == null) return;
            try
            {
                double cash       = account.Get(AccountItem.CashValue, Currency.UsDollar);
                double buying     = account.Get(AccountItem.BuyingPower, Currency.UsDollar);
                double realized   = account.Get(AccountItem.RealizedProfitLoss, Currency.UsDollar);
                double unrealized = account.Get(AccountItem.UnrealizedProfitLoss, Currency.UsDollar);
                var payload = new Dictionary<string, object>
                {
                    ["account"]         = account.Name,
                    ["cash_value"]      = cash,
                    ["buying_power"]    = buying,
                    ["realized_pnl"]    = realized,
                    ["unrealized_pnl"]  = unrealized,
                    ["net_liquidation"] = cash + unrealized
                };
                WriteEnvelope("account_balance", payload);
            }
            catch (Exception ex)
            {
                LogWarn("VLTraderTCPClient: account_balance emit failed: " + ex.Message);
            }
        }

        private void OnAccountItemUpdate(object sender, AccountItemEventArgs e)
        {
            // Emit only on cash/PnL changes to limit frame churn.
            if (e.AccountItem == AccountItem.CashValue
                || e.AccountItem == AccountItem.BuyingPower
                || e.AccountItem == AccountItem.RealizedProfitLoss
                || e.AccountItem == AccountItem.UnrealizedProfitLoss)
            {
                SendAccountBalance();
            }
        }

        private void SendAck(string acks)
        {
            WriteEnvelope("ack", new Dictionary<string, object> { ["acks"] = acks });
        }

        /// <summary>
        /// Plan 4.4 Stage 1 — entry point for VLBarsSubscriptionManager.
        /// Routes its bar frames (bars_historical, bar_update) through the
        /// SAME WriteEnvelope path the signal/fill/heartbeat code uses, so
        /// bar bytes inherit the writeLock and the hand-rolled encoder.
        /// </summary>
        internal void SendFrame(string type, Dictionary<string, object> payload)
        {
            WriteEnvelope(type, payload);
        }

        private void WriteEnvelope(string type, Dictionary<string, object> payload)
        {
            if (stream == null) return;
            try
            {
                string json = EncodeFrame(type, payload);
                byte[] body = Encoding.UTF8.GetBytes(json);
                if (body.Length > MAX_FRAME_BYTES)
                {
                    LogWarn("VLTraderTCPClient: outgoing frame too large " + body.Length);
                    return;
                }
                byte[] header = new byte[]
                {
                    (byte)((body.Length >> 24) & 0xFF),
                    (byte)((body.Length >> 16) & 0xFF),
                    (byte)((body.Length >> 8)  & 0xFF),
                    (byte)( body.Length        & 0xFF)
                };
                lock (writeLock)
                {
                    stream.Write(header, 0, 4);
                    stream.Write(body, 0, body.Length);
                    stream.Flush();
                }
            }
            catch (Exception ex)
            {
                LogWarn("VLTraderTCPClient: write err " + ex.Message);
            }
        }

        // === Heartbeat loop (spec L4408: 30s interval) ===
        private void RunHeartbeatLoop(CancellationToken ct)
        {
            int beatCount = 0;
            while (!ct.IsCancellationRequested)
            {
                Thread.Sleep(HEARTBEAT_INTERVAL_MS);
                if (ct.IsCancellationRequested) return;
                WriteEnvelope("heartbeat", new Dictionary<string, object>());

                // Periodic re-emit of accounts every 3 heartbeats (90s) to ensure
                // Go always has them after a restart (not just on connect/change)
                beatCount++;
                if (beatCount % 3 == 0)
                {
                    SendAccountsList();
                }

                // Periodic poll of account balance every heartbeat (30s) — Tradovate's
                // AccountItemUpdate doesn't fire reliably, so poll to ensure real balance
                // populates even if AccountItemUpdate misses + connect-seed failed.
                // If account not ready yet, skip + retry next beat.
                try
                {
                    SendAccountBalance();
                }
                catch (Exception ex)
                {
                    LogWarn($"RunHeartbeatLoop: SendAccountBalance poll failed (will retry): {ex.Message}");
                }
            }
        }

        // === Log helpers — NT8 Log API varies slightly across versions; isolate ===
        private static void LogInfo(string msg)
        {
            try { NinjaScript.Log(msg, LogLevel.Information); } catch { }
        }

        private static void LogWarn(string msg)
        {
            try { NinjaScript.Log(msg, LogLevel.Warning); } catch { }
        }

        // ==============================================================
        // Hand-rolled JSON encoder + parser (no external deps).
        // NT8 user AddOns cannot resolve Newtonsoft.Json reliably, so we
        // implement only what the 4-message wire protocol needs:
        // {type, payload} envelopes with flat snake_case payloads.
        // On-wire bytes must match Go encoding/json: snake_case keys,
        // invariant-culture numbers, compact (no pretty-print), UTF-8.
        // ==============================================================

        private static string EncodeFrame(string type, Dictionary<string, object> payload)
        {
            var sb = new StringBuilder();
            sb.Append('{');
            sb.Append("\"type\":");
            AppendString(sb, type);
            sb.Append(",\"payload\":");
            AppendObject(sb, payload);
            sb.Append('}');
            return sb.ToString();
        }

        private static void AppendObject(StringBuilder sb, Dictionary<string, object> obj)
        {
            sb.Append('{');
            if (obj != null)
            {
                bool first = true;
                foreach (var kv in obj)
                {
                    if (!first) sb.Append(',');
                    first = false;
                    AppendString(sb, kv.Key);
                    sb.Append(':');
                    AppendValue(sb, kv.Value);
                }
            }
            sb.Append('}');
        }

        private static void AppendValue(StringBuilder sb, object v)
        {
            if (v == null) { sb.Append("null"); return; }
            if (v is string s) { AppendString(sb, s); return; }
            if (v is bool b) { sb.Append(b ? "true" : "false"); return; }
            if (v is int i) { sb.Append(i.ToString(CultureInfo.InvariantCulture)); return; }
            if (v is long l) { sb.Append(l.ToString(CultureInfo.InvariantCulture)); return; }
            if (v is double d) { sb.Append(d.ToString("R", CultureInfo.InvariantCulture)); return; }
            if (v is float f) { sb.Append(((double)f).ToString("R", CultureInfo.InvariantCulture)); return; }
            if (v is decimal m) { sb.Append(m.ToString(CultureInfo.InvariantCulture)); return; }
            if (v is Dictionary<string, object> nested) { AppendObject(sb, nested); return; }
            // Plan 4.4 bar-payload fix: List<object> / arrays. Must come AFTER
            // string + Dictionary (string is IEnumerable<char>; Dictionary is
            // IEnumerable<KeyValuePair>). Without this, bars_historical /
            // bar_update payloads (["bars"] = List<object>) get stringified
            // via the .ToString() fallback to "System.Collections.Generic.
            // List`1[System.Object]", and Go rejects them with
            // "cannot unmarshal string into ... []Bar".
            if (v is System.Collections.IEnumerable e)
            {
                sb.Append('[');
                bool firstItem = true;
                foreach (var item in e)
                {
                    if (!firstItem) sb.Append(',');
                    firstItem = false;
                    AppendValue(sb, item);
                }
                sb.Append(']');
                return;
            }
            AppendString(sb, v.ToString());
        }

        private static void AppendString(StringBuilder sb, string s)
        {
            sb.Append('"');
            if (s != null)
            {
                foreach (char c in s)
                {
                    switch (c)
                    {
                        case '"':  sb.Append("\\\""); break;
                        case '\\': sb.Append("\\\\"); break;
                        case '\n': sb.Append("\\n"); break;
                        case '\r': sb.Append("\\r"); break;
                        case '\t': sb.Append("\\t"); break;
                        case '\b': sb.Append("\\b"); break;
                        case '\f': sb.Append("\\f"); break;
                        default:
                            if (c < 0x20) sb.AppendFormat("\\u{0:x4}", (int)c);
                            else sb.Append(c);
                            break;
                    }
                }
            }
            sb.Append('"');
        }

        // === Field-extraction helpers (typed reads from parsed dictionary) ===
        private static string GetString(Dictionary<string, object> obj, string key)
        {
            if (obj != null && obj.TryGetValue(key, out var v) && v != null) return v.ToString();
            return null;
        }

        private static double GetDouble(Dictionary<string, object> obj, string key)
        {
            if (obj != null && obj.TryGetValue(key, out var v) && v != null)
            {
                if (v is double d) return d;
                if (v is long l) return (double)l;
                if (v is int i) return (double)i;
                if (v is decimal m) return (double)m;
                return Convert.ToDouble(v, CultureInfo.InvariantCulture);
            }
            return 0.0;
        }

        private static int GetInt(Dictionary<string, object> obj, string key)
        {
            if (obj != null && obj.TryGetValue(key, out var v) && v != null)
            {
                if (v is long l) return (int)l;
                if (v is int i) return i;
                if (v is double d) return (int)d;
                return Convert.ToInt32(v, CultureInfo.InvariantCulture);
            }
            return 0;
        }

        // === Recursive-descent JSON parser ===
        private class JsonParser
        {
            private readonly string s;
            private int pos;

            public JsonParser(string input)
            {
                s = input ?? string.Empty;
                pos = 0;
            }

            public object Parse()
            {
                SkipWs();
                return ParseValue();
            }

            private object ParseValue()
            {
                SkipWs();
                if (pos >= s.Length) throw new Exception("unexpected EOF");
                char c = s[pos];
                if (c == '{') return ParseObject();
                if (c == '"') return ParseString();
                if (c == 't' || c == 'f') return ParseBool();
                if (c == 'n') return ParseNull();
                if (c == '-' || (c >= '0' && c <= '9')) return ParseNumber();
                if (c == '[') return ParseArray();
                throw new Exception("unexpected char '" + c + "' at " + pos);
            }

            private Dictionary<string, object> ParseObject()
            {
                var result = new Dictionary<string, object>();
                pos++; // consume '{'
                SkipWs();
                if (pos < s.Length && s[pos] == '}') { pos++; return result; }
                while (true)
                {
                    SkipWs();
                    string key = ParseString();
                    SkipWs();
                    if (pos >= s.Length || s[pos] != ':') throw new Exception("expected ':' at " + pos);
                    pos++; // consume ':'
                    SkipWs();
                    object value = ParseValue();
                    result[key] = value;
                    SkipWs();
                    if (pos < s.Length && s[pos] == ',') { pos++; continue; }
                    if (pos < s.Length && s[pos] == '}') { pos++; return result; }
                    throw new Exception("expected ',' or '}' at " + pos);
                }
            }

            private string ParseString()
            {
                if (pos >= s.Length || s[pos] != '"') throw new Exception("expected string at " + pos);
                pos++; // consume opening '"'
                var sb = new StringBuilder();
                while (pos < s.Length && s[pos] != '"')
                {
                    if (s[pos] == '\\' && pos + 1 < s.Length)
                    {
                        char esc = s[pos + 1];
                        switch (esc)
                        {
                            case '"':  sb.Append('"'); pos += 2; break;
                            case '\\': sb.Append('\\'); pos += 2; break;
                            case '/':  sb.Append('/'); pos += 2; break;
                            case 'n':  sb.Append('\n'); pos += 2; break;
                            case 'r':  sb.Append('\r'); pos += 2; break;
                            case 't':  sb.Append('\t'); pos += 2; break;
                            case 'b':  sb.Append('\b'); pos += 2; break;
                            case 'f':  sb.Append('\f'); pos += 2; break;
                            case 'u':
                                if (pos + 5 < s.Length)
                                {
                                    sb.Append((char)Convert.ToInt32(s.Substring(pos + 2, 4), 16));
                                    pos += 6;
                                }
                                else
                                {
                                    pos++;
                                }
                                break;
                            default: sb.Append(esc); pos += 2; break;
                        }
                    }
                    else
                    {
                        sb.Append(s[pos]);
                        pos++;
                    }
                }
                if (pos >= s.Length) throw new Exception("unterminated string");
                pos++; // consume closing '"'
                return sb.ToString();
            }

            private object ParseNumber()
            {
                int start = pos;
                if (s[pos] == '-') pos++;
                while (pos < s.Length &&
                       (char.IsDigit(s[pos]) || s[pos] == '.' ||
                        s[pos] == 'e' || s[pos] == 'E' ||
                        s[pos] == '+' || s[pos] == '-'))
                {
                    pos++;
                }
                string num = s.Substring(start, pos - start);
                if (num.IndexOfAny(new[] { '.', 'e', 'E' }) >= 0)
                {
                    return double.Parse(num, CultureInfo.InvariantCulture);
                }
                if (long.TryParse(num, NumberStyles.Integer, CultureInfo.InvariantCulture, out long l))
                {
                    return l;
                }
                return double.Parse(num, CultureInfo.InvariantCulture);
            }

            private bool ParseBool()
            {
                if (pos + 4 <= s.Length && s.Substring(pos, 4) == "true") { pos += 4; return true; }
                if (pos + 5 <= s.Length && s.Substring(pos, 5) == "false") { pos += 5; return false; }
                throw new Exception("invalid bool at " + pos);
            }

            private object ParseNull()
            {
                if (pos + 4 <= s.Length && s.Substring(pos, 4) == "null") { pos += 4; return null; }
                throw new Exception("invalid null at " + pos);
            }

            private List<object> ParseArray()
            {
                var result = new List<object>();
                pos++; // consume '['
                SkipWs();
                if (pos < s.Length && s[pos] == ']') { pos++; return result; }
                while (true)
                {
                    SkipWs();
                    result.Add(ParseValue());
                    SkipWs();
                    if (pos < s.Length && s[pos] == ',') { pos++; continue; }
                    if (pos < s.Length && s[pos] == ']') { pos++; return result; }
                    throw new Exception("expected ',' or ']' at " + pos);
                }
            }

            private void SkipWs()
            {
                while (pos < s.Length &&
                       (s[pos] == ' ' || s[pos] == '\t' ||
                        s[pos] == '\n' || s[pos] == '\r'))
                {
                    pos++;
                }
            }
        }
    }
}
