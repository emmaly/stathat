package stathat

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// ListAlerts returns all configured alerts.
func (c *Client) ListAlerts(ctx context.Context) ([]Alert, error) {
	u, err := c.exportPath("/alerts")
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, fmt.Errorf("stathat: creating list alerts request: %w", err)
	}

	var alerts []Alert
	if err := c.doJSON(ctx, req, &alerts); err != nil {
		return nil, fmt.Errorf("stathat: listing alerts: %w", err)
	}
	return alerts, nil
}

// GetAlert returns a single alert by ID.
// Returns ErrAlertNotFound if the alert does not exist.
func (c *Client) GetAlert(ctx context.Context, alertID int) (Alert, error) {
	u, err := c.exportPath("/alerts/" + strconv.Itoa(alertID))
	if err != nil {
		return Alert{}, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return Alert{}, fmt.Errorf("stathat: creating get alert request: %w", err)
	}

	var alert Alert
	if err := c.doJSON(ctx, req, &alert); err != nil {
		var apiErr *APIError
		if errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusNotFound {
			return Alert{}, ErrAlertNotFound
		}
		return Alert{}, fmt.Errorf("stathat: getting alert: %w", err)
	}
	return alert, nil
}

// CreateAlert creates a new alert and returns it.
func (c *Client) CreateAlert(ctx context.Context, params CreateAlertParams) (Alert, error) {
	u, err := c.exportPath("/alerts")
	if err != nil {
		return Alert{}, err
	}

	vals := url.Values{
		"stat_id":     {params.StatID},
		"kind":        {string(params.Kind)},
		"time_window": {string(params.TimeWindow)},
	}
	if params.Operator != "" {
		vals.Set("operator", params.Operator)
	}
	if params.Kind == AlertKindValue && params.Threshold != nil {
		vals.Set("threshold", strconv.FormatFloat(*params.Threshold, 'f', -1, 64))
	}
	if params.Kind == AlertKindDelta {
		if params.Percentage != nil {
			vals.Set("percentage", strconv.FormatFloat(*params.Percentage, 'f', -1, 64))
		}
		if params.TimeDelta != "" {
			vals.Set("time_delta", params.TimeDelta)
		}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, strings.NewReader(vals.Encode()))
	if err != nil {
		return Alert{}, fmt.Errorf("stathat: creating alert request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	var alert Alert
	if err := c.doJSON(ctx, req, &alert); err != nil {
		return Alert{}, fmt.Errorf("stathat: creating alert: %w", err)
	}
	return alert, nil
}

// DeleteAlert deletes an alert by ID.
func (c *Client) DeleteAlert(ctx context.Context, alertID int) error {
	u, err := c.exportPath("/alerts/" + strconv.Itoa(alertID))
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, u, nil)
	if err != nil {
		return fmt.Errorf("stathat: creating delete alert request: %w", err)
	}

	return c.doJSON(ctx, req, nil)
}
