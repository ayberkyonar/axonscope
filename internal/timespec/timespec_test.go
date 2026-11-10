package timespec

import (
	"testing"
	"time"
)

func TestParse(t *testing.T) {
	cet := time.FixedZone("CET", 3600)
	now := time.Date(2026, 11, 10, 12, 0, 0, 0, cet)

	tests := []struct {
		in   string
		want time.Time
	}{
		{"", time.Time{}},
		{"2026-11-05T20:08:36Z", time.Date(2026, 11, 5, 20, 8, 36, 0, time.UTC)},
		{"2026-11-05T21:00", time.Date(2026, 11, 5, 21, 0, 0, 0, cet)},
		{"2026-11-05 21:00", time.Date(2026, 11, 5, 21, 0, 0, 0, cet)},
		{"2026-11-05", time.Date(2026, 11, 5, 0, 0, 0, 0, cet)},
		{"90m", now.Add(-90 * time.Minute)},
		{"24h", now.Add(-24 * time.Hour)},
		{"7d", now.AddDate(0, 0, -7)},
		{"2w", now.AddDate(0, 0, -14)},
	}
	for _, tt := range tests {
		got, err := Parse(tt.in, now)
		if err != nil {
			t.Errorf("Parse(%q) returned error: %v", tt.in, err)
		} else if !got.Equal(tt.want) {
			t.Errorf("Parse(%q) = %v, want %v", tt.in, got, tt.want)
		}
	}
}

func TestParseRejectsInvalid(t *testing.T) {
	for _, in := range []string{"yesterday", "7", "-1h", "-3d", "d", "2026-13-01", "7days"} {
		if _, err := Parse(in, time.Now()); err == nil {
			t.Errorf("Parse(%q) returned no error", in)
		}
	}
}
