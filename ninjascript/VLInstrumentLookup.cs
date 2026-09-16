// VLInstrumentLookup.cs — the ONE place the AddOn turns a Go-side symbol into
// an NT8 Instrument. Bars and orders must resolve through the same door so they
// can never be on different contracts (2026-09-10: bars and orders each called
// the date rule at their own moment and disagreed for 135 minutes; then both
// were on December while the platform was on September).
//
// Resolution order:
//   1. NT8's rolling name ("MNQ ##-##") — the platform's own front month.
//   2. ONLY if the build rejects the rolling name: the old date rule, logged
//      by name so the operator can see the fallback ran.
//
// ContractName() names the contract NT8 actually resolved ("MNQ 09-26"),
// derived from the instrument's expiry — never the rolling literal — so the
// subscribed ACK and every bar frame carry the real name and a roll shows up
// as a name change on the wire.

#region Using declarations
using System;
using System.Globalization;
using NinjaTrader.Cbi;
#endregion

namespace NinjaTrader.NinjaScript.AddOns
{
    public static class VLInstrumentLookup
    {
        /// <summary>
        /// Resolve returns the Instrument for a Go-side symbol, or null. `how`
        /// receives "rolling" or "date-rule-fallback" for the caller's log line.
        /// </summary>
        public static Instrument Resolve(string symbol, Action<string> logWarn, out string how)
        {
            how = "rolling";
            string rolling = VLContractResolver.RollingContractName(symbol);
            Instrument instrument = null;
            try { instrument = Instrument.GetInstrument(rolling); }
            catch (Exception ex)
            {
                if (logWarn != null) logWarn("VLInstrumentLookup: GetInstrument(\"" + rolling + "\") threw: " + ex.Message);
            }
            if (instrument != null)
            {
                // The rolling instrument names the front month but a BarsRequest
                // against it returned ZERO bars on this build (2026-09-11 00:33 CT,
                // every timeframe, 22 minutes blind). Ask its master for the
                // rollover-aware front-month EXPIRY and resolve the CONCRETE
                // contract NT8 says is current — that one has data and takes orders.
                Instrument concrete = null;
                string concreteName = "";
                try
                {
                    var mi = instrument.MasterInstrument;
                    DateTime exp = mi.GetNextExpiry(DateTime.Now);
                    concreteName = string.Format(CultureInfo.InvariantCulture, "{0} {1:D2}-{2:D2}", mi.Name, exp.Month, exp.Year % 100);
                    concrete = Instrument.GetInstrument(concreteName);
                }
                catch (Exception ex)
                {
                    if (logWarn != null) logWarn("VLInstrumentLookup: front-month expiry lookup for '" + rolling + "' threw: " + ex.Message);
                }
                if (concrete != null)
                {
                    how = "rolling->" + concreteName;
                    return concrete;
                }
                if (logWarn != null)
                {
                    logWarn("VLInstrumentLookup: rolling '" + rolling + "' resolved but its concrete front month '"
                            + concreteName + "' did not — falling back to the DATE RULE (owner ruling 2026-09-11: say so)");
                }
            }
            // The rolling name was refused. Fall back to the computed expiry and SAY SO:
            // a silent fallback would put us back on a contract the platform is not on.
            string dated = VLContractResolver.DateRuleContract(symbol);
            if (string.Equals(dated, rolling, StringComparison.Ordinal) || string.Equals(dated, symbol, StringComparison.Ordinal))
            {
                return null; // non-quarterly root or passthrough — nothing else to try
            }
            how = "date-rule-fallback";
            if (logWarn != null)
            {
                logWarn("VLInstrumentLookup: NT8 refused the rolling name '" + rolling
                        + "' — falling back to the DATE RULE '" + dated
                        + "'. This contract may not be the platform's front month (owner ruling 2026-09-11).");
            }
            try { instrument = Instrument.GetInstrument(dated); }
            catch (Exception ex)
            {
                if (logWarn != null) logWarn("VLInstrumentLookup: GetInstrument(\"" + dated + "\") threw: " + ex.Message);
            }
            return instrument;
        }

        /// <summary>
        /// ContractName is the qualified name of the contract NT8 resolved —
        /// "<root> <MM-yy>" from the instrument's expiry — so a rolling lookup
        /// still names the real contract on the wire.
        /// </summary>
        public static string ContractName(Instrument instrument)
        {
            if (instrument == null) return "";
            try
            {
                string root = instrument.MasterInstrument != null ? instrument.MasterInstrument.Name : "";
                DateTime exp = instrument.Expiry;
                if (!string.IsNullOrEmpty(root) && exp.Year > 1971)
                {
                    return string.Format(CultureInfo.InvariantCulture, "{0} {1:D2}-{2:D2}", root, exp.Month, exp.Year % 100);
                }
            }
            catch (Exception) { }
            return instrument.FullName;
        }
    }
}
