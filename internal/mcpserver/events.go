package mcpserver

import (
	"context"
	"iter"

	"github.com/ayberkyonar/axonscope/internal/eventjson"
	"github.com/ayberkyonar/axonscope/internal/eventstore"
)

const (
	defaultLimit = 20
	// maxLimit keeps a single result small enough for an agent's context window.
	maxLimit = 100
)

type listEventsInput struct {
	From  int64             `json:"from,omitempty" jsonschema:"sequence number of the first event, default 0"`
	Limit int               `json:"limit,omitempty" jsonschema:"maximum number of events to return, default 20, at most 100"`
	Names []string          `json:"names,omitempty" jsonschema:"only events with one of these names; full names and short names such as AccountRegistered both work"`
	Tags  map[string]string `json:"tags,omitempty" jsonschema:"only events that have all of these tags, as key-value pairs such as userId: 42"`
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

	query := eventstore.Query{From: in.From, Names: in.Names}
	for key, value := range in.Tags {
		query.Tags = append(query.Tags, eventstore.Tag{Key: key, Value: value})
	}
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
