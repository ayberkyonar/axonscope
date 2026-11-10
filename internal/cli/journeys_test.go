package cli

import (
	"strings"
	"testing"
	"time"

	"github.com/ayberkyonar/axonscope/internal/insight"
)

func TestWriteJourneysText(t *testing.T) {
	js := insight.Journeys{
		Key: "orderId", Events: 500, Entities: 120, Matching: 20, Distinct: 3, Other: 4,
		Top: []insight.Journey{
			{
				Steps:          []insight.Step{{Name: "shop.Started"}, {Name: "shop.ItemAdded", Repeated: true}, {Name: "shop.Cancelled"}},
				Count:          12,
				MedianDuration: 3*time.Minute + 20*time.Second,
				Examples:       []string{"o1", "o7"},
			},
			{
				Steps:          []insight.Step{{Name: "shop.Started"}, {Name: "shop.Cancelled"}},
				Count:          4,
				MedianDuration: 40 * time.Second,
				Examples:       []string{"o3"},
			},
		},
	}

	var out strings.Builder
	if err := writeJourneysText(&out, js); err != nil {
		t.Fatalf("writeJourneysText returned error: %v", err)
	}

	want := "" +
		"500 events, 120 journeys by orderId, 20 matching, 3 different paths\n" +
		"\n" +
		"12  60%  Started → ItemAdded (repeated) → Cancelled  median 3m20s  e.g. o1, o7\n" +
		"4   20%  Started → Cancelled                         median 40s    e.g. o3\n" +
		"4   20%  (other paths)\n"
	if out.String() != want {
		t.Errorf("output mismatch\ngot:\n%s\nwant:\n%s", out.String(), want)
	}
}

func TestWriteJourneysTextWithoutJourneys(t *testing.T) {
	var out strings.Builder
	if err := writeJourneysText(&out, insight.Journeys{Events: 41}); err != nil {
		t.Fatalf("writeJourneysText returned error: %v", err)
	}
	if want := "no journeys (41 events read)\n"; out.String() != want {
		t.Errorf("got %q, want %q", out.String(), want)
	}
}

func TestFormatStepsMarksLongPaths(t *testing.T) {
	j := insight.Journey{Steps: []insight.Step{{Name: "a.LoggedIn"}, {Name: "a.LoggedOut"}}, MoreSteps: true}
	if got, want := formatSteps(j), "LoggedIn → LoggedOut → …"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}
