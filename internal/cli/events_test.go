package cli

import (
	"encoding/json"
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

func TestWriteEventsText(t *testing.T) {
	var out strings.Builder
	if err := writeEventsText(&out, fakeEvents(testEvents, nil)); err != nil {
		t.Fatalf("writeEventsText returned error: %v", err)
	}

	want := "" +
		"0  2026-11-05 20:08:36  AccountRegistered  userId=42 email=a@b.nl\n" +
		"1  2026-11-05 20:09:58  LoggedOut          userId=42\n"
	if out.String() != want {
		t.Errorf("output mismatch\ngot:\n%s\nwant:\n%s", out.String(), want)
	}
}

func TestWriteEventsJSONWritesOneObjectPerLine(t *testing.T) {
	var out strings.Builder
	if err := writeEventsJSON(&out, fakeEvents(testEvents, nil)); err != nil {
		t.Fatalf("writeEventsJSON returned error: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	if len(lines) != len(testEvents) {
		t.Fatalf("got %d lines, want %d:\n%s", len(lines), len(testEvents), out.String())
	}
	for i, line := range lines {
		var e struct{ Sequence int64 }
		if err := json.Unmarshal([]byte(line), &e); err != nil {
			t.Errorf("line %d is not valid JSON: %v", i, err)
		} else if e.Sequence != testEvents[i].Sequence {
			t.Errorf("line %d has sequence %d, want %d", i, e.Sequence, testEvents[i].Sequence)
		}
	}
}

func TestFirstN(t *testing.T) {
	tests := []struct {
		n    int
		want int
	}{
		{0, 2}, // 0 means no limit
		{1, 1},
		{5, 2},
	}
	for _, tt := range tests {
		count := 0
		for range firstN(fakeEvents(testEvents, nil), tt.n) {
			count++
		}
		if count != tt.want {
			t.Errorf("firstN(2 events, %d) gave %d events, want %d", tt.n, count, tt.want)
		}
	}
}

func TestWriteEventsTextShowsEventsBeforeError(t *testing.T) {
	readErr := errors.New("connection lost")
	var out strings.Builder

	err := writeEventsText(&out, fakeEvents(testEvents[:1], readErr))

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
