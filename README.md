# stathat

Full-featured Go client for the [StatHat](https://www.stathat.com/) API — posting, data export, alerts, and stat management.

The official StatHat Go clients only cover stat posting. This package adds the export/data API, alerts API, and stat management, with emphasis on getting data back out of StatHat.

```
go get github.com/emmaly/stathat
```

## Features

- **Posting** — EZ API (single + JSON batch) and Classic API, with optional timestamps
- **Data export** — time-series dataset retrieval with typed timeframe DSL
- **Stat management** — list (with auto-paginating iterator), lookup by name, delete
- **Alerts** — full CRUD for value, delta, and data alerts
- Zero external dependencies (standard library only)
- `context.Context` on every method
- Configurable `*http.Client` via functional options

## Usage

### Client Setup

```go
import "github.com/emmaly/stathat"

// Posting only
c := stathat.New(stathat.WithEZKey("you@example.com"))

// Export/alerts only
c := stathat.New(stathat.WithAccessToken("your-access-token"))

// Everything
c := stathat.New(
    stathat.WithEZKey("you@example.com"),
    stathat.WithAccessToken("your-access-token"),
)
```

### Post Stats

```go
// EZ API
c.PostCount(ctx, "page views", 1)
c.PostValue(ctx, "load average", 0.75)
c.PostValueAt(ctx, "temperature", 72.5, time.Now().Add(-time.Hour))

// Batch (single HTTP request)
c.PostBatch(ctx,
    stathat.CountReport("hits", 10),
    stathat.ValueReport("latency ms", 42.5),
    stathat.CountReportAt("signups", 1, yesterday),
)

// Classic API
c := stathat.New(stathat.WithUserKey("your-user-key"))
c.PostClassicCount(ctx, "stat-private-key", 1)
```

### Read Data

```go
// Single stat, 1 week at 3-hour intervals
ds, _ := c.GetStatData(ctx, "statID", stathat.NewTimeframe(1, stathat.Week, 3, stathat.Hour))
for _, p := range ds.Points {
    fmt.Printf("%s: %.2f\n", p.GoTime(), p.Value)
}

// Multiple stats
datasets, _ := c.GetData(ctx, stathat.DataQuery{
    StatIDs:   []string{"id1", "id2"},
    Timeframe: stathat.NewTimeframe(1, stathat.Month, 1, stathat.Day),
})

// Daily summary
summary := stathat.NewSummaryTimeframe(7, stathat.Day)
datasets, _ = c.GetData(ctx, stathat.DataQuery{
    StatIDs: []string{"id1"},
    Summary: &summary,
})

// Raw timeframe string
ds, _ = c.GetStatData(ctx, "statID", stathat.RawTimeframe("2M1d"))
```

### List Stats

```go
// All at once (auto-paginates)
stats, _ := c.StatList(ctx)

// Iterator (fetches pages on demand)
for s, err := range c.StatIter(ctx) {
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println(s.ID, s.Name)
}

// Lookup by name
stat, _ := c.StatInfo(ctx, "page views")

// Delete
c.DeleteStat(ctx, "statID")
```

### Alerts

```go
// List
alerts, _ := c.ListAlerts(ctx)

// Create a value alert
threshold := 100.0
alert, _ := c.CreateAlert(ctx, stathat.CreateAlertParams{
    StatID:     "statID",
    Kind:       stathat.AlertKindValue,
    TimeWindow: stathat.TimeWindow1h,
    Operator:   "greater than",
    Threshold:  &threshold,
})

// Create a delta alert
pct := 50.0
alert, _ = c.CreateAlert(ctx, stathat.CreateAlertParams{
    StatID:     "statID",
    Kind:       stathat.AlertKindDelta,
    TimeWindow: stathat.TimeWindow1d,
    Operator:   "different than",
    Percentage: &pct,
    TimeDelta:  "1d",
})

// Create a no-data alert
alert, _ = c.CreateAlert(ctx, stathat.CreateAlertParams{
    StatID:     "statID",
    Kind:       stathat.AlertKindData,
    TimeWindow: stathat.TimeWindow5m,
})

// Delete
c.DeleteAlert(ctx, alert.ID)
```

## Timeframe DSL

StatHat uses a compact timeframe syntax: `1w3h` means "1 week of data at 3-hour intervals".

```go
// Typed builder (validated)
tf := stathat.NewTimeframe(1, stathat.Week, 3, stathat.Hour) // "1w3h"

// Summary (duration only)
tf = stathat.NewSummaryTimeframe(7, stathat.Day) // "7d"

// Raw passthrough
tf = stathat.RawTimeframe("2M1d")

// Parse
parts, _ := stathat.ParseTimeframe("1w3h")
// parts.DurationN=1, parts.DurationUnit=Week, parts.IntervalN=3, parts.IntervalUnit=Hour
```

Units: `m` (minute), `h` (hour), `d` (day), `w` (week), `M` (month), `y` (year).

## Error Handling

```go
// Sentinel errors
if errors.Is(err, stathat.ErrStatNotFound) { ... }
if errors.Is(err, stathat.ErrNoAccessToken) { ... }
if errors.Is(err, stathat.ErrNoEZKey) { ... }

// API errors
var apiErr *stathat.APIError
if errors.As(err, &apiErr) {
    fmt.Println(apiErr.StatusCode, apiErr.Message)
}
```

## License

MIT
