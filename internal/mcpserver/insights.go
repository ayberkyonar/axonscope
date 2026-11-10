package mcpserver

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ayberkyonar/axonscope/internal/eventjson"
	"github.com/ayberkyonar/axonscope/internal/eventstore"
	"github.com/ayberkyonar/axonscope/internal/insight"
)

type trendInput struct {
	filterInput
	By       string `json:"by,omitempty" jsonschema:"period length: hour, day or week; default day"`
	TimeZone string `json:"timeZone,omitempty" jsonschema:"IANA time zone for day and week boundaries, like Europe/Amsterdam; default UTC"`
}

type statsInput struct {
	filterInput
	ByTag     string `json:"byTag,omitempty" jsonschema:"count per value of this tag key instead of per event type, like customerId"`
	Top       int    `json:"top,omitempty" jsonschema:"with byTag: how many of the most frequent values to return, default 10, at most 100"`
	MaxEvents int    `json:"maxEvents,omitempty" jsonschema:"with byTag: stop after this many events, default 10000, at most 100000"`
}

const (
	defaultTop = 10
	maxTop     = 100
	// maxMaxEvents caps how many events one question may read with their
	// tags, so an agent can't put unbounded load on the event store.
	maxMaxEvents = 100_000
)

func eventStats(ctx context.Context, store *eventstore.Store, in statsInput, now time.Time) (eventjson.Stats, error) {
	query, err := in.query(now)
	if err != nil {
		return eventjson.Stats{}, err
	}

	if in.ByTag != "" {
		top, maxEvents := limits(in.Top, in.MaxEvents)
		stats, err := insight.GroupByTag(store.Events(ctx, query), in.ByTag, top, maxEvents)
		if err != nil {
			return eventjson.Stats{}, err
		}
		return eventjson.FromTagStats(stats), nil
	}

	query.SkipTags = true // counting per name needs no tags
	stats, err := insight.Summarize(store.Events(ctx, query))
	if err != nil {
		return eventjson.Stats{}, err
	}
	return eventjson.FromStats(stats), nil
}

type journeysInput struct {
	filterInput
	ByTag        string `json:"byTag" jsonschema:"the tag that ties events into one journey, like orderId or userId"`
	Containing   string `json:"containing,omitempty" jsonschema:"only journeys with an event of this name, like OrderCancelled"`
	StartingWith string `json:"startingWith,omitempty" jsonschema:"only journeys whose first event has this name; leaves out journeys that began before 'since'"`
	Top          int    `json:"top,omitempty" jsonschema:"how many of the most common paths to return, default 10, at most 100"`
	MaxEvents    int    `json:"maxEvents,omitempty" jsonschema:"stop after this many events, default 10000, at most 100000"`
}

// journeyExamples is how many example tag values to return per path.
const journeyExamples = 3

func eventJourneys(ctx context.Context, store *eventstore.Store, in journeysInput, now time.Time) (eventjson.Journeys, error) {
	if in.ByTag == "" {
		return eventjson.Journeys{}, errors.New("byTag is required: the tag that ties events into one journey, like orderId")
	}
	query, err := in.query(now)
	if err != nil {
		return eventjson.Journeys{}, err
	}

	top, maxEvents := limits(in.Top, in.MaxEvents)
	journeys, err := insight.FindJourneys(store.Events(ctx, query), insight.JourneyOptions{
		Key: in.ByTag, Containing: in.Containing, StartingWith: in.StartingWith,
		Top: top, Examples: journeyExamples, MaxEvents: maxEvents,
	})
	if err != nil {
		return eventjson.Journeys{}, err
	}
	return eventjson.FromJourneys(journeys), nil
}

// limits returns how many results to return and how many events to read,
// using the defaults for zero and capping both.
func limits(top, maxEvents int) (int, int) {
	if top <= 0 {
		top = defaultTop
	}
	if maxEvents <= 0 {
		maxEvents = insight.DefaultMaxEvents
	}
	return min(top, maxTop), min(maxEvents, maxMaxEvents)
}

func eventTrend(ctx context.Context, store *eventstore.Store, in trendInput, now time.Time) (eventjson.Trend, error) {
	query, periods, err := in.trend(now)
	if err != nil {
		return eventjson.Trend{}, err
	}
	buckets, err := insight.Trend(store.Events(ctx, query), periods)
	if err != nil {
		return eventjson.Trend{}, err
	}
	return eventjson.FromTrend(periods.Unit, periods.Location, buckets), nil
}

// trend turns the input into a query and the periods to split it into.
// Dates in 'since' and 'until' are read in the trend's time zone, so they
// line up with its days. Without 'until', the trend runs up to now, so
// recent quiet periods show.
func (in trendInput) trend(now time.Time) (eventstore.Query, insight.Periods, error) {
	by := in.By
	if by == "" {
		by = "day"
	}
	unit, err := insight.ParseUnit(by)
	if err != nil {
		return eventstore.Query{}, insight.Periods{}, fmt.Errorf("by: %w", err)
	}

	loc := time.UTC
	if in.TimeZone != "" {
		if loc, err = time.LoadLocation(in.TimeZone); err != nil {
			return eventstore.Query{}, insight.Periods{}, fmt.Errorf("timeZone: unknown time zone %q: want an IANA name like Europe/Amsterdam", in.TimeZone)
		}
	}

	query, err := in.query(now.In(loc))
	if err != nil {
		return eventstore.Query{}, insight.Periods{}, err
	}
	query.SkipTags = true // counting needs no tags

	periods := insight.Periods{Unit: unit, Location: loc, From: query.Since, To: query.Until}
	if periods.To.IsZero() {
		periods.To = now
	}
	return query, periods, nil
}
