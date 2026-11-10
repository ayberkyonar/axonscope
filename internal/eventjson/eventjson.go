// Package eventjson defines the JSON form of events. The CLI and the MCP
// server both use it, so they always output the same shape.
package eventjson

import (
	"encoding/json"
	"time"

	"github.com/ayberkyonar/axonscope/internal/eventstore"
)

// Event is the JSON form of an eventstore.Event.
type Event struct {
	Sequence  int64             `json:"sequence"`
	ID        string            `json:"id"`
	Name      string            `json:"name"`
	Version   string            `json:"version,omitempty"`
	Timestamp string            `json:"timestamp" jsonschema:"RFC 3339 timestamp in UTC"`
	Tags      []Tag             `json:"tags"`
	Metadata  map[string]string `json:"metadata,omitempty"`
	Payload   any               `json:"payload" jsonschema:"the event payload: a JSON value, or a string if it is not JSON"`
}

// Tag is the JSON form of an eventstore.Tag.
type Tag struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// From converts an event to its JSON form.
func From(e eventstore.Event) Event {
	tags := make([]Tag, len(e.Tags))
	for i, t := range e.Tags {
		tags[i] = Tag{Key: t.Key, Value: t.Value}
	}

	return Event{
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

// decodePayload returns the payload as a JSON value, so it is embedded as
// real JSON instead of an escaped string. Payloads that are not JSON are
// returned as text.
func decodePayload(payload []byte) any {
	var v any
	if err := json.Unmarshal(payload, &v); err != nil {
		return string(payload)
	}
	return v
}
