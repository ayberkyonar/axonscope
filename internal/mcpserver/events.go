package mcpserver

import (
	"context"
	"fmt"
	"iter"
	"time"

	"github.com/ayberkyonar/axonscope/internal/eventjson"
	"github.com/ayberkyonar/axonscope/internal/eventstore"
	"github.com/ayberkyonar/axonscope/internal/timespec"
)

const (
	defaultLimit = 20
	// maxLimit keeps a single result small enough for an agent's context window.
	maxLimit = 100
)

// filterInput holds the inputs that select events, shared by several tools.
type filterInput struct {
	Names []string          `json:"names,omitempty" jsonschema:"only events with one of these names; full names and short names such as AccountRegistered both work"`
	Tags  map[string]string `json:"tags,omitempty" jsonschema:"only events that have all of these tags, as key-value pairs such as userId: 42"`
	Since string            `json:"since,omitempty" jsonschema:"only events at or after this time: a date (2026-11-05), date and time (2026-11-05T21:00), RFC 3339 timestamp, or duration ago (90m, 24h, 7d, 2w)"`
	Until string            `json:"until,omitempty" jsonschema:"only events before this time, same formats as since"`
}

// query turns the inputs into a query. Relative times count back from now.
func (f filterInput) query(now time.Time) (eventstore.Query, error) {
	since, err := timespec.Parse(f.Since, now)
	if err != nil {
		return eventstore.Query{}, fmt.Errorf("since: %w", err)
	}
	until, err := timespec.Parse(f.Until, now)
	if err != nil {
		return eventstore.Query{}, fmt.Errorf("until: %w", err)
	}

	q := eventstore.Query{Names: f.Names, Since: since, Until: until}
	for key, value := range f.Tags {
		q.Tags = append(q.Tags, eventstore.Tag{Key: key, Value: value})
	}
	return q, nil
}

type listEventsInput struct {
	filterInput
	From  int64 `json:"from,omitempty" jsonschema:"sequence number of the first event, default 0"`
	Limit int   `json:"limit,omitempty" jsonschema:"maximum number of events to return, default 20, at most 100"`
}

type listEventsOutput struct {
	Events []eventjson.Event `json:"events"`
	Next   *int64            `json:"next,omitempty" jsonschema:"pass as 'from' to read the next page; absent when there are no more events"`
}

type getEventInput struct {
	Sequence int64 `json:"sequence" jsonschema:"sequence number of the event"`
}

func listEvents(ctx context.Context, store *eventstore.Store, in listEventsInput) (listEventsOutput, error) {
	limit := in.Limit
	if limit <= 0 {
		limit = defaultLimit
	}
	limit = min(limit, maxLimit)

	query, err := in.query(time.Now())
	if err != nil {
		return listEventsOutput{}, err
	}
	query.From = in.From
	return collectPage(store.Events(ctx, query), limit)
}

// collectPage reads up to limit events. It reads one extra event to find
// out whether a next page exists, without returning that event.
func collectPage(events iter.Seq2[eventstore.Event, error], limit int) (listEventsOutput, error) {
	out := listEventsOutput{Events: []eventjson.Event{}}
	for e, err := range events {
		if err != nil {
			return listEventsOutput{}, err
		}
		if len(out.Events) == limit {
			out.Next = &e.Sequence
			break
		}
		out.Events = append(out.Events, eventjson.From(e))
	}
	return out, nil
}
