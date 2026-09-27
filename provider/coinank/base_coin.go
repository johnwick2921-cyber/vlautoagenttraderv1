package coinank

type SymbolResp struct {
	Symbol       string `json:"symbol"`       // symbol,such as:`BTCUSDT`
	BaseCoin     string `json:"baseCoin"`     // baseCoin from symbol,such as `BTC`
	ExchangeName string `json:"exchangeName"` // symbol source ,such as:`Binance`
	ExpireAt     int    `json:"expireAt"`
	UpdateAt     int    `json:"updateAt"`
}
