// Package insight turns events into overviews, such as counts and trends.
//
// It works on any iterator of events, not on Axon Server directly, so the same
// logic can serve live reads now and a stored projection later.
package insight

import (
	"cmp"
	"iter"
	"slices"
	"strings"
	"time"

	"github.com/ayberkyonar/axonscope/internal/eventstore"
)

// Stats summarizes a set of events.
type Stats struct {
	Total int
	// First and Last are the timestamps of the oldest and newest event.
	// Both are zero when Total is 0.
	First, Last time.Time
	// ByName counts the events per name, most frequent first.
	ByName []NameCount
}

// NameCount is the number of events with one name.
type NameCount struct {
	Name  string
	Count int
}

// ShortName returns the name without its package, like eventstore.ShortName.
func (nc NameCount) ShortName() string {
	return eventstore.ShortName(nc.Name)
}

// Summarize counts events per name. It doesn't use tags, so read the events
// with eventstore.Query.SkipTags to make it much faster.
func Summarize(events iter.Seq2[eventstore.Event, error]) (Stats, error) {
	var stats Stats
	counts := map[string]int{}
	for e, err := range events {
		if err != nil {
			return Stats{}, err
		}
		if stats.Total == 0 {
			stats.First = e.Timestamp
		}
		stats.Last = e.Timestamp
		stats.Total++
		counts[e.Name]++
	}
	stats.ByName = byCount(counts)
	return stats, nil
}

// byCount turns counts per name into a list, most frequent first.
// Names with the same count are sorted alphabetically, so the order is stable.
func byCount(counts map[string]int) []NameCount {
	list := make([]NameCount, 0, len(counts))
	for name, count := range counts {
		list = append(list, NameCount{Name: name, Count: count})
	}
	slices.SortFunc(list, func(a, b NameCount) int {
		return cmp.Or(cmp.Compare(b.Count, a.Count), strings.Compare(a.Name, b.Name))
	})
	return list
}
