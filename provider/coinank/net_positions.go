package coinank

type NetPositionsResponse struct {
	Begin          int64  `json:"begin"` // begin timestamp
	Interval       string `json:"interval"`
	NetLongsHigh   int    `json:"netLongsHigh"`   // net long high
	NetLongsClose  int    `json:"netLongsClose"`  // net long close
	NetLongsLow    int    `json:"netLongsLow"`    // net long close
	NetShortsClose int    `json:"netShortsClose"` // net short close
	NetShortsHigh  int    `json:"netShortsHigh"`  // net short high
	NetShortsLow   int    `json:"netShortsLow"`   // net short low
}
