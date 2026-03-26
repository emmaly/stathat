package stathat

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

// PostCount posts a count via the EZ API.
func (c *Client) PostCount(ctx context.Context, stat string, count int) error {
	return c.postEZ(ctx, stat, url.Values{"count": {strconv.Itoa(count)}})
}

// PostCountAt posts a count with a specific timestamp via the EZ API.
func (c *Client) PostCountAt(ctx context.Context, stat string, count int, t time.Time) error {
	return c.postEZ(ctx, stat, url.Values{
		"count": {strconv.Itoa(count)},
		"t":     {strconv.FormatInt(t.Unix(), 10)},
	})
}

// PostValue posts a value via the EZ API.
func (c *Client) PostValue(ctx context.Context, stat string, value float64) error {
	return c.postEZ(ctx, stat, url.Values{"value": {strconv.FormatFloat(value, 'f', -1, 64)}})
}

// PostValueAt posts a value with a specific timestamp via the EZ API.
func (c *Client) PostValueAt(ctx context.Context, stat string, value float64, t time.Time) error {
	return c.postEZ(ctx, stat, url.Values{
		"value": {strconv.FormatFloat(value, 'f', -1, 64)},
		"t":     {strconv.FormatInt(t.Unix(), 10)},
	})
}

// PostBatch posts multiple stats in a single JSON request via the EZ API.
func (c *Client) PostBatch(ctx context.Context, reports ...Report) error {
	if len(reports) == 0 {
		return ErrEmptyBatch
	}
	if c.ezKey == "" {
		return ErrNoEZKey
	}

	items := make([]ezPostItem, len(reports))
	for i, r := range reports {
		item := ezPostItem{Stat: r.Stat, Count: r.Count, Value: r.Value}
		if r.Time != nil {
			ts := r.Time.Unix()
			item.T = &ts
		}
		items[i] = item
	}

	body, err := json.Marshal(ezPostRequest{EZKey: c.ezKey, Data: items})
	if err != nil {
		return fmt.Errorf("stathat: marshaling batch: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.postURL+"/ez", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("stathat: creating batch request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	return c.doJSON(ctx, req, nil)
}

// PostClassicCount posts a count via the Classic API.
func (c *Client) PostClassicCount(ctx context.Context, statKey string, count int) error {
	return c.postClassic(ctx, "/c", statKey, url.Values{"count": {strconv.Itoa(count)}})
}

// PostClassicCountAt posts a count with a timestamp via the Classic API.
func (c *Client) PostClassicCountAt(ctx context.Context, statKey string, count int, t time.Time) error {
	return c.postClassic(ctx, "/c", statKey, url.Values{
		"count": {strconv.Itoa(count)},
		"t":     {strconv.FormatInt(t.Unix(), 10)},
	})
}

// PostClassicValue posts a value via the Classic API.
func (c *Client) PostClassicValue(ctx context.Context, statKey string, value float64) error {
	return c.postClassic(ctx, "/v", statKey, url.Values{"value": {strconv.FormatFloat(value, 'f', -1, 64)}})
}

// PostClassicValueAt posts a value with a timestamp via the Classic API.
func (c *Client) PostClassicValueAt(ctx context.Context, statKey string, value float64, t time.Time) error {
	return c.postClassic(ctx, "/v", statKey, url.Values{
		"value": {strconv.FormatFloat(value, 'f', -1, 64)},
		"t":     {strconv.FormatInt(t.Unix(), 10)},
	})
}

// postEZ posts a single stat via EZ API form encoding.
func (c *Client) postEZ(ctx context.Context, stat string, vals url.Values) error {
	if c.ezKey == "" {
		return ErrNoEZKey
	}
	vals.Set("ezkey", c.ezKey)
	vals.Set("stat", stat)
	return c.postForm(ctx, c.postURL+"/ez", vals)
}

// postClassic posts via Classic API form encoding.
func (c *Client) postClassic(ctx context.Context, path string, statKey string, vals url.Values) error {
	if c.userKey == "" {
		return ErrNoUserKey
	}
	vals.Set("key", statKey)
	vals.Set("ukey", c.userKey)
	return c.postForm(ctx, c.postURL+path, vals)
}
