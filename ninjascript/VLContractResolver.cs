// VLContractResolver.cs — shared front-month contract resolver for the
// VL Trader NT8 AddOn.
//
// Maps Go-side symbols (bare root like "MNQ", or Databento continuous form
// like "NQ.c.0") to NT8's qualified instrument name ("<root> <MM-YY>"). Used
// by both VLBarsSubscriptionManager (bars_subscribe path) and
// VLTraderTCPClient (signal path) so the two stay in lockstep.
//
// This file CANNOT be compiled in WSL2 — the operator compiles on Windows
// via NT8's NinjaScript editor (F5). Pure C# only: no Newtonsoft, no LINQ,
// no NT8 API dependency, so it's easy to unit-test in a side project.
//
// Spec: ResolveFrontMonthContract behavior table in fix/nt8-frontmonth-
// contract-resolution PR description; verification table authored by
// Agent B in the same branch.

#region Using declarations
using System;
using System.Globalization;
#endregion

namespace NinjaTrader.NinjaScript.AddOns
{
    /// <summary>
    /// Static helper that maps bare CME futures roots and Databento
    /// continuous symbols to NT8 qualified front-month instrument names.
    /// Pure C#; no NT8 API dependency.
    /// </summary>
    public static class VLContractResolver
    {
        // Whitelist of CME roots whose bare-root forms get mapped to the current
        // front-month QUARTERLY (H/M/U/Z) contract by the date logic below. Only
        // quarterly-cycle families belong here. ENERGY (CL/MCL/NG — monthly) and
        // METALS (GC/MGC — Feb/Apr/Jun/Aug/Oct/Dec; SI — Jan/Mar/May/Jul/Sep/Dec)
        // are deliberately NOT listed: their front month is NOT quarterly, so this
        // computation would pick the wrong contract. They are recognized in Go
        // (Phase 1) but their NT8 front-month resolution needs the per-family roll
        // calendar (MasterInstrument.GetNextExpiry) — a later phase. Until then an
        // energy/metal root passes through unchanged and the BarsRequest path logs
        // it as unresolved (see VLBarsSubscriptionManager) rather than mis-resolving.
        private static readonly string[] CMEFuturesRoots = new[]
        {
            "MNQ", "NQ",   // E-mini + Micro E-mini Nasdaq-100
            "MES", "ES",   // E-mini + Micro E-mini S&P 500
            "MYM", "YM",   // E-mini + Micro E-mini Dow
            "M2K", "RTY",  // Micro + E-mini Russell 2000
            "ZB", "ZN",    // 30Y T-Bond + 10Y T-Note (CBOT, quarterly H/M/U/Z)
            "ZF", "ZT",    // 5Y + 2Y T-Note (CBOT, quarterly H/M/U/Z)
        };

        // Days before the 3rd-Friday expiry at which we roll to the next
        // quarterly. Set to 8 so the operator's chart + the AddOn's
        // subscription line up well before the actual expiry date and the
        // liquidity moves to the next contract.
        private const int RollDaysBeforeExpiry = 8;

        /// <summary>
        /// ResolveFrontMonthContract maps a Go-side symbol to the NT8
        /// instrument name (`<root> <MM-YY>`). Behavior:
        ///
        /// 1. If the symbol already contains a space (already qualified
        ///    like "MNQ 06-26"), return it unchanged (idempotent).
        /// 2. Strip Databento continuous suffix: "NQ.c.0" -> "NQ",
        ///    "MNQ.c.0" -> "MNQ".
        /// 3. If the stripped root is not in CMEFuturesRoots (e.g. a
        ///    crypto symbol like "BTCUSDT"), return the ORIGINAL symbol
        ///    unchanged — don't mangle non-CME symbols.
        /// 4. Derive the current front-month quarterly from
        ///    DateTime.UtcNow:
        ///    - Quarterly months: H=Mar(3), M=Jun(6), U=Sep(9), Z=Dec(12).
        ///    - Expiry = 3rd Friday of that month at end-of-day.
        ///    - "Roll" = (expiry - RollDaysBeforeExpiry).
        ///    - Pick the nearest quarterly whose roll date is >= today.
        ///    - If past December's roll, advance to next year's March.
        /// 5. Format using CultureInfo.InvariantCulture as
        ///    "{root} {MM:D2}-{YY:D2}" (year is year%100, both zero-padded).
        /// </summary>
        public static string ResolveFrontMonthContract(string symbol)
        {
            return ResolveFrontMonthContractAt(symbol, DateTime.UtcNow);
        }

        /// <summary>
        /// ResolveFrontMonthContractAt is a testable variant accepting an
        /// explicit `now` so unit tests / Agent B's verification table can
        /// pin the date. Production code calls ResolveFrontMonthContract
        /// which internally uses DateTime.UtcNow.
        /// </summary>
        public static string ResolveFrontMonthContractAt(string symbol, DateTime now)
        {
            // Null/empty guard — return as-is rather than throw so callers
            // can do best-effort logging upstream.
            if (string.IsNullOrEmpty(symbol))
            {
                return symbol;
            }

            // (1) Already qualified ("MNQ 06-26") — idempotent passthrough.
            if (symbol.IndexOf(' ') >= 0)
            {
                return symbol;
            }

            // (2) Strip Databento continuous suffix ".c.0", ".c.1", etc.
            // Match case-insensitively; take everything before ".c.".
            string root = symbol;
            int continuousIdx = symbol.IndexOf(".c.", StringComparison.OrdinalIgnoreCase);
            if (continuousIdx > 0)
            {
                root = symbol.Substring(0, continuousIdx);
            }

            // (3) Whitelist check — non-CME symbols pass through unchanged.
            bool isCME = false;
            for (int i = 0; i < CMEFuturesRoots.Length; i++)
            {
                if (string.Equals(root, CMEFuturesRoots[i], StringComparison.OrdinalIgnoreCase))
                {
                    isCME = true;
                    break;
                }
            }
            if (!isCME)
            {
                return symbol;
            }

            // (4) Walk quarterly months {3, 6, 9, 12} of current year, then
            // March of next year as the wrap. First quarterly whose roll
            // date is >= today.Date wins.
            DateTime today = now.Date;
            int[] quarterlyMonths = new[] { 3, 6, 9, 12 };
            int year = now.Year;
            int frontMonth = 0;
            int frontYear = 0;

            for (int i = 0; i < quarterlyMonths.Length; i++)
            {
                int qm = quarterlyMonths[i];
                DateTime expiry = ThirdFridayOf(year, qm);
                DateTime rollDate = expiry.AddDays(-RollDaysBeforeExpiry);
                if (rollDate >= today)
                {
                    frontMonth = qm;
                    frontYear = year;
                    break;
                }
            }

            // Past December's roll → wrap to next year's March.
            if (frontMonth == 0)
            {
                frontMonth = 3;
                frontYear = year + 1;
            }

            // (5) Format "{root} {MM:D2}-{YY:D2}" — uppercase the root so
            // "nq.c.0" still yields "NQ 06-26", and use invariant culture so
            // the operator's regional locale can't inject ',' separators.
            string upperRoot = root.ToUpperInvariant();
            int yy = frontYear % 100;
            return string.Format(
                CultureInfo.InvariantCulture,
                "{0} {1:D2}-{2:D2}",
                upperRoot,
                frontMonth,
                yy);
        }

        /// <summary>
        /// ThirdFridayOf returns the 3rd Friday of the given (year, month).
        /// Exposed internally for testability.
        /// </summary>
        internal static DateTime ThirdFridayOf(int year, int month)
        {
            var first = new DateTime(year, month, 1);
            int offset = ((int)DayOfWeek.Friday - (int)first.DayOfWeek + 7) % 7;
            return first.AddDays(offset + 14);
        }
    }
}
