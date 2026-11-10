package eventstore

import (
	"slices"
	"strings"

	"github.com/ayberkyonar/axonscope/internal/eventstore/internal/axonpb"
)

// Query selects events. The zero value selects all events.
type Query struct {
	// From is the sequence number of the first event to read.
	From int64

	// Names selects events with one of these names. A name matches the full
	// event name or its short form (see Event.ShortName), so
	// "AccountRegistered" matches "com.example.events.AccountRegistered".
	// Empty selects all names.
	//
	// Axon Server only matches full names, so short names are matched while
	// reading. A query with short names and no tags reads the whole event store.
	Names []string

	// Tags selects events that have all of these tags.
	Tags []Tag
}

// request builds the Axon Server request for q. Filters Axon Server cannot
// apply are left to matchesName.
func (q Query) request() *axonpb.SourceEventsRequest {
	criterion := &axonpb.TagsAndNamesCriterion{}
	for _, tag := range q.Tags {
		criterion.Tag = append(criterion.Tag, &axonpb.Tag{Key: []byte(tag.Key), Value: []byte(tag.Value)})
	}
	if !slices.ContainsFunc(q.Names, isShortName) {
		criterion.Name = q.Names
	}

	req := &axonpb.SourceEventsRequest{FromSequence: q.From}
	if len(criterion.Name) > 0 || len(criterion.Tag) > 0 {
		req.Criterion = []*axonpb.Criterion{{TagsAndNames: criterion}}
	}
	return req
}

// matchesName reports whether an event with the given full name is selected by q.
func (q Query) matchesName(name string) bool {
	if len(q.Names) == 0 {
		return true
	}
	short := shortName(name)
	return slices.ContainsFunc(q.Names, func(n string) bool {
		return n == name || n == short
	})
}

func isShortName(name string) bool {
	return !strings.Contains(name, ".")
}

func shortName(name string) string {
	return name[strings.LastIndex(name, ".")+1:]
}
