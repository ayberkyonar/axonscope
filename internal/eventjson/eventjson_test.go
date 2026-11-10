package eventjson

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"

	"github.com/ayberkyonar/axonscope/internal/eventstore"
)

func TestFrom(t *testing.T) {
	e := eventstore.Event{
		Sequence:  12,
		ID:        "abc",
		Name:      "com.example.LoggedOut",
		Timestamp: time.Date(2026, 11, 5, 21, 0, 0, 0, time.FixedZone("CET", 3600)),
		Tags:      []eventstore.Tag{{Key: "userId", Value: "42"}},
		Payload:   []byte(`{"userId":"42"}`),
	}

	got, err := json.Marshal(From(e))
	if err != nil {
		t.Fatal(err)
	}

	// The timestamp is converted to UTC, and the payload is embedded as JSON.
	want := `{"sequence":12,"id":"abc","name":"com.example.LoggedOut",` +
		`"timestamp":"2026-11-05T20:00:00Z","tags":[{"key":"userId","value":"42"}],` +
		`"payload":{"userId":"42"}}`
	if string(got) != want {
		t.Errorf("got  %s\nwant %s", got, want)
	}
}

func TestDecodePayload(t *testing.T) {
	tests := []struct {
		payload string
		want    any
	}{
		{`{"userId":"42"}`, map[string]any{"userId": "42"}},
		{`not json`, "not json"},
		{``, ""},
	}
	for _, tt := range tests {
		if got := decodePayload([]byte(tt.payload)); !reflect.DeepEqual(got, tt.want) {
			t.Errorf("decodePayload(%q) = %#v, want %#v", tt.payload, got, tt.want)
		}
	}
}
