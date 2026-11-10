package eventstore

import (
	"context"
	"errors"
	"fmt"
	"io"
	"iter"
	"strings"
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

// ShortName returns the event name without its package, see ShortName.
func (e Event) ShortName() string {
	return ShortName(e.Name)
}

// ShortName returns an event name without its package:
// "com.example.events.AccountRegistered" becomes "AccountRegistered".
func ShortName(name string) string {
	return name[strings.LastIndex(name, ".")+1:]
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

		for sequenced, err := range s.selected(ctx, q) {
			if err != nil {
				yield(Event{}, err)
				return
			}
			event := toEvent(sequenced)
			if !q.SkipTags {
				event.Tags, err = s.tags(ctx, event.Sequence)
			}
			if !yield(event, err) || err != nil {
				return
			}
		}
	}
}

// selected returns the events selected by q as Axon Server sends them,
// without their tags. Filters Axon Server cannot apply are applied here,
// before anyone spends a call on an event's tags.
func (s *Store) selected(ctx context.Context, q Query) iter.Seq2[*axonpb.SequencedEvent, error] {
	return func(yield func(*axonpb.SequencedEvent, error) bool) {
		from, err := s.startSequence(ctx, q)
		if err != nil {
			yield(nil, err)
			return
		}
		q.From = from // q is a copy, so this does not affect the caller

		stream, err := s.events.Source(ctx, q.request())
		if err != nil {
			yield(nil, s.rpcError("read events", err))
			return
		}

		for {
			resp, err := stream.Recv()
			// Axon Server rejects a start past the head; for us that just means no events.
			if errors.Is(err, io.EOF) || status.Code(err) == codes.OutOfRange {
				return
			}
			if err != nil {
				yield(nil, s.rpcError("read events", err))
				return
			}

			sequenced := resp.GetEvent()
			if sequenced == nil {
				continue // a consistency marker, which only matters for appending
			}
			e := sequenced.GetEvent()
			timestamp := time.UnixMilli(e.GetTimestamp())
			if q.pastUntil(timestamp) {
				return
			}
			if q.beforeSince(timestamp) || !q.matchesName(e.GetName()) {
				continue
			}
			if !yield(sequenced, nil) {
				return
			}
		}
	}
}

// startSequence returns the sequence number to start reading at: q.From,
// or later when q.Since allows skipping older events.
func (s *Store) startSequence(ctx context.Context, q Query) (int64, error) {
	if q.Since.IsZero() {
		return q.From, nil
	}
	resp, err := s.events.GetSequenceAt(ctx, &axonpb.GetSequenceAtRequest{Timestamp: q.Since.UnixMilli()})
	if err != nil {
		return 0, s.rpcError("find events since "+q.Since.Format(time.RFC3339), err)
	}
	// Axon Server's answer is approximate: it can be one event too far.
	// Starting one earlier is safe, because older events are skipped while reading.
	return max(q.From, resp.GetSequence()-1), nil
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

// toEvent converts a protobuf event to an Event, without its tags:
// the event store sends those separately.
func toEvent(sequenced *axonpb.SequencedEvent) Event {
	e := sequenced.GetEvent()
	return Event{
		Sequence:  sequenced.GetSequence(),
		ID:        e.GetIdentifier(),
		Name:      e.GetName(),
		Version:   e.GetVersion(),
		Timestamp: time.UnixMilli(e.GetTimestamp()),
		Metadata:  e.GetMetadata(),
		Payload:   e.GetPayload(),
	}
}

// tags fetches the tags of the event at sequence.
func (s *Store) tags(ctx context.Context, sequence int64) ([]Tag, error) {
	resp, err := s.events.GetTags(ctx, &axonpb.GetTagsRequest{Sequence: sequence})
	if err != nil {
		return nil, s.rpcError(fmt.Sprintf("read tags of event %d", sequence), err)
	}
	tags := make([]Tag, len(resp.GetTag()))
	for i, tag := range resp.GetTag() {
		tags[i] = Tag{Key: string(tag.GetKey()), Value: string(tag.GetValue())}
	}
	return tags, nil
}
