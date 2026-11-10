package eventstore

import (
	"testing"

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
