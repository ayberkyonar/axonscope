package insight

import (
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/ayberkyonar/axonscope/internal/eventstore"
)

var journeyStart = time.Date(2026, 11, 5, 20, 0, 0, 0, time.UTC)

// step returns an event with a full name like "shop.Started", minute
// minutes after journeyStart, with tags written as "key=value".
func step(name string, minute int, tags ...string) eventstore.Event {
	e := eventstore.Event{Name: "shop." + name, Timestamp: journeyStart.Add(time.Duration(minute) * time.Minute)}
	for _, tag := range tags {
		key, value, _ := strings.Cut(tag, "=")
		e.Tags = append(e.Tags, eventstore.Tag{Key: key, Value: value})
	}
	return e
}

// paths writes each journey as "count: Step Step+ …" (+ marks a repeated step).
func paths(js Journeys) []string {
	out := make([]string, len(js.Top))
	for i, j := range js.Top {
		var names []string
		for _, s := range j.Steps {
			n := strings.TrimPrefix(s.Name, "shop.")
			if s.Repeated {
				n += "+"
			}
			names = append(names, n)
		}
		if j.MoreSteps {
			names = append(names, "…")
		}
		out[i] = strings.Repeat("|", j.Count) + " " + strings.Join(names, " ")
	}
	return out
}

// orders is a small restaurant: three orders, one of them cancelled.
var orders = []eventstore.Event{
	step("Started", 0, "orderId=o1"),
	step("Started", 1, "orderId=o2"),
	step("ItemAdded", 2, "orderId=o1"),
	step("ItemAdded", 3, "orderId=o1"),
	step("ItemAdded", 4, "orderId=o2"),
	step("Placed", 5, "orderId=o1"),
	step("Cancelled", 6, "orderId=o2"),
	step("Started", 7, "orderId=o3"),
	step("ItemAdded", 8, "orderId=o3"),
	step("Placed", 9, "orderId=o3"),
	step("Restocked", 10), // no orderId: not part of any journey
}

func TestFindJourneys(t *testing.T) {
	got, err := FindJourneys(events(nil, orders...), JourneyOptions{Key: "orderId", Top: 10, Examples: 5})
	if err != nil {
		t.Fatal(err)
	}

	// o1 added two items and o3 one: one item and several items are different
	// paths (see TestFindJourneysMergesRepeats for two vs. three).
	// Equally common paths are sorted alphabetically.
	want := []string{"| Started ItemAdded Cancelled", "| Started ItemAdded Placed", "| Started ItemAdded+ Placed"}
	if !reflect.DeepEqual(paths(got), want) {
		t.Errorf("paths = %q, want %q", paths(got), want)
	}
	if got.Events != 11 || got.Entities != 3 || got.Matching != 3 || got.Distinct != 3 || got.Other != 0 || got.Partial {
		t.Errorf("counts = %+v", got)
	}
}

func TestFindJourneysMergesRepeats(t *testing.T) {
	got, err := FindJourneys(events(nil,
		step("ItemAdded", 0, "orderId=a"), step("ItemAdded", 1, "orderId=a"), step("Placed", 2, "orderId=a"),
		step("ItemAdded", 0, "orderId=b"), step("ItemAdded", 1, "orderId=b"), step("ItemAdded", 2, "orderId=b"), step("Placed", 3, "orderId=b"),
	), JourneyOptions{Key: "orderId", Top: 10})
	if err != nil {
		t.Fatal(err)
	}

	if want := []string{"|| ItemAdded+ Placed"}; !reflect.DeepEqual(paths(got), want) {
		t.Errorf("paths = %q, want %q", paths(got), want)
	}
}

func TestFindJourneysContaining(t *testing.T) {
	got, err := FindJourneys(events(nil, orders...), JourneyOptions{Key: "orderId", Containing: "Cancelled", Top: 10, Examples: 5})
	if err != nil {
		t.Fatal(err)
	}

	if want := []string{"| Started ItemAdded Cancelled"}; !reflect.DeepEqual(paths(got), want) {
		t.Errorf("paths = %q, want %q", paths(got), want)
	}
	if got.Entities != 3 || got.Matching != 1 {
		t.Errorf("entities %d, matching %d; want 3 and 1", got.Entities, got.Matching)
	}
	if want := []string{"o2"}; !reflect.DeepEqual(got.Top[0].Examples, want) {
		t.Errorf("examples = %q, want %q", got.Top[0].Examples, want)
	}
}

func TestFindJourneysStartingWith(t *testing.T) {
	// o4's journey started before the events read: we only see its end.
	stream := append([]eventstore.Event{step("Placed", 0, "orderId=o4")}, orders...)

	got, err := FindJourneys(events(nil, stream...), JourneyOptions{Key: "orderId", StartingWith: "shop.Started", Top: 10})
	if err != nil {
		t.Fatal(err)
	}
	if got.Entities != 4 || got.Matching != 3 {
		t.Errorf("entities %d, matching %d; want 4 and 3 (o4 left out)", got.Entities, got.Matching)
	}
}

func TestFindJourneysTopAndOther(t *testing.T) {
	got, err := FindJourneys(events(nil, orders...), JourneyOptions{Key: "orderId", Top: 1, Examples: 1})
	if err != nil {
		t.Fatal(err)
	}

	if len(got.Top) != 1 || got.Other != 2 || got.Distinct != 3 {
		t.Errorf("top %d, other %d, distinct %d; want 1, 2, 3", len(got.Top), got.Other, got.Distinct)
	}
}

func TestFindJourneysMedianDuration(t *testing.T) {
	got, err := FindJourneys(events(nil,
		step("Started", 0, "orderId=a"), step("Placed", 2, "orderId=a"),
		step("Started", 0, "orderId=b"), step("Placed", 4, "orderId=b"),
		step("Started", 0, "orderId=c"), step("Placed", 30, "orderId=c"),
	), JourneyOptions{Key: "orderId", Top: 10})
	if err != nil {
		t.Fatal(err)
	}

	// The median ignores the one slow order, unlike an average would.
	if want := 4 * time.Minute; got.Top[0].MedianDuration != want {
		t.Errorf("median = %v, want %v", got.Top[0].MedianDuration, want)
	}
}

func TestFindJourneysCutsLongPaths(t *testing.T) {
	var stream []eventstore.Event
	for i := range MaxSteps + 5 {
		name := "LoggedIn"
		if i%2 == 1 {
			name = "LoggedOut"
		}
		stream = append(stream, step(name, i, "userId=u1"))
	}

	got, err := FindJourneys(events(nil, stream...), JourneyOptions{Key: "userId", Top: 10})
	if err != nil {
		t.Fatal(err)
	}
	if j := got.Top[0]; len(j.Steps) != MaxSteps || !j.MoreSteps {
		t.Errorf("got %d steps, more %v; want %d steps and more", len(j.Steps), j.MoreSteps, MaxSteps)
	}
}

func TestFindJourneysStopsAtTheLimit(t *testing.T) {
	got, err := FindJourneys(events(nil, orders...), JourneyOptions{Key: "orderId", Top: 10, MaxEvents: 4})
	if err != nil {
		t.Fatal(err)
	}
	if !got.Partial || got.Events != 4 {
		t.Errorf("partial %v, events %d; want partial after 4 events", got.Partial, got.Events)
	}
}

func TestFindJourneysReturnsError(t *testing.T) {
	readErr := errors.New("connection lost")
	if _, err := FindJourneys(events(readErr), JourneyOptions{Key: "orderId"}); !errors.Is(err, readErr) {
		t.Errorf("got error %v, want %v", err, readErr)
	}
}
