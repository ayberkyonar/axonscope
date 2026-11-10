// Package timespec parses the points in time that people and agents pass to
// axonscope: absolute dates and times, or durations meaning "that long ago".
package timespec

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Help describes the accepted formats, for flag and tool descriptions.
const Help = "a date (2026-11-05), date and time (2026-11-05T21:00), " +
	"RFC 3339 timestamp, or duration ago (90m, 24h, 7d, 2w)"

// Parse parses s as a point in time. Durations count back from now, and
// dates without a time zone are in now's time zone. An empty s returns the
// zero Time, meaning "no time given".
func Parse(s string, now time.Time) (time.Time, error) {
	if s == "" {
		return time.Time{}, nil
	}
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, nil
	}
	for _, layout := range []string{"2006-01-02T15:04", "2006-01-02 15:04", time.DateOnly} {
		if t, err := time.ParseInLocation(layout, s, now.Location()); err == nil {
			return t, nil
		}
	}
	if d, ok := parseDuration(s); ok {
		return now.Add(-d), nil
	}
	return time.Time{}, fmt.Errorf("invalid time %q: want %s", s, Help)
}

// parseDuration parses durations like "90m" and "24h", and adds days ("7d")
// and weeks ("2w"), which time.ParseDuration does not know.
func parseDuration(s string) (time.Duration, bool) {
	units := []struct {
		suffix string
		length time.Duration
	}{
		{"d", 24 * time.Hour},
		{"w", 7 * 24 * time.Hour},
	}
	for _, u := range units {
		if n, ok := strings.CutSuffix(s, u.suffix); ok {
			count, err := strconv.Atoi(n)
			if err != nil || count < 0 {
				return 0, false
			}
			return time.Duration(count) * u.length, true
		}
	}

	d, err := time.ParseDuration(s)
	if err != nil || d < 0 {
		return 0, false
	}
	return d, true
}
