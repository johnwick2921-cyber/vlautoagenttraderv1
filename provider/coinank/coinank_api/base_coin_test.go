package coinank_api

import (
	"encoding/json"
	"testing"
)

func TestBaseCoinSymbolsNoArgs(t *testing.T) {
	resp, err := BaseCoinSymbols(liveNetworkGate(t), "", "", "")
	if err != nil {
		t.Error(err)
	}
	res, err := json.Marshal(resp)
	if err != nil {
		t.Error(err)
	}
	t.Logf("%s", res)
}

func TestBaseCoinSymbolsBTC(t *testing.T) {
	resp, err := BaseCoinSymbols(liveNetworkGate(t), "", "", "BTC")
	if err != nil {
		t.Error(err)
	}
	res, err := json.Marshal(resp)
	if err != nil {
		t.Error(err)
	}
	t.Logf("%s", res)
}

func TestBaseCoinSymbolsBTCUSDT(t *testing.T) {
	resp, err := BaseCoinSymbols(liveNetworkGate(t), "", "BTCUSDT", "")
	if err != nil {
		t.Error(err)
	}
	res, err := json.Marshal(resp)
	if err != nil {
		t.Error(err)
	}
	t.Logf("%s", res)
}
