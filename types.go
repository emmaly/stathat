package stathat

import "time"

// Stat represents a tracked statistic in StatHat.
type Stat struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Public  bool   `json:"public"`
	Counter bool   `json:"counter"`
}

// DataPoint is a single time-value observation in a dataset.
type DataPoint struct {
	Time  int64   `json:"time"`
	Value float64 `json:"value"`
}

// GoTime returns the DataPoint's timestamp as a time.Time.
func (dp DataPoint) GoTime() time.Time {
	return time.Unix(dp.Time, 0)
}

// Dataset is a time-series for one stat over a queried timeframe.
type Dataset struct {
	Name      string      `json:"name"`
	Timeframe string      `json:"timeframe"`
	Points    []DataPoint `json:"points"`
}

// AlertKind is the type of alert trigger.
type AlertKind string

const (
	// AlertKindValue triggers when a stat crosses a threshold.
	AlertKindValue AlertKind = "value"
	// AlertKindDelta triggers when a stat changes by a percentage.
	AlertKindDelta AlertKind = "delta"
	// AlertKindData triggers when no data is received.
	AlertKindData AlertKind = "data"
)

// TimeWindow is a valid alert time window duration.
type TimeWindow string

const (
	TimeWindow5m TimeWindow = "5m"
	TimeWindow1h TimeWindow = "1h"
	TimeWindow3h TimeWindow = "3h"
	TimeWindow1d TimeWindow = "1d"
	TimeWindow1w TimeWindow = "1w"
	TimeWindow1M TimeWindow = "1M"
	TimeWindow1y TimeWindow = "1y"
)

// Alert represents an alert configuration on a stat.
type Alert struct {
	ID         int       `json:"id"`
	StatID     string    `json:"stat_id"`
	StatName   string    `json:"stat_name"`
	Kind       AlertKind `json:"kind"`
	TimeWindow string    `json:"time_window"`
	Operator   string    `json:"operator,omitempty"`
	Threshold  float64   `json:"threshold,omitempty"`
	Percentage float64   `json:"percentage,omitempty"`
	TimeDelta  string    `json:"time_delta,omitempty"`
}

// CreateAlertParams holds parameters for creating a new alert.
type CreateAlertParams struct {
	StatID     string     // Required: stat to alert on.
	Kind       AlertKind  // Required: value, delta, or data.
	TimeWindow TimeWindow // Required: duration window.
	Operator   string     // For value alerts: "greater than" or "less than". For delta: also "different than".
	Threshold  *float64   // For value alerts: the comparison value. Pointer to distinguish unset from zero.
	Percentage *float64   // For delta alerts: the change percentage threshold. Pointer to distinguish unset from zero.
	TimeDelta  string     // For delta alerts: the comparison period (same values as TimeWindow).
}

// Report is a stat report for batch posting via the EZ API.
type Report struct {
	Stat  string
	Count *int
	Value *float64
	Time  *time.Time
}

// CountReport creates a Report for a counter stat.
func CountReport(stat string, count int) Report {
	return Report{Stat: stat, Count: &count}
}

// ValueReport creates a Report for a value stat.
func ValueReport(stat string, value float64) Report {
	return Report{Stat: stat, Value: &value}
}

// CountReportAt creates a timestamped Report for a counter stat.
func CountReportAt(stat string, count int, t time.Time) Report {
	return Report{Stat: stat, Count: &count, Time: &t}
}

// ValueReportAt creates a timestamped Report for a value stat.
func ValueReportAt(stat string, value float64, t time.Time) Report {
	return Report{Stat: stat, Value: &value, Time: &t}
}

// ezPostRequest is the JSON body for bulk EZ posting.
type ezPostRequest struct {
	EZKey string       `json:"ezkey"`
	Data  []ezPostItem `json:"data"`
}

// ezPostItem is a single report within a bulk EZ post.
type ezPostItem struct {
	Stat  string   `json:"stat"`
	Count *int     `json:"count,omitempty"`
	Value *float64 `json:"value,omitempty"`
	T     *int64   `json:"t,omitempty"`
}

// deleteResponse is used for parsing delete confirmations.
type deleteResponse struct {
	Msg string `json:"msg"`
}
