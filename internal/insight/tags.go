package insight

import (
	"cmp"
	"iter"
	"slices"
	"strings"
	"time"

	"github.com/ayberkyonar/axonscope/internal/eventstore"
)

// DefaultMaxEvents is how many events GroupByTag reads by default.
//
// Axon Server sends each event's tags in a separate call, so grouping by tag
// costs one call per event. The limit keeps a single question from putting
// heavy load on a production event store.
const DefaultMaxEvents = 10_000

// TagStats counts events per value of one tag key.
type TagStats struct {
	Key string
	// Total is the number of events read, including those without the tag.
	Total int
	// First and Last are the timestamps of the oldest and newest event read.
	First, Last time.Time
	// Distinct is the number of different values of the tag.
	Distinct int
	// Top holds the most frequent values, most frequent first.
	Top []TagCount
	// Untagged is the number of events without the tag.
	Untagged int
	// Partial reports that reading stopped at the limit, so more events
	// exist and the counts cover only the first Total of them.
	Partial bool
}

// TagCount is the number of events with one tag value.
type TagCount struct {
	Value string
	Count int
}

// GroupByTag counts events per value of the tag key and keeps the top most
// frequent values. It reads at most maxEvents events (0 means no limit) and
// sets Partial when there were more. The events need their tags, so don't
// read them with eventstore.Query.SkipTags.
func GroupByTag(events iter.Seq2[eventstore.Event, error], key string, top, maxEvents int) (TagStats, error) {
	stats := TagStats{Key: key}
	counts := map[string]int{}
	for e, err := range events {
		if err != nil {
			return TagStats{}, err
		}
		if maxEvents > 0 && stats.Total == maxEvents {
			stats.Partial = true // this event is one too many
			break
		}

		if stats.Total == 0 {
			stats.First = e.Timestamp
		}
		stats.Last = e.Timestamp
		stats.Total++

		tagged := false
		for _, tag := range e.Tags {
			if tag.Key == key {
				counts[tag.Value]++
				tagged = true
			}
		}
		if !tagged {
			stats.Untagged++
		}
	}

	stats.Distinct = len(counts)
	stats.Top = topValues(counts, top)
	return stats, nil
}

// topValues returns the n most frequent values, most frequent first.
// Values with the same count are sorted alphabetically, so the order is stable.
func topValues(counts map[string]int, n int) []TagCount {
	list := make([]TagCount, 0, len(counts))
	for value, count := range counts {
		list = append(list, TagCount{Value: value, Count: count})
	}
	slices.SortFunc(list, func(a, b TagCount) int {
		return cmp.Or(cmp.Compare(b.Count, a.Count), strings.Compare(a.Value, b.Value))
	})
	return list[:min(n, len(list))]
}
