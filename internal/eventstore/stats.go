package eventstore

import (
	"cmp"
	"context"
	"slices"
	"strings"
	"time"
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

// ShortName returns the name without its package, like Event.ShortName.
func (nc NameCount) ShortName() string {
	return shortName(nc.Name)
}

// Stats counts the events selected by q.
//
// It reads every selected event, so narrow q down on a large event store,
// for example with Since. It skips fetching tags, which makes it much
// faster than ranging over Events.
func (s *Store) Stats(ctx context.Context, q Query) (Stats, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	var stats Stats
	counts := map[string]int{}
	for sequenced, err := range s.selected(ctx, q) {
		if err != nil {
			return Stats{}, err
		}
		e := sequenced.GetEvent()
		timestamp := time.UnixMilli(e.GetTimestamp())
		if stats.Total == 0 {
			stats.First = timestamp
		}
		stats.Last = timestamp
		stats.Total++
		counts[e.GetName()]++
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
