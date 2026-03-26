package stathat

import (
	"context"
	"errors"
	"fmt"
	"iter"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const pageSize = 10000

// StatList returns all stats, automatically paginating through results.
func (c *Client) StatList(ctx context.Context) ([]Stat, error) {
	var all []Stat
	for s, err := range c.StatIter(ctx) {
		if err != nil {
			return all, err
		}
		all = append(all, s)
	}
	return all, nil
}

// StatIter returns an iterator over all stats, fetching pages on demand.
// On error the iterator yields a zero Stat and the error, then stops.
func (c *Client) StatIter(ctx context.Context) iter.Seq2[Stat, error] {
	return func(yield func(Stat, error) bool) {
		offset := 0
		for {
			page, err := c.statListPage(ctx, offset)
			if err != nil {
				yield(Stat{}, err)
				return
			}
			for _, s := range page {
				if !yield(s, nil) {
					return
				}
			}
			if len(page) < pageSize {
				return
			}
			offset += len(page)
		}
	}
}

// StatInfo returns stat details by name.
// Returns ErrStatNotFound if the stat does not exist.
func (c *Client) StatInfo(ctx context.Context, name string) (Stat, error) {
	base, err := c.exportPath("/stat")
	if err != nil {
		return Stat{}, err
	}

	u := base + "?" + url.Values{"name": {name}}.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return Stat{}, fmt.Errorf("stathat: creating stat info request: %w", err)
	}

	var stat Stat
	if err := c.doJSON(ctx, req, &stat); err != nil {
		var apiErr *APIError
		if errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusNotFound {
			return Stat{}, ErrStatNotFound
		}
		return Stat{}, fmt.Errorf("stathat: getting stat info: %w", err)
	}
	return stat, nil
}

// DeleteStat deletes a stat by ID.
func (c *Client) DeleteStat(ctx context.Context, statID string) error {
	u, err := c.exportPath("/stats/" + url.PathEscape(statID))
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, u, nil)
	if err != nil {
		return fmt.Errorf("stathat: creating delete request: %w", err)
	}

	return c.doJSON(ctx, req, nil)
}

// DataQuery configures a data retrieval request.
type DataQuery struct {
	// StatIDs is the list of stat IDs to retrieve data for.
	StatIDs []string
	// Timeframe specifies the duration and interval (e.g. "1w3h").
	Timeframe Timeframe
	// Start optionally pins the dataset to a specific start time.
	Start *time.Time
	// Summary optionally requests daily summary data (e.g. "7d", "2M").
	Summary *Timeframe
}

// GetData retrieves time-series datasets for the given query.
func (c *Client) GetData(ctx context.Context, q DataQuery) ([]Dataset, error) {
	if len(q.StatIDs) == 0 {
		return nil, fmt.Errorf("stathat: GetData requires at least one stat ID")
	}

	path := "/data/" + strings.Join(q.StatIDs, "/")
	base, err := c.exportPath(path)
	if err != nil {
		return nil, err
	}

	params := url.Values{}
	if !q.Timeframe.IsZero() {
		params.Set("t", q.Timeframe.String())
	}
	if q.Start != nil {
		params.Set("start", strconv.FormatInt(q.Start.Unix(), 10))
	}
	if q.Summary != nil && !q.Summary.IsZero() {
		params.Set("summary", q.Summary.String())
	}

	u := base
	if len(params) > 0 {
		u += "?" + params.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, fmt.Errorf("stathat: creating data request: %w", err)
	}

	var datasets []Dataset
	if err := c.doJSON(ctx, req, &datasets); err != nil {
		return nil, fmt.Errorf("stathat: getting data: %w", err)
	}
	return datasets, nil
}

// GetStatData is a convenience method for retrieving data for a single stat.
func (c *Client) GetStatData(ctx context.Context, statID string, tf Timeframe) (Dataset, error) {
	datasets, err := c.GetData(ctx, DataQuery{
		StatIDs:   []string{statID},
		Timeframe: tf,
	})
	if err != nil {
		return Dataset{}, err
	}
	if len(datasets) == 0 {
		return Dataset{}, fmt.Errorf("stathat: no dataset returned for stat %s", statID)
	}
	return datasets[0], nil
}

// statListPage fetches a single page of stats at the given offset.
func (c *Client) statListPage(ctx context.Context, offset int) ([]Stat, error) {
	base, err := c.exportPath("/statlist")
	if err != nil {
		return nil, err
	}

	u := base
	if offset > 0 {
		u += "?" + url.Values{"offset": {strconv.Itoa(offset)}}.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, fmt.Errorf("stathat: creating stat list request: %w", err)
	}

	var stats []Stat
	if err := c.doJSON(ctx, req, &stats); err != nil {
		return nil, fmt.Errorf("stathat: listing stats: %w", err)
	}
	return stats, nil
}
