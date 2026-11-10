package cli

import (
	"errors"
	"iter"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/ayberkyonar/axonscope/internal/eventstore"
)

// fakeEvents returns an iterator over events, followed by err if it is not nil.
func fakeEvents(events []eventstore.Event, err error) iter.Seq2[eventstore.Event, error] {
	return func(yield func(eventstore.Event, error) bool) {
		for _, e := range events {
			if !yield(e, nil) {
				return
			}
		}
		if err != nil {
			yield(eventstore.Event{}, err)
		}
	}
}

var testEvents = []eventstore.Event{
	{
		Sequence:  0,
		Name:      "com.example.AccountRegistered",
		Timestamp: time.Date(2026, 11, 5, 20, 8, 36, 0, time.Local),
		Tags:      []eventstore.Tag{{Key: "userId", Value: "42"}, {Key: "email", Value: "a@b.nl"}},
	},
	{
		Sequence:  1,
		Name:      "com.example.LoggedOut",
		Timestamp: time.Date(2026, 11, 5, 20, 9, 58, 0, time.Local),
		Tags:      []eventstore.Tag{{Key: "userId", Value: "42"}},
	},
}

func TestPrintEvents(t *testing.T) {
	var out strings.Builder
	if err := printEvents(&out, fakeEvents(testEvents, nil), 0); err != nil {
		t.Fatalf("printEvents returned error: %v", err)
	}

	want := "" +
		"0  2026-11-05 20:08:36  AccountRegistered  userId=42 email=a@b.nl\n" +
		"1  2026-11-05 20:09:58  LoggedOut          userId=42\n"
	if out.String() != want {
		t.Errorf("output mismatch\ngot:\n%s\nwant:\n%s", out.String(), want)
	}
}

func TestPrintEventsStopsAtLimit(t *testing.T) {
	var out strings.Builder
	if err := printEvents(&out, fakeEvents(testEvents, nil), 1); err != nil {
		t.Fatalf("printEvents returned error: %v", err)
	}

	if lines := strings.Count(out.String(), "\n"); lines != 1 {
		t.Errorf("got %d lines, want 1:\n%s", lines, out.String())
	}
}

func TestPrintEventsShowsEventsBeforeError(t *testing.T) {
	readErr := errors.New("connection lost")
	var out strings.Builder

	err := printEvents(&out, fakeEvents(testEvents[:1], readErr), 0)

	if !errors.Is(err, readErr) {
		t.Errorf("got error %v, want %v", err, readErr)
	}
	if !strings.Contains(out.String(), "AccountRegistered") {
		t.Errorf("event read before the error is missing:\n%s", out.String())
	}
}

func TestParseTags(t *testing.T) {
	got, err := parseTags([]string{"userId=42", "email=a=b@c.nl", "empty="})
	if err != nil {
		t.Fatalf("parseTags returned error: %v", err)
	}

	want := []eventstore.Tag{
		{Key: "userId", Value: "42"},
		{Key: "email", Value: "a=b@c.nl"}, // only the first "=" separates key and value
		{Key: "empty", Value: ""},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("parseTags = %v, want %v", got, want)
	}
}

func TestParseTagsRejectsInvalid(t *testing.T) {
	for _, raw := range []string{"userId", "=42"} {
		if _, err := parseTags([]string{raw}); err == nil {
			t.Errorf("parseTags(%q) returned no error", raw)
		}
	}
}
