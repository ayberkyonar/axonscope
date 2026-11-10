package insight

import (
	"fmt"
	"iter"
	"time"

	"github.com/ayberkyonar/axonscope/internal/eventstore"
)

// Unit is the length of one period in a trend.
type Unit int

const (
	Hour Unit = iota + 1
	Day
	Week
)

// ParseUnit parses "hour", "day" or "week".
func ParseUnit(s string) (Unit, error) {
	switch s {
	case "hour":
		return Hour, nil
	case "day":
		return Day, nil
	case "week":
		return Week, nil
	}
	return 0, fmt.Errorf("invalid period %q: want hour, day or week", s)
}

func (u Unit) String() string {
	switch u {
	case Hour:
		return "hour"
	case Day:
		return "day"
	case Week:
		return "week"
	}
	return fmt.Sprintf("Unit(%d)", int(u))
}

// start returns the start of the period that contains t, in t's time zone.
// Weeks start on Monday.
func (u Unit) start(t time.Time) time.Time {
	y, m, d := t.Date()
	switch u {
	case Hour:
		return time.Date(y, m, d, t.Hour(), 0, 0, 0, t.Location())
	case Week:
		daysSinceMonday := (int(t.Weekday()) + 6) % 7
		return time.Date(y, m, d-daysSinceMonday, 0, 0, 0, 0, t.Location())
	default:
		return time.Date(y, m, d, 0, 0, 0, 0, t.Location())
	}
}

// next returns the start of the period after the one starting at start.
// Days and weeks follow the calendar, so they stay correct across
// daylight saving time changes.
func (u Unit) next(start time.Time) time.Time {
	switch u {
	case Hour:
		return start.Add(time.Hour)
	case Week:
		return start.AddDate(0, 0, 7)
	default:
		return start.AddDate(0, 0, 1)
	}
}

// Periods describes how a trend splits time.
type Periods struct {
	Unit Unit
	// Location is the time zone whose calendar sets the boundaries:
	// days start at midnight there. Nil means UTC.
	Location *time.Location
	// From and To bound the trend. Every period from the one containing From
	// up to To (exclusive) is included, also periods without events, so gaps
	// are visible. A zero From starts at the first event, a zero To ends after
	// the last.
	From, To time.Time
}

// Bucket is one period of a trend.
type Bucket struct {
	Start time.Time
	Total int
	// ByName counts the period's events per name, most frequent first.
	ByName []NameCount
}

// MaxBuckets limits the length of a trend, so a short unit over a long time
// range fails with a clear message instead of producing a huge result.
const MaxBuckets = 1000

// Trend counts events per period, oldest period first. It doesn't use tags,
// so read the events with eventstore.Query.SkipTags to make it much faster.
func Trend(events iter.Seq2[eventstore.Event, error], p Periods) ([]Bucket, error) {
	loc := p.Location
	if loc == nil {
		loc = time.UTC
	}

	// Counts per name, keyed by the start of each period as Unix seconds.
	counts := map[int64]map[string]int{}
	var first, last time.Time
	for e, err := range events {
		if err != nil {
			return nil, err
		}
		key := p.Unit.start(e.Timestamp.In(loc)).Unix()
		if counts[key] == nil {
			counts[key] = map[string]int{}
		}
		counts[key][e.Name]++
		if first.IsZero() {
			first = e.Timestamp
		}
		last = e.Timestamp
	}

	from, to := p.From, p.To
	if from.IsZero() {
		from = first
	}
	if to.IsZero() && !last.IsZero() {
		to = last.Add(time.Nanosecond) // To is exclusive, so end just after the last event
	}
	if from.IsZero() || to.IsZero() {
		return []Bucket{}, nil // no events and no bounds: nothing to show
	}

	buckets := []Bucket{}
	for start := p.Unit.start(from.In(loc)); start.Before(to); start = p.Unit.next(start) {
		if len(buckets) == MaxBuckets {
			return nil, fmt.Errorf("the trend would have more than %d periods: use a longer period or a shorter time range",
				MaxBuckets)
		}
		bucket := Bucket{Start: start, ByName: byCount(counts[start.Unix()])}
		for _, nc := range bucket.ByName {
			bucket.Total += nc.Count
		}
		buckets = append(buckets, bucket)
	}
	return buckets, nil
}
