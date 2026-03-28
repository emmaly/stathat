// Command stathat-export dumps a full export of a StatHat account to text files.
//
// Usage:
//
//	stathat-export -token ACCESS_TOKEN [-out DIR] [-timeframe TF] [-workers N]
//
// Or set STATHAT_ACCESS_TOKEN in the environment.
//
// For each stat, it writes a CSV file into the output directory containing
// the time-series data. It also writes a stats.csv index of all stats and
// an alerts.csv of all configured alerts.
package main

import (
	"context"
	"encoding/csv"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/emmaly/stathat"
)

func main() {
	token := flag.String("token", os.Getenv("STATHAT_ACCESS_TOKEN"), "StatHat access token (or set STATHAT_ACCESS_TOKEN)")
	outDir := flag.String("out", "stathat-export", "output directory")
	timeframe := flag.String("timeframe", "1y1d", "timeframe for data export (e.g. 1y1d, 1M1h)")
	workers := flag.Int("workers", 5, "concurrent data fetch workers")
	flag.Parse()

	if *token == "" {
		fmt.Fprintln(os.Stderr, "error: access token required (-token or STATHAT_ACCESS_TOKEN)")
		flag.Usage()
		os.Exit(1)
	}

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))

	if err := run(context.Background(), logger, *token, *outDir, *timeframe, *workers); err != nil {
		logger.Error("export failed", "error", err)
		os.Exit(1)
	}
}

// run performs the full export.
func run(ctx context.Context, logger *slog.Logger, token, outDir, tf string, workers int) error {
	c := stathat.New(
		stathat.WithAccessToken(token),
		stathat.WithLogger(logger),
	)

	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return fmt.Errorf("creating output directory: %w", err)
	}

	// Export stats list
	logger.Info("fetching stat list")
	stats, err := c.StatList(ctx)
	if err != nil {
		return fmt.Errorf("listing stats: %w", err)
	}
	logger.Info("found stats", "count", len(stats))

	if err := writeStatsIndex(filepath.Join(outDir, "stats.csv"), stats); err != nil {
		return fmt.Errorf("writing stats index: %w", err)
	}

	// Export alerts
	logger.Info("fetching alerts")
	alerts, err := c.ListAlerts(ctx)
	if err != nil {
		return fmt.Errorf("listing alerts: %w", err)
	}
	logger.Info("found alerts", "count", len(alerts))

	if err := writeAlerts(filepath.Join(outDir, "alerts.csv"), alerts); err != nil {
		return fmt.Errorf("writing alerts: %w", err)
	}

	// Export data for each stat
	dataDir := filepath.Join(outDir, "data")
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return fmt.Errorf("creating data directory: %w", err)
	}

	timeframeVal := stathat.RawTimeframe(tf)

	var (
		wg      sync.WaitGroup
		sem     = make(chan struct{}, workers)
		mu      sync.Mutex
		errList []error
	)

	for i, s := range stats {
		wg.Add(1)
		go func(idx int, stat stathat.Stat) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			logger.Info("exporting stat", "name", stat.Name, "progress", fmt.Sprintf("%d/%d", idx+1, len(stats)))

			ds, err := c.GetStatData(ctx, stat.ID, timeframeVal)
			if err != nil {
				mu.Lock()
				errList = append(errList, fmt.Errorf("stat %q (%s): %w", stat.Name, stat.ID, err))
				mu.Unlock()
				logger.Warn("failed to export stat", "name", stat.Name, "error", err)
				return
			}

			filename := sanitizeFilename(stat.Name) + ".csv"
			path := filepath.Join(dataDir, filename)
			if err := writeDataset(path, stat, ds); err != nil {
				mu.Lock()
				errList = append(errList, fmt.Errorf("writing %q: %w", stat.Name, err))
				mu.Unlock()
				logger.Warn("failed to write stat data", "name", stat.Name, "error", err)
			}
		}(i, s)
	}

	wg.Wait()

	if len(errList) > 0 {
		logger.Warn("export completed with errors", "errors", len(errList), "total", len(stats))
		for _, e := range errList {
			logger.Warn("  error", "detail", e)
		}
	} else {
		logger.Info("export complete", "stats", len(stats), "alerts", len(alerts), "output", outDir)
	}

	return nil
}

// writeStatsIndex writes the stats index CSV.
func writeStatsIndex(path string, stats []stathat.Stat) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	w := csv.NewWriter(f)
	defer w.Flush()

	w.Write([]string{"id", "name", "type", "public"})
	for _, s := range stats {
		typ := "value"
		if s.Counter {
			typ = "counter"
		}
		pub := "false"
		if s.Public {
			pub = "true"
		}
		w.Write([]string{s.ID, s.Name, typ, pub})
	}
	return w.Error()
}

// writeAlerts writes the alerts CSV.
func writeAlerts(path string, alerts []stathat.Alert) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	w := csv.NewWriter(f)
	defer w.Flush()

	w.Write([]string{"id", "stat_id", "stat_name", "kind", "time_window", "operator", "threshold", "percentage", "time_delta"})
	for _, a := range alerts {
		w.Write([]string{
			fmt.Sprintf("%d", a.ID),
			a.StatID,
			a.StatName,
			string(a.Kind),
			a.TimeWindow,
			a.Operator,
			floatOrEmpty(a.Threshold),
			floatOrEmpty(a.Percentage),
			a.TimeDelta,
		})
	}
	return w.Error()
}

// writeDataset writes a single stat's time-series data as CSV.
func writeDataset(path string, stat stathat.Stat, ds stathat.Dataset) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	w := csv.NewWriter(f)
	defer w.Flush()

	// Header comment line with stat metadata
	w.Write([]string{"# stat_id: " + stat.ID})
	w.Write([]string{"# stat_name: " + stat.Name})
	typ := "value"
	if stat.Counter {
		typ = "counter"
	}
	w.Write([]string{"# stat_type: " + typ})
	w.Write([]string{"# timeframe: " + ds.Timeframe})

	w.Write([]string{"timestamp", "datetime", "value"})
	for _, p := range ds.Points {
		w.Write([]string{
			fmt.Sprintf("%d", p.Time),
			p.GoTime().UTC().Format(time.RFC3339),
			fmt.Sprintf("%g", p.Value),
		})
	}
	return w.Error()
}

// sanitizeFilename replaces characters unsafe for filenames.
func sanitizeFilename(name string) string {
	r := strings.NewReplacer(
		"/", "_",
		"\\", "_",
		":", "_",
		"*", "_",
		"?", "_",
		"\"", "_",
		"<", "_",
		">", "_",
		"|", "_",
		" ", "_",
	)
	s := r.Replace(name)
	// Collapse repeated underscores
	for strings.Contains(s, "__") {
		s = strings.ReplaceAll(s, "__", "_")
	}
	return strings.Trim(s, "_")
}

// floatOrEmpty formats a float64 or returns empty string for zero.
func floatOrEmpty(v float64) string {
	if v == 0 {
		return ""
	}
	return fmt.Sprintf("%g", v)
}
