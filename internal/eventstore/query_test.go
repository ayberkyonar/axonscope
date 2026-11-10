package eventstore

import (
	"testing"
	"time"

	"google.golang.org/protobuf/proto"

	"github.com/ayberkyonar/axonscope/internal/eventstore/internal/axonpb"
)

const fullName = "com.example.events.AccountRegistered"

func TestQueryRequest(t *testing.T) {
	userTag := &axonpb.Tag{Key: []byte("userId"), Value: []byte("42")}

	tests := []struct {
		name  string
		query Query
		want  *axonpb.SourceEventsRequest
	}{
		{
			name:  "zero value selects everything",
			query: Query{},
			want:  &axonpb.SourceEventsRequest{},
		},
		{
			name:  "from is passed on",
			query: Query{From: 7},
			want:  &axonpb.SourceEventsRequest{FromSequence: 7},
		},
		{
			name:  "full names and tags are filtered by the server",
			query: Query{Names: []string{fullName}, Tags: []Tag{{Key: "userId", Value: "42"}}},
			want: &axonpb.SourceEventsRequest{Criterion: []*axonpb.Criterion{{
				TagsAndNames: &axonpb.TagsAndNamesCriterion{Name: []string{fullName}, Tag: []*axonpb.Tag{userTag}},
			}}},
		},
		{
			name:  "short names are left out of the request",
			query: Query{Names: []string{fullName, "LoggedOut"}, Tags: []Tag{{Key: "userId", Value: "42"}}},
			want: &axonpb.SourceEventsRequest{Criterion: []*axonpb.Criterion{{
				TagsAndNames: &axonpb.TagsAndNamesCriterion{Tag: []*axonpb.Tag{userTag}},
			}}},
		},
		{
			name:  "only short names means no criterion at all",
			query: Query{Names: []string{"LoggedOut"}},
			want:  &axonpb.SourceEventsRequest{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Protobuf messages hold internal state, so compare them with proto.Equal.
			if got := tt.query.request(); !proto.Equal(got, tt.want) {
				t.Errorf("request() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestQueryMatchesName(t *testing.T) {
	tests := []struct {
		names []string
		want  bool
	}{
		{nil, true},
		{[]string{fullName}, true},
		{[]string{"AccountRegistered"}, true},
		{[]string{"LoggedOut", "AccountRegistered"}, true},
		{[]string{"LoggedOut"}, false},
		{[]string{"Registered"}, false},
	}
	for _, tt := range tests {
		if got := (Query{Names: tt.names}).matchesName(fullName); got != tt.want {
			t.Errorf("Query{Names: %q}.matchesName(%q) = %v, want %v", tt.names, fullName, got, tt.want)
		}
	}
}

func TestEventShortName(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{fullName, "AccountRegistered"},
		{"AccountRegistered", "AccountRegistered"},
		{"", ""},
	}
	for _, tt := range tests {
		if got := (Event{Name: tt.name}).ShortName(); got != tt.want {
			t.Errorf("Event{Name: %q}.ShortName() = %q, want %q", tt.name, got, tt.want)
		}
	}
}

func TestQueryTimeBounds(t *testing.T) {
	at := func(hour int) time.Time { return time.Date(2026, 11, 5, hour, 0, 0, 0, time.UTC) }
	q := Query{Since: at(10), Until: at(12)}

	tests := []struct {
		t           time.Time
		beforeSince bool
		pastUntil   bool
		description string
	}{
		{at(9), true, false, "before the range"},
		{at(10), false, false, "since is inclusive"},
		{at(11), false, false, "inside the range"},
		{at(12), false, true, "until is exclusive"},
		{at(13), false, true, "after the range"},
	}
	for _, tt := range tests {
		if got := q.beforeSince(tt.t); got != tt.beforeSince {
			t.Errorf("%s: beforeSince = %v, want %v", tt.description, got, tt.beforeSince)
		}
		if got := q.pastUntil(tt.t); got != tt.pastUntil {
			t.Errorf("%s: pastUntil = %v, want %v", tt.description, got, tt.pastUntil)
		}
	}

	var unbounded Query
	if unbounded.beforeSince(at(0)) || unbounded.pastUntil(at(23)) {
		t.Error("a zero Query must not exclude any time")
	}
}
