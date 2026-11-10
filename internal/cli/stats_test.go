package cli

import (
	"strings"
	"testing"
	"time"

	"github.com/ayberkyonar/axonscope/internal/insight"
)

func TestWriteStatsText(t *testing.T) {
	stats := insight.Stats{
		Total: 112,
		First: time.Date(2026, 11, 5, 20, 0, 0, 0, time.Local),
		Last:  time.Date(2026, 11, 5, 22, 30, 0, 0, time.Local),
		ByName: []insight.NameCount{
			{Name: "com.example.LoginSucceeded", Count: 100},
			{Name: "com.example.LoggedOut", Count: 12},
		},
	}

	var out strings.Builder
	if err := writeStatsText(&out, stats); err != nil {
		t.Fatalf("writeStatsText returned error: %v", err)
	}

	want := "" +
		"112 events from 2026-11-05 20:00:00 to 2026-11-05 22:30:00\n" +
		"\n" +
		"100  LoginSucceeded\n" +
		" 12  LoggedOut\n"
	if out.String() != want {
		t.Errorf("output mismatch\ngot:\n%s\nwant:\n%s", out.String(), want)
	}
}

func TestWriteStatsTextWithoutEvents(t *testing.T) {
	var out strings.Builder
	if err := writeStatsText(&out, insight.Stats{}); err != nil {
		t.Fatalf("writeStatsText returned error: %v", err)
	}
	if out.String() != "no events\n" {
		t.Errorf("got %q, want %q", out.String(), "no events\n")
	}
}

func TestQueryFlags(t *testing.T) {
	now := time.Date(2026, 11, 10, 12, 0, 0, 0, time.UTC)
	f := queryFlags{
		from:  3,
		names: []string{"LoggedOut"},
		tags:  []string{"userId=42"},
		since: "7d",
		until: "2026-11-09T00:00:00Z",
	}

	q, err := f.query(now)
	if err != nil {
		t.Fatalf("query returned error: %v", err)
	}

	if q.From != 3 || len(q.Names) != 1 || len(q.Tags) != 1 || q.Tags[0].Key != "userId" {
		t.Errorf("filters not passed on: %+v", q)
	}
	if want := now.AddDate(0, 0, -7); !q.Since.Equal(want) {
		t.Errorf("Since = %v, want %v", q.Since, want)
	}
	if want := time.Date(2026, 11, 9, 0, 0, 0, 0, time.UTC); !q.Until.Equal(want) {
		t.Errorf("Until = %v, want %v", q.Until, want)
	}
}

func TestQueryFlagsNamesTheBadFlag(t *testing.T) {
	_, err := (&queryFlags{until: "soon"}).query(time.Now())
	if err == nil || !strings.HasPrefix(err.Error(), "--until:") {
		t.Errorf("got error %v, want one starting with --until:", err)
	}
}

func TestWriteTagStatsText(t *testing.T) {
	at := time.Date(2026, 11, 5, 20, 0, 0, 0, time.Local)
	stats := insight.TagStats{
		Key: "customerId", Total: 10000, First: at, Last: at.Add(time.Hour), Distinct: 187,
		Top:      []insight.TagCount{{Value: "c-1042", Count: 123}, {Value: "c-0007", Count: 17}},
		Untagged: 4, Partial: true,
	}

	var out strings.Builder
	if err := writeTagStatsText(&out, stats); err != nil {
		t.Fatalf("writeTagStatsText returned error: %v", err)
	}

	want := "" +
		"10000 events from 2026-11-05 20:00:00 to 2026-11-05 21:00:00, 187 different customerId values\n" +
		"Only the first 10000 events were counted. For a complete answer, narrow it down " +
		"with --since, --name or --tag, or raise --max-events.\n" +
		"\n" +
		"123  c-1042\n" +
		" 17  c-0007\n" +
		"  4  (no customerId tag)\n"
	if out.String() != want {
		t.Errorf("output mismatch\ngot:\n%s\nwant:\n%s", out.String(), want)
	}
}
