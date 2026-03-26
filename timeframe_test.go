package stathat

import (
	"testing"
)

func TestNewTimeframe(t *testing.T) {
	tests := []struct {
		name     string
		tf       Timeframe
		expected string
	}{
		{"week at 3h", NewTimeframe(1, Week, 3, Hour), "1w3h"},
		{"month at 1d", NewTimeframe(2, Month, 1, Day), "2M1d"},
		{"year at 1w", NewTimeframe(1, Year, 1, Week), "1y1w"},
		{"day at 5m", NewTimeframe(1, Day, 5, Minute), "1d5m"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.tf.String(); got != tt.expected {
				t.Errorf("got %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestNewSummaryTimeframe(t *testing.T) {
	tests := []struct {
		name     string
		tf       Timeframe
		expected string
	}{
		{"7 days", NewSummaryTimeframe(7, Day), "7d"},
		{"2 months", NewSummaryTimeframe(2, Month), "2M"},
		{"1 year", NewSummaryTimeframe(1, Year), "1y"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.tf.String(); got != tt.expected {
				t.Errorf("got %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestRawTimeframe(t *testing.T) {
	tf := RawTimeframe("1w3h")
	if tf.String() != "1w3h" {
		t.Errorf("got %q, want %q", tf.String(), "1w3h")
	}
}

func TestTimeframeIsZero(t *testing.T) {
	if !(Timeframe{}).IsZero() {
		t.Error("empty Timeframe should be zero")
	}
	if RawTimeframe("1d").IsZero() {
		t.Error("non-empty Timeframe should not be zero")
	}
}

func TestParseTimeframe(t *testing.T) {
	tests := []struct {
		input   string
		durN    int
		durUnit Unit
		intN    int
		intUnit Unit
		wantErr bool
	}{
		{"1w3h", 1, Week, 3, Hour, false},
		{"2M1d", 2, Month, 1, Day, false},
		{"7d", 7, Day, 0, 0, false},
		{"1y1w", 1, Year, 1, Week, false},
		{"30m", 30, Minute, 0, 0, false},
		{"", 0, 0, 0, 0, true},
		{"abc", 0, 0, 0, 0, true},
		{"1x", 0, 0, 0, 0, true},
		{"1w3x", 0, 0, 0, 0, true},
		{"1w3h5m", 0, 0, 0, 0, true}, // trailing text
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			parts, err := ParseTimeframe(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("err=%v, wantErr=%v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if parts.DurationN != tt.durN || parts.DurationUnit != tt.durUnit {
				t.Errorf("duration: got %d%c, want %d%c", parts.DurationN, parts.DurationUnit, tt.durN, tt.durUnit)
			}
			if parts.IntervalN != tt.intN || parts.IntervalUnit != tt.intUnit {
				t.Errorf("interval: got %d%c, want %d%c", parts.IntervalN, parts.IntervalUnit, tt.intN, tt.intUnit)
			}
		})
	}
}

func TestParseTimeframeRoundTrip(t *testing.T) {
	tf := NewTimeframe(1, Week, 3, Hour)
	parts, err := ParseTimeframe(tf.String())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if parts.DurationN != 1 || parts.DurationUnit != Week || parts.IntervalN != 3 || parts.IntervalUnit != Hour {
		t.Errorf("round-trip failed: %+v", parts)
	}
}

func TestMonthVsMinuteDistinction(t *testing.T) {
	// M = Month, m = Minute — case-sensitive
	partsMonth, err := ParseTimeframe("2M")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if partsMonth.DurationUnit != Month {
		t.Errorf("expected Month, got %c", partsMonth.DurationUnit)
	}

	partsMinute, err := ParseTimeframe("30m")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if partsMinute.DurationUnit != Minute {
		t.Errorf("expected Minute, got %c", partsMinute.DurationUnit)
	}
}
