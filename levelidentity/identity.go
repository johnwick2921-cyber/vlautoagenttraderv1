// Package levelidentity owns the recording-only candidate identity contract.
// It has no dependency on detector, scoring, gate or execution code.
package levelidentity

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math"
	"strings"
)

// Inputs preserves presence independently of value. Formation is the separately
// captured CLOSE; the trading system's FormedAtMs remains candle open.
type Inputs struct {
	Symbol        string   `json:"symbol"`
	Kind          string   `json:"kind"`
	Lo            *float64 `json:"lo"`
	Hi            *float64 `json:"hi"`
	OriginDate    string   `json:"origin_date"`
	TF            string   `json:"tf"`
	FormedCloseMs *int64   `json:"formed_close_ms"`
}

// ID uses Stage A's seven-field hash layout, with the approved formation CLOSE
// in the formation position. A missing input never becomes a partial hash.
func ID(in Inputs) (*string, string) {
	var missing []string
	if strings.TrimSpace(in.Symbol) == "" {
		missing = append(missing, "symbol")
	}
	if strings.TrimSpace(in.Kind) == "" {
		missing = append(missing, "kind")
	}
	if in.Lo == nil || *in.Lo <= 0 || math.IsNaN(*in.Lo) || math.IsInf(*in.Lo, 0) {
		missing = append(missing, "lo")
	}
	if in.Hi == nil || *in.Hi <= 0 || math.IsNaN(*in.Hi) || math.IsInf(*in.Hi, 0) {
		missing = append(missing, "hi")
	}
	if strings.TrimSpace(in.OriginDate) == "" {
		missing = append(missing, "origin_date")
	}
	if strings.TrimSpace(in.TF) == "" {
		missing = append(missing, "tf")
	}
	if in.FormedCloseMs == nil || *in.FormedCloseMs <= 0 {
		missing = append(missing, "formed_close_ms")
	}
	if len(missing) > 0 {
		return nil, strings.Join(missing, ",")
	}
	identity := fmt.Sprintf("%s|%s|%g|%g|%s|%s|%d", in.Symbol, in.Kind, *in.Lo, *in.Hi, in.OriginDate, in.TF, *in.FormedCloseMs)
	h := sha256.Sum256([]byte(identity))
	id := hex.EncodeToString(h[:])
	return &id, ""
}
