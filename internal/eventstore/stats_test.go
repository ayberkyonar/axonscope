package eventstore

import (
	"reflect"
	"testing"
	"time"
)

func TestByCount(t *testing.T) {
	got := byCount(map[string]int{"b.LoggedOut": 2, "a.LoginSucceeded": 5, "c.AccountRegistered": 2})

	want := []NameCount{
		{Name: "a.LoginSucceeded", Count: 5},
		{Name: "b.LoggedOut", Count: 2}, // same count: alphabetical
		{Name: "c.AccountRegistered", Count: 2},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("byCount = %v, want %v", got, want)
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
