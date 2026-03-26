package stathat

import (
	"fmt"
	"strconv"
	"unicode"
)

// Unit represents a time unit in the StatHat timeframe syntax.
type Unit byte

const (
	Minute Unit = 'm'
	Hour   Unit = 'h'
	Day    Unit = 'd'
	Week   Unit = 'w'
	Month  Unit = 'M'
	Year   Unit = 'y'
)

// String returns the single-character representation of the unit.
func (u Unit) String() string {
	return string(u)
}

// validUnit reports whether b is a valid unit character.
func validUnit(b byte) bool {
	switch Unit(b) {
	case Minute, Hour, Day, Week, Month, Year:
		return true
	}
	return false
}

// Timeframe represents a StatHat timeframe specification.
//
// For data queries it combines a duration and interval: "1w3h" means
// "1 week of data at 3-hour intervals".
//
// For summaries it is a duration only: "7d" means "7-day daily summary".
type Timeframe struct {
	raw string
}

// NewTimeframe creates a data query timeframe with explicit duration and interval.
// Example: NewTimeframe(1, Week, 3, Hour) produces "1w3h".
func NewTimeframe(durationN int, durationUnit Unit, intervalN int, intervalUnit Unit) Timeframe {
	return Timeframe{
		raw: fmt.Sprintf("%d%c%d%c", durationN, durationUnit, intervalN, intervalUnit),
	}
}

// NewSummaryTimeframe creates a summary timeframe with duration only.
// Example: NewSummaryTimeframe(7, Day) produces "7d".
func NewSummaryTimeframe(n int, unit Unit) Timeframe {
	return Timeframe{
		raw: fmt.Sprintf("%d%c", n, unit),
	}
}

// RawTimeframe wraps a pre-formatted timeframe string without validation.
func RawTimeframe(s string) Timeframe {
	return Timeframe{raw: s}
}

// String returns the timeframe string representation.
func (tf Timeframe) String() string {
	return tf.raw
}

// IsZero reports whether the timeframe is unset.
func (tf Timeframe) IsZero() bool {
	return tf.raw == ""
}

// TimeframeParts holds the parsed components of a timeframe string.
type TimeframeParts struct {
	DurationN    int
	DurationUnit Unit
	IntervalN    int  // 0 if summary-only
	IntervalUnit Unit // 0 if summary-only
}

// ParseTimeframe parses a timeframe string like "1w3h" into its components.
// For summary-only timeframes like "7d", IntervalN and IntervalUnit are zero.
func ParseTimeframe(s string) (TimeframeParts, error) {
	if s == "" {
		return TimeframeParts{}, fmt.Errorf("stathat: empty timeframe string")
	}

	n, unit, rest, err := parseSegment(s)
	if err != nil {
		return TimeframeParts{}, fmt.Errorf("stathat: parsing timeframe %q duration: %w", s, err)
	}

	parts := TimeframeParts{
		DurationN:    n,
		DurationUnit: Unit(unit),
	}

	if rest == "" {
		return parts, nil
	}

	n2, unit2, rest2, err := parseSegment(rest)
	if err != nil {
		return TimeframeParts{}, fmt.Errorf("stathat: parsing timeframe %q interval: %w", s, err)
	}
	if rest2 != "" {
		return TimeframeParts{}, fmt.Errorf("stathat: unexpected trailing text in timeframe %q: %q", s, rest2)
	}

	parts.IntervalN = n2
	parts.IntervalUnit = Unit(unit2)
	return parts, nil
}

// parseSegment parses a leading "N<unit>" from s, returning the number, unit byte,
// and remaining string.
func parseSegment(s string) (int, byte, string, error) {
	i := 0
	for i < len(s) && unicode.IsDigit(rune(s[i])) {
		i++
	}
	if i == 0 {
		return 0, 0, "", fmt.Errorf("expected digits, got %q", s)
	}
	if i >= len(s) {
		return 0, 0, "", fmt.Errorf("expected unit after %q", s[:i])
	}

	n, err := strconv.Atoi(s[:i])
	if err != nil {
		return 0, 0, "", fmt.Errorf("invalid number %q: %w", s[:i], err)
	}

	unit := s[i]
	if !validUnit(unit) {
		return 0, 0, "", fmt.Errorf("invalid unit %q (valid: m, h, d, w, M, y)", string(unit))
	}

	return n, unit, s[i+1:], nil
}
