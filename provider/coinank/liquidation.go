package coinank

type LiquidationOrdersResponse struct {
	ExchangeName  string  `json:"exchangeName"`
	BaseCoin      string  `json:"baseCoin"`
	ContractCode  string  `json:"contractCode"` //contract code
	PosSide       string  `json:"posSide"`      // `long`: long ,`short`:short
	Amount        float64 `json:"amount"`       //liquidation amount
	Price         float64 `json:"price"`        //liquidation price
	AvgPrice      float64 `json:"avgPrice"`
	TradeTurnover float64 `json:"tradeTurnover"` // liquidation turnover
	Ts            int64   `json:"ts"`
}

type LiquidationSymbol struct {
	Symbol        string  `json:"symbol"`
	ExchangeName  string  `json:"exchangeName"`
	Ts            int64   `json:"ts"`            // timestamp
	LongTurnover  float64 `json:"longTurnover"`  //long turnover
	ShortTurnover float64 `json:"shortTurnover"` //short turnover
	ShortAmount   float64 `json:"shortAmount"`   //short amount
	LongAmount    float64 `json:"longAmount"`    // long amount
}

type LiquidationStatistic struct {
	All struct {
		LongTurnover  float64 `json:"longTurnover"`
		ShortTurnover float64 `json:"shortTurnover"`
		ShortAmount   float64 `json:"shortAmount"`
		LongAmount    float64 `json:"longAmount"`
	} `json:"all"` //coin liquidation aggregated with all exchanges
	Ts int64 `json:"ts"` // timestamp
}

type LiquidationExchangeStatisticsResponse struct {
	TopOrder struct {
		Symbol        string  `json:"symbol"`
		PosSide       string  `json:"posSide"`       //side
		ExchangeName  string  `json:"exchangeName"`  //exchangeName
		TradeTurnover float64 `json:"tradeTurnover"` //turnover
		BaseCoin      string  `json:"baseCoin"`
		Ts            int64   `json:"ts"`
	} `json:"topOrder"` // 24 hour liquidation top order
	Total int      `json:"total"` // 24 hour total liquidation number
	Two4H struct { // 24 hour liquidation data
		BaseCoin      string  `json:"baseCoin"`
		TotalTurnover float64 `json:"totalTurnover"`
		LongTurnover  float64 `json:"longTurnover"`
		ShortTurnover float64 `json:"shortTurnover"`
		Percentage    float64 `json:"percentage"`
		LongRatio     float64 `json:"longRatio"`
		ShortRatio    float64 `json:"shortRatio"`
		Interval      string  `json:"interval"`
	} `json:"24h"`
	OneH struct { // 1 hour liquidation data
		BaseCoin      string  `json:"baseCoin"`
		TotalTurnover float64 `json:"totalTurnover"`
		LongTurnover  float64 `json:"longTurnover"`
		ShortTurnover float64 `json:"shortTurnover"`
		Percentage    float64 `json:"percentage"`
		LongRatio     float64 `json:"longRatio"`
		ShortRatio    float64 `json:"shortRatio"`
		Interval      string  `json:"interval"`
	} `json:"1h"`
}
