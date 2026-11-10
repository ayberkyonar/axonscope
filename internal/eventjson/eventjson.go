// Package eventjson defines the JSON form of events and event statistics.
// The CLI and the MCP server both use it, so they always output the same shape.
package eventjson

import (
	"encoding/json"
	"time"

	"github.com/ayberkyonar/axonscope/internal/eventstore"
	"github.com/ayberkyonar/axonscope/internal/insight"
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

// Stats is the JSON form of insight.Stats, or of insight.TagStats when
// counting per tag value: then ByTag is set instead of ByName.
type Stats struct {
	Total  int         `json:"total"`
	First  string      `json:"first,omitempty" jsonschema:"timestamp of the oldest counted event; absent when there are none"`
	Last   string      `json:"last,omitempty" jsonschema:"timestamp of the newest counted event; absent when there are none"`
	ByName []NameCount `json:"byName,omitempty" jsonschema:"number of events per name, most frequent first"`
	ByTag  *TagGroups  `json:"byTag,omitempty" jsonschema:"number of events per value of one tag, when counting by tag"`
}

// TagGroups is the JSON form of the per-value counts in insight.TagStats.
type TagGroups struct {
	Key      string     `json:"key"`
	Distinct int        `json:"distinct" jsonschema:"number of different values of the tag"`
	Top      []TagCount `json:"top" jsonschema:"the most frequent values, most frequent first"`
	Untagged int        `json:"untagged" jsonschema:"number of counted events without this tag"`
	Partial  bool       `json:"partial" jsonschema:"true when only the first 'total' events were read because of the event limit: counts are incomplete, narrow the selection for a full answer"`
}

// TagCount is the JSON form of insight.TagCount.
type TagCount struct {
	Value string `json:"value"`
	Count int    `json:"count"`
}

// FromTagStats converts per-tag counts to their JSON form.
func FromTagStats(s insight.TagStats) Stats {
	groups := &TagGroups{Key: s.Key, Distinct: s.Distinct, Top: make([]TagCount, len(s.Top)), Untagged: s.Untagged, Partial: s.Partial}
	for i, tc := range s.Top {
		groups.Top[i] = TagCount{Value: tc.Value, Count: tc.Count}
	}
	out := Stats{Total: s.Total, ByTag: groups}
	if s.Total > 0 {
		out.First = formatTime(s.First)
		out.Last = formatTime(s.Last)
	}
	return out
}

// NameCount is the JSON form of insight.NameCount.
type NameCount struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

// FromStats converts stats to their JSON form.
func FromStats(s insight.Stats) Stats {
	out := Stats{Total: s.Total, ByName: fromNameCounts(s.ByName)}
	if s.Total > 0 {
		out.First = formatTime(s.First)
		out.Last = formatTime(s.Last)
	}
	return out
}

// Trend is the JSON form of a trend.
type Trend struct {
	Unit     string        `json:"unit" jsonschema:"length of each period: hour, day or week"`
	TimeZone string        `json:"timeZone" jsonschema:"time zone whose calendar sets the period boundaries"`
	Periods  []TrendPeriod `json:"periods" jsonschema:"one entry per period, oldest first, including periods without events"`
}

// TrendPeriod is the JSON form of insight.Bucket.
type TrendPeriod struct {
	Start  string      `json:"start" jsonschema:"start of the period, RFC 3339 in the trend's time zone"`
	Total  int         `json:"total"`
	ByName []NameCount `json:"byName,omitempty" jsonschema:"events per name in this period, most frequent first"`
}

// FromTrend converts a trend to its JSON form. Period starts keep their
// time zone, so a day reads as local midnight, like 2026-11-05T00:00:00+01:00.
func FromTrend(unit insight.Unit, loc *time.Location, buckets []insight.Bucket) Trend {
	out := Trend{Unit: unit.String(), TimeZone: loc.String(), Periods: make([]TrendPeriod, len(buckets))}
	for i, b := range buckets {
		out.Periods[i] = TrendPeriod{Start: b.Start.Format(time.RFC3339), Total: b.Total, ByName: fromNameCounts(b.ByName)}
	}
	return out
}

func fromNameCounts(list []insight.NameCount) []NameCount {
	out := make([]NameCount, len(list))
	for i, nc := range list {
		out[i] = NameCount{Name: nc.Name, Count: nc.Count}
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
