// Task 12 / Cluster D — verify normalizeSymbols preserves the case of CME
// futures symbols and does not append USDT-like noise. Crypto behavior
// remains uppercased + trimmed as before.

package store

import (
	"reflect"
	"testing"
)

func TestNormalizeSymbols_CMEFutures_CasePreserved(t *testing.T) {
	in := []string{"NQ.c.0", "MNQ.c.0", "  ES.c.0  "}
	want := []string{"NQ.c.0", "MNQ.c.0", "ES.c.0"}
	got := normalizeSymbols(in)
	if !reflect.DeepEqual(got, want) {
		t.Errorf("normalizeSymbols(%v) = %v, want %v (CME futures must NOT be uppercased on save)", in, got, want)
	}
}

func TestNormalizeSymbols_Crypto_UnchangedByTask12(t *testing.T) {
	// Crypto symbols still get ToUpper as before.
	in := []string{"btcusdt", " ethusdt ", "SOLUSDT"}
	want := []string{"BTCUSDT", "ETHUSDT", "SOLUSDT"}
	got := normalizeSymbols(in)
	if !reflect.DeepEqual(got, want) {
		t.Errorf("normalizeSymbols(%v) = %v, want %v (crypto path must remain uppercased)", in, got, want)
	}
}

func TestNormalizeSymbols_Mixed(t *testing.T) {
	// Mix of CME + crypto in one list — each should follow its own path.
	in := []string{"NQ.c.0", "btcusdt", "MNQ.c.0", "ethusdt"}
	want := []string{"NQ.c.0", "BTCUSDT", "MNQ.c.0", "ETHUSDT"}
	got := normalizeSymbols(in)
	if !reflect.DeepEqual(got, want) {
		t.Errorf("normalizeSymbols(%v) = %v, want %v", in, got, want)
	}
}
