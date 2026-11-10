// Package eventjson defines the JSON form of events and event statistics.
// The CLI and the MCP server both use it, so they always output the same shape.
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
		Timestamp: formatTime(e.Timestamp),
		Tags:      tags,
		Metadata:  e.Metadata,
		Payload:   decodePayload(e.Payload),
	}
}

// Stats is the JSON form of eventstore.Stats.
type Stats struct {
	Total  int         `json:"total"`
	First  string      `json:"first,omitempty" jsonschema:"timestamp of the oldest counted event; absent when there are none"`
	Last   string      `json:"last,omitempty" jsonschema:"timestamp of the newest counted event; absent when there are none"`
	ByName []NameCount `json:"byName" jsonschema:"number of events per name, most frequent first"`
}

// NameCount is the JSON form of eventstore.NameCount.
type NameCount struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

// FromStats converts stats to their JSON form.
func FromStats(s eventstore.Stats) Stats {
	out := Stats{Total: s.Total, ByName: make([]NameCount, len(s.ByName))}
	if s.Total > 0 {
		out.First = formatTime(s.First)
		out.Last = formatTime(s.Last)
	}
	for i, nc := range s.ByName {
		out.ByName[i] = NameCount{Name: nc.Name, Count: nc.Count}
	}
	return out
}

func formatTime(t time.Time) string {
	return t.UTC().Format(time.RFC3339Nano)
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
