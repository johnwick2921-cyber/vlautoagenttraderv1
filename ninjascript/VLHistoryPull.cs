// VLHistoryPull.cs — HISTORY IMPORT (wave 101, 2026-09-11).
//
// Serves the Go server's bars_history_request frames: one NAMED contract, one
// timeframe, one [from, to) window, answered as chunked bars_history_data
// frames (seq/last) or a bars_history_error naming why not.
//
// WHY A SEPARATE CLASS. VLBarsSubscriptionManager owns the LIVE BarsRequest
// subscriptions and their reconnect/reseed discipline; this class must never
// touch that state. A history pull is an independent, disposable BarsRequest
// against an EXPLICIT instrument name — "MNQ 09-23" — resolved directly via
// Instrument.GetInstrument, never through VLInstrumentLookup (which answers
// "the platform's front month") and never from a date rule (that is the
// 2026-09-10 roll bug, one scale per contract).
//
// RULES THIS CLASS OBEYS:
//   - MergePolicy.DoNotMerge ALWAYS: a back-adjusted series is a different
//     instrument wearing the same label (owner ruling 2026-09-11).
//   - Every data frame echoes the contract from the instrument's real expiry
//     (VLInstrumentLookup.ContractName), never the request string.
//   - Bars are sent ascending by time, chunked at ~8000 per frame to stay
//     under the 1 MB envelope.
//   - A pull that cannot be served is ANSWERED (bars_history_error), never
//     silent — three-state honesty.

#region Using declarations
using System;
using System.Collections.Generic;
using System.Globalization;
using NinjaTrader.Cbi;
using NinjaTrader.Data;
#endregion

namespace NinjaTrader.NinjaScript.AddOns
{
    /// <summary>One in-flight named-contract pull.</summary>
    public class VLHistoryPullEntry
    {
        public string RequestId;
        public string Symbol;
        public string Contract;
        public string Timeframe;
        public BarsRequest Request;
    }

    public class VLHistoryPullManager
    {
        private const int CHUNK_BARS = 8000; // ~640 KB at ~80 B/bar — under the 1 MB envelope

        private readonly Action<string, Dictionary<string, object>> sendFrame;
        private readonly Action<string> logInfo;
        private readonly Action<string> logWarn;
        private readonly Dictionary<string, VLHistoryPullEntry> inFlight =
            new Dictionary<string, VLHistoryPullEntry>();

        public VLHistoryPullManager(
            Action<string, Dictionary<string, object>> sendFrame,
            Action<string> logInfo,
            Action<string> logWarn)
        {
            this.sendFrame = sendFrame;
            this.logInfo = logInfo;
            this.logWarn = logWarn;
        }

        /// <summary>Handles one bars_history_request frame.</summary>
        public void HandleBarsHistoryRequest(Dictionary<string, object> p)
        {
            if (p == null) { logWarn("VLHistoryPull: bars_history_request payload missing"); return; }
            string requestId = TryGetString(p, "request_id");
            string symbol = TryGetString(p, "symbol");
            string contract = TryGetString(p, "contract");
            string timeframe = TryGetString(p, "timeframe");
            long fromMs = TryGetLong(p, "from_ms");
            long toMs = TryGetLong(p, "to_ms");
            if (string.IsNullOrEmpty(requestId))
            {
                logWarn("VLHistoryPull: bars_history_request without request_id — refused");
                return;
            }
            if (string.IsNullOrEmpty(contract))
            {
                SendError(requestId, contract, "a contract must be NAMED — importing by date inference is the 09-10 bug");
                return;
            }
            if (inFlight.ContainsKey(requestId))
            {
                logWarn("VLHistoryPull: duplicate request_id " + requestId + " — refused");
                return;
            }

            Instrument instrument = null;
            try { instrument = Instrument.GetInstrument(contract); }
            catch (Exception ex)
            {
                logWarn("VLHistoryPull: GetInstrument(\"" + contract + "\") threw: " + ex.Message);
            }
            if (instrument == null)
            {
                SendError(requestId, contract, "unavailable: instrument " + contract + " not found on this platform");
                return;
            }

            BarsPeriod period = BarsPeriodFor(timeframe);
            if (period == null)
            {
                SendError(requestId, contract, "unavailable: unsupported timeframe " + timeframe);
                return;
            }

            DateTime from = fromMs > 0 ? DateTimeFromUtcMs(fromMs) : instrument.Expiry.AddMonths(-4);
            DateTime to = toMs > 0 ? DateTimeFromUtcMs(toMs) : instrument.Expiry.AddDays(1);
            if (to <= from)
            {
                SendError(requestId, contract, "unavailable: empty window (to <= from)");
                return;
            }

            string realName = VLInstrumentLookup.ContractName(instrument);

            // FLAG: NT8 API — the (instrument, from, to) BarsRequest constructor
            // is the documented 8.1 form for a dated window. If the operator's
            // build only exposes the (instrument, barsBack) form, this line is
            // the one to adapt.
            BarsRequest request;
            try
            {
                request = new BarsRequest(instrument, from, to);
                request.BarsPeriod = period;
                // DO NOT MERGE (owner ruling 2026-09-11): a back-adjusted
                // history is a different price scale wearing the same label.
                request.MergePolicy = MergePolicy.DoNotMerge;
                var hours = TradingHours.Get("CME US Index Futures ETH");
                if (hours != null) request.TradingHours = hours;
            }
            catch (Exception ex)
            {
                SendError(requestId, contract, "unavailable: BarsRequest construction failed: " + ex.Message);
                return;
            }

            inFlight[requestId] = new VLHistoryPullEntry
            {
                RequestId = requestId, Symbol = string.IsNullOrEmpty(symbol) ? "MNQ" : symbol,
                Contract = realName, Timeframe = timeframe, Request = request,
            };

            logInfo("VLHistoryPull: requesting " + realName + " " + timeframe
                    + " [" + from.ToString("yyyy-MM-dd", CultureInfo.InvariantCulture)
                    + ", " + to.ToString("yyyy-MM-dd", CultureInfo.InvariantCulture) + ")");

            // FLAG: NT8 API — the Request callback's FIRST parameter is the
            // BarsRequest itself, not the Bars collection (the same signature
            // the compiling VLBarsSubscriptionManager.cs:420 uses). The bars
            // hang off `req.Bars`. CS1503 is exactly this mismatch.
            request.Request((req, errorCode, errorMessage) =>
            {
                VLHistoryPullEntry entry;
                lock (inFlight) { inFlight.TryGetValue(requestId, out entry); }
                if (entry == null) return; // torn down while in flight
                try
                {
                    if (errorCode != ErrorCode.NoError || req.Bars == null)
                    {
                        SendError(requestId, realName, "unavailable: NT8 error " + errorCode
                                  + (string.IsNullOrEmpty(errorMessage) ? "" : ": " + errorMessage));
                        return;
                    }
                    EmitChunks(entry, req.Bars);
                }
                catch (Exception ex)
                {
                    SendError(requestId, realName, "unavailable: " + ex.Message);
                }
                finally
                {
                    lock (inFlight) { inFlight.Remove(requestId); }
                    try { request.Dispose(); } catch { }
                }
            });
        }

        /// <summary>Splits the loaded bars into ascending chunks and emits them.</summary>
        private void EmitChunks(VLHistoryPullEntry entry, Bars bars)
        {
            int n = bars.Count;
            if (n == 0)
            {
                // Zero bars is an answer, not silence: last=true, no bars.
                var done = new Dictionary<string, object>
                {
                    ["request_id"] = entry.RequestId, ["symbol"] = entry.Symbol,
                    ["contract"] = entry.Contract, ["timeframe"] = entry.Timeframe,
                    ["seq"] = 1, ["last"] = true, ["bars"] = new List<object>(),
                };
                sendFrame("bars_history_data", done);
                logInfo("VLHistoryPull: " + entry.Contract + " " + entry.Timeframe + " — zero bars served");
                return;
            }

            var chunk = new List<object>(CHUNK_BARS);
            int seq = 0;
            long lastT = 0;
            for (int i = 0; i < n; i++)
            {
                long t = ToUtcEpochMs(bars.GetTime(i), bars.TradingHours);
                if (t <= lastT) continue; // ascending, dedup guard
                lastT = t;
                chunk.Add(BuildBarObject(t, bars.GetOpen(i), bars.GetHigh(i),
                    bars.GetLow(i), bars.GetClose(i), bars.GetVolume(i)));
                if (chunk.Count < CHUNK_BARS && i < n - 1) continue;
                seq++;
                var payload = new Dictionary<string, object>
                {
                    ["request_id"] = entry.RequestId, ["symbol"] = entry.Symbol,
                    ["contract"] = entry.Contract, ["timeframe"] = entry.Timeframe,
                    ["seq"] = seq, ["last"] = (i >= n - 1), ["bars"] = chunk,
                };
                sendFrame("bars_history_data", payload);
                logInfo("VLHistoryPull: " + entry.Contract + " " + entry.Timeframe
                        + " chunk " + seq + " bars=" + chunk.Count + (i >= n - 1 ? " (last)" : ""));
                chunk = new List<object>(CHUNK_BARS);
            }
        }

        private void SendError(string requestId, string contract, string reason)
        {
            var payload = new Dictionary<string, object>
            {
                ["request_id"] = requestId, ["contract"] = contract ?? "", ["reason"] = reason,
            };
            sendFrame("bars_history_error", payload);
            logWarn("VLHistoryPull: " + contract + " — " + reason);
        }

        /// <summary>Tears down every in-flight pull (AddOn terminate).</summary>
        public void DisposeAll()
        {
            lock (inFlight)
            {
                foreach (var kv in inFlight)
                {
                    try { kv.Value.Request.Dispose(); } catch { }
                }
                inFlight.Clear();
            }
        }

        private static BarsPeriod BarsPeriodFor(string tf)
        {
            switch (string.IsNullOrEmpty(tf) ? "" : tf.Trim())
            {
                case "1m": return new BarsPeriod { BarsPeriodType = BarsPeriodType.Minute, Value = 1 };
                case "3m": return new BarsPeriod { BarsPeriodType = BarsPeriodType.Minute, Value = 3 };
                case "5m": return new BarsPeriod { BarsPeriodType = BarsPeriodType.Minute, Value = 5 };
                case "15m": return new BarsPeriod { BarsPeriodType = BarsPeriodType.Minute, Value = 15 };
                case "30m": return new BarsPeriod { BarsPeriodType = BarsPeriodType.Minute, Value = 30 };
                case "1h": return new BarsPeriod { BarsPeriodType = BarsPeriodType.Minute, Value = 60 };
                case "2h": return new BarsPeriod { BarsPeriodType = BarsPeriodType.Minute, Value = 120 };
                case "4h": return new BarsPeriod { BarsPeriodType = BarsPeriodType.Minute, Value = 240 };
                case "6h": return new BarsPeriod { BarsPeriodType = BarsPeriodType.Minute, Value = 360 };
                case "8h": return new BarsPeriod { BarsPeriodType = BarsPeriodType.Minute, Value = 480 };
                case "12h": return new BarsPeriod { BarsPeriodType = BarsPeriodType.Minute, Value = 720 };
                case "1d": return new BarsPeriod { BarsPeriodType = BarsPeriodType.Day, Value = 1 };
                default: return null;
            }
        }

        private static object BuildBarObject(long t, double o, double h, double l, double c, double v)
        {
            return new Dictionary<string, object>
            {
                ["t"] = t, ["o"] = o, ["h"] = h, ["l"] = l, ["c"] = c, ["v"] = v,
            };
        }

        private static DateTime DateTimeFromUtcMs(long ms)
        {
            return new DateTime(1970, 1, 1, 0, 0, 0, DateTimeKind.Utc).AddMilliseconds(ms);
        }

        // Mirror of VLBarsSubscriptionManager's time conversion: the bar's
        // timestamp in the session's zone, expressed as UTC epoch ms.
        // FLAG: NT8 API — on this build `TradingHours.TimeZoneInfo` is a
        // TimeZoneInfo OBJECT (not a string); assigning it directly is the
        // pattern the compiling manager uses. Passing it to
        // FindSystemTimeZoneById(string) is CS1503.
        private static long ToUtcEpochMs(DateTime localTime, TradingHours tradingHours)
        {
            DateTime utc;
            if (localTime.Kind == DateTimeKind.Utc)
            {
                utc = localTime;
            }
            else
            {
                TimeZoneInfo tz = null;
                if (tradingHours != null)
                {
                    tz = tradingHours.TimeZoneInfo;
                }
                if (tz == null)
                {
                    // Fall back to the machine's local zone — wrong if the
                    // bars were stamped in CT and the machine is in another
                    // zone, but better than throwing.
                    tz = TimeZoneInfo.Local;
                }
                utc = TimeZoneInfo.ConvertTimeToUtc(
                    DateTime.SpecifyKind(localTime, DateTimeKind.Unspecified), tz);
            }
            var unixEpoch = new DateTime(1970, 1, 1, 0, 0, 0, DateTimeKind.Utc);
            return (long)((utc - unixEpoch).TotalMilliseconds);
        }

        private static string TryGetString(Dictionary<string, object> p, string key)
        {
            object v; if (p != null && p.TryGetValue(key, out v) && v != null) return v.ToString();
            return "";
        }

        private static long TryGetLong(Dictionary<string, object> p, string key)
        {
            string s = TryGetString(p, key);
            long n; if (long.TryParse(s, out n)) return n;
            return 0;
        }
    }
}
