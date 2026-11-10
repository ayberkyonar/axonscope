package eventstore

import (
	"context"
	"errors"
	"fmt"
	"io"
	"iter"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/ayberkyonar/axonscope/internal/eventstore/internal/axonpb"
)

// Event is one event read from the event store.
type Event struct {
	// Sequence is the event's position in the event store, starting at 0.
	Sequence int64
	ID       string
	// Name is the event type, usually a fully qualified class name
	// such as "com.example.events.AccountRegistered".
	Name      string
	Version   string
	Timestamp time.Time
	Tags      []Tag
	Metadata  map[string]string
	// Payload is the serialized event, usually JSON.
	Payload []byte
}

// ShortName returns the event name without its package:
// "com.example.events.AccountRegistered" becomes "AccountRegistered".
func (e Event) ShortName() string {
	return shortName(e.Name)
}

// Tag links an event to a domain concept, such as userId=123.
// Applications use tags to find the events that belong together.
type Tag struct {
	Key   string
	Value string
}

// Events returns the events selected by q, oldest first,
// up to the event store's current head.
//
// Range over the result to read the events. Stopping early is cheap:
// the stream is closed and no further events are fetched.
// After an error, the iteration stops.
func (s *Store) Events(ctx context.Context, q Query) iter.Seq2[Event, error] {
	return func(yield func(Event, error) bool) {
		// Cancelling the context closes the stream, also when the caller stops early.
		ctx, cancel := context.WithCancel(ctx)
		defer cancel()

		stream, err := s.events.Source(ctx, q.request())
		if err != nil {
			yield(Event{}, s.rpcError("read events", err))
			return
		}

		for {
			resp, err := stream.Recv()
			// Axon Server rejects a start past the head; for us that just means no events.
			if errors.Is(err, io.EOF) || status.Code(err) == codes.OutOfRange {
				return
			}
			if err != nil {
				yield(Event{}, s.rpcError("read events", err))
				return
			}

			sequenced := resp.GetEvent()
			if sequenced == nil {
				continue // a consistency marker, which only matters for appending
			}
			// Checked before toEvent, so skipped events cost no extra call for their tags.
			if !q.matchesName(sequenced.GetEvent().GetName()) {
				continue
			}

			event, err := s.toEvent(ctx, sequenced)
			if !yield(event, err) || err != nil {
				return
			}
		}
	}
}

// ErrNotFound means no event exists at the requested sequence number.
var ErrNotFound = errors.New("event not found")

// Event returns the event at the given sequence number,
// or an error wrapping ErrNotFound if there is none.
func (s *Store) Event(ctx context.Context, sequence int64) (Event, error) {
	for e, err := range s.Events(ctx, Query{From: sequence}) {
		if err != nil {
			return Event{}, err
		}
		if e.Sequence == sequence {
			return e, nil
		}
		break // the first event is past sequence, so it does not exist
	}
	return Event{}, fmt.Errorf("event %d: %w", sequence, ErrNotFound)
}

// toEvent converts a protobuf event to an Event.
// The event store sends tags separately, so this fetches them too.
func (s *Store) toEvent(ctx context.Context, sequenced *axonpb.SequencedEvent) (Event, error) {
	resp, err := s.events.GetTags(ctx, &axonpb.GetTagsRequest{Sequence: sequenced.GetSequence()})
	if err != nil {
		return Event{}, s.rpcError(fmt.Sprintf("read tags of event %d", sequenced.GetSequence()), err)
	}

	tags := make([]Tag, len(resp.GetTag()))
	for i, tag := range resp.GetTag() {
		tags[i] = Tag{Key: string(tag.GetKey()), Value: string(tag.GetValue())}
	}

	e := sequenced.GetEvent()
	return Event{
		Sequence:  sequenced.GetSequence(),
		ID:        e.GetIdentifier(),
		Name:      e.GetName(),
		Version:   e.GetVersion(),
		Timestamp: time.UnixMilli(e.GetTimestamp()),
		Tags:      tags,
		Metadata:  e.GetMetadata(),
		Payload:   e.GetPayload(),
	}, nil
}
