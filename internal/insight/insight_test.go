package insight

import (
	"errors"
	"iter"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/ayberkyonar/axonscope/internal/eventstore"
)

// events returns an iterator over events with the given names and times,
// followed by err if it is not nil.
func events(err error, list ...eventstore.Event) iter.Seq2[eventstore.Event, error] {
	return func(yield func(eventstore.Event, error) bool) {
		for _, e := range list {
			if !yield(e, nil) {
				return
			}
		}
		if err != nil {
			yield(eventstore.Event{}, err)
		}
	}
}

func event(name string, t time.Time) eventstore.Event {
	return eventstore.Event{Name: name, Timestamp: t}
}

var amsterdam = mustLoad("Europe/Amsterdam")

func mustLoad(name string) *time.Location {
	loc, err := time.LoadLocation(name)
	if err != nil {
		panic(err)
	}
	return loc
}

func TestSummarize(t *testing.T) {
	t1 := time.Date(2026, 11, 5, 20, 0, 0, 0, time.UTC)
	t2 := t1.Add(time.Hour)

	stats, err := Summarize(events(nil, event("b.Placed", t1), event("a.Cancelled", t1), event("b.Placed", t2)))
	if err != nil {
		t.Fatal(err)
	}

	want := Stats{
		Total: 3, First: t1, Last: t2,
		ByName: []NameCount{{"b.Placed", 2}, {"a.Cancelled", 1}},
	}
	if !reflect.DeepEqual(stats, want) {
		t.Errorf("got %+v, want %+v", stats, want)
	}
}

func TestSummarizeReturnsError(t *testing.T) {
	readErr := errors.New("connection lost")
	if _, err := Summarize(events(readErr)); !errors.Is(err, readErr) {
		t.Errorf("got error %v, want %v", err, readErr)
	}
}

func TestByCount(t *testing.T) {
	got := byCount(map[string]int{"b.LoggedOut": 2, "a.LoginSucceeded": 5, "c.AccountRegistered": 2})

	want := []NameCount{
		{Name: "a.LoginSucceeded", Count: 5},
		{Name: "b.LoggedOut", Count: 2}, // same count: alphabetical
		{Name: "c.AccountRegistered", Count: 2},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("byCount = %v, want %v", got, want)
	}
}

// totals returns the start (formatted) and total of each bucket.
func totals(buckets []Bucket, layout string) []string {
	out := make([]string, len(buckets))
	for i, b := range buckets {
		out[i] = b.Start.Format(layout) + "=" + strings.Repeat("|", b.Total)
	}
	return out
}

func TestTrendIncludesEmptyPeriods(t *testing.T) {
	day := func(d, h int) time.Time { return time.Date(2026, 11, d, h, 0, 0, 0, time.UTC) }

	buckets, err := Trend(events(nil, event("Placed", day(2, 9)), event("Placed", day(2, 17)), event("Placed", day(4, 12))),
		Periods{Unit: Day, From: day(1, 0), To: day(5, 0)})
	if err != nil {
		t.Fatal(err)
	}

	want := []string{"11-01=", "11-02=||", "11-03=", "11-04=|"}
	if got := totals(buckets, "01-02"); !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestTrendUsesTheTimeZoneForDays(t *testing.T) {
	// 23:30 UTC on Nov 2 is already Nov 3 in Amsterdam (UTC+1).
	late := time.Date(2026, 11, 2, 23, 30, 0, 0, time.UTC)

	inUTC, _ := Trend(events(nil, event("Placed", late)), Periods{Unit: Day})
	inAmsterdam, _ := Trend(events(nil, event("Placed", late)), Periods{Unit: Day, Location: amsterdam})

	if got := inUTC[0].Start.Format("01-02"); got != "11-02" {
		t.Errorf("UTC: day %s, want 11-02", got)
	}
	if got := inAmsterdam[0].Start.Format("01-02"); got != "11-03" {
		t.Errorf("Amsterdam: day %s, want 11-03", got)
	}
}

func TestTrendWeeksStartOnMonday(t *testing.T) {
	sunday := time.Date(2026, 11, 8, 12, 0, 0, 0, time.UTC)
	monday := sunday.AddDate(0, 0, 1)

	buckets, err := Trend(events(nil, event("Placed", sunday), event("Placed", monday)), Periods{Unit: Week})
	if err != nil {
		t.Fatal(err)
	}

	want := []string{"Mon 11-02=|", "Mon 11-09=|"}
	if got := totals(buckets, "Mon 01-02"); !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestTrendDaysFollowTheCalendarAcrossDaylightSaving(t *testing.T) {
	// Amsterdam switches to winter time on 25 October 2026: that day has 25 hours.
	from := time.Date(2026, 10, 24, 0, 0, 0, 0, amsterdam)
	to := time.Date(2026, 10, 27, 0, 0, 0, 0, amsterdam)

	buckets, err := Trend(events(nil), Periods{Unit: Day, Location: amsterdam, From: from, To: to})
	if err != nil {
		t.Fatal(err)
	}

	want := []string{"10-24 00:00=", "10-25 00:00=", "10-26 00:00="}
	if got := totals(buckets, "01-02 15:04"); !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestTrendCountsPerName(t *testing.T) {
	at := time.Date(2026, 11, 5, 20, 15, 0, 0, time.UTC)

	buckets, err := Trend(events(nil, event("Placed", at), event("Cancelled", at), event("Placed", at)), Periods{Unit: Hour})
	if err != nil {
		t.Fatal(err)
	}

	want := []Bucket{{
		Start:  time.Date(2026, 11, 5, 20, 0, 0, 0, time.UTC),
		Total:  3,
		ByName: []NameCount{{"Placed", 2}, {"Cancelled", 1}},
	}}
	if !reflect.DeepEqual(buckets, want) {
		t.Errorf("got %+v, want %+v", buckets, want)
	}
}

func TestTrendWithoutEventsOrBounds(t *testing.T) {
	buckets, err := Trend(events(nil), Periods{Unit: Day})
	if err != nil || len(buckets) != 0 {
		t.Errorf("got %v, %v; want no buckets and no error", buckets, err)
	}
}

func TestTrendRejectsTooManyPeriods(t *testing.T) {
	from := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	_, err := Trend(events(nil), Periods{Unit: Hour, From: from, To: from.AddDate(1, 0, 0)})
	if err == nil || !strings.Contains(err.Error(), "longer period") {
		t.Errorf("got error %v, want one suggesting a longer period", err)
	}
}

func TestParseUnit(t *testing.T) {
	for _, u := range []Unit{Hour, Day, Week} {
		if got, err := ParseUnit(u.String()); err != nil || got != u {
			t.Errorf("ParseUnit(%q) = %v, %v", u.String(), got, err)
		}
	}
	if _, err := ParseUnit("month"); err == nil {
		t.Error(`ParseUnit("month") returned no error`)
	}
}

func tagged(t time.Time, tags ...string) eventstore.Event {
	e := eventstore.Event{Name: "Placed", Timestamp: t}
	for _, tag := range tags {
		key, value, _ := strings.Cut(tag, "=")
		e.Tags = append(e.Tags, eventstore.Tag{Key: key, Value: value})
	}
	return e
}

func TestGroupByTag(t *testing.T) {
	t0 := time.Date(2026, 11, 5, 20, 0, 0, 0, time.UTC)
	stream := events(nil,
		tagged(t0, "customerId=c1", "orderId=o1"),
		tagged(t0, "customerId=c2"),
		tagged(t0, "customerId=c1"),
		tagged(t0, "orderId=o9"), // no customerId
		tagged(t0.Add(time.Hour), "customerId=c3"),
	)

	got, err := GroupByTag(stream, "customerId", 2, 0)
	if err != nil {
		t.Fatal(err)
	}

	want := TagStats{
		Key: "customerId", Total: 5, First: t0, Last: t0.Add(time.Hour),
		Distinct: 3,
		Top:      []TagCount{{"c1", 2}, {"c2", 1}}, // c2 before c3: same count, alphabetical
		Untagged: 1,
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got  %+v\nwant %+v", got, want)
	}
}

func TestGroupByTagStopsAtTheLimit(t *testing.T) {
	t0 := time.Date(2026, 11, 5, 20, 0, 0, 0, time.UTC)
	stream := events(nil, tagged(t0, "k=a"), tagged(t0, "k=a"), tagged(t0, "k=b"))

	got, err := GroupByTag(stream, "k", 10, 2)
	if err != nil {
		t.Fatal(err)
	}
	if !got.Partial || got.Total != 2 || !reflect.DeepEqual(got.Top, []TagCount{{"a", 2}}) {
		t.Errorf("got %+v, want 2 events read, partial, only a counted", got)
	}

	exact, _ := GroupByTag(events(nil, tagged(t0, "k=a"), tagged(t0, "k=a")), "k", 10, 2)
	if exact.Partial {
		t.Error("reading exactly the limit, with no more events, must not be partial")
	}
}

func TestGroupByTagReturnsError(t *testing.T) {
	readErr := errors.New("connection lost")
	if _, err := GroupByTag(events(readErr), "k", 10, 0); !errors.Is(err, readErr) {
		t.Errorf("got error %v, want %v", err, readErr)
	}
}
