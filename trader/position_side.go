package trader

import "strings"

// ── W-EXEC-TRUTH W0 (Q10/Q12, checklist canon 28) ──────────────────────────
// ONE canonicalizer for a position's side, called where the value ENTERS the
// decision logic. Writers store "LONG"/"SHORT"; NT8's positionMap emits
// "LONG"/"SHORT" with a SIGNED positionAmt; the crypto brokers emit lowercase
// with a signed amount. Comparing any of them raw against "long" was the cause
// of two dead safety legs (EntryGate leg 7, the executeOpen* same-side guards)
// and of reconcileBeforeOpenNT reading a held short as flat.

// positionSide canonicalizes a side string: "long", "short", or "" for
// anything else.
func positionSide(s string) string {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "long":
		return "long"
	case "short":
		return "short"
	}
	return ""
}

// brokerPositionSide reads a broker position map's side canonically; with no
// recognizable side string it falls back to the SIGN of positionAmt (every
// broker signs a short negative). "" = no position side can be established.
func brokerPositionSide(pos map[string]interface{}) string {
	s, _ := pos["side"].(string)
	if c := positionSide(s); c != "" {
		return c
	}
	if amt, ok := pos["positionAmt"].(float64); ok {
		switch {
		case amt > 0:
			return "long"
		case amt < 0:
			return "short"
		}
	}
	return ""
}
