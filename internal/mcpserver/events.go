package mcpserver

import (
	"context"
	"encoding/json"
	"iter"
	"time"

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
	Events []event `json:"events"`
	Next   *int64  `json:"next,omitempty" jsonschema:"pass as 'from' to read the next page; absent when there are no more events"`
}

// event is the JSON form of an eventstore.Event.
type event struct {
	Sequence  int64             `json:"sequence"`
	ID        string            `json:"id"`
	Name      string            `json:"name"`
	Version   string            `json:"version,omitempty"`
	Timestamp string            `json:"timestamp" jsonschema:"RFC 3339 timestamp in UTC"`
	Tags      []tag             `json:"tags"`
	Metadata  map[string]string `json:"metadata,omitempty"`
	Payload   any               `json:"payload" jsonschema:"the event payload: a JSON value, or a string if it is not JSON"`
}

type tag struct {
	Key   string `json:"key"`
	Value string `json:"value"`
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
	out := listEventsOutput{Events: []event{}}
	for e, err := range events {
		if err != nil {
			return listEventsOutput{}, err
		}
		if len(out.Events) == limit {
			out.Next = &e.Sequence
			break
		}
		out.Events = append(out.Events, toJSON(e))
	}
	return out, nil
}

func toJSON(e eventstore.Event) event {
	tags := make([]tag, len(e.Tags))
	for i, t := range e.Tags {
		tags[i] = tag{Key: t.Key, Value: t.Value}
	}

	return event{
		Sequence:  e.Sequence,
		ID:        e.ID,
		Name:      e.Name,
		Version:   e.Version,
		Timestamp: e.Timestamp.UTC().Format(time.RFC3339Nano),
		Tags:      tags,
		Metadata:  e.Metadata,
		Payload:   decodePayload(e.Payload),
	}
}

// decodePayload returns the payload as a JSON value, so agents see real
// JSON instead of an escaped string. Payloads that are not JSON are
// returned as text.
func decodePayload(payload []byte) any {
	var v any
	if err := json.Unmarshal(payload, &v); err != nil {
		return string(payload)
	}
	return v
}
