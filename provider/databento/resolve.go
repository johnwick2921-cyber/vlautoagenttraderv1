package databento

import (
	"encoding/json"
	"fmt"
	"net/url"
	"time"
)

// ResolveContinuous returns the specific contract symbol that a continuous
// symbol (e.g. "NQ.c.0") points to today. For non-continuous symbols this
// is a passthrough.
func (c *Client) ResolveContinuous(symbol string) (string, error) {
	today := time.Now().UTC().Format("2006-01-02")
	params := url.Values{}
	params.Set("dataset", DefaultDataset)
	params.Set("symbols", symbol)
	params.Set("stype_in", "continuous")
	params.Set("stype_out", "raw_symbol")
	params.Set("start_date", today)
	params.Set("end_date", today)

	body, err := c.doRequest("/symbology.resolve", params)
	if err != nil {
		return "", err
	}
	return parseResolveResponse(body, symbol)
}

type resolveResponse struct {
	Result   map[string][]resolveEntry `json:"result"`
	NotFound []string                  `json:"not_found"`
}

type resolveEntry struct {
	D0 string `json:"d0"`
	D1 string `json:"d1"`
	S  string `json:"s"`
}

func parseResolveResponse(body []byte, symbol string) (string, error) {
	var resp resolveResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return "", fmt.Errorf("databento resolve: parse: %w", err)
	}
	for _, nf := range resp.NotFound {
		if nf == symbol {
			return "", fmt.Errorf("databento resolve: symbol not found: %s", symbol)
		}
	}
	entries, ok := resp.Result[symbol]
	if !ok || len(entries) == 0 {
		return "", fmt.Errorf("databento resolve: no entries for %s", symbol)
	}
	return entries[0].S, nil
}
