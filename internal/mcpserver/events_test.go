package mcpserver

import (
	"errors"
	"iter"
	"reflect"
	"testing"
	"time"

	"github.com/ayberkyonar/axonscope/internal/eventstore"
)

// eventsWithSequences returns an iterator over empty events with the given
// sequence numbers, followed by err if it is not nil.
func eventsWithSequences(sequences []int64, err error) iter.Seq2[eventstore.Event, error] {
	return func(yield func(eventstore.Event, error) bool) {
		for _, seq := range sequences {
			if !yield(eventstore.Event{Sequence: seq, Payload: []byte("{}")}, nil) {
				return
			}
		}
		if err != nil {
			yield(eventstore.Event{}, err)
		}
	}
}

func TestCollectPage(t *testing.T) {
	tests := []struct {
		name          string
		sequences     []int64
		limit         int
		wantSequences []int64
		wantNext      *int64
	}{
		{"fewer events than limit", []int64{0, 1}, 5, []int64{0, 1}, nil},
		{"exactly limit events", []int64{0, 1}, 2, []int64{0, 1}, nil},
		{"more events than limit", []int64{0, 1, 2}, 2, []int64{0, 1}, new(int64(2))},
		{"no events", nil, 5, []int64{}, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, err := collectPage(eventsWithSequences(tt.sequences, nil), tt.limit)
			if err != nil {
				t.Fatalf("collectPage returned error: %v", err)
			}

			got := []int64{}
			for _, e := range out.Events {
				got = append(got, e.Sequence)
			}
			if !reflect.DeepEqual(got, tt.wantSequences) {
				t.Errorf("sequences = %v, want %v", got, tt.wantSequences)
			}
			if !reflect.DeepEqual(out.Next, tt.wantNext) {
				t.Errorf("next = %v, want %v", deref(out.Next), deref(tt.wantNext))
			}
		})
	}
}

func TestCollectPageReturnsError(t *testing.T) {
	readErr := errors.New("connection lost")

	_, err := collectPage(eventsWithSequences([]int64{0}, readErr), 5)

	if !errors.Is(err, readErr) {
		t.Errorf("got error %v, want %v", err, readErr)
	}
}

// deref makes test failures print the value instead of a memory address.
func deref(p *int64) any {
	if p == nil {
		return nil
	}
	return *p
}

func TestFilterInputQuery(t *testing.T) {
	now := time.Date(2026, 11, 10, 12, 0, 0, 0, time.UTC)
	in := filterInput{
		Names: []string{"LoggedOut"},
		Tags:  map[string]string{"userId": "42"},
		Since: "24h",
	}

	q, err := in.query(now)
	if err != nil {
		t.Fatalf("query returned error: %v", err)
	}

	if len(q.Names) != 1 || !reflect.DeepEqual(q.Tags, []eventstore.Tag{{Key: "userId", Value: "42"}}) {
		t.Errorf("filters not passed on: %+v", q)
	}
	if want := now.Add(-24 * time.Hour); !q.Since.Equal(want) {
		t.Errorf("Since = %v, want %v", q.Since, want)
	}
	if !q.Until.IsZero() {
		t.Errorf("Until = %v, want zero", q.Until)
	}
}

func TestFilterInputQueryRejectsInvalidTime(t *testing.T) {
	if _, err := (filterInput{Since: "last week"}).query(time.Now()); err == nil {
		t.Error("query returned no error")
	}
}
