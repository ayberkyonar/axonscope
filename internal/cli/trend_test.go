package cli

import (
	"strings"
	"testing"
	"time"

	"github.com/ayberkyonar/axonscope/internal/insight"
)

func TestWriteTrendText(t *testing.T) {
	day := func(d int) time.Time { return time.Date(2026, 11, d, 0, 0, 0, 0, time.UTC) }
	buckets := []insight.Bucket{
		{Start: day(2), Total: 30},
		{Start: day(3), Total: 0},
		{Start: day(4), Total: 1},
	}

	var out strings.Builder
	if err := writeTrendText(&out, insight.Day, buckets); err != nil {
		t.Fatalf("writeTrendText returned error: %v", err)
	}

	want := "" +
		"Mon 2026-11-02  30  " + strings.Repeat("█", barWidth) + "\n" +
		"Tue 2026-11-03   0\n" +
		"Wed 2026-11-04   1  █\n" // small counts still get one block
	if out.String() != want {
		t.Errorf("output mismatch\ngot:\n%s\nwant:\n%s", out.String(), want)
	}
}

func TestPeriodLabel(t *testing.T) {
	start := time.Date(2026, 11, 2, 21, 0, 0, 0, time.UTC)
	tests := map[insight.Unit]string{
		insight.Hour: "Mon 2026-11-02 21:00",
		insight.Day:  "Mon 2026-11-02",
		insight.Week: "week of 2026-11-02",
	}
	for unit, want := range tests {
		if got := periodLabel(unit, start); got != want {
			t.Errorf("periodLabel(%v) = %q, want %q", unit, got, want)
		}
	}
}
