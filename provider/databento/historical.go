package databento

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"time"
)

// Bar represents one OHLCV record returned by Databento.
type Bar struct {
	Timestamp time.Time
	Open      float64
	High      float64
	Low       float64
	Close     float64
	Volume    float64
}

// rawBar mirrors the JSON shape Databento returns for ohlcv-* schemas.
// Numeric fields arrive as integer fixed-point with 1e9 divisor.
// ts_event and rtype are nested in the "hd" record header — they are NOT
// at the top level. (A prior parser version assumed top-level ts_event;
// the fabricated unit-test fixture matched that assumption and shipped a
// silent bug, caught by the live smoke test on 2026-05-25.)
type rawBar struct {
	Hd struct {
		TsEvent string `json:"ts_event"`
		Rtype   int    `json:"rtype"`
	} `json:"hd"`
	Open   string `json:"open"`
	High   string `json:"high"`
	Low    string `json:"low"`
	Close  string `json:"close"`
	Volume string `json:"volume"`
}

// GetOHLCV fetches OHLCV bars for one symbol over [start, end).
// interval must be one of "1m", "1h", "1d" — maps to schema "ohlcv-<interval>".
// symbol can be a continuous code like "NQ.c.0" or a specific contract like "NQM6".
func (c *Client) GetOHLCV(symbol, interval string, start, end time.Time) ([]Bar, error) {
	schema := "ohlcv-" + interval
	params := url.Values{}
	params.Set("dataset", DefaultDataset)
	params.Set("symbols", symbol)
	params.Set("schema", schema)
	params.Set("stype_in", "continuous") // NQ.c.0 is a continuous symbol
	params.Set("start", start.UTC().Format(time.RFC3339))
	params.Set("end", end.UTC().Format(time.RFC3339))
	params.Set("encoding", "json")

	body, err := c.doRequest("/timeseries.get_range", params)
	if err != nil {
		return nil, err
	}
	return parseOHLCVResponse(body)
}

// parseOHLCVResponse decodes Databento's JSON-lines body into []Bar.
func parseOHLCVResponse(body []byte) ([]Bar, error) {
	var bars []Bar
	sc := bufio.NewScanner(bytes.NewReader(body))
	sc.Buffer(make([]byte, 1024*1024), 8*1024*1024) // tolerate long lines
	for sc.Scan() {
		line := sc.Bytes()
		if len(line) == 0 {
			continue
		}
		var r rawBar
		if err := json.Unmarshal(line, &r); err != nil {
			return nil, fmt.Errorf("databento: parse bar: %w", err)
		}
		bar, err := r.toBar()
		if err != nil {
			return nil, err
		}
		bars = append(bars, bar)
	}
	if err := sc.Err(); err != nil {
		return nil, fmt.Errorf("databento: scan body: %w", err)
	}
	return bars, nil
}

func (r rawBar) toBar() (Bar, error) {
	if r.Hd.Rtype != 33 {
		return Bar{}, fmt.Errorf("unexpected rtype %d (want 33 for ohlcv-1m)", r.Hd.Rtype)
	}
	tsNs, err := strconv.ParseInt(r.Hd.TsEvent, 10, 64)
	if err != nil {
		return Bar{}, fmt.Errorf("databento: parse ts_event %q: %w", r.Hd.TsEvent, err)
	}
	open, err := scaledFloat(r.Open)
	if err != nil {
		return Bar{}, err
	}
	high, err := scaledFloat(r.High)
	if err != nil {
		return Bar{}, err
	}
	low, err := scaledFloat(r.Low)
	if err != nil {
		return Bar{}, err
	}
	closeP, err := scaledFloat(r.Close)
	if err != nil {
		return Bar{}, err
	}
	vol, err := strconv.ParseFloat(r.Volume, 64)
	if err != nil {
		return Bar{}, fmt.Errorf("databento: parse volume %q: %w", r.Volume, err)
	}
	return Bar{
		Timestamp: time.Unix(0, tsNs).UTC(),
		Open:      open,
		High:      high,
		Low:       low,
		Close:     closeP,
		Volume:    vol,
	}, nil
}

// Databento ohlcv schemas use integer fixed-point with 1e9 divisor.
func scaledFloat(s string) (float64, error) {
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("databento: parse scaled int %q: %w", s, err)
	}
	return float64(n) / 1e9, nil
}
