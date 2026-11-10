package insight

import (
	"cmp"
	"iter"
	"slices"
	"strings"
	"time"

	"github.com/ayberkyonar/axonscope/internal/eventstore"
)

// JourneyOptions describes which journeys to collect and how many to return.
type JourneyOptions struct {
	// Key is the tag that ties events into one journey, like orderId:
	// all events with the same orderId value form that order's journey.
	Key string
	// Containing keeps only journeys with an event of this name. Empty keeps all.
	Containing string
	// StartingWith keeps only journeys whose first event has this name, which
	// leaves out journeys that began before the events read. Empty keeps all.
	StartingWith string
	// Top is how many of the most common journeys to return.
	Top int
	// Examples is how many tag values to give as examples per journey.
	Examples int
	// MaxEvents stops reading after this many events (0 means no limit).
	MaxEvents int
}

// Names in Containing and StartingWith may be full or short, like in
// eventstore.Query.Names.
func nameMatches(want, name string) bool {
	return want == name || want == eventstore.ShortName(name)
}

// Journeys summarizes the paths that entities, like orders or users, take
// through the events.
type Journeys struct {
	Key string
	// Events is the number of events read.
	Events int
	// Entities is the number of different tag values seen, each one journey.
	Entities int
	// Matching is the number of journeys that passed Containing and StartingWith.
	Matching int
	// Distinct is the number of different paths among the matching journeys.
	Distinct int
	// Top holds the most common paths, most common first.
	Top []Journey
	// Other is the number of matching journeys whose path is not in Top.
	Other int
	// Partial reports that reading stopped at MaxEvents, so the journeys are
	// incomplete: there were more events.
	Partial bool
}

// Journey is one path and how many entities took it.
type Journey struct {
	Steps []Step
	// MoreSteps reports that the path is longer than MaxSteps; only its first
	// MaxSteps steps are in Steps.
	MoreSteps bool
	Count     int
	// MedianDuration is the median time from the first to the last event.
	MedianDuration time.Duration
	// Examples are tag values of entities that took this path, in the order
	// they first appeared.
	Examples []string
}

// Step is one step of a journey. Repeated means the event occurred several
// times in a row; repeats are merged so that, for example, adding two or
// three items counts as the same path.
type Step struct {
	Name     string
	Repeated bool
}

// MaxSteps limits the length of a path, so very long journeys (a user who
// logs in every day) don't each become their own unreadable path.
const MaxSteps = 12

// entity collects the journey of one tag value while reading.
type entity struct {
	steps       []Step
	first, last time.Time
}

func (en *entity) add(name string, t time.Time) {
	if n := len(en.steps); n > 0 && en.steps[n-1].Name == name {
		en.steps[n-1].Repeated = true
	} else {
		en.steps = append(en.steps, Step{Name: name})
	}
	if en.first.IsZero() {
		en.first = t
	}
	en.last = t
}

func (en *entity) matches(opts JourneyOptions) bool {
	if opts.StartingWith != "" && !nameMatches(opts.StartingWith, en.steps[0].Name) {
		return false
	}
	if opts.Containing != "" && !slices.ContainsFunc(en.steps, func(s Step) bool {
		return nameMatches(opts.Containing, s.Name)
	}) {
		return false
	}
	return true
}

// FindJourneys collects each entity's journey, the events sharing one value
// of opts.Key in the order they happened, and counts the common paths.
// Events without the tag are ignored. The events need their tags, so don't
// read them with eventstore.Query.SkipTags.
func FindJourneys(events iter.Seq2[eventstore.Event, error], opts JourneyOptions) (Journeys, error) {
	result := Journeys{Key: opts.Key}
	entities := map[string]*entity{}
	var order []string // tag values in the order they first appeared, for stable examples

	for e, err := range events {
		if err != nil {
			return Journeys{}, err
		}
		if opts.MaxEvents > 0 && result.Events == opts.MaxEvents {
			result.Partial = true // this event is one too many
			break
		}
		result.Events++

		for _, tag := range e.Tags {
			if tag.Key != opts.Key {
				continue
			}
			en := entities[tag.Value]
			if en == nil {
				en = &entity{}
				entities[tag.Value] = en
				order = append(order, tag.Value)
			}
			en.add(e.Name, e.Timestamp)
		}
	}
	result.Entities = len(entities)

	// Group the matching journeys by path.
	type group struct {
		journey   Journey
		durations []time.Duration
	}
	groups := map[string]*group{}
	for _, value := range order {
		en := entities[value]
		if !en.matches(opts) {
			continue
		}
		result.Matching++

		steps, more := en.steps, false
		if len(steps) > MaxSteps {
			steps, more = steps[:MaxSteps], true
		}
		key := pathKey(steps, more)
		g := groups[key]
		if g == nil {
			g = &group{journey: Journey{Steps: steps, MoreSteps: more}}
			groups[key] = g
		}
		g.journey.Count++
		g.durations = append(g.durations, en.last.Sub(en.first))
		if len(g.journey.Examples) < opts.Examples {
			g.journey.Examples = append(g.journey.Examples, value)
		}
	}
	result.Distinct = len(groups)

	keys := make([]string, 0, len(groups))
	for key, g := range groups {
		g.journey.MedianDuration = median(g.durations)
		keys = append(keys, key)
	}
	// Most common first; equally common paths alphabetically, so the order is stable.
	slices.SortFunc(keys, func(a, b string) int {
		return cmp.Or(cmp.Compare(groups[b].journey.Count, groups[a].journey.Count), strings.Compare(a, b))
	})
	for _, key := range keys[:min(opts.Top, len(keys))] {
		result.Top = append(result.Top, groups[key].journey)
	}

	result.Other = result.Matching
	for _, j := range result.Top {
		result.Other -= j.Count
	}
	return result, nil
}

// pathKey identifies a path, so equal paths can be grouped.
func pathKey(steps []Step, more bool) string {
	var b strings.Builder
	for _, s := range steps {
		b.WriteString(s.Name)
		if s.Repeated {
			b.WriteString("+")
		}
		b.WriteString(" ")
	}
	if more {
		b.WriteString("…")
	}
	return b.String()
}

func median(durations []time.Duration) time.Duration {
	slices.Sort(durations)
	n := len(durations)
	if n%2 == 1 {
		return durations[n/2]
	}
	return (durations[n/2-1] + durations[n/2]) / 2
}
